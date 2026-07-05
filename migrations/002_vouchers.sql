-- Voucher packages and voucher-enabled subscribers.
-- Extends radius_users with voucher tracking columns and adds the
-- voucher_packages table. All DDL uses IF NOT EXISTS so re-runs are safe.

CREATE TABLE IF NOT EXISTS voucher_packages (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                  TEXT NOT NULL,
    description           TEXT NOT NULL DEFAULT '',
    price                 NUMERIC(12,2) NOT NULL DEFAULT 0,
    speed_upload_kbps     INTEGER NOT NULL DEFAULT 0,
    speed_download_kbps   INTEGER NOT NULL DEFAULT 0,
    data_cap_bytes        BIGINT NOT NULL DEFAULT 0,
    time_limit_type       TEXT NOT NULL DEFAULT 'usage',
    time_limit_seconds    INTEGER NOT NULL DEFAULT 0,
    max_concurrent_users  INTEGER NOT NULL DEFAULT 0,
    enabled               BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE radius_users ADD COLUMN IF NOT EXISTS is_voucher BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE radius_users ADD COLUMN IF NOT EXISTS voucher_package_id UUID;
ALTER TABLE radius_users ADD COLUMN IF NOT EXISTS first_login_at TIMESTAMPTZ;
ALTER TABLE radius_users ADD COLUMN IF NOT EXISTS expires_at TIMESTAMPTZ;
ALTER TABLE radius_users ADD COLUMN IF NOT EXISTS usage_seconds_used INTEGER NOT NULL DEFAULT 0;
ALTER TABLE radius_users ADD COLUMN IF NOT EXISTS data_bytes_used BIGINT NOT NULL DEFAULT 0;
ALTER TABLE radius_users ADD COLUMN IF NOT EXISTS speed_upload_kbps INTEGER NOT NULL DEFAULT 0;
ALTER TABLE radius_users ADD COLUMN IF NOT EXISTS speed_download_kbps INTEGER NOT NULL DEFAULT 0;
ALTER TABLE radius_users ADD COLUMN IF NOT EXISTS voucher_time_limit_type TEXT NOT NULL DEFAULT '';
ALTER TABLE radius_users ADD COLUMN IF NOT EXISTS voucher_time_limit_seconds INTEGER NOT NULL DEFAULT 0;
ALTER TABLE radius_users ADD COLUMN IF NOT EXISTS voucher_data_cap_bytes BIGINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_voucher_packages_enabled ON voucher_packages(enabled);
CREATE INDEX IF NOT EXISTS idx_radius_users_is_voucher ON radius_users(is_voucher);
CREATE INDEX IF NOT EXISTS idx_radius_users_expires_at ON radius_users(expires_at);
