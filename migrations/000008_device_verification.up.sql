CREATE TABLE IF NOT EXISTS trusted_devices (
    id BIGSERIAL PRIMARY KEY,
    fingerprint_hash CHAR(64) NOT NULL UNIQUE,
    browser_name VARCHAR(64),
    browser_version VARCHAR(32),
    os_name VARCHAR(64),
    os_version VARCHAR(32),
    device_name VARCHAR(128),
    pending_token_hash CHAR(64),
    pending_expires_at TIMESTAMPTZ,
    verify_target VARCHAR(190),
    verify_code_hash CHAR(64),
    verify_expires_at TIMESTAMPTZ,
    trusted_token_hash CHAR(64) UNIQUE,
    trusted_at TIMESTAMPTZ,
    last_seen_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS trusted_devices_pending_idx ON trusted_devices (pending_token_hash, pending_expires_at);
CREATE INDEX IF NOT EXISTS trusted_devices_trusted_idx ON trusted_devices (trusted_token_hash);
CREATE INDEX IF NOT EXISTS trusted_devices_last_seen_idx ON trusted_devices (last_seen_at);

