// Package radiusctl provides the radiusctl CLI: an HTTP client over the
// radius-go management API. Types here are CLI-owned and intentionally
// decoupled from the server's internal domain types so the CLI binary
// does not import server packages (pgx, bcrypt, echo, ...).
package radiusctl

import (
	"encoding/json"
	"time"
)

// Envelope mirrors the server's runtime.Envelope response wrapper.
// On success Data holds the payload; on failure Error holds details.
type Envelope struct {
	Success bool `json:"success"`
	// Data is the raw JSON of the Data field; client.do leaves it as-is so
	// --json can render it verbatim and subcommands can unmarshal it into
	// concrete types as needed.
	Data json.RawMessage `json:"data"`
	Err  *APIError       `json:"error"`
}

// APIError is the error object inside an Envelope on failure.
type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details"`
}

// NAS mirrors domain.NAS minus the secret (server omits it).
type NAS struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	IPAddress   string    `json:"ip_address"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateNASRequest is the body for POST /api/v1/radius/nases.
type CreateNASRequest struct {
	Name        string `json:"name"`
	IPAddress   string `json:"ip_address"`
	Secret      string `json:"secret"`
	Description string `json:"description"`
}

// UpdateNASRequest is the body for PUT /api/v1/radius/nases/:id.
// Pointers mark "provided on the command line" so we omit unset fields.
type UpdateNASRequest struct {
	Name        string `json:"name,omitempty"`
	IPAddress   string `json:"ip_address,omitempty"`
	Secret      string `json:"secret,omitempty"`
	Description string `json:"description,omitempty"`
	Enabled     *bool  `json:"enabled,omitempty"`
}

// Subscriber mirrors domain.Subscriber (no password hash exposed).
type Subscriber struct {
	ID               string     `json:"id"`
	Username         string     `json:"username"`
	FullName         string     `json:"full_name"`
	Email            string     `json:"email"`
	Enabled          bool       `json:"enabled"`
	SimultaneousUse  int        `json:"simultaneous_use"`
	SessionTimeout   int        `json:"session_timeout"`
	IdleTimeout      int        `json:"idle_timeout"`
	FramedIP         string     `json:"framed_ip"`
	MikrotikGroup    string     `json:"mikrotik_group"`
	RateLimit        string     `json:"rate_limit"`
	BandwidthMaxUp   uint32     `json:"bandwidth_max_up"`
	BandwidthMaxDown uint32     `json:"bandwidth_max_down"`
	MaxTotalOctets   uint32     `json:"max_total_octets"`
	ServiceType      string     `json:"service_type"`
	PPPoEProfileID   string     `json:"pppoe_profile_id,omitempty"`
	IsVoucher        bool       `json:"is_voucher"`
	VoucherPackageID string     `json:"voucher_package_id,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// CreateSubscriberRequest is the body for POST /api/v1/radius/subscribers.
type CreateSubscriberRequest struct {
	Username         string `json:"username"`
	Password         string `json:"password"`
	FullName         string `json:"full_name"`
	Email            string `json:"email"`
	SimultaneousUse  int    `json:"simultaneous_use"`
	SessionTimeout   int    `json:"session_timeout"`
	IdleTimeout      int    `json:"idle_timeout"`
	FramedIP         string `json:"framed_ip"`
	MikrotikGroup    string `json:"mikrotik_group"`
	RateLimit        string `json:"rate_limit"`
	BandwidthMaxUp   uint32 `json:"bandwidth_max_up"`
	BandwidthMaxDown uint32 `json:"bandwidth_max_down"`
	MaxTotalOctets   uint32 `json:"max_total_octets"`
	ServiceType      string `json:"service_type"`
	PPPoEProfileID   string `json:"pppoe_profile_id"`
}

