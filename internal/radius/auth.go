package radius

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"fmt"
	"net"
	"time"

	"github.com/your-org/radius-go/internal/domain"

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

	if user.SessionTimeout > 0 {
		rfc2865.SessionTimeout_Add(resp, rfc2865.SessionTimeout(user.SessionTimeout))
	}
	if user.IdleTimeout > 0 {
		rfc2865.IdleTimeout_Add(resp, rfc2865.IdleTimeout(user.IdleTimeout))
	}
	if user.FramedIP != "" {
		if ip := net.ParseIP(user.FramedIP); ip != nil {
			rfc2865.FramedIPAddress_Add(resp, ip)
		}
	}

	// MikroTik rate-limit from per-user speed fields.
	rateLimit := formatRateLimit(user)
	if rateLimit != "" {
		MikrotikRateLimit_SetString(resp, rateLimit)
	} else if user.RateLimit != "" {
		MikrotikRateLimit_SetString(resp, user.RateLimit)
	}
	if user.MikrotikGroup != "" {
		MikrotikGroup_SetString(resp, user.MikrotikGroup)
	}

	// pfSense/OPNsense VSAs.
	if user.BandwidthMaxUp > 0 {
		PfSenseBandwidthMaxUp_Set(resp, user.BandwidthMaxUp)
	}
	if user.BandwidthMaxDown > 0 {
		PfSenseBandwidthMaxDown_Set(resp, user.BandwidthMaxDown)
	}
	if user.MaxTotalOctets > 0 {
		PfSenseMaxTotalOctets_Set(resp, user.MaxTotalOctets)
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