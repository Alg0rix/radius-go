-- radius-go initial schema

CREATE TABLE IF NOT EXISTS radius_users (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username         TEXT NOT NULL UNIQUE,
    password_hash    TEXT NOT NULL,
    full_name        TEXT NOT NULL DEFAULT '',
    email            TEXT NOT NULL DEFAULT '',
    enabled          BOOLEAN NOT NULL DEFAULT true,
    simultaneous_use INTEGER NOT NULL DEFAULT 0,
    session_timeout  INTEGER NOT NULL DEFAULT 0,
    idle_timeout     INTEGER NOT NULL DEFAULT 0,
    framed_ip        TEXT NOT NULL DEFAULT '',
    mikrotik_group   TEXT NOT NULL DEFAULT '',
    rate_limit       TEXT NOT NULL DEFAULT '',
    bandwidth_max_up   INTEGER NOT NULL DEFAULT 0,
    bandwidth_max_down INTEGER NOT NULL DEFAULT 0,
    max_total_octets   INTEGER NOT NULL DEFAULT 0,
    service_type     TEXT NOT NULL DEFAULT 'framed',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS radius_nas (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    ip_address  TEXT NOT NULL,
    secret      TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    enabled     BOOLEAN NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS radius_sessions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id       TEXT NOT NULL,
    username         TEXT NOT NULL,
    nas_id           TEXT NOT NULL DEFAULT '',
    nas_ip           TEXT NOT NULL DEFAULT '',
    nas_identifier   TEXT NOT NULL DEFAULT '',
    framed_ip        TEXT NOT NULL DEFAULT '',
    calling_station  TEXT NOT NULL DEFAULT '',
    called_station   TEXT NOT NULL DEFAULT '',
    service_type     TEXT NOT NULL DEFAULT 'framed',
    input_octets     BIGINT NOT NULL DEFAULT 0,
    output_octets    BIGINT NOT NULL DEFAULT 0,
    session_time     BIGINT NOT NULL DEFAULT 0,
    session_status   TEXT NOT NULL DEFAULT 'active',
    mikrotik_group   TEXT NOT NULL DEFAULT '',
    rate_limit       TEXT NOT NULL DEFAULT '',
    bandwidth_max_up   INTEGER NOT NULL DEFAULT 0,
    bandwidth_max_down INTEGER NOT NULL DEFAULT 0,
    max_total_octets   INTEGER NOT NULL DEFAULT 0,
    start_time       TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_update      TIMESTAMPTZ NOT NULL DEFAULT now(),
    stop_time        TIMESTAMPTZ,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_radius_sessions_session_id ON radius_sessions(session_id);
CREATE INDEX IF NOT EXISTS idx_radius_sessions_username ON radius_sessions(username);
CREATE INDEX IF NOT EXISTS idx_radius_sessions_session_status ON radius_sessions(session_status);
CREATE INDEX IF NOT EXISTS idx_radius_nas_ip_address ON radius_nas(ip_address);