CREATE TABLE IF NOT EXISTS warehouses (
    id BIGSERIAL PRIMARY KEY,
    company_id BIGINT NOT NULL,
    name VARCHAR(200) NOT NULL,
    address TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT warehouses_status_check CHECK (status IN ('active', 'inactive'))
);

CREATE INDEX IF NOT EXISTS warehouses_company_idx ON warehouses (company_id, status);
CREATE UNIQUE INDEX IF NOT EXISTS warehouses_company_name_uidx ON warehouses (company_id, name);

CREATE TABLE IF NOT EXISTS warehouse_stocks (
    id BIGSERIAL PRIMARY KEY,
    company_id BIGINT NOT NULL,
    warehouse_id BIGINT NOT NULL REFERENCES warehouses (id) ON DELETE RESTRICT,
    product_id BIGINT NOT NULL REFERENCES products (id) ON DELETE RESTRICT,
    quantity DOUBLE PRECISION NOT NULL DEFAULT 0,
    cost_price DOUBLE PRECISION NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT warehouse_stocks_qty_check CHECK (quantity >= 0),
    CONSTRAINT warehouse_stocks_warehouse_product_uniq UNIQUE (warehouse_id, product_id)
);

CREATE INDEX IF NOT EXISTS warehouse_stocks_company_idx ON warehouse_stocks (company_id);
CREATE INDEX IF NOT EXISTS warehouse_stocks_warehouse_idx ON warehouse_stocks (warehouse_id);

CREATE TABLE IF NOT EXISTS inventory_orders (
    id BIGSERIAL PRIMARY KEY,
    company_id BIGINT NOT NULL,
    warehouse_id BIGINT NOT NULL REFERENCES warehouses (id) ON DELETE RESTRICT,
    document_no VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    surplus_deficit DOUBLE PRECISION NOT NULL DEFAULT 0,
    note TEXT,
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    updated_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT inventory_orders_status_check CHECK (status IN ('draft', 'posted', 'cancelled'))
);

CREATE UNIQUE INDEX IF NOT EXISTS inventory_orders_company_document_uidx
    ON inventory_orders (company_id, document_no);
CREATE INDEX IF NOT EXISTS inventory_orders_company_created_idx
    ON inventory_orders (company_id, created_at DESC);

CREATE TABLE IF NOT EXISTS inventory_order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES inventory_orders (id) ON DELETE CASCADE,
    product_id BIGINT REFERENCES products (id) ON DELETE SET NULL,
    product_name VARCHAR(200) NOT NULL,
    expected_qty DOUBLE PRECISION NOT NULL DEFAULT 0,
    actual_qty DOUBLE PRECISION NOT NULL DEFAULT 0,
    cost_price DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS inventory_order_items_order_idx ON inventory_order_items (order_id);

CREATE TABLE IF NOT EXISTS transfer_orders (
    id BIGSERIAL PRIMARY KEY,
    company_id BIGINT NOT NULL,
    document_no VARCHAR(64) NOT NULL,
    from_warehouse_id BIGINT NOT NULL REFERENCES warehouses (id) ON DELETE RESTRICT,
    to_warehouse_id BIGINT NOT NULL REFERENCES warehouses (id) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    note TEXT,
    transferred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    received_at TIMESTAMPTZ,
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT transfer_orders_status_check CHECK (status IN ('draft', 'posted', 'cancelled')),
    CONSTRAINT transfer_orders_diff_warehouse_check CHECK (from_warehouse_id != to_warehouse_id)
);

CREATE UNIQUE INDEX IF NOT EXISTS transfer_orders_company_document_uidx
    ON transfer_orders (company_id, document_no);
CREATE INDEX IF NOT EXISTS transfer_orders_company_transferred_idx
    ON transfer_orders (company_id, transferred_at DESC);

CREATE TABLE IF NOT EXISTS transfer_order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES transfer_orders (id) ON DELETE CASCADE,
    product_id BIGINT REFERENCES products (id) ON DELETE SET NULL,
    product_name VARCHAR(200) NOT NULL,
    quantity DOUBLE PRECISION NOT NULL,
    cost_price DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT transfer_order_items_qty_check CHECK (quantity > 0)
);

CREATE INDEX IF NOT EXISTS transfer_order_items_order_idx ON transfer_order_items (order_id);

