package radius

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/Alg0rix/radius-go/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type VoucherService struct {
	db  *pgxpool.Pool
	repo *Repository
}

func NewVoucherService(db *pgxpool.Pool, repo *Repository) *VoucherService {
	return &VoucherService{db: db, repo: repo}
}

// --- Voucher Package CRUD ---

const voucherPackageSelect = `SELECT id, name, description, price, speed_upload_kbps,
	speed_download_kbps, data_cap_bytes, time_limit_type, time_limit_seconds,
	max_concurrent_users, enabled, created_at, updated_at`

func scanVoucherPackages(rows pgx.Rows) ([]domain.VoucherPackage, error) {
	var packages []domain.VoucherPackage
	for rows.Next() {
		var p domain.VoucherPackage
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.Price,
			&p.SpeedUploadKbps, &p.SpeedDownloadKbps, &p.DataCapBytes,
			&p.TimeLimitType, &p.TimeLimitSeconds, &p.MaxConcurrentUsers, &p.Enabled,
			&p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan voucher package: %w", err)
		}
		packages = append(packages, p)
	}
	return packages, rows.Err()
}

func (v *VoucherService) CreatePackage(ctx context.Context, req domain.CreateVoucherPackageRequest) (domain.VoucherPackage, error) {
	timeLimitType := domain.TimeLimitUsage
	if req.TimeLimitType == string(domain.TimeLimitCalendar) {
		timeLimitType = domain.TimeLimitCalendar
	}

	pkg := domain.VoucherPackage{
		ID:                 uuid.New().String(),
		Name:               req.Name,
		Description:        req.Description,
		Price:              req.Price,
		SpeedUploadKbps:    req.SpeedUploadKbps,
		SpeedDownloadKbps:  req.SpeedDownloadKbps,
		DataCapBytes:       req.DataCapBytes,
		TimeLimitType:      timeLimitType,
		TimeLimitSeconds:   req.TimeLimitSeconds,
		MaxConcurrentUsers: req.MaxConcurrentUsers,
		Enabled:            true,
	}

	_, err := v.db.Exec(ctx, `INSERT INTO voucher_packages (id, name, description, price,
		speed_upload_kbps, speed_download_kbps, data_cap_bytes, time_limit_type,
		time_limit_seconds, max_concurrent_users, enabled)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
		pkg.ID, pkg.Name, pkg.Description, pkg.Price,
		pkg.SpeedUploadKbps, pkg.SpeedDownloadKbps, pkg.DataCapBytes,
		string(pkg.TimeLimitType), pkg.TimeLimitSeconds, pkg.MaxConcurrentUsers, pkg.Enabled)
	if err != nil {
		return pkg, fmt.Errorf("create voucher package: %w", err)
	}
	return pkg, nil
}

func (v *VoucherService) ListPackages(ctx context.Context) ([]domain.VoucherPackage, error) {
	rows, err := v.db.Query(ctx, voucherPackageSelect+` FROM voucher_packages ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list voucher packages: %w", err)
	}
	defer rows.Close()
	return scanVoucherPackages(rows)
}

func (v *VoucherService) GetPackage(ctx context.Context, id string) (*domain.VoucherPackage, error) {
	row := v.db.QueryRow(ctx, voucherPackageSelect+` FROM voucher_packages WHERE id=$1`, id)
	var p domain.VoucherPackage
	err := row.Scan(&p.ID, &p.Name, &p.Description, &p.Price,
		&p.SpeedUploadKbps, &p.SpeedDownloadKbps, &p.DataCapBytes,
		&p.TimeLimitType, &p.TimeLimitSeconds, &p.MaxConcurrentUsers, &p.Enabled,
		&p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get voucher package: %w", err)
	}
	return &p, nil
}

func (v *VoucherService) UpdatePackage(ctx context.Context, id string, req domain.UpdateVoucherPackageRequest) (domain.VoucherPackage, error) {
	existing, err := v.GetPackage(ctx, id)
	if err != nil {
		return domain.VoucherPackage{}, err
	}
	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.Description != nil {
		existing.Description = *req.Description
	}
	if req.Price != nil {
		existing.Price = *req.Price
	}
	if req.SpeedUploadKbps != nil {
		existing.SpeedUploadKbps = *req.SpeedUploadKbps
	}
	if req.SpeedDownloadKbps != nil {
		existing.SpeedDownloadKbps = *req.SpeedDownloadKbps
	}
	if req.DataCapBytes != nil {
		existing.DataCapBytes = *req.DataCapBytes
	}
	if req.TimeLimitType != nil {
		existing.TimeLimitType = domain.TimeLimitType(*req.TimeLimitType)
	}
	if req.TimeLimitSeconds != nil {
		existing.TimeLimitSeconds = *req.TimeLimitSeconds
	}
	if req.MaxConcurrentUsers != nil {
		existing.MaxConcurrentUsers = *req.MaxConcurrentUsers
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}

	_, err = v.db.Exec(ctx, `UPDATE voucher_packages SET name=$1, description=$2, price=$3,
		speed_upload_kbps=$4, speed_download_kbps=$5, data_cap_bytes=$6, time_limit_type=$7,
		time_limit_seconds=$8, max_concurrent_users=$9, enabled=$10, updated_at=now()
		WHERE id=$11`,
		existing.Name, existing.Description, existing.Price,
		existing.SpeedUploadKbps, existing.SpeedDownloadKbps, existing.DataCapBytes,
		string(existing.TimeLimitType), existing.TimeLimitSeconds, existing.MaxConcurrentUsers,
		existing.Enabled, id)
	if err != nil {
		return domain.VoucherPackage{}, fmt.Errorf("update voucher package: %w", err)
	}
	return *existing, nil
}