// UpdateSubscriberRequest is the body for PUT /api/v1/radius/subscribers/:id.
// Pointer fields let us distinguish "not provided" from "set to zero".
type UpdateSubscriberRequest struct {
	Username         string   `json:"username,omitempty"`
	Password         string   `json:"password,omitempty"`
	FullName         string   `json:"full_name,omitempty"`
	Email            string   `json:"email,omitempty"`
	Enabled          *bool    `json:"enabled,omitempty"`
	SimultaneousUse  *int     `json:"simultaneous_use,omitempty"`
	SessionTimeout   *int     `json:"session_timeout,omitempty"`
	IdleTimeout      *int     `json:"idle_timeout,omitempty"`
	FramedIP         string   `json:"framed_ip,omitempty"`
	MikrotikGroup    string   `json:"mikrotik_group,omitempty"`
	RateLimit        string   `json:"rate_limit,omitempty"`
	BandwidthMaxUp   *uint32  `json:"bandwidth_max_up,omitempty"`
	BandwidthMaxDown *uint32  `json:"bandwidth_max_down,omitempty"`
	MaxTotalOctets   *uint32  `json:"max_total_octets,omitempty"`
	ServiceType      string   `json:"service_type,omitempty"`
	PPPoEProfileID   *string  `json:"pppoe_profile_id,omitempty"`
}

// Session mirrors domain.RadiusSession.
type Session struct {
	ID             string     `json:"id"`
	SessionID      string     `json:"session_id"`
	Username       string     `json:"username"`
	NASID          string     `json:"nas_id"`
	NASIP          string     `json:"nas_ip"`
	NASIdentifier  string     `json:"nas_identifier"`
	FramedIP       string     `json:"framed_ip"`
	CallingStation string     `json:"calling_station"`
	CalledStation  string     `json:"called_station"`
	ServiceType    string     `json:"service_type"`
	InputOctets    int64      `json:"input_octets"`
	OutputOctets   int64      `json:"output_octets"`
	SessionTime    int64      `json:"session_time"`
	SessionStatus  string     `json:"session_status"`
	MikrotikGroup  string     `json:"mikrotik_group"`
	RateLimit      string     `json:"rate_limit"`
	StartTime      time.Time  `json:"start_time"`
	LastUpdate     time.Time  `json:"last_update"`
	StopTime       *time.Time `json:"stop_time"`
}

// Status mirrors radius.Status.
type Status struct {
	StartedAt       time.Time `json:"started_at"`
	NASCount        int       `json:"nas_count"`
	SubscriberCount int       `json:"subscriber_count"`
	ActiveSessions  int       `json:"active_sessions"`
	AuthRequests    uint64    `json:"auth_requests"`
	AuthAccepts     uint64    `json:"auth_accepts"`
	AuthRejects     uint64    `json:"auth_rejects"`
	AcctRequests    uint64    `json:"acct_requests"`
	Health          string    `json:"health"`
}

// DisconnectRequest is the body for POST /api/v1/radius/sessions/disconnect.
type DisconnectRequest struct {
	Username string `json:"username"`
	Reason   string `json:"reason,omitempty"`
}

// CoAChangeRequest is the body for POST /api/v1/radius/subscribers/coa-change.
type CoAChangeRequest struct {
	Username         string  `json:"username"`
	RateLimit        string  `json:"rate_limit,omitempty"`
	Group            string  `json:"mikrotik_group,omitempty"`
	BandwidthMaxUp   *uint32 `json:"bandwidth_max_up,omitempty"`
	BandwidthMaxDown *uint32 `json:"bandwidth_max_down,omitempty"`
	MaxTotalOctets   *uint32 `json:"max_total_octets,omitempty"`
}

// CoaChangeResult mirrors radius.CoaChangeResult.
type CoaChangeResult struct {
	DisconnectedCount int      `json:"disconnected_count"`
	FailedNAS         []string `json:"failed_nas"`
}

// CleanupResult mirrors radius.CleanupResult.
type CleanupResult struct {
	StaleSessionsCleaned int       `json:"stale_sessions_cleaned"`
	ActiveSessionsKept   int       `json:"active_sessions_kept"`
	CleanedAt            time.Time `json:"cleaned_at"`
}

