package database

import (
	"strings"
	"testing"
)

func TestInitSQLEmbedded_Tables(t *testing.T) {
	if initSQL == "" {
		t.Fatal("expected embedded initSQL to be non-empty")
	}

	expectedTables := []string{
		"users", "accounts", "account_access", "categories", "reconcile_reports", "checking_records",
		"assets", "asset_values", "asset_stores", "asset_store_records",
		"account_assets", "account_snapshots", "captchas", "subscriptions", "user_settings",
		"perspectives", "perspective_settings", "perspective_accounts", "perspective_recurring_transactions", "perspective_access", "recurring_transactions",
	}
	for _, table := range expectedTables {
		if !strings.Contains(initSQL, "CREATE TABLE IF NOT EXISTS "+table) {
			t.Errorf("expected initSQL to contain CREATE TABLE IF NOT EXISTS for %s", table)
		}
	}
}

func TestInitSQLEmbedded_Subscriptions(t *testing.T) {
	expectedSubSnippets := []string{
		"CREATE TABLE IF NOT EXISTS subscriptions",
		"user_id UUID NOT NULL REFERENCES users(id)",
		"paid_date DATE NOT NULL",
		"subscriber_ends DATE NOT NULL",
		"idx_subscriptions_user_id",
		"idx_subscriptions_user_ends",
	}
	for _, snippet := range expectedSubSnippets {
		if !strings.Contains(initSQL, snippet) {
			t.Errorf("expected initSQL to contain %q", snippet)
		}
	}
}

func TestInitSQLEmbedded_Constraints(t *testing.T) {
	expectedConstraints := []string{
		"uq_asset_stores_account_asset_label UNIQUE (account_id, asset_id, label)",
		"uq_asset_stores_account_asset_lower_label ON asset_stores (account_id, asset_id, LOWER(label))",
		"uq_asset_store_records_store_date UNIQUE (asset_store_id, record_date)",
		"uq_account_snapshots_account_date UNIQUE (account_id, snapshot_date)",
		"uq_account_assets_account_asset_date UNIQUE (account_id, asset_id, asset_date)",
		"uq_assets_user_label UNIQUE NULLS NOT DISTINCT (user_id, label)",
		"uq_assets_user_lower_label ON assets (user_id, LOWER(label)) NULLS NOT DISTINCT",
		"uq_asset_values_asset_date UNIQUE (asset_id, value_date)",
		"uq_perspective_settings_perspective UNIQUE (perspective_id)",
		"uq_perspective_accounts_perspective_account UNIQUE (perspective_id, account_id)",
		"uq_perspective_recurring_tx UNIQUE (perspective_id, recurring_transaction_id)",
		"ALTER TABLE perspective_settings ADD COLUMN IF NOT EXISTS show_categories",
		"ALTER TABLE perspective_settings DROP COLUMN IF EXISTS calculate_average_return_yoy",
		"uq_perspective_access_perspective_user UNIQUE (perspective_id, user_id)",
		"uq_categories_user_name UNIQUE NULLS NOT DISTINCT (user_id, name)",
		"uq_categories_user_lower_name ON categories (user_id, LOWER(name)) NULLS NOT DISTINCT",
		"uq_user_settings_user UNIQUE (user_id)",
	}
	for _, c := range expectedConstraints {
		if !strings.Contains(initSQL, c) {
			t.Errorf("expected initSQL to contain constraint: %s", c)
		}
	}
}

func TestInitSQLEmbedded_Categories(t *testing.T) {
	expectedSnippets := []string{
		"CREATE TABLE IF NOT EXISTS categories",
		"idx_categories_user_id",
		"idx_checking_records_category_id",
		"category_id UUID REFERENCES categories(id) ON DELETE SET NULL",
		"'Housing'",
		"'Transportation'",
		"'Groceries'",
		"'Utilities'",
		"'Healthcare'",
		"'Insurance'",
		"'Debt'",
		"'Income'",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(initSQL, snippet) {
			t.Errorf("expected initSQL to contain %q", snippet)
		}
	}
}

func TestInitSQLEmbedded_ReconcileReports(t *testing.T) {
	expectedSnippets := []string{
		"CREATE TABLE IF NOT EXISTS reconcile_reports",
		"report_date DATE NOT NULL",
		"starting_balance DOUBLE PRECISION NOT NULL",
		"ending_balance DOUBLE PRECISION NOT NULL",
		"idx_reconcile_reports_account_date",
		"reconcile_report_id UUID REFERENCES reconcile_reports(id) ON DELETE SET NULL",
		"idx_checking_records_reconcile_report_id",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(initSQL, snippet) {
			t.Errorf("expected initSQL to contain %q", snippet)
		}
	}
}

