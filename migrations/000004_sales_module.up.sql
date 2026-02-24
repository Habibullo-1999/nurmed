CREATE TABLE IF NOT EXISTS sales_orders (
    id BIGSERIAL PRIMARY KEY,
    company_id BIGINT NOT NULL,
    channel VARCHAR(20) NOT NULL,
    document_no VARCHAR(64) NOT NULL,
    customer_name VARCHAR(200),
    currency VARCHAR(8) NOT NULL DEFAULT 'UZS',
    status VARCHAR(20) NOT NULL DEFAULT 'posted',
    total_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
    item_count INTEGER NOT NULL DEFAULT 0,
    sold_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sales_orders_channel_check CHECK (channel IN ('realization', 'mobile', 'pos')),
    CONSTRAINT sales_orders_status_check CHECK (status IN ('draft', 'posted', 'cancelled'))
);

CREATE UNIQUE INDEX IF NOT EXISTS sales_orders_company_document_uidx
    ON sales_orders (company_id, document_no);
CREATE INDEX IF NOT EXISTS sales_orders_company_channel_idx
    ON sales_orders (company_id, channel, sold_at DESC);

CREATE TABLE IF NOT EXISTS sales_order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES sales_orders (id) ON DELETE CASCADE,
    product_id BIGINT,
    product_name VARCHAR(200) NOT NULL,
    quantity DOUBLE PRECISION NOT NULL,
    price DOUBLE PRECISION NOT NULL,
    amount DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sales_order_items_qty_check CHECK (quantity > 0),
    CONSTRAINT sales_order_items_price_check CHECK (price >= 0),
    CONSTRAINT sales_order_items_amount_check CHECK (amount >= 0)
);

CREATE INDEX IF NOT EXISTS sales_order_items_order_id_idx ON sales_order_items (order_id);

CREATE TABLE IF NOT EXISTS sales_returns (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES sales_orders (id) ON DELETE RESTRICT,
    company_id BIGINT NOT NULL,
    reason TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'posted',
    total_amount DOUBLE PRECISION NOT NULL,
    returned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT sales_returns_status_check CHECK (status IN ('draft', 'posted', 'cancelled')),
    CONSTRAINT sales_returns_amount_check CHECK (total_amount > 0)
);

CREATE INDEX IF NOT EXISTS sales_returns_company_idx
    ON sales_returns (company_id, returned_at DESC);
CREATE INDEX IF NOT EXISTS sales_returns_order_idx ON sales_returns (order_id);

INSERT INTO permissions (code, module, resource, action)
VALUES
    ('sales.realization.read', 'sales', 'realization', 'read'),
    ('sales.realization.create', 'sales', 'realization', 'create'),
    ('sales.registry.read', 'sales', 'registry', 'read'),
    ('sales.mobile.read', 'sales', 'mobile', 'read'),
    ('sales.mobile.create', 'sales', 'mobile', 'create'),
    ('sales.pos.read', 'sales', 'pos', 'read'),
    ('sales.pos.create', 'sales', 'pos', 'create'),
    ('sales.return.read', 'sales', 'return', 'read'),
    ('sales.return.create', 'sales', 'return', 'create')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN (
    'sales.realization.read',
    'sales.realization.create',
    'sales.registry.read',
    'sales.mobile.read',
    'sales.mobile.create',
    'sales.pos.read',
    'sales.pos.create',
    'sales.return.read',
    'sales.return.create'
)
WHERE r.code IN ('owner', 'sales_manager')
ON CONFLICT (role_id, permission_id) DO NOTHING;
