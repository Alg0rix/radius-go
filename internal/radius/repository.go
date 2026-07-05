package radius

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/your-org/radius-go/internal/domain"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

// userSelectCols is the shared SELECT column list + scan order for radius_users.
const userSelectCols = `id, username, password_hash, full_name, email, enabled,
	simultaneous_use, session_timeout, idle_timeout, framed_ip, mikrotik_group,
	rate_limit, bandwidth_max_up, bandwidth_max_down, max_total_octets,
	is_voucher, voucher_package_id, first_login_at, expires_at,
	usage_seconds_used, data_bytes_used, speed_upload_kbps, speed_download_kbps,
	voucher_time_limit_type, voucher_time_limit_seconds, voucher_data_cap_bytes,
	service_type, created_at, updated_at`

// scanUser scans a single row into a RadiusUser.
func scanUser(row pgx.Row) (*domain.RadiusUser, error) {
	var u domain.RadiusUser
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.FullName, &u.Email, &u.Enabled,
		&u.SimultaneousUse, &u.SessionTimeout, &u.IdleTimeout, &u.FramedIP, &u.MikrotikGroup,
		&u.RateLimit, &u.BandwidthMaxUp, &u.BandwidthMaxDown, &u.MaxTotalOctets,
		&u.IsVoucher, &u.VoucherPackageID, &u.FirstLoginAt, &u.ExpiresAt,
		&u.UsageSecondsUsed, &u.DataBytesUsed, &u.SpeedUploadKbps, &u.SpeedDownloadKbps,
		&u.VoucherTimeLimitType, &u.VoucherTimeLimitSeconds, &u.VoucherDataCapBytes,
		&u.ServiceType, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("repo: scan user: %w", err)
	}
	return &u, nil
}

