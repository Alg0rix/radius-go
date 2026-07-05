package radius

import (
	"testing"

	"github.com/Alg0rix/radius-go/internal/domain"
)

func ptr[T any](v T) *T { return &v }

func TestValidIPv4(t *testing.T) {
	tests := []struct {
		name string
		ip   string
		want bool
	}{
		{"empty", "", true},
		{"valid ipv4", "1.2.3.4", true},
		{"valid netmask", "255.255.255.0", true},
		{"ipv6", "::1", false},
		{"invalid", "not-an-ip", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := validIPv4(tt.ip)
			if got != tt.want {
				t.Errorf("validIPv4(%q) = %v, want %v", tt.ip, got, tt.want)
			}
		})
	}
}

func TestValidateMutualExclusion(t *testing.T) {
	tests := []struct {
		name    string
		voucher *string
		pppoe   *string
		wantErr bool
	}{
		{"both nil", nil, nil, false},
		{"only voucher", ptr("vp-1"), nil, false},
		{"only pppoe", nil, ptr("pp-1"), false},
		{"both empty", ptr(""), ptr(""), false},
		{"voucher empty pppoe set", ptr(""), ptr("pp-1"), false},
		{"both set", ptr("vp-1"), ptr("pp-1"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateMutualExclusion(tt.voucher, tt.pppoe)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMutualExclusion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidatePPPoEProfile(t *testing.T) {
	base := domain.CreatePPPoEProfileRequest{Name: "Test"}

	tests := []struct {
		name    string
		req     domain.CreatePPPoEProfileRequest
		wantErr bool
	}{
		{"valid base", base, false},
		{"empty name", domain.CreatePPPoEProfileRequest{}, true},
		{"invalid primary dns", func(r domain.CreatePPPoEProfileRequest) domain.CreatePPPoEProfileRequest { r.PrimaryDNS = "bad"; return r }(base), true},
		{"invalid secondary dns", func(r domain.CreatePPPoEProfileRequest) domain.CreatePPPoEProfileRequest { r.SecondaryDNS = "bad"; return r }(base), true},
		{"invalid netmask", func(r domain.CreatePPPoEProfileRequest) domain.CreatePPPoEProfileRequest { r.FramedIPNetmask = "bad"; return r }(base), true},
		{"mtu too high", func(r domain.CreatePPPoEProfileRequest) domain.CreatePPPoEProfileRequest { r.MTU = 1501; return r }(base), true},
		{"mtu negative", func(r domain.CreatePPPoEProfileRequest) domain.CreatePPPoEProfileRequest { r.MTU = -1; return r }(base), true},
		{"session timeout negative", func(r domain.CreatePPPoEProfileRequest) domain.CreatePPPoEProfileRequest { r.SessionTimeout = -1; return r }(base), true},
		{"idle timeout negative", func(r domain.CreatePPPoEProfileRequest) domain.CreatePPPoEProfileRequest { r.IdleTimeout = -1; return r }(base), true},
		{"bandwidth negative", func(r domain.CreatePPPoEProfileRequest) domain.CreatePPPoEProfileRequest { r.BandwidthMaxUp = -1; return r }(base), true},
		{"max octets negative", func(r domain.CreatePPPoEProfileRequest) domain.CreatePPPoEProfileRequest { r.MaxTotalOctets = -1; return r }(base), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePPPoEProfile(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePPPoEProfile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateUpdatePPPoEProfile(t *testing.T) {
	tests := []struct {
		name    string
		req     domain.UpdatePPPoEProfileRequest
		wantErr bool
	}{
		{"empty", domain.UpdatePPPoEProfileRequest{}, false},
		{"valid name", domain.UpdatePPPoEProfileRequest{Name: ptr("Test")}, false},
		{"invalid name", domain.UpdatePPPoEProfileRequest{Name: ptr("")}, true},
		{"invalid primary dns", domain.UpdatePPPoEProfileRequest{PrimaryDNS: ptr("bad")}, true},
		{"invalid mtu", domain.UpdatePPPoEProfileRequest{MTU: ptr(1501)}, true},
		{"negative bandwidth", domain.UpdatePPPoEProfileRequest{BandwidthMaxDown: ptr(-1)}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateUpdatePPPoEProfile(tt.req)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateUpdatePPPoEProfile() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
