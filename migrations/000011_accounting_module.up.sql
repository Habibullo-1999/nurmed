CREATE TABLE IF NOT EXISTS cash_registers (
    id         BIGSERIAL PRIMARY KEY,
    company_id BIGINT       NOT NULL REFERENCES companies (id) ON DELETE CASCADE,
    name       VARCHAR(200) NOT NULL,
    currency   VARCHAR(8)   NOT NULL DEFAULT 'TJS',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS cash_transactions (
    id               BIGSERIAL PRIMARY KEY,
    company_id       BIGINT      NOT NULL,
    cash_register_id BIGINT      NOT NULL REFERENCES cash_registers (id) ON DELETE CASCADE,
    doc_type         VARCHAR(50),
    doc_no           VARCHAR(64),
    income           DOUBLE PRECISION NOT NULL DEFAULT 0,
    expense          DOUBLE PRECISION NOT NULL DEFAULT 0,
    category         VARCHAR(200),
    note             TEXT,
    created_by       BIGINT REFERENCES users (id),
    transacted_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS payment_documents (
    id             BIGSERIAL PRIMARY KEY,
    company_id     BIGINT      NOT NULL,
    document_no    VARCHAR(64) NOT NULL,
    doc_date       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    doc_type       VARCHAR(50) NOT NULL,
    debit_account  VARCHAR(100),
    credit_account VARCHAR(100),
    income         DOUBLE PRECISION NOT NULL DEFAULT 0,
    expense        DOUBLE PRECISION NOT NULL DEFAULT 0,
    category       VARCHAR(200),
    note           TEXT,
    organization   VARCHAR(200),
    created_by     BIGINT REFERENCES users (id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS counterparties (
    id         BIGSERIAL PRIMARY KEY,
    company_id BIGINT       NOT NULL,
    name       VARCHAR(200) NOT NULL,
    group_type VARCHAR(50),
    phone      VARCHAR(50),
    region     VARCHAR(100),
    currency   VARCHAR(8)   NOT NULL DEFAULT 'TJS',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS counterparty_transactions (
    id              BIGSERIAL PRIMARY KEY,
    company_id      BIGINT           NOT NULL,
    counterparty_id BIGINT           NOT NULL REFERENCES counterparties (id) ON DELETE CASCADE,
    amount          DOUBLE PRECISION NOT NULL DEFAULT 0,
    currency        VARCHAR(8)       NOT NULL DEFAULT 'TJS',
    transacted_at   TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    created_at      TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS debt_records (
    id                BIGSERIAL PRIMARY KEY,
    company_id        BIGINT      NOT NULL,
    counterparty_id   BIGINT REFERENCES counterparties (id),
    client_name       VARCHAR(200),
    phone             VARCHAR(50),
    period            VARCHAR(50),
    start_date        TIMESTAMPTZ,
    next_payment_date TIMESTAMPTZ,
    last_payment_date TIMESTAMPTZ,
    balance           DOUBLE PRECISION NOT NULL DEFAULT 0,
    client_text       TEXT,
    admin_text        TEXT,
    note              TEXT,
    channels          VARCHAR(200),
    status            VARCHAR(50) NOT NULL DEFAULT 'active',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS price_lists (
    id              BIGSERIAL PRIMARY KEY,
    company_id      BIGINT       NOT NULL,
    name            VARCHAR(200) NOT NULL,
    note            TEXT,
    price_list_type VARCHAR(50)  NOT NULL DEFAULT 'nomenclature',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS currency_rates (
    id         BIGSERIAL PRIMARY KEY,
    company_id BIGINT           NOT NULL,
    currency   VARCHAR(8)       NOT NULL,
    rate       DOUBLE PRECISION NOT NULL DEFAULT 1,
    rate_date  TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    created_by BIGINT REFERENCES users (id),
    created_at TIMESTAMPTZ      NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payment_documents_company_date ON payment_documents (company_id, doc_date DESC);
CREATE INDEX IF NOT EXISTS idx_cash_transactions_company_register ON cash_transactions (company_id, cash_register_id, transacted_at DESC);
CREATE INDEX IF NOT EXISTS idx_counterparties_company ON counterparties (company_id);
CREATE INDEX IF NOT EXISTS idx_counterparty_transactions_company ON counterparty_transactions (company_id, counterparty_id);
CREATE INDEX IF NOT EXISTS idx_debt_records_company_status ON debt_records (company_id, status);
CREATE INDEX IF NOT EXISTS idx_currency_rates_company ON currency_rates (company_id, currency, rate_date DESC);
CREATE INDEX IF NOT EXISTS idx_cash_registers_company ON cash_registers (company_id);