// scanUsers iterates pgx.Rows into a slice of RadiusUser.
func scanUsers(rows pgx.Rows) ([]domain.RadiusUser, error) {
	var users []domain.RadiusUser
	for rows.Next() {
		var u domain.RadiusUser
		if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.FullName, &u.Email, &u.Enabled,
			&u.SimultaneousUse, &u.SessionTimeout, &u.IdleTimeout, &u.FramedIP, &u.MikrotikGroup,
			&u.RateLimit, &u.BandwidthMaxUp, &u.BandwidthMaxDown, &u.MaxTotalOctets,
			&u.IsVoucher, &u.VoucherPackageID, &u.FirstLoginAt, &u.ExpiresAt,
			&u.UsageSecondsUsed, &u.DataBytesUsed, &u.SpeedUploadKbps, &u.SpeedDownloadKbps,
			&u.VoucherTimeLimitType, &u.VoucherTimeLimitSeconds, &u.VoucherDataCapBytes,
			&u.ServiceType, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, fmt.Errorf("repo: scan users: %w", err)
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

// --- Users ---

func (r *Repository) ListUsers(ctx context.Context) ([]domain.RadiusUser, error) {
	rows, err := r.db.Query(ctx, `SELECT `+userSelectCols+` FROM radius_users ORDER BY username`)
	if err != nil {
		return nil, fmt.Errorf("repo: list users: %w", err)
	}
	defer rows.Close()
	return scanUsers(rows)
}

func (r *Repository) CreateUser(ctx context.Context, user domain.RadiusUser) error {
	_, err := r.db.Exec(ctx, `INSERT INTO radius_users (id, username, password_hash, full_name, email, enabled,
		simultaneous_use, session_timeout, idle_timeout, framed_ip, mikrotik_group,
		rate_limit, bandwidth_max_up, bandwidth_max_down, max_total_octets,
		is_voucher, voucher_package_id, first_login_at, expires_at,
		usage_seconds_used, data_bytes_used, speed_upload_kbps, speed_download_kbps,
		voucher_time_limit_type, voucher_time_limit_seconds, voucher_data_cap_bytes,
		service_type)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,
		        $16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27)`,
		user.ID, user.Username, user.PasswordHash, user.FullName, user.Email, user.Enabled,
		user.SimultaneousUse, user.SessionTimeout, user.IdleTimeout, user.FramedIP, user.MikrotikGroup,
		user.RateLimit, user.BandwidthMaxUp, user.BandwidthMaxDown, user.MaxTotalOctets,
		user.IsVoucher, nullStr(user.VoucherPackageID), user.FirstLoginAt, user.ExpiresAt,
		user.UsageSecondsUsed, user.DataBytesUsed, user.SpeedUploadKbps, user.SpeedDownloadKbps,
		user.VoucherTimeLimitType, user.VoucherTimeLimitSeconds, user.VoucherDataCapBytes,
		string(user.ServiceType))
	return err
}

func (r *Repository) UpdateUser(ctx context.Context, user domain.RadiusUser) error {
	_, err := r.db.Exec(ctx, `UPDATE radius_users SET username=$1, password_hash=$2, full_name=$3, email=$4,
		enabled=$5, simultaneous_use=$6, session_timeout=$7, idle_timeout=$8, framed_ip=$9,
		mikrotik_group=$10, rate_limit=$11, bandwidth_max_up=$12, bandwidth_max_down=$13,
		max_total_octets=$14, is_voucher=$15, voucher_package_id=$16,
		usage_seconds_used=$17, data_bytes_used=$18, speed_upload_kbps=$19, speed_download_kbps=$20,
		voucher_time_limit_type=$21, voucher_time_limit_seconds=$22, voucher_data_cap_bytes=$23,
		service_type=$24, updated_at=now() WHERE id=$25`,
		user.Username, user.PasswordHash, user.FullName, user.Email, user.Enabled,
		user.SimultaneousUse, user.SessionTimeout, user.IdleTimeout, user.FramedIP, user.MikrotikGroup,
		user.RateLimit, user.BandwidthMaxUp, user.BandwidthMaxDown, user.MaxTotalOctets,
		user.IsVoucher, nullStr(user.VoucherPackageID),
		user.UsageSecondsUsed, user.DataBytesUsed, user.SpeedUploadKbps, user.SpeedDownloadKbps,
		user.VoucherTimeLimitType, user.VoucherTimeLimitSeconds, user.VoucherDataCapBytes,
		string(user.ServiceType), user.ID)
	return err
}

func (r *Repository) GetUserByID(ctx context.Context, id string) (*domain.RadiusUser, error) {
	row := r.db.QueryRow(ctx, `SELECT `+userSelectCols+` FROM radius_users WHERE id=$1`, id)
	return scanUser(row)
}

func (r *Repository) GetUserByUsername(ctx context.Context, username string) (*domain.RadiusUser, error) {
	row := r.db.QueryRow(ctx, `SELECT `+userSelectCols+` FROM radius_users WHERE username=$1`, username)
	return scanUser(row)
}

func (r *Repository) DeleteUser(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM radius_users WHERE id=$1`, id)
	return err
}

// RecordFirstLogin sets first_login_at and expires_at (for calendar vouchers) on first auth.
func (r *Repository) RecordFirstLogin(ctx context.Context, userID string, timeLimitType domain.TimeLimitType, timeLimitSeconds int) error {
	if timeLimitType != domain.TimeLimitCalendar {
		_, err := r.db.Exec(ctx, `UPDATE radius_users SET first_login_at=now(), updated_at=now() WHERE id=$1`, userID)
		return err
	}
	_, err := r.db.Exec(ctx, `UPDATE radius_users SET first_login_at=now(), expires_at=now() + $2::interval, updated_at=now() WHERE id=$1`,
		userID, fmt.Sprintf("%d seconds", timeLimitSeconds))
	return err
}

// AddUsageDelta increments usage_seconds_used and data_bytes_used.
func (r *Repository) AddUsageDelta(ctx context.Context, userID string, secs int, octets int64) error {
	_, err := r.db.Exec(ctx, `UPDATE radius_users SET usage_seconds_used=usage_seconds_used+$2, data_bytes_used=data_bytes_used+$3, updated_at=now() WHERE id=$1`,
		userID, secs, octets)
	return err
}

// DisableUser sets enabled=false (voucher expired / limit hit).
func (r *Repository) DisableUser(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE radius_users SET enabled=false, updated_at=now() WHERE id=$1`, userID)
	return err
}

// --- Voucher listing ---

func (r *Repository) ListVoucherUsers(ctx context.Context) ([]domain.RadiusUser, error) {
	rows, err := r.db.Query(ctx, `SELECT `+userSelectCols+` FROM radius_users WHERE is_voucher=true ORDER BY username`)
	if err != nil {
		return nil, fmt.Errorf("repo: list vouchers: %w", err)
	}
	defer rows.Close()
	return scanUsers(rows)
}

// --- NAS ---

func (r *Repository) ListNAS(ctx context.Context) ([]domain.NAS, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name, ip_address, secret, description, enabled, created_at, updated_at FROM radius_nas ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("repo: list nas: %w", err)
	}
	defer rows.Close()

	var nases []domain.NAS
	for rows.Next() {
		var n domain.NAS
		if err := rows.Scan(&n.ID, &n.Name, &n.IPAddress, &n.Secret, &n.Description, &n.Enabled, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, fmt.Errorf("repo: scan nas: %w", err)
		}
		nases = append(nases, n)
	}
	return nases, rows.Err()
}

