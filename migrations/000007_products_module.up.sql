CREATE TABLE IF NOT EXISTS products (
    id BIGSERIAL PRIMARY KEY,
    company_id BIGINT NOT NULL,
    name VARCHAR(200) NOT NULL,
    sku VARCHAR(64),
    barcode VARCHAR(64),
    unit VARCHAR(16) NOT NULL DEFAULT 'pcs',
    purchase_price DOUBLE PRECISION NOT NULL DEFAULT 0,
    sale_price DOUBLE PRECISION NOT NULL DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_by BIGINT REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT products_status_check CHECK (status IN ('active', 'inactive')),
    CONSTRAINT products_purchase_price_check CHECK (purchase_price >= 0),
    CONSTRAINT products_sale_price_check CHECK (sale_price >= 0)
);

CREATE INDEX IF NOT EXISTS products_company_status_idx ON products (company_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS products_company_name_idx ON products (company_id, name);
CREATE UNIQUE INDEX IF NOT EXISTS products_company_sku_uidx ON products (company_id, sku) WHERE sku IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS products_company_barcode_uidx ON products (company_id, barcode) WHERE barcode IS NOT NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'sales_order_items_product_id_fkey'
    ) THEN
        ALTER TABLE sales_order_items
            ADD CONSTRAINT sales_order_items_product_id_fkey
                FOREIGN KEY (product_id)
                    REFERENCES products (id)
                    ON DELETE SET NULL
                    NOT VALID;
    END IF;
END $$;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'purchase_order_items_product_id_fkey'
    ) THEN
        ALTER TABLE purchase_order_items
            ADD CONSTRAINT purchase_order_items_product_id_fkey
                FOREIGN KEY (product_id)
                    REFERENCES products (id)
                    ON DELETE SET NULL
                    NOT VALID;
    END IF;
END $$;

INSERT INTO permissions (code, module, resource, action)
VALUES
    ('products.read', 'products', 'product', 'read'),
    ('products.create', 'products', 'product', 'create')
ON CONFLICT (code) DO NOTHING;

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
JOIN permissions p ON p.code IN (
    'products.read',
    'products.create'
)
WHERE r.code IN ('owner', 'sales_manager', 'warehouse_operator')
ON CONFLICT (role_id, permission_id) DO NOTHING;
