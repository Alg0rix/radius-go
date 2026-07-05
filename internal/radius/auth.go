package radius

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"fmt"
	"math"
	"net"
	"time"

	"github.com/Alg0rix/radius-go/internal/domain"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"

	"golang.org/x/crypto/bcrypt"
)

// RADIUS attribute type 80 (Message-Authenticator).
const attrMessageAuthenticator radius.Type = 80

func (s *Service) handleAuth(w radius.ResponseWriter, r *radius.Request) {
	s.incAuthRequests()

	username := rfc2865.UserName_GetString(r.Packet)

	passwords, err := rfc2865.UserPassword_GetStrings(r.Packet)
	if err != nil || len(passwords) == 0 {
		s.rejectAuth(w, r)
		return
	}
	password := passwords[0]

	s.mu.RLock()
	user, ok := s.subscribers[username]
	s.mu.RUnlock()

	if !ok || !user.Enabled || !passwordMatches(user.PasswordHash, password) {
		s.rejectAuth(w, r)
		return
	}

	// --- voucher expiry checks ---
	if user.IsVoucher {
		// First login: record timestamp and calculate calendar expiry.
		if user.FirstLoginAt == nil {
			go s.repo.RecordFirstLogin(context.Background(), user.ID,
				domain.TimeLimitType(user.VoucherTimeLimitType), user.VoucherTimeLimitSeconds)
			now := time.Now()
			user.FirstLoginAt = &now
			s.mu.Lock()
			if stored, ok := s.subscribers[username]; ok {
				stored.FirstLoginAt = &now
				if user.VoucherTimeLimitType == string(domain.TimeLimitCalendar) {
					exp := now.Add(time.Duration(user.VoucherTimeLimitSeconds) * time.Second)
					stored.ExpiresAt = &exp
				}
				s.subscribers[username] = stored
			}
			s.mu.Unlock()
		}

		now := time.Now()

		// Calendar expiry: reject if expired.
		if user.ExpiresAt != nil && now.After(*user.ExpiresAt) {
			s.rejectAndDisableVoucher(username, user.ID)
			s.rejectAuth(w, r)
			return
		}

		// Usage-based time limit: reject if used >= limit.
		if user.VoucherTimeLimitType == string(domain.TimeLimitUsage) &&
			user.VoucherTimeLimitSeconds > 0 &&
			user.UsageSecondsUsed >= user.VoucherTimeLimitSeconds {
			s.rejectAndDisableVoucher(username, user.ID)
			s.rejectAuth(w, r)
			return
		}

		// Data cap: reject if already exceeded.
		if user.VoucherDataCapBytes > 0 && user.DataBytesUsed >= user.VoucherDataCapBytes {
			s.rejectAndDisableVoucher(username, user.ID)
			s.rejectAuth(w, r)
			return
		}
	}

	// --- simultaneous use ---
	if user.SimultaneousUse > 0 {
		s.mu.RLock()
		count := 0
		for _, sess := range s.sessions {
			if sess.Username == username && sess.SessionStatus == domain.SessionStateActive {
				count++
			}
		}
		s.mu.RUnlock()
		if count >= user.SimultaneousUse {
			s.rejectAuth(w, r)
			return
		}
	}

	resp := r.Response(radius.CodeAccessAccept)

	// Service-Type
	if user.ServiceType == domain.ServiceTypeLogin {
		rfc2865.ServiceType_Add(resp, rfc2865.ServiceType_Value_LoginUser)
	} else {
		rfc2865.ServiceType_Add(resp, rfc2865.ServiceType_Value_FramedUser)
	}

	rateLimit := effectiveRateLimit(user)
	bandwidthUp, bandwidthDown := effectiveBandwidth(user)
	maxTotalOctets := effectiveMaxTotalOctets(user)
	sessionTimeout := effectiveSessionTimeout(user)
	idleTimeout := effectiveIdleTimeout(user)

	// PPP-layer attributes for PPPoE profiles.
	if user.ServiceType == domain.ServiceTypeLogin && user.PPPoEProfile != nil {
		profile := user.PPPoEProfile
		FramedProtocol_AddPPP(resp)
		if profile.FramedIPPool != "" {
			FramedPool_SetString(resp, profile.FramedIPPool)
			MikrotikAddressPool_SetString(resp, profile.FramedIPPool)
		} else if user.FramedIP != "" {
			if ip := net.ParseIP(user.FramedIP); ip != nil {
				rfc2865.FramedIPAddress_Add(resp, ip)
			}
		}
		if profile.FramedIPNetmask != "" {
			FramedIPNetmask_Add(resp, profile.FramedIPNetmask)
		}
		if profile.MTU > 0 {
			FramedMTU_Set(resp, profile.MTU)
		}
		if profile.PPPCompression {
			FramedCompression_AddStac(resp)
		}
		emitDNS(resp, profile.PrimaryDNS, profile.SecondaryDNS)
	} else {
		if user.FramedIP != "" {
			if ip := net.ParseIP(user.FramedIP); ip != nil {
				rfc2865.FramedIPAddress_Add(resp, ip)
			}
		}
	}

	// Hotspot voucher package extras.
	if user.ServiceType == domain.ServiceTypeFramed && user.VoucherPackage != nil {
		pkg := user.VoucherPackage
		if pkg.AddressPool != "" {
			FramedPool_SetString(resp, pkg.AddressPool)
			MikrotikAddressPool_SetString(resp, pkg.AddressPool)
		}
		emitDNS(resp, pkg.PrimaryDNS, pkg.SecondaryDNS)
	}

	if sessionTimeout > 0 {
		rfc2865.SessionTimeout_Add(resp, rfc2865.SessionTimeout(sessionTimeout))
	}
	if idleTimeout > 0 {
		rfc2865.IdleTimeout_Add(resp, rfc2865.IdleTimeout(idleTimeout))
	}

	if rateLimit != "" {
		MikrotikRateLimit_SetString(resp, rateLimit)
	}
	if user.MikrotikGroup != "" {
		MikrotikGroup_SetString(resp, user.MikrotikGroup)
	}

	if bandwidthUp > 0 {
		PfSenseBandwidthMaxUp_Set(resp, bandwidthUp)
	}
	if bandwidthDown > 0 {
		PfSenseBandwidthMaxDown_Set(resp, bandwidthDown)
	}
	if maxTotalOctets > 0 {
		PfSenseMaxTotalOctets_Set(resp, maxTotalOctets)
		MikrotikTotalLimit_Set(resp, maxTotalOctets)
	}

	// Session-Timeout for near-expiry vouchers (usage-based).
	if user.IsVoucher && user.VoucherTimeLimitType == string(domain.TimeLimitUsage) &&
		user.VoucherTimeLimitSeconds > 0 {
		remaining := user.VoucherTimeLimitSeconds - user.UsageSecondsUsed
		if remaining > 0 {
			rfc2865.SessionTimeout_Add(resp, rfc2865.SessionTimeout(remaining))
		}
	}
	// Session-Timeout for calendar vouchers: remaining until expiry.
	if user.IsVoucher && user.VoucherTimeLimitType == string(domain.TimeLimitCalendar) &&
		user.ExpiresAt != nil {
		remaining := int(time.Until(*user.ExpiresAt).Seconds())
		if remaining > 0 {
			rfc2865.SessionTimeout_Add(resp, rfc2865.SessionTimeout(remaining))
		}
	}

	if err := addMessageAuthenticator(resp); err != nil {
		s.deps.Logger.Error().Err(err).Msg("message authenticator failed")
		s.rejectAuth(w, r)
		return
	}

	s.incAuthAccepts()
	w.Write(resp)
}