func (r *Repository) CreateNAS(ctx context.Context, nas domain.NAS) error {
	_, err := r.db.Exec(ctx, `INSERT INTO radius_nas (id, name, ip_address, secret, description, enabled) VALUES ($1,$2,$3,$4,$5,$6)`,
		nas.ID, nas.Name, nas.IPAddress, nas.Secret, nas.Description, nas.Enabled)
	return err
}

func (r *Repository) GetNASByID(ctx context.Context, id string) (*domain.NAS, error) {
	row := r.db.QueryRow(ctx, `SELECT id, name, ip_address, secret, description, enabled, created_at, updated_at FROM radius_nas WHERE id=$1`, id)
	var n domain.NAS
	err := row.Scan(&n.ID, &n.Name, &n.IPAddress, &n.Secret, &n.Description, &n.Enabled, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("repo: get nas: %w", err)
	}
	return &n, nil
}

func (r *Repository) UpdateNAS(ctx context.Context, nas domain.NAS) error {
	_, err := r.db.Exec(ctx, `UPDATE radius_nas SET name=$1, ip_address=$2, secret=$3, description=$4, enabled=$5, updated_at=now() WHERE id=$6`,
		nas.Name, nas.IPAddress, nas.Secret, nas.Description, nas.Enabled, nas.ID)
	return err
}

