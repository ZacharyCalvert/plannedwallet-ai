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

CREATE TABLE IF NOT EXISTS checking_records (
    id UUID PRIMARY KEY,
    account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    value DOUBLE PRECISION NOT NULL,
    date_sent DATE NOT NULL,
    date_cleared DATE,
    entity VARCHAR(255) NOT NULL,
    description VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_checking_records_account_datesent ON checking_records(account_id, date_sent DESC);

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




