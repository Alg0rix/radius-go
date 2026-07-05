package radius

import (
	"errors"
	"net"
	"net/mail"
	"strings"

	"github.com/google/uuid"
	"github.com/Alg0rix/radius-go/internal/domain"
)

const (
	maxNameLen       = 255
	maxUsernameLen   = 128
	maxPasswordLen   = 128
	maxSecretLen     = 512
	maxIPLen         = 45
	maxRateLimitLen  = 64
	maxGroupLen      = 128
	maxEmailLen      = 255
	maxDescriptionLen = 512
)

func validUUID(s string) bool {
	_, err := uuid.Parse(s)
	return err == nil
}

func validIP(s string) bool {
	if s == "" {
		return false
	}
	if net.ParseIP(s) != nil {
		return true
	}
	host, _, err := net.SplitHostPort(s)
	return err == nil && net.ParseIP(host) != nil
}

func validEmail(s string) bool {
	if s == "" {
		return true
	}
	addr, err := mail.ParseAddress(s)
	return err == nil && addr.Address == s && len(s) <= maxEmailLen
}

func validServiceType(s string) bool {
	return s == "" || s == string(domain.ServiceTypeFramed) || s == string(domain.ServiceTypeLogin)
}

func validTimeLimitType(s string) bool {
	return s == "" || s == string(domain.TimeLimitCalendar) || s == string(domain.TimeLimitUsage)
}

func validPPPoEProfileName(s string) bool {
	return nonEmpty(s) && len(s) <= maxNameLen
}

func validIPv4(s string) bool {
	if s == "" {
		return true
	}
	ip := net.ParseIP(s)
	if ip == nil {
		return false
	}
	return ip.To4() != nil
}

func validatePPPoEProfile(req domain.CreatePPPoEProfileRequest) error {
	if !validPPPoEProfileName(req.Name) {
		return errors.New("invalid name")
	}
	if !validIPv4(req.PrimaryDNS) {
		return errors.New("invalid primary_dns")
	}
	if !validIPv4(req.SecondaryDNS) {
		return errors.New("invalid secondary_dns")
	}
	if !validIPv4(req.FramedIPNetmask) {
		return errors.New("invalid framed_ip_netmask")
	}
	if req.MTU < 0 || req.MTU > 1500 {
		return errors.New("invalid mtu")
	}
	if req.MRU < 0 || req.MRU > 1500 {
		return errors.New("invalid mru")
	}
	if req.SessionTimeout < 0 {
		return errors.New("invalid session_timeout")
	}
	if req.IdleTimeout < 0 {
		return errors.New("invalid idle_timeout")
	}
	if req.KeepaliveInterval < 0 {
		return errors.New("invalid keepalive_interval")
	}
	if req.BandwidthMaxUp < 0 || req.BandwidthMaxDown < 0 {
		return errors.New("invalid bandwidth")
	}
	if req.MaxTotalOctets < 0 {
		return errors.New("invalid max_total_octets")
	}
	return nil
}

func validateUpdatePPPoEProfile(req domain.UpdatePPPoEProfileRequest) error {
	if req.Name != nil && !validPPPoEProfileName(*req.Name) {
		return errors.New("invalid name")
	}
	if req.PrimaryDNS != nil && !validIPv4(*req.PrimaryDNS) {
		return errors.New("invalid primary_dns")
	}
	if req.SecondaryDNS != nil && !validIPv4(*req.SecondaryDNS) {
		return errors.New("invalid secondary_dns")
	}
	if req.FramedIPNetmask != nil && !validIPv4(*req.FramedIPNetmask) {
		return errors.New("invalid framed_ip_netmask")
	}
	if req.MTU != nil && (*req.MTU < 0 || *req.MTU > 1500) {
		return errors.New("invalid mtu")
	}
	if req.MRU != nil && (*req.MRU < 0 || *req.MRU > 1500) {
		return errors.New("invalid mru")
	}
	if req.SessionTimeout != nil && *req.SessionTimeout < 0 {
		return errors.New("invalid session_timeout")
	}
	if req.IdleTimeout != nil && *req.IdleTimeout < 0 {
		return errors.New("invalid idle_timeout")
	}
	if req.KeepaliveInterval != nil && *req.KeepaliveInterval < 0 {
		return errors.New("invalid keepalive_interval")
	}
	if req.BandwidthMaxUp != nil && *req.BandwidthMaxUp < 0 {
		return errors.New("invalid bandwidth_max_up")
	}
	if req.BandwidthMaxDown != nil && *req.BandwidthMaxDown < 0 {
		return errors.New("invalid bandwidth_max_down")
	}
	if req.MaxTotalOctets != nil && *req.MaxTotalOctets < 0 {
		return errors.New("invalid max_total_octets")
	}
	return nil
}

func validateMutualExclusion(voucherPackageID, pppoeProfileID *string) error {
	hasVoucher := voucherPackageID != nil && *voucherPackageID != ""
	hasPPPoE := pppoeProfileID != nil && *pppoeProfileID != ""
	if hasVoucher && hasPPPoE {
		return errors.New("voucher_package_id and pppoe_profile_id are mutually exclusive")
	}
	return nil
}

func limitString(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

func nonEmpty(s string) bool { return strings.TrimSpace(s) != "" }