func (s *Service) rejectAuth(w radius.ResponseWriter, r *radius.Request) {
	s.incAuthRejects()
	resp := r.Response(radius.CodeAccessReject)
	addMessageAuthenticator(resp)
	w.Write(resp)
}

// rejectAndDisableVoucher marks a voucher as disabled so future auths are rejected quickly.
func (s *Service) rejectAndDisableVoucher(username, userID string) {
	s.mu.Lock()
	if u, ok := s.subscribers[username]; ok {
		u.Enabled = false
		s.subscribers[username] = u
	}
	s.mu.Unlock()
	go s.repo.DisableUser(context.Background(), userID)
}

// formatRateLimit builds a MikroTik rate-limit string from per-user kbps fields.
func formatRateLimit(u domain.RadiusUser) string {
	if u.SpeedUploadKbps <= 0 && u.SpeedDownloadKbps <= 0 {
		return ""
	}
	up := formatKbps(u.SpeedUploadKbps)
	down := formatKbps(u.SpeedDownloadKbps)
	return fmt.Sprintf("%s/%s", up, down)
}

func formatBandwidthKbps(up, down int) string {
	if up <= 0 && down <= 0 {
		return ""
	}
	return fmt.Sprintf("%s/%s", formatKbps(up), formatKbps(down))
}

func effectiveRateLimit(user domain.RadiusUser) string {
	if user.RateLimit != "" {
		return user.RateLimit
	}
	if rl := formatRateLimit(user); rl != "" {
		return rl
	}
	if user.PPPoEProfile != nil {
		if user.PPPoEProfile.RateLimit != "" {
			return user.PPPoEProfile.RateLimit
		}
		if rl := formatBandwidthKbps(user.PPPoEProfile.BandwidthMaxUp, user.PPPoEProfile.BandwidthMaxDown); rl != "" {
			return rl
		}
	}
	return ""
}

func effectiveBandwidth(user domain.RadiusUser) (uint32, uint32) {
	up, down := user.BandwidthMaxUp, user.BandwidthMaxDown
	if user.PPPoEProfile != nil {
		if up == 0 && user.PPPoEProfile.BandwidthMaxUp > 0 {
			up = uint32(user.PPPoEProfile.BandwidthMaxUp)
		}
		if down == 0 && user.PPPoEProfile.BandwidthMaxDown > 0 {
			down = uint32(user.PPPoEProfile.BandwidthMaxDown)
		}
	}
	return up, down
}

func effectiveMaxTotalOctets(user domain.RadiusUser) uint32 {
	if user.MaxTotalOctets > 0 {
		return user.MaxTotalOctets
	}
	if user.PPPoEProfile != nil && user.PPPoEProfile.MaxTotalOctets > 0 {
		// RADIUS attributes are 32-bit; Acct-Input/Output-Gigawords would be needed to exceed 4 GiB.
		if user.PPPoEProfile.MaxTotalOctets > math.MaxUint32 {
			return math.MaxUint32
		}
		return uint32(user.PPPoEProfile.MaxTotalOctets)
	}
	return 0
}

func effectiveSessionTimeout(user domain.RadiusUser) int {
	if user.SessionTimeout > 0 {
		return user.SessionTimeout
	}
	if user.PPPoEProfile != nil && user.PPPoEProfile.SessionTimeout > 0 {
		return user.PPPoEProfile.SessionTimeout
	}
	return 0
}

func effectiveIdleTimeout(user domain.RadiusUser) int {
	if user.IdleTimeout > 0 {
		return user.IdleTimeout
	}
	if user.PPPoEProfile != nil && user.PPPoEProfile.IdleTimeout > 0 {
		return user.PPPoEProfile.IdleTimeout
	}
	return 0
}

func formatKbps(kbps int) string {
	if kbps <= 0 {
		return "0K"
	}
	if kbps >= 1000000 {
		return fmt.Sprintf("%dM", kbps/1000)
	}
	return fmt.Sprintf("%dK", kbps)
}

// passwordMatches compares a bcrypt hash against a plaintext password.
func passwordMatches(hash, supplied string) bool {
	if supplied == "" || hash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(supplied)) == nil
}

// addMessageAuthenticator computes and sets the HMAC-MD5 Message-Authenticator
// (attribute type 80) on the response packet.
func addMessageAuthenticator(p *radius.Packet) error {
	p.Add(attrMessageAuthenticator, make(radius.Attribute, 16))

	encoded, err := p.MarshalBinary()
	if err != nil {
		return err
	}

	mac := hmac.New(md5.New, p.Secret)
	mac.Write(encoded)
	hash := mac.Sum(nil)

	for _, avp := range p.Attributes {
		if avp.Type == attrMessageAuthenticator {
			copy(avp.Attribute, hash[:16])
			break
		}
	}
	return nil
}

func timePtr() *time.Time {
	t := time.Now()
	return &t
}