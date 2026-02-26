CREATE TABLE IF NOT EXISTS purchase_orders (
    id BIGSERIAL PRIMARY KEY,
    company_id BIGINT NOT NULL,
    document_no VARCHAR(64) NOT NULL,
    supplier_name VARCHAR(200),
    currency VARCHAR(8) NOT NULL DEFAULT 'UZS',
    status VARCHAR(20) NOT NULL DEFAULT 'posted',
    total_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
    item_count INTEGER NOT NULL DEFAULT 0,
    purchased_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT purchase_orders_status_check CHECK (status IN ('draft', 'posted', 'cancelled'))
);

CREATE UNIQUE INDEX IF NOT EXISTS purchase_orders_company_document_uidx
    ON purchase_orders (company_id, document_no);
CREATE UNIQUE INDEX IF NOT EXISTS purchase_orders_id_company_uidx
    ON purchase_orders (id, company_id);
CREATE INDEX IF NOT EXISTS purchase_orders_company_purchased_idx
    ON purchase_orders (company_id, purchased_at DESC);

CREATE TABLE IF NOT EXISTS purchase_order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES purchase_orders (id) ON DELETE CASCADE,
    product_id BIGINT,
    product_name VARCHAR(200) NOT NULL,
    quantity DOUBLE PRECISION NOT NULL,
    price DOUBLE PRECISION NOT NULL,
    amount DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT purchase_order_items_qty_check CHECK (quantity > 0),
    CONSTRAINT purchase_order_items_price_check CHECK (price >= 0),
    CONSTRAINT purchase_order_items_amount_check CHECK (amount >= 0)
);

CREATE INDEX IF NOT EXISTS purchase_order_items_order_id_idx ON purchase_order_items (order_id);

CREATE TABLE IF NOT EXISTS purchase_returns (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL,
    company_id BIGINT NOT NULL,
    reason TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'posted',
    total_amount DOUBLE PRECISION NOT NULL,
    returned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT purchase_returns_status_check CHECK (status IN ('draft', 'posted', 'cancelled')),
    CONSTRAINT purchase_returns_amount_check CHECK (total_amount > 0),
    CONSTRAINT purchase_returns_order_company_fkey
        FOREIGN KEY (order_id, company_id)
            REFERENCES purchase_orders (id, company_id)
            ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS purchase_returns_company_idx
    ON purchase_returns (company_id, returned_at DESC);
CREATE INDEX IF NOT EXISTS purchase_returns_order_idx ON purchase_returns (order_id);

INSERT INTO permissions (code, module, resource, action)
VALUES
    ('purchases.acquisition.read', 'purchases', 'acquisition', 'read'),
    ('purchases.acquisition.create', 'purchases', 'acquisition', 'create'),
    ('purchases.registry.read', 'purchases', 'registry', 'read'),
    ('purchases.return.read', 'purchases', 'return', 'read'),
    ('purchases.return.create', 'purchases', 'return', 'create')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN (
    'purchases.acquisition.read',
    'purchases.acquisition.create',
    'purchases.registry.read',
    'purchases.return.read',
    'purchases.return.create'
)
WHERE r.code IN ('owner', 'warehouse_operator')
ON CONFLICT (role_id, permission_id) DO NOTHING;
