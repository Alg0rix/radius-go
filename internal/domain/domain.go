package domain

import "time"

type SessionState string

const (
	SessionStateActive SessionState = "active"
	SessionStateStop   SessionState = "stopped"
	SessionStateStale  SessionState = "stale"
)

type ServiceType string

const (
	ServiceTypeFramed ServiceType = "framed"
	ServiceTypeLogin  ServiceType = "login"
)

type RadiusUser struct {
	ID               string      `json:"id"`
	Username         string      `json:"username"`
	PasswordHash     string      `json:"-"`
	FullName         string      `json:"full_name"`
	Email            string      `json:"email"`
	Enabled          bool        `json:"enabled"`
	SimultaneousUse  int         `json:"simultaneous_use"`
	SessionTimeout   int         `json:"session_timeout"`
	IdleTimeout      int         `json:"idle_timeout"`
	FramedIP         string      `json:"framed_ip"`
	MikrotikGroup    string      `json:"mikrotik_group"`
	RateLimit        string      `json:"rate_limit"`
	BandwidthMaxUp   uint32      `json:"bandwidth_max_up"`
	BandwidthMaxDown uint32      `json:"bandwidth_max_down"`
	MaxTotalOctets   uint32      `json:"max_total_octets"`
	ServiceType      ServiceType `json:"service_type"`
	// Voucher tracking — zero values mean "not a voucher".
	IsVoucher               bool       `json:"is_voucher"`
	VoucherPackageID        string     `json:"voucher_package_id,omitempty"`
	FirstLoginAt            *time.Time `json:"first_login_at,omitempty"`
	ExpiresAt               *time.Time `json:"expires_at,omitempty"`
	UsageSecondsUsed        int        `json:"usage_seconds_used"`
	DataBytesUsed           int64      `json:"data_bytes_used"`
	SpeedUploadKbps         int        `json:"speed_upload_kbps"`
	SpeedDownloadKbps       int        `json:"speed_download_kbps"`
	VoucherTimeLimitType    string      `json:"voucher_time_limit_type,omitempty"`
	VoucherTimeLimitSeconds int        `json:"voucher_time_limit_seconds"`
	VoucherDataCapBytes     int64      `json:"voucher_data_cap_bytes"`
	CreatedAt               time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type NAS struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	IPAddress   string    `json:"ip_address"`
	Secret      string    `json:"-"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type RadiusSession struct {
	ID              string       `json:"id"`
	SessionID       string       `json:"session_id"`
	Username        string       `json:"username"`
	NASID           string       `json:"nas_id"`
	NASIP           string       `json:"nas_ip"`
	NASIdentifier   string       `json:"nas_identifier"`
	FramedIP        string       `json:"framed_ip"`
	CallingStation  string       `json:"calling_station"`
	CalledStation   string       `json:"called_station"`
	ServiceType     ServiceType  `json:"service_type"`
	InputOctets     int64        `json:"input_octets"`
	OutputOctets    int64        `json:"output_octets"`
	SessionTime     int64        `json:"session_time"`
	SessionStatus   SessionState `json:"session_status"`
	MikrotikGroup   string       `json:"mikrotik_group"`
	RateLimit       string       `json:"rate_limit"`
	BandwidthMaxUp   uint32       `json:"bandwidth_max_up"`
	BandwidthMaxDown uint32       `json:"bandwidth_max_down"`
	MaxTotalOctets   uint32       `json:"max_total_octets"`
	StartTime       time.Time    `json:"start_time"`
	LastUpdate      time.Time    `json:"last_update"`
	StopTime        *time.Time   `json:"stop_time"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

type Subscriber struct {
	ID               string      `json:"id"`
	Username         string      `json:"username"`
	FullName         string      `json:"full_name"`
	Email            string      `json:"email"`
	Enabled          bool        `json:"enabled"`
	SimultaneousUse  int         `json:"simultaneous_use"`
	SessionTimeout   int         `json:"session_timeout"`
	IdleTimeout      int         `json:"idle_timeout"`
	FramedIP         string      `json:"framed_ip"`
	MikrotikGroup    string      `json:"mikrotik_group"`
	RateLimit        string      `json:"rate_limit"`
	BandwidthMaxUp   uint32      `json:"bandwidth_max_up"`
	BandwidthMaxDown uint32      `json:"bandwidth_max_down"`
	MaxTotalOctets   uint32      `json:"max_total_octets"`
	ServiceType      ServiceType `json:"service_type"`
	IsVoucher        bool        `json:"is_voucher"`
	VoucherPackageID string      `json:"voucher_package_id,omitempty"`
	ExpiresAt        *time.Time  `json:"expires_at,omitempty"`
	CreatedAt        time.Time   `json:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at"`
}

func SubscriberFromUser(u RadiusUser) Subscriber {
	return Subscriber{
		ID:               u.ID,
		Username:         u.Username,
		FullName:         u.FullName,
		Email:            u.Email,
		Enabled:          u.Enabled,
		SimultaneousUse:  u.SimultaneousUse,
		SessionTimeout:   u.SessionTimeout,
		IdleTimeout:      u.IdleTimeout,
		FramedIP:         u.FramedIP,
		MikrotikGroup:    u.MikrotikGroup,
		RateLimit:        u.RateLimit,
		BandwidthMaxUp:   u.BandwidthMaxUp,
		BandwidthMaxDown: u.BandwidthMaxDown,
		MaxTotalOctets:   u.MaxTotalOctets,
		ServiceType:      u.ServiceType,
		IsVoucher:        u.IsVoucher,
		VoucherPackageID: u.VoucherPackageID,
		ExpiresAt:        u.ExpiresAt,
		CreatedAt:        u.CreatedAt,
		UpdatedAt:        u.UpdatedAt,
	}
}

type CreateUserRequest struct {
	Username        string `json:"username"`
	Password        string `json:"password"`
	FullName        string `json:"full_name"`
	Email           string `json:"email"`
	SimultaneousUse int    `json:"simultaneous_use"`
	SessionTimeout  int    `json:"session_timeout"`
	IdleTimeout     int    `json:"idle_timeout"`
	FramedIP        string `json:"framed_ip"`
	MikrotikGroup   string `json:"mikrotik_group"`
	RateLimit       string `json:"rate_limit"`
	BandwidthMaxUp   uint32 `json:"bandwidth_max_up"`
	BandwidthMaxDown uint32 `json:"bandwidth_max_down"`
	MaxTotalOctets   uint32 `json:"max_total_octets"`
	ServiceType     string `json:"service_type"`
}

type UpdateUserRequest struct {
	Username          string  `json:"username"`
	Password          string  `json:"password"`
	FullName          string  `json:"full_name"`
	Email             string  `json:"email"`
	Enabled           *bool   `json:"enabled"`
	SimultaneousUse   *int    `json:"simultaneous_use"`
	SessionTimeout    *int    `json:"session_timeout"`
	IdleTimeout       *int    `json:"idle_timeout"`
	FramedIP          string  `json:"framed_ip"`
	MikrotikGroup     string  `json:"mikrotik_group"`
	RateLimit         string  `json:"rate_limit"`
	BandwidthMaxUp    *uint32 `json:"bandwidth_max_up"`
	BandwidthMaxDown  *uint32 `json:"bandwidth_max_down"`
	MaxTotalOctets    *uint32 `json:"max_total_octets"`
	ServiceType       string  `json:"service_type"`
}

type CreateNASRequest struct {
	Name        string `json:"name"`
	IPAddress   string `json:"ip_address"`
	Secret      string `json:"secret"`
	Description string `json:"description"`
}

type UpdateNASRequest struct {
	Name        string `json:"name"`
	IPAddress   string `json:"ip_address"`
	Secret      string `json:"secret"`
	Description string `json:"description"`
	Enabled     *bool  `json:"enabled"`
}

type DisconnectRequest struct {
	Username string `json:"username"`
	Reason   string `json:"reason"`
}

type CoAChangeRequest struct {
	Username    string `json:"username"`
	RateLimit   string `json:"rate_limit"`
	Group       string `json:"mikrotik_group"`
	BandwidthMaxUp   *uint32 `json:"bandwidth_max_up"`
	BandwidthMaxDown *uint32 `json:"bandwidth_max_down"`
	MaxTotalOctets   *uint32 `json:"max_total_octets"`
}

// --- Voucher types ---

type TimeLimitType string

const (
	TimeLimitCalendar TimeLimitType = "calendar"
	TimeLimitUsage    TimeLimitType = "usage"
)

type VoucherPackage struct {
	ID                  string        `json:"id"`
	Name                string        `json:"name"`
	Description         string        `json:"description"`
	Price               float64       `json:"price"`
	SpeedUploadKbps     int           `json:"speed_upload_kbps"`
	SpeedDownloadKbps   int           `json:"speed_download_kbps"`
	DataCapBytes        int64         `json:"data_cap_bytes"`
	TimeLimitType       TimeLimitType `json:"time_limit_type"`
	TimeLimitSeconds    int           `json:"time_limit_seconds"`
	MaxConcurrentUsers  int           `json:"max_concurrent_users"`
	Enabled             bool          `json:"enabled"`
	CreatedAt           time.Time     `json:"created_at"`
	UpdatedAt           time.Time     `json:"updated_at"`
}

type CreateVoucherPackageRequest struct {
	Name               string `json:"name"`
	Description        string `json:"description"`
	Price              float64 `json:"price"`
	SpeedUploadKbps     int    `json:"speed_upload_kbps"`
	SpeedDownloadKbps   int    `json:"speed_download_kbps"`
	DataCapBytes       int64  `json:"data_cap_bytes"`
	TimeLimitType       string `json:"time_limit_type"`
	TimeLimitSeconds    int    `json:"time_limit_seconds"`
	MaxConcurrentUsers  int    `json:"max_concurrent_users"`
}

type UpdateVoucherPackageRequest struct {
	Name               *string  `json:"name"`
	Description        *string  `json:"description"`
	Price              *float64 `json:"price"`
	SpeedUploadKbps     *int     `json:"speed_upload_kbps"`
	SpeedDownloadKbps   *int     `json:"speed_download_kbps"`
	DataCapBytes       *int64   `json:"data_cap_bytes"`
	TimeLimitType       *string  `json:"time_limit_type"`
	TimeLimitSeconds    *int     `json:"time_limit_seconds"`
	MaxConcurrentUsers  *int     `json:"max_concurrent_users"`
	Enabled            *bool    `json:"enabled"`
}

type GenerateVoucherRequest struct {
	PackageID      string `json:"package_id"`
	Count          int    `json:"count"`
	CodeFormat     string `json:"code_format"`   // "random" or "custom"
	CodeLength     int    `json:"code_length"`   // for random
	CustomCode     string `json:"custom_code"`   // for custom
	PasswordMode   string `json:"password_mode"` // "same_as_user", "random", "custom"
	CustomPassword string `json:"custom_password"` // for custom
}

type VoucherBalance struct {
	Username             string `json:"username"`
	PackageName          string `json:"package_name"`
	Enabled              bool   `json:"enabled"`
	FirstLoginAt         *time.Time `json:"first_login_at"`
	ExpiresAt            *time.Time `json:"expires_at"`
	TimeLimitType        string `json:"time_limit_type"`
	TimeLimitSeconds     int    `json:"time_limit_seconds"`
	UsageSecondsUsed     int    `json:"usage_seconds_used"`
	UsageSecondsRemaining int   `json:"usage_seconds_remaining"`
	DataCapBytes         int64  `json:"data_cap_bytes"`
	DataBytesUsed        int64  `json:"data_bytes_used"`
	DataBytesRemaining   int64  `json:"data_bytes_remaining"`
}