// VoucherPackage mirrors domain.VoucherPackage.
type VoucherPackage struct {
	ID                 string    `json:"id"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	Price              float64   `json:"price"`
	SpeedUploadKbps    int       `json:"speed_upload_kbps"`
	SpeedDownloadKbps  int       `json:"speed_download_kbps"`
	DataCapBytes       int64     `json:"data_cap_bytes"`
	TimeLimitType      string    `json:"time_limit_type"`
	TimeLimitSeconds   int       `json:"time_limit_seconds"`
	MaxConcurrentUsers int       `json:"max_concurrent_users"`
	AddressPool        string    `json:"address_pool"`
	PrimaryDNS         string    `json:"primary_dns"`
	SecondaryDNS       string    `json:"secondary_dns"`
	Enabled            bool      `json:"enabled"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// CreateVoucherPackageRequest is the body for POST /api/v1/voucher-packages.
type CreateVoucherPackageRequest struct {
	Name               string  `json:"name"`
	Description        string  `json:"description"`
	Price              float64 `json:"price"`
	SpeedUploadKbps    int     `json:"speed_upload_kbps"`
	SpeedDownloadKbps  int     `json:"speed_download_kbps"`
	DataCapBytes       int64   `json:"data_cap_bytes"`
	TimeLimitType      string  `json:"time_limit_type"`
	TimeLimitSeconds   int     `json:"time_limit_seconds"`
	MaxConcurrentUsers int     `json:"max_concurrent_users"`
	AddressPool        string  `json:"address_pool"`
	PrimaryDNS         string  `json:"primary_dns"`
	SecondaryDNS       string  `json:"secondary_dns"`
}

// UpdateVoucherPackageRequest is the body for PUT /api/v1/voucher-packages/:id.
type UpdateVoucherPackageRequest struct {
	Name               *string  `json:"name,omitempty"`
	Description        *string  `json:"description,omitempty"`
	Price              *float64 `json:"price,omitempty"`
	SpeedUploadKbps    *int     `json:"speed_upload_kbps,omitempty"`
	SpeedDownloadKbps  *int     `json:"speed_download_kbps,omitempty"`
	DataCapBytes       *int64   `json:"data_cap_bytes,omitempty"`
	TimeLimitType      *string  `json:"time_limit_type,omitempty"`
	TimeLimitSeconds   *int     `json:"time_limit_seconds,omitempty"`
	MaxConcurrentUsers *int     `json:"max_concurrent_users,omitempty"`
	AddressPool        *string  `json:"address_pool,omitempty"`
	PrimaryDNS         *string  `json:"primary_dns,omitempty"`
	SecondaryDNS       *string  `json:"secondary_dns,omitempty"`
	Enabled            *bool    `json:"enabled,omitempty"`
}

// GenerateVoucherRequest is the body for POST /api/v1/vouchers/generate.
type GenerateVoucherRequest struct {
	PackageID      string `json:"package_id"`
	Count          int    `json:"count"`
	CodeFormat     string `json:"code_format"`
	CodeLength     int    `json:"code_length"`
	CustomCode     string `json:"custom_code,omitempty"`
	PasswordMode   string `json:"password_mode"`
	CustomPassword string `json:"custom_password,omitempty"`
}

// GeneratedVoucher mirrors radius.GeneratedVoucher (Subscriber + plaintext password).
type GeneratedVoucher struct {
	Subscriber
	Password string `json:"password"`
}

// VoucherBalance mirrors domain.VoucherBalance.
type VoucherBalance struct {
	Username              string     `json:"username"`
	PackageName           string     `json:"package_name"`
	Enabled               bool       `json:"enabled"`
	FirstLoginAt          *time.Time `json:"first_login_at"`
	ExpiresAt             *time.Time `json:"expires_at"`
	TimeLimitType         string     `json:"time_limit_type"`
	TimeLimitSeconds      int        `json:"time_limit_seconds"`
	UsageSecondsUsed      int        `json:"usage_seconds_used"`
	UsageSecondsRemaining int        `json:"usage_seconds_remaining"`
	DataCapBytes          int64      `json:"data_cap_bytes"`
	DataBytesUsed         int64      `json:"data_bytes_used"`
	DataBytesRemaining    int64      `json:"data_bytes_remaining"`
}

// DeleteResult is the {id: ...} body returned by delete endpoints.
type DeleteResult struct {
	ID string `json:"id"`
}

// PPPoEProfile mirrors domain.PPPoEProfile.
type PPPoEProfile struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Description       string    `json:"description"`
	FramedIPPool      string    `json:"framed_ip_pool"`
	FramedIPNetmask   string    `json:"framed_ip_netmask"`
	PrimaryDNS        string    `json:"primary_dns"`
	SecondaryDNS      string    `json:"secondary_dns"`
	PPPCompression    bool      `json:"ppp_compression"`
	MTU               int       `json:"mtu"`
	MRU               int       `json:"mru"`
	KeepaliveInterval int       `json:"keepalive_interval"`
	RateLimit         string    `json:"rate_limit"`
	BandwidthMaxUp    int       `json:"bandwidth_max_up"`
	BandwidthMaxDown  int       `json:"bandwidth_max_down"`
	SessionTimeout    int       `json:"session_timeout"`
	IdleTimeout       int       `json:"idle_timeout"`
	MaxTotalOctets    int64     `json:"max_total_octets"`
	Enabled           bool      `json:"enabled"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// CreatePPPoEProfileRequest is the body for POST /api/v1/pppoe-profiles.
type CreatePPPoEProfileRequest struct {
	Name              string `json:"name"`
	Description       string `json:"description"`
	FramedIPPool      string `json:"framed_ip_pool"`
	FramedIPNetmask   string `json:"framed_ip_netmask"`
	PrimaryDNS        string `json:"primary_dns"`
	SecondaryDNS      string `json:"secondary_dns"`
	PPPCompression    bool   `json:"ppp_compression"`
	MTU               int    `json:"mtu"`
	MRU               int    `json:"mru"`
	KeepaliveInterval int    `json:"keepalive_interval"`
	RateLimit         string `json:"rate_limit"`
	BandwidthMaxUp    int    `json:"bandwidth_max_up"`
	BandwidthMaxDown  int    `json:"bandwidth_max_down"`
	SessionTimeout    int    `json:"session_timeout"`
	IdleTimeout       int    `json:"idle_timeout"`
	MaxTotalOctets    int64  `json:"max_total_octets"`
}

// UpdatePPPoEProfileRequest is the body for PUT /api/v1/pppoe-profiles/:id.
type UpdatePPPoEProfileRequest struct {
	Name              *string `json:"name,omitempty"`
	Description       *string `json:"description,omitempty"`
	FramedIPPool      *string `json:"framed_ip_pool,omitempty"`
	FramedIPNetmask   *string `json:"framed_ip_netmask,omitempty"`
	PrimaryDNS        *string `json:"primary_dns,omitempty"`
	SecondaryDNS      *string `json:"secondary_dns,omitempty"`
	PPPCompression    *bool   `json:"ppp_compression,omitempty"`
	MTU               *int    `json:"mtu,omitempty"`
	MRU               *int    `json:"mru,omitempty"`
	KeepaliveInterval *int    `json:"keepalive_interval,omitempty"`
	RateLimit         *string `json:"rate_limit,omitempty"`
	BandwidthMaxUp    *int    `json:"bandwidth_max_up,omitempty"`
	BandwidthMaxDown  *int    `json:"bandwidth_max_down,omitempty"`
	SessionTimeout    *int    `json:"session_timeout,omitempty"`
	IdleTimeout       *int    `json:"idle_timeout,omitempty"`
	MaxTotalOctets    *int64  `json:"max_total_octets,omitempty"`
	Enabled           *bool   `json:"enabled,omitempty"`
}
