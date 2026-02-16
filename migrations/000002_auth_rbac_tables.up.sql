ALTER TABLE users
    ADD COLUMN IF NOT EXISTS failed_login_count INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS locked_until TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS password_changed_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS roles (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(128) NOT NULL,
    description TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS permissions (
    id BIGSERIAL PRIMARY KEY,
    code VARCHAR(150) NOT NULL UNIQUE,
    module VARCHAR(64) NOT NULL,
    resource VARCHAR(64) NOT NULL,
    action VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id BIGINT NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    permission_id BIGINT NOT NULL REFERENCES permissions (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE IF NOT EXISTS user_roles (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_id BIGINT NOT NULL REFERENCES roles (id) ON DELETE CASCADE,
    scope_type VARCHAR(20) NOT NULL,
    scope_id BIGINT,
    own_only BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT user_roles_scope_type_check CHECK (scope_type IN ('global', 'company', 'branch', 'warehouse')),
    CONSTRAINT user_roles_scope_required_check CHECK (
        scope_type = 'global' OR scope_id IS NOT NULL
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS user_roles_unique_idx
    ON user_roles (user_id, role_id, scope_type, COALESCE(scope_id, 0), own_only);
CREATE INDEX IF NOT EXISTS user_roles_user_id_idx ON user_roles (user_id);
CREATE INDEX IF NOT EXISTS user_roles_scope_idx ON user_roles (scope_type, scope_id);

CREATE TABLE IF NOT EXISTS auth_sessions (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    refresh_hash CHAR(64) NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    ip VARCHAR(64),
    user_agent TEXT,
    revoked_at TIMESTAMPTZ,
    rotated_from_session_id BIGINT REFERENCES auth_sessions (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS auth_sessions_user_idx ON auth_sessions (user_id);
CREATE INDEX IF NOT EXISTS auth_sessions_active_idx ON auth_sessions (user_id, revoked_at, expires_at);

CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users (id) ON DELETE SET NULL,
    action VARCHAR(120) NOT NULL,
    module VARCHAR(64) NOT NULL,
    resource VARCHAR(64) NOT NULL,
    resource_id VARCHAR(128),
    meta JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS audit_logs_user_created_idx ON audit_logs (user_id, created_at DESC);
CREATE INDEX IF NOT EXISTS audit_logs_action_created_idx ON audit_logs (action, created_at DESC);

INSERT INTO roles (code, name, description)
VALUES
    ('owner', 'Owner', 'Full company access'),
    ('accountant', 'Accountant', 'Accounting and reporting'),
    ('sales_manager', 'Sales Manager', 'Sales and CRM operations'),
    ('warehouse_operator', 'Warehouse Operator', 'Warehouse operations')
ON CONFLICT (code) DO NOTHING;

INSERT INTO permissions (code, module, resource, action)
VALUES
    ('users.read', 'users', 'user', 'read'),
    ('accounting.entry.manage', 'accounting', 'entry', 'manage'),
    ('sales.order.create', 'sales', 'order', 'create'),
    ('warehouse.stock.update', 'warehouse', 'stock', 'update'),
    ('reports.view', 'reports', 'report', 'view')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code = 'owner'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('users.read', 'accounting.entry.manage', 'reports.view')
WHERE r.code = 'accountant'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('users.read', 'sales.order.create', 'reports.view')
WHERE r.code = 'sales_manager'
ON CONFLICT (role_id, permission_id) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN ('users.read', 'warehouse.stock.update', 'reports.view')
WHERE r.code = 'warehouse_operator'
ON CONFLICT (role_id, permission_id) DO NOTHING;
