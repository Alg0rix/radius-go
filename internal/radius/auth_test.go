package radius

import (
	"math"
	"testing"

	"github.com/Alg0rix/radius-go/internal/domain"
)

func TestEffectiveRateLimit(t *testing.T) {
	profile := &domain.PPPoEProfile{
		RateLimit:      "5M/5M",
		BandwidthMaxUp: 1000,
		BandwidthMaxDown: 2000,
	}

	tests := []struct {
		name string
		user domain.RadiusUser
		want string
	}{
		{
			name: "per-user rate limit wins",
			user: domain.RadiusUser{RateLimit: "10M/10M", PPPoEProfile: profile},
			want: "10M/10M",
		},
		{
			name: "per-user speed bandwidth wins over profile rate limit",
			user: domain.RadiusUser{SpeedUploadKbps: 2000, SpeedDownloadKbps: 4000, PPPoEProfile: &domain.PPPoEProfile{RateLimit: "5M/5M"}},
			want: "2000K/4000K",
		},
		{
			name: "profile rate limit falls through",
			user: domain.RadiusUser{PPPoEProfile: profile},
			want: "5M/5M",
		},
		{
			name: "profile bandwidth falls through",
			user: domain.RadiusUser{PPPoEProfile: &domain.PPPoEProfile{BandwidthMaxUp: 1000, BandwidthMaxDown: 2000}},
			want: "1000K/2000K",
		},
		{
			name: "empty returns empty",
			user: domain.RadiusUser{},
			want: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := effectiveRateLimit(tt.user)
			if got != tt.want {
				t.Errorf("effectiveRateLimit() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestEffectiveBandwidth(t *testing.T) {
	profile := &domain.PPPoEProfile{BandwidthMaxUp: 1000, BandwidthMaxDown: 2000}

	tests := []struct {
		name    string
		user    domain.RadiusUser
		wantUp  uint32
		wantDown uint32
	}{
		{
			name:     "per-user bandwidth overrides profile",
			user:     domain.RadiusUser{BandwidthMaxUp: 3000, BandwidthMaxDown: 4000, PPPoEProfile: profile},
			wantUp:   3000,
			wantDown: 4000,
		},
		{
			name:     "profile fills missing per-user bandwidth",
			user:     domain.RadiusUser{BandwidthMaxUp: 3000, PPPoEProfile: profile},
			wantUp:   3000,
			wantDown: 2000,
		},
		{
			name:     "profile used when per-user empty",
			user:     domain.RadiusUser{PPPoEProfile: profile},
			wantUp:   1000,
			wantDown: 2000,
		},
		{
			name:     "empty profile returns user zeros",
			user:     domain.RadiusUser{},
			wantUp:   0,
			wantDown: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotUp, gotDown := effectiveBandwidth(tt.user)
			if gotUp != tt.wantUp || gotDown != tt.wantDown {
				t.Errorf("effectiveBandwidth() = (%d, %d), want (%d, %d)", gotUp, gotDown, tt.wantUp, tt.wantDown)
			}
		})
	}
}

func TestEffectiveMaxTotalOctets(t *testing.T) {
	tests := []struct {
		name string
		user domain.RadiusUser
		want uint32
	}{
		{
			name: "per-user value wins",
			user: domain.RadiusUser{MaxTotalOctets: 1000, PPPoEProfile: &domain.PPPoEProfile{MaxTotalOctets: 2000}},
			want: 1000,
		},
		{
			name: "profile value falls through",
			user: domain.RadiusUser{PPPoEProfile: &domain.PPPoEProfile{MaxTotalOctets: 2000}},
			want: 2000,
		},
		{
			name: "large profile value clamps to max uint32",
			user: domain.RadiusUser{PPPoEProfile: &domain.PPPoEProfile{MaxTotalOctets: math.MaxUint32 + 1}},
			want: math.MaxUint32,
		},
		{
			name: "empty returns zero",
			user: domain.RadiusUser{},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := effectiveMaxTotalOctets(tt.user)
			if got != tt.want {
				t.Errorf("effectiveMaxTotalOctets() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestEffectiveSessionTimeout(t *testing.T) {
	tests := []struct {
		name string
		user domain.RadiusUser
		want int
	}{
		{
			name: "per-user wins",
			user: domain.RadiusUser{SessionTimeout: 100, PPPoEProfile: &domain.PPPoEProfile{SessionTimeout: 200}},
			want: 100,
		},
		{
			name: "profile falls through",
			user: domain.RadiusUser{PPPoEProfile: &domain.PPPoEProfile{SessionTimeout: 200}},
			want: 200,
		},
		{
			name: "empty returns zero",
			user: domain.RadiusUser{},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := effectiveSessionTimeout(tt.user)
			if got != tt.want {
				t.Errorf("effectiveSessionTimeout() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestEffectiveIdleTimeout(t *testing.T) {
	tests := []struct {
		name string
		user domain.RadiusUser
		want int
	}{
		{
			name: "per-user wins",
			user: domain.RadiusUser{IdleTimeout: 100, PPPoEProfile: &domain.PPPoEProfile{IdleTimeout: 200}},
			want: 100,
		},
		{
			name: "profile falls through",
			user: domain.RadiusUser{PPPoEProfile: &domain.PPPoEProfile{IdleTimeout: 200}},
			want: 200,
		},
		{
			name: "empty returns zero",
			user: domain.RadiusUser{},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := effectiveIdleTimeout(tt.user)
			if got != tt.want {
				t.Errorf("effectiveIdleTimeout() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestFormatBandwidthKbps(t *testing.T) {
	tests := []struct {
		name   string
		up     int
		down   int
		want   string
	}{
		{"both zero", 0, 0, ""},
		{"both positive", 1000, 2000, "1000K/2000K"},
		{"one positive", 0, 2000, "0K/2000K"},
		{"megabit", 1000000, 2000000, "1000M/2000M"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatBandwidthKbps(tt.up, tt.down)
			if got != tt.want {
				t.Errorf("formatBandwidthKbps() = %q, want %q", got, tt.want)
			}
		})
	}
}
