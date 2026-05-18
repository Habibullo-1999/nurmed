CREATE TABLE IF NOT EXISTS companies (
    id         BIGSERIAL    PRIMARY KEY,
    name       VARCHAR(255) NOT NULL,
    status     VARCHAR(50)  NOT NULL DEFAULT 'active',
    deleted_at TIMESTAMP    NULL,
    created_at TIMESTAMP    NOT NULL DEFAULT now(),
    updated_at TIMESTAMP    NOT NULL DEFAULT now()
);

INSERT INTO companies (name) VALUES ('Default');

ALTER TABLE users
    ADD CONSTRAINT fk_users_company_id
    FOREIGN KEY (company_id) REFERENCES companies(id);