func (v *VoucherService) DeletePackage(ctx context.Context, id string) error {
	_, err := v.db.Exec(ctx, `DELETE FROM voucher_packages WHERE id=$1`, id)
	return err
}

// --- Voucher Generation ---

// GeneratedVoucher carries the subscriber + the plaintext password returned once at creation time.
type GeneratedVoucher struct {
	domain.Subscriber
	Password string `json:"password"`
}

func (v *VoucherService) GenerateVouchers(ctx context.Context, req domain.GenerateVoucherRequest) ([]GeneratedVoucher, error) {
	pkg, err := v.GetPackage(ctx, req.PackageID)
	if err != nil {
		return nil, fmt.Errorf("generate vouchers: %w", err)
	}
	if !pkg.Enabled {
		return nil, fmt.Errorf("generate vouchers: package %s is disabled", pkg.Name)
	}

	count := req.Count
	if count <= 0 {
		count = 1
	}
	if req.CodeLength <= 0 {
		req.CodeLength = 12
	}

	results := make([]GeneratedVoucher, 0, count)
	for i := 0; i < count; i++ {
		gv, err := v.generateOneVoucher(ctx, pkg, req, i)
		if err != nil {
			return results, err
		}
		results = append(results, gv)
	}
	return results, nil
}

func (v *VoucherService) generateOneVoucher(ctx context.Context, pkg *domain.VoucherPackage, req domain.GenerateVoucherRequest, index int) (GeneratedVoucher, error) {
	code := req.CustomCode
	if req.CodeFormat != "custom" {
		code = generateRandomCode(req.CodeLength)
	} else if req.Count > 1 && req.CustomCode != "" {
		code = fmt.Sprintf("%s-%d", req.CustomCode, index+1)
	}

	password := code
	switch req.PasswordMode {
	case "random":
		password = generateRandomCode(16)
	case "custom":
		password = req.CustomPassword
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return GeneratedVoucher{}, fmt.Errorf("hash password: %w", err)
	}

	simUse := pkg.MaxConcurrentUsers
	if simUse < 0 {
		simUse = 0
	}

	user := domain.RadiusUser{
		ID:                      uuid.New().String(),
		Username:                code,
		PasswordHash:            string(hash),
		FullName:                fmt.Sprintf("Voucher: %s", pkg.Name),
		Enabled:                 true,
		SimultaneousUse:         simUse,
		SpeedUploadKbps:         pkg.SpeedUploadKbps,
		SpeedDownloadKbps:       pkg.SpeedDownloadKbps,
		IsVoucher:               true,
		VoucherPackageID:        &pkg.ID,
		VoucherTimeLimitType:    string(pkg.TimeLimitType),
		VoucherTimeLimitSeconds: pkg.TimeLimitSeconds,
		VoucherDataCapBytes:     pkg.DataCapBytes,
		ServiceType:             domain.ServiceTypeFramed,
	}

	// pfSense-Max-Total-Octets is uint32; clip data cap if it fits.
	if pkg.DataCapBytes > 0 && pkg.DataCapBytes <= 4294967295 {
		user.MaxTotalOctets = uint32(pkg.DataCapBytes)
	}

	if err := v.repo.CreateUser(ctx, user); err != nil {
		return GeneratedVoucher{}, fmt.Errorf("create voucher user: %w", err)
	}

	return GeneratedVoucher{
		Subscriber: domain.SubscriberFromUser(user),
		Password:   password,
	}, nil
}

// GetBalance returns the current remaining time/data for a voucher subscriber.
func (v *VoucherService) GetBalance(ctx context.Context, username string) (domain.VoucherBalance, error) {
	user, err := v.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return domain.VoucherBalance{}, err
	}
	if !user.IsVoucher {
		return domain.VoucherBalance{}, fmt.Errorf("user %s is not a voucher subscriber", username)
	}

	bal := domain.VoucherBalance{
		Username:          user.Username,
		Enabled:           user.Enabled,
		FirstLoginAt:      user.FirstLoginAt,
		ExpiresAt:         user.ExpiresAt,
		UsageSecondsUsed:  user.UsageSecondsUsed,
		DataBytesUsed:     user.DataBytesUsed,
	}

	if user.VoucherPackageID != nil && *user.VoucherPackageID != "" {
		if pkg, err := v.GetPackage(ctx, *user.VoucherPackageID); err == nil {
			bal.PackageName = pkg.Name
			bal.TimeLimitType = string(pkg.TimeLimitType)
			bal.TimeLimitSeconds = pkg.TimeLimitSeconds
			bal.UsageSecondsRemaining = pkg.TimeLimitSeconds - user.UsageSecondsUsed
			if bal.UsageSecondsRemaining < 0 {
				bal.UsageSecondsRemaining = 0
			}
			bal.DataCapBytes = pkg.DataCapBytes
			bal.DataBytesRemaining = pkg.DataCapBytes - user.DataBytesUsed
			if bal.DataBytesRemaining < 0 {
				bal.DataBytesRemaining = 0
			}
		}
	}

	return bal, nil
}

// ListVouchers returns all voucher subscribers.
func (v *VoucherService) ListVouchers(ctx context.Context) ([]domain.Subscriber, error) {
	users, err := v.repo.ListVoucherUsers(ctx)
	if err != nil {
		return nil, err
	}
	subs := make([]domain.Subscriber, 0, len(users))
	for _, u := range users {
		subs = append(subs, domain.SubscriberFromUser(u))
	}
	return subs, nil
}

// generateRandomCode produces a mixed-case alphanumeric code of the given length.
func generateRandomCode(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		b[i] = chars[n.Int64()]
	}
	return string(b)
}