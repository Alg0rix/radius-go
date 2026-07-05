CREATE TABLE IF NOT EXISTS pppoe_profiles (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name               TEXT NOT NULL UNIQUE,
    description        TEXT NOT NULL DEFAULT '',

    framed_ip_pool     TEXT NOT NULL DEFAULT '',
    framed_ip_netmask  TEXT NOT NULL DEFAULT '',
    primary_dns        TEXT NOT NULL DEFAULT '',
    secondary_dns      TEXT NOT NULL DEFAULT '',
    ppp_compression    BOOLEAN NOT NULL DEFAULT false,
    mtu                INTEGER NOT NULL DEFAULT 0,
    mru                INTEGER NOT NULL DEFAULT 0,
    keepalive_interval INTEGER NOT NULL DEFAULT 0,

    rate_limit         TEXT NOT NULL DEFAULT '',
    bandwidth_max_up   INTEGER NOT NULL DEFAULT 0,
    bandwidth_max_down INTEGER NOT NULL DEFAULT 0,
    session_timeout    INTEGER NOT NULL DEFAULT 0,
    idle_timeout       INTEGER NOT NULL DEFAULT 0,
    max_total_octets   BIGINT NOT NULL DEFAULT 0,

    enabled            BOOLEAN NOT NULL DEFAULT true,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_pppoe_profiles_enabled ON pppoe_profiles(enabled);

ALTER TABLE voucher_packages
    ADD COLUMN IF NOT EXISTS address_pool  TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS primary_dns   TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS secondary_dns TEXT NOT NULL DEFAULT '';

ALTER TABLE radius_users
    ADD COLUMN IF NOT EXISTS pppoe_profile_id UUID REFERENCES pppoe_profiles(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_radius_users_pppoe_profile_id ON radius_users(pppoe_profile_id);
