CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    totp_secret VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_users_lower_email ON users (LOWER(email));

CREATE TABLE IF NOT EXISTS accounts (
    id UUID PRIMARY KEY,
    admin_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    type VARCHAR(50) NOT NULL,
    description VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS account_access (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    access_level VARCHAR(20) NOT NULL CHECK (access_level IN ('read', 'edit', 'admin')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_account_access_account_user UNIQUE (account_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_account_access_user_id ON account_access(user_id);
CREATE INDEX IF NOT EXISTS idx_account_access_account_id ON account_access(account_id);

CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_categories_user_name UNIQUE NULLS NOT DISTINCT (user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_categories_user_id ON categories(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_categories_user_lower_name ON categories (user_id, LOWER(name)) NULLS NOT DISTINCT;

INSERT INTO categories (id, user_id, name) VALUES
    ('00000000-0000-4000-8000-000000000001', NULL, 'Housing'),
    ('00000000-0000-4000-8000-000000000002', NULL, 'Transportation'),
    ('00000000-0000-4000-8000-000000000003', NULL, 'Groceries'),
    ('00000000-0000-4000-8000-000000000004', NULL, 'Utilities'),
    ('00000000-0000-4000-8000-000000000005', NULL, 'Healthcare'),
    ('00000000-0000-4000-8000-000000000006', NULL, 'Insurance'),
    ('00000000-0000-4000-8000-000000000007', NULL, 'Debt')
ON CONFLICT DO NOTHING;

CREATE TABLE IF NOT EXISTS reconcile_reports (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL DEFAULT '',
    report_date DATE NOT NULL,
    starting_balance DOUBLE PRECISION NOT NULL,
    ending_balance DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_reconcile_reports_account_date ON reconcile_reports(account_id, report_date DESC);

CREATE TABLE IF NOT EXISTS checking_records (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    reconcile_report_id UUID REFERENCES reconcile_reports(id) ON DELETE SET NULL,
    value DOUBLE PRECISION NOT NULL,
    date_sent DATE NOT NULL,
    date_cleared DATE,
    entity VARCHAR(255) NOT NULL,
    description VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE checking_records ADD COLUMN IF NOT EXISTS category_id UUID REFERENCES categories(id) ON DELETE SET NULL;
ALTER TABLE checking_records ADD COLUMN IF NOT EXISTS reconcile_report_id UUID REFERENCES reconcile_reports(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_checking_records_account_datesent ON checking_records(account_id, date_sent DESC);
CREATE INDEX IF NOT EXISTS idx_checking_records_category_id ON checking_records(category_id);
CREATE INDEX IF NOT EXISTS idx_checking_records_reconcile_report_id ON checking_records(reconcile_report_id);

CREATE TABLE IF NOT EXISTS assets (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    label VARCHAR(100) NOT NULL,
    description VARCHAR(200) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_assets_user_label UNIQUE NULLS NOT DISTINCT (user_id, label)
);

CREATE INDEX IF NOT EXISTS idx_assets_user_id ON assets(user_id);
CREATE INDEX IF NOT EXISTS idx_assets_label ON assets(label);
CREATE UNIQUE INDEX IF NOT EXISTS uq_assets_user_lower_label ON assets (user_id, LOWER(label)) NULLS NOT DISTINCT;

CREATE TABLE IF NOT EXISTS asset_values (
    id UUID PRIMARY KEY,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    value DOUBLE PRECISION NOT NULL,
    value_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_asset_values_asset_date UNIQUE (asset_id, value_date)
);

CREATE INDEX IF NOT EXISTS idx_asset_values_asset_date ON asset_values(asset_id, value_date DESC);

CREATE TABLE IF NOT EXISTS asset_stores (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    label VARCHAR(100) NOT NULL,
    description VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_asset_stores_account_label UNIQUE (account_id, label)
);

CREATE INDEX IF NOT EXISTS idx_asset_stores_account_id ON asset_stores(account_id);
CREATE INDEX IF NOT EXISTS idx_asset_stores_asset_id ON asset_stores(asset_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_asset_stores_account_lower_label ON asset_stores (account_id, LOWER(label));

CREATE TABLE IF NOT EXISTS asset_store_records (
    id UUID PRIMARY KEY,
    asset_store_id UUID NOT NULL REFERENCES asset_stores(id) ON DELETE CASCADE,
    quantity DOUBLE PRECISION NOT NULL CHECK (quantity > 0),
    record_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_asset_store_records_store_date UNIQUE (asset_store_id, record_date)
);

CREATE INDEX IF NOT EXISTS idx_asset_store_records_store_id ON asset_store_records(asset_store_id);
CREATE INDEX IF NOT EXISTS idx_asset_store_records_store_date ON asset_store_records(asset_store_id, record_date DESC);

CREATE TABLE IF NOT EXISTS account_assets (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    quantity DOUBLE PRECISION NOT NULL CHECK (quantity > 0),
    asset_date DATE NOT NULL DEFAULT CURRENT_DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_account_assets_account_asset_date UNIQUE (account_id, asset_id, asset_date)
);

CREATE INDEX IF NOT EXISTS idx_account_assets_account_id ON account_assets(account_id);
CREATE INDEX IF NOT EXISTS idx_account_assets_asset_id ON account_assets(asset_id);
CREATE INDEX IF NOT EXISTS idx_account_assets_account_date ON account_assets(account_id, asset_date DESC);

CREATE TABLE IF NOT EXISTS account_snapshots (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    value DOUBLE PRECISION NOT NULL,
    snapshot_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_account_snapshots_account_date UNIQUE (account_id, snapshot_date)
);

CREATE INDEX IF NOT EXISTS idx_account_snapshots_account_date ON account_snapshots(account_id, snapshot_date ASC);

CREATE TABLE IF NOT EXISTS captchas (
    id VARCHAR(64) PRIMARY KEY,
    digits BYTEA NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_captchas_created_at ON captchas(created_at);

CREATE TABLE IF NOT EXISTS subscriptions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    paid_date DATE NOT NULL,
    subscriber_ends DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_user_id ON subscriptions(user_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_ends ON subscriptions(user_id, subscriber_ends DESC);

CREATE TABLE IF NOT EXISTS policy_agreements (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    terms_sha256 VARCHAR(64) NOT NULL,
    privacy_sha256 VARCHAR(64) NOT NULL DEFAULT '',
    terms_approved BOOLEAN NOT NULL DEFAULT FALSE,
    privacy_approved BOOLEAN NOT NULL DEFAULT FALSE,
    us_adult_consumer_confirmed BOOLEAN NOT NULL DEFAULT FALSE,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    client_ip VARCHAR(64) NOT NULL,
    agreed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE policy_agreements ADD COLUMN IF NOT EXISTS privacy_sha256 VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE policy_agreements ADD COLUMN IF NOT EXISTS us_adult_consumer_confirmed BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE policy_agreements DROP COLUMN IF EXISTS us_customer;

CREATE INDEX IF NOT EXISTS idx_policy_agreements_user_active ON policy_agreements(user_id, active);
CREATE INDEX IF NOT EXISTS idx_policy_agreements_user_agreed_at ON policy_agreements(user_id, agreed_at DESC);

CREATE TABLE IF NOT EXISTS user_recovery_keys (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key_hash VARCHAR(255) NOT NULL,
    key_hint VARCHAR(32) NOT NULL DEFAULT '',
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    used_at TIMESTAMPTZ,
    used_ip VARCHAR(64) NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_user_recovery_keys_user_active ON user_recovery_keys(user_id, active);
CREATE INDEX IF NOT EXISTS idx_user_recovery_keys_user_created ON user_recovery_keys(user_id, created_at DESC);

CREATE TABLE IF NOT EXISTS perspectives (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(255) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_perspectives_user_id ON perspectives(user_id);

CREATE TABLE IF NOT EXISTS perspective_settings (
    id UUID PRIMARY KEY,
    perspective_id UUID NOT NULL REFERENCES perspectives(id) ON DELETE CASCADE,
    show_financial_trend BOOLEAN NOT NULL DEFAULT TRUE,
    calculate_average_daily_spend BOOLEAN NOT NULL DEFAULT FALSE,
    calculate_average_return_yoy BOOLEAN NOT NULL DEFAULT FALSE,
    track_ruined_date BOOLEAN NOT NULL DEFAULT FALSE,
    custom_daily_spend DOUBLE PRECISION,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_perspective_settings_perspective UNIQUE (perspective_id)
);

CREATE INDEX IF NOT EXISTS idx_perspective_settings_perspective_id ON perspective_settings(perspective_id);

CREATE TABLE IF NOT EXISTS perspective_accounts (
    id UUID PRIMARY KEY,
    perspective_id UUID NOT NULL REFERENCES perspectives(id) ON DELETE CASCADE,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_perspective_accounts_perspective_account UNIQUE (perspective_id, account_id)
);

CREATE INDEX IF NOT EXISTS idx_perspective_accounts_perspective_id ON perspective_accounts(perspective_id);
CREATE INDEX IF NOT EXISTS idx_perspective_accounts_account_id ON perspective_accounts(account_id);

CREATE TABLE IF NOT EXISTS perspective_access (
    id UUID PRIMARY KEY,
    perspective_id UUID NOT NULL REFERENCES perspectives(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    access_level VARCHAR(20) NOT NULL CHECK (access_level IN ('read', 'edit', 'admin')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_perspective_access_perspective_user UNIQUE (perspective_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_perspective_access_perspective_id ON perspective_access(perspective_id);
CREATE INDEX IF NOT EXISTS idx_perspective_access_user_id ON perspective_access(user_id);

CREATE TABLE IF NOT EXISTS recurring_transactions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    account_id UUID REFERENCES accounts(id) ON DELETE SET NULL,
    category_id UUID REFERENCES categories(id) ON DELETE SET NULL,
    amount DOUBLE PRECISION NOT NULL CHECK (amount <> 0),
    last_date_applied DATE,
    frequency VARCHAR(20) NOT NULL CONSTRAINT chk_recurring_frequency CHECK (frequency IN ('weekly', 'bi-monthly', 'monthly', 'yearly')),
    interval INTEGER NOT NULL DEFAULT 1 CHECK (interval >= 1),
    schedule JSONB,
    name VARCHAR(19) NOT NULL DEFAULT '' CHECK (char_length(name) < 20),
    entity VARCHAR(255) NOT NULL,
    description VARCHAR(500) NOT NULL DEFAULT '',
    reminder BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recurring_transactions_user_id ON recurring_transactions(user_id);
CREATE INDEX IF NOT EXISTS idx_recurring_transactions_account_id ON recurring_transactions(account_id);
CREATE INDEX IF NOT EXISTS idx_recurring_transactions_category_id ON recurring_transactions(category_id);

ALTER TABLE recurring_transactions DROP CONSTRAINT IF EXISTS recurring_transactions_frequency_check;
ALTER TABLE recurring_transactions DROP CONSTRAINT IF EXISTS chk_recurring_frequency;
ALTER TABLE recurring_transactions ADD CONSTRAINT chk_recurring_frequency CHECK (frequency IN ('weekly', 'bi-monthly', 'monthly', 'yearly'));

DROP INDEX IF EXISTS idx_bills_category_id;
DROP INDEX IF EXISTS idx_bills_account_id;
DROP INDEX IF EXISTS idx_bills_user_id;
DROP TABLE IF EXISTS bills CASCADE;
DROP INDEX IF EXISTS idx_incomes_category_id;
DROP INDEX IF EXISTS idx_incomes_account_id;
DROP INDEX IF EXISTS idx_incomes_user_id;
DROP TABLE IF EXISTS incomes CASCADE;





