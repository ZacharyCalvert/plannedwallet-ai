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
		"users", "accounts", "account_access", "checking_records",
		"assets", "asset_values", "asset_stores", "asset_store_records",
		"account_assets", "account_snapshots", "captchas", "subscriptions",
		"perspectives", "perspective_settings", "perspective_accounts", "perspective_access",
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
		"uq_asset_stores_account_label UNIQUE (account_id, label)",
		"uq_asset_store_records_store_date UNIQUE (asset_store_id, record_date)",
		"uq_account_snapshots_account_date UNIQUE (account_id, snapshot_date)",
		"uq_account_assets_account_asset_date UNIQUE (account_id, asset_id, asset_date)",
		"uq_assets_user_label UNIQUE NULLS NOT DISTINCT (user_id, label)",
		"uq_assets_user_lower_label ON assets (user_id, LOWER(label)) NULLS NOT DISTINCT",
		"uq_asset_values_asset_date UNIQUE (asset_id, value_date)",
		"uq_perspective_settings_perspective UNIQUE (perspective_id)",
		"uq_perspective_accounts_perspective_account UNIQUE (perspective_id, account_id)",
		"uq_perspective_access_perspective_user UNIQUE (perspective_id, user_id)",
	}
	for _, c := range expectedConstraints {
		if !strings.Contains(initSQL, c) {
			t.Errorf("expected initSQL to contain constraint: %s", c)
		}
	}
}