func (r *Repository) DeleteNAS(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM radius_nas WHERE id=$1`, id)
	return err
}

// --- Sessions ---

func (r *Repository) UpsertSession(ctx context.Context, session domain.RadiusSession) error {
	_, err := r.db.Exec(ctx, `INSERT INTO radius_sessions (id, session_id, username, nas_id, nas_ip, nas_identifier,
		framed_ip, calling_station, called_station, service_type, input_octets, output_octets, session_time,
		session_status, mikrotik_group, rate_limit, bandwidth_max_up, bandwidth_max_down, max_total_octets,
		start_time, last_update, stop_time)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22)
		ON CONFLICT (id) DO UPDATE SET
		session_id=$2, username=$3, nas_id=$4, nas_ip=$5, nas_identifier=$6,
		framed_ip=$7, calling_station=$8, called_station=$9, service_type=$10,
		input_octets=$11, output_octets=$12, session_time=$13, session_status=$14,
		mikrotik_group=$15, rate_limit=$16, bandwidth_max_up=$17, bandwidth_max_down=$18,
		max_total_octets=$19, start_time=$20, last_update=$21, stop_time=$22,
		updated_at=now()`,
		session.ID, session.SessionID, session.Username, session.NASID, session.NASIP, session.NASIdentifier,
		session.FramedIP, session.CallingStation, session.CalledStation, string(session.ServiceType),
		session.InputOctets, session.OutputOctets, session.SessionTime, string(session.SessionStatus),
		session.MikrotikGroup, session.RateLimit, session.BandwidthMaxUp, session.BandwidthMaxDown,
		session.MaxTotalOctets, session.StartTime, session.LastUpdate, session.StopTime)
	return err
}

func (r *Repository) ListActiveSessions(ctx context.Context) ([]domain.RadiusSession, error) {
	return r.listSessionsByStatus(ctx, domain.SessionStateActive)
}

func (r *Repository) ListActiveSessionsByUsername(ctx context.Context, username string) ([]domain.RadiusSession, error) {
	rows, err := r.db.Query(ctx, sessionSelectCols+` FROM radius_sessions WHERE username=$1 AND session_status='active'`, username)
	if err != nil {
		return nil, fmt.Errorf("repo: list sessions by username: %w", err)
	}
	defer rows.Close()
	return scanSessions(rows)
}

func (r *Repository) listSessionsByStatus(ctx context.Context, status domain.SessionState) ([]domain.RadiusSession, error) {
	rows, err := r.db.Query(ctx, sessionSelectCols+` FROM radius_sessions WHERE session_status=$1`, string(status))
	if err != nil {
		return nil, fmt.Errorf("repo: list sessions: %w", err)
	}
	defer rows.Close()
	return scanSessions(rows)
}

func (r *Repository) ListAllSessions(ctx context.Context) ([]domain.RadiusSession, error) {
	rows, err := r.db.Query(ctx, sessionSelectCols+` FROM radius_sessions ORDER BY start_time DESC LIMIT 500`)
	if err != nil {
		return nil, fmt.Errorf("repo: list all sessions: %w", err)
	}
	defer rows.Close()
	return scanSessions(rows)
}

func (r *Repository) UpdateSessionStatus(ctx context.Context, sessionID string, status domain.SessionState, stopTime *time.Time) error {
	_, err := r.db.Exec(ctx, `UPDATE radius_sessions SET session_status=$1, stop_time=$2, updated_at=now() WHERE id=$3`,
		string(status), stopTime, sessionID)
	return err
}

const sessionSelectCols = `SELECT id, session_id, username, nas_id, nas_ip, nas_identifier,
	framed_ip, calling_station, called_station, service_type, input_octets, output_octets,
	session_time, session_status, mikrotik_group, rate_limit, bandwidth_max_up, bandwidth_max_down,
	max_total_octets, start_time, last_update, stop_time, created_at, updated_at`

func scanSessions(rows pgx.Rows) ([]domain.RadiusSession, error) {
	var sessions []domain.RadiusSession
	for rows.Next() {
		var s domain.RadiusSession
		if err := rows.Scan(&s.ID, &s.SessionID, &s.Username, &s.NASID, &s.NASIP, &s.NASIdentifier,
			&s.FramedIP, &s.CallingStation, &s.CalledStation, &s.ServiceType, &s.InputOctets,
			&s.OutputOctets, &s.SessionTime, &s.SessionStatus, &s.MikrotikGroup, &s.RateLimit,
			&s.BandwidthMaxUp, &s.BandwidthMaxDown, &s.MaxTotalOctets,
			&s.StartTime, &s.LastUpdate, &s.StopTime, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("repo: scan session: %w", err)
		}
		sessions = append(sessions, s)
	}
	return sessions, rows.Err()
}

// nullStr returns a string representation that passes NULL to pgx for empty strings.
func nullStr(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}