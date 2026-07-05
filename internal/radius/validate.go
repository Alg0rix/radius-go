package radius

import (
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

func limitString(s string, max int) string {
	if len(s) > max {
		return s[:max]
	}
	return s
}

func nonEmpty(s string) bool { return strings.TrimSpace(s) != "" }
