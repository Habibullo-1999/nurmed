CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,
    company_id BIGINT,
    username VARCHAR(100) NOT NULL,
    phone VARCHAR(32),
    email VARCHAR(255),
    password_hash TEXT NOT NULL,
    first_name VARCHAR(100) NOT NULL,
    last_name VARCHAR(100),
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    is_super_admin BOOLEAN NOT NULL DEFAULT FALSE,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT users_status_check CHECK (status IN ('active', 'blocked', 'invited', 'deleted'))
);

CREATE UNIQUE INDEX IF NOT EXISTS users_username_uidx ON users (username);
CREATE UNIQUE INDEX IF NOT EXISTS users_phone_uidx ON users (phone) WHERE phone IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS users_email_uidx ON users (email) WHERE email IS NOT NULL;
CREATE INDEX IF NOT EXISTS users_company_id_idx ON users (company_id);