CREATE TABLE IF NOT EXISTS writeoff_orders (
    id BIGSERIAL PRIMARY KEY,
    company_id BIGINT NOT NULL,
    warehouse_id BIGINT NOT NULL REFERENCES warehouses (id) ON DELETE RESTRICT,
    document_no VARCHAR(64) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    total_amount DOUBLE PRECISION NOT NULL DEFAULT 0,
    object_name VARCHAR(200),
    counterparty_name VARCHAR(200),
    note TEXT,
    written_off_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT writeoff_orders_status_check CHECK (status IN ('draft', 'posted', 'cancelled')),
    CONSTRAINT writeoff_orders_amount_check CHECK (total_amount >= 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS writeoff_orders_company_document_uidx
    ON writeoff_orders (company_id, document_no);
CREATE INDEX IF NOT EXISTS writeoff_orders_company_written_idx
    ON writeoff_orders (company_id, written_off_at DESC);

CREATE TABLE IF NOT EXISTS writeoff_order_items (
    id BIGSERIAL PRIMARY KEY,
    order_id BIGINT NOT NULL REFERENCES writeoff_orders (id) ON DELETE CASCADE,
    product_id BIGINT REFERENCES products (id) ON DELETE SET NULL,
    product_name VARCHAR(200) NOT NULL,
    quantity DOUBLE PRECISION NOT NULL,
    cost_price DOUBLE PRECISION NOT NULL DEFAULT 0,
    amount DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT writeoff_order_items_qty_check CHECK (quantity > 0)
);

CREATE INDEX IF NOT EXISTS writeoff_order_items_order_idx ON writeoff_order_items (order_id);

-- Seed default warehouse per company from existing purchase orders
INSERT INTO warehouses (company_id, name, status)
SELECT DISTINCT company_id, 'Основной склад', 'active'
FROM purchase_orders
ON CONFLICT (company_id, name) DO NOTHING;

-- Initialize stock from purchases minus sales
INSERT INTO warehouse_stocks (company_id, warehouse_id, product_id, quantity, cost_price)
WITH purchase_totals AS (
    SELECT
        po.company_id,
        poi.product_id,
        SUM(poi.quantity) AS purchased_qty,
        AVG(poi.price)   AS avg_price
    FROM purchase_order_items poi
    JOIN purchase_orders po ON po.id = poi.order_id AND po.status = 'posted'
    WHERE poi.product_id IS NOT NULL
    GROUP BY po.company_id, poi.product_id
),
sales_totals AS (
    SELECT
        so.company_id,
        soi.product_id,
        SUM(soi.quantity) AS sold_qty
    FROM sales_order_items soi
    JOIN sales_orders so ON so.id = soi.order_id AND so.status = 'posted'
    WHERE soi.product_id IS NOT NULL
    GROUP BY so.company_id, soi.product_id
)
SELECT
    pt.company_id,
    w.id AS warehouse_id,
    pt.product_id,
    GREATEST(pt.purchased_qty - COALESCE(st.sold_qty, 0), 0) AS quantity,
    pt.avg_price AS cost_price
FROM purchase_totals pt
JOIN warehouses w ON w.company_id = pt.company_id AND w.name = 'Основной склад'
LEFT JOIN sales_totals st ON st.company_id = pt.company_id AND st.product_id = pt.product_id
WHERE GREATEST(pt.purchased_qty - COALESCE(st.sold_qty, 0), 0) > 0
ON CONFLICT (warehouse_id, product_id) DO UPDATE
    SET quantity   = EXCLUDED.quantity,
        cost_price = EXCLUDED.cost_price,
        updated_at = NOW();

INSERT INTO permissions (code, module, resource, action)
VALUES
    ('warehouse.stock.read',       'warehouse', 'stock',     'read'),
    ('warehouse.inventory.read',   'warehouse', 'inventory', 'read'),
    ('warehouse.inventory.create', 'warehouse', 'inventory', 'create'),
    ('warehouse.inventory.post',   'warehouse', 'inventory', 'post'),
    ('warehouse.transfer.read',    'warehouse', 'transfer',  'read'),
    ('warehouse.transfer.create',  'warehouse', 'transfer',  'create'),
    ('warehouse.transfer.post',    'warehouse', 'transfer',  'post'),
    ('warehouse.writeoff.read',    'warehouse', 'writeoff',  'read'),
    ('warehouse.writeoff.create',  'warehouse', 'writeoff',  'create'),
    ('warehouse.writeoff.post',    'warehouse', 'writeoff',  'post'),
    ('warehouse.warehouses.read',  'warehouse', 'warehouses','read')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN (
    'warehouse.stock.read',
    'warehouse.inventory.read',
    'warehouse.inventory.create',
    'warehouse.inventory.post',
    'warehouse.transfer.read',
    'warehouse.transfer.create',
    'warehouse.transfer.post',
    'warehouse.writeoff.read',
    'warehouse.writeoff.create',
    'warehouse.writeoff.post',
    'warehouse.warehouses.read'
)
WHERE r.code IN ('owner', 'warehouse_operator')
ON CONFLICT (role_id, permission_id) DO NOTHING;