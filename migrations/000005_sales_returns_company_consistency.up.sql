CREATE UNIQUE INDEX IF NOT EXISTS sales_orders_id_company_uidx
    ON sales_orders (id, company_id);

ALTER TABLE sales_returns
    DROP CONSTRAINT IF EXISTS sales_returns_order_id_fkey;

ALTER TABLE sales_returns
    ADD CONSTRAINT sales_returns_order_company_fkey
        FOREIGN KEY (order_id, company_id)
            REFERENCES sales_orders (id, company_id)
            ON DELETE RESTRICT;