func TestInitSQLEmbedded_RecurringTransactions(t *testing.T) {
	expectedSnippets := []string{
		"CREATE TABLE IF NOT EXISTS recurring_transactions",
		"user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE",
		"account_id UUID REFERENCES accounts(id) ON DELETE SET NULL",
		"category_id UUID REFERENCES categories(id) ON DELETE SET NULL",
		"amount DOUBLE PRECISION NOT NULL CHECK (amount <> 0)",
		"last_date_applied DATE",
		"frequency VARCHAR(20) NOT NULL CONSTRAINT chk_recurring_frequency CHECK (frequency IN ('weekly', 'bi-monthly', 'monthly', 'yearly'))",
		"interval INTEGER NOT NULL DEFAULT 1 CHECK (interval >= 1)",
		"schedule JSONB",
		"name VARCHAR(19) NOT NULL DEFAULT '' CHECK (char_length(name) < 20)",
		"entity VARCHAR(255) NOT NULL",
		"reminder BOOLEAN NOT NULL DEFAULT FALSE",
		"idx_recurring_transactions_user_id",
		"idx_recurring_transactions_account_id",
		"idx_recurring_transactions_category_id",
		"DROP INDEX IF EXISTS idx_bills_category_id",
		"DROP INDEX IF EXISTS idx_bills_account_id",
		"DROP INDEX IF EXISTS idx_bills_user_id",
		"DROP TABLE IF EXISTS bills CASCADE",
		"DROP INDEX IF EXISTS idx_incomes_category_id",
		"DROP INDEX IF EXISTS idx_incomes_account_id",
		"DROP INDEX IF EXISTS idx_incomes_user_id",
		"DROP TABLE IF EXISTS incomes CASCADE",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(initSQL, snippet) {
			t.Errorf("expected initSQL to contain %q", snippet)
		}
	}
}

func TestInitSQLEmbedded_UserDeleteCascades(t *testing.T) {
	expected := []string{
		"admin_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE",
		"user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE",
		"user_id UUID REFERENCES users(id) ON DELETE CASCADE",
	}
	for _, snippet := range expected {
		if !strings.Contains(initSQL, snippet) {
			t.Errorf("expected initSQL to contain cascade snippet %q", snippet)
		}
	}
}

func TestInitSQLEmbedded_PolicyAgreements(t *testing.T) {
	expectedSnippets := []string{
		"CREATE TABLE IF NOT EXISTS policy_agreements",
		"user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE",
		"terms_sha256 VARCHAR(64) NOT NULL",
		"privacy_sha256 VARCHAR(64) NOT NULL",
		"terms_approved BOOLEAN NOT NULL",
		"privacy_approved BOOLEAN NOT NULL",
		"us_adult_consumer_confirmed BOOLEAN NOT NULL",
		"active BOOLEAN NOT NULL DEFAULT TRUE",
		"client_ip VARCHAR(64) NOT NULL",
		"agreed_at TIMESTAMPTZ NOT NULL DEFAULT NOW()",
		"idx_policy_agreements_user_active",
		"idx_policy_agreements_user_agreed_at",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(initSQL, snippet) {
			t.Errorf("expected initSQL to contain %q", snippet)
		}
	}
}

func TestInitSQLEmbedded_UserRecoveryKeys(t *testing.T) {
	expectedSnippets := []string{
		"CREATE TABLE IF NOT EXISTS user_recovery_keys",
		"user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE",
		"key_hash VARCHAR(255) NOT NULL",
		"key_hint VARCHAR(32) NOT NULL DEFAULT ''",
		"active BOOLEAN NOT NULL DEFAULT TRUE",
		"created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()",
		"used_at TIMESTAMPTZ",
		"used_ip VARCHAR(64) NOT NULL DEFAULT ''",
		"idx_user_recovery_keys_user_active",
		"idx_user_recovery_keys_user_created",
	}
	for _, snippet := range expectedSnippets {
		if !strings.Contains(initSQL, snippet) {
			t.Errorf("expected initSQL to contain %q", snippet)
		}
	}
}
