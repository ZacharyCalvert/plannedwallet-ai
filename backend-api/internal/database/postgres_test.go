package database

import (
	"strings"
	"testing"
)

func TestInitSQLEmbedded(t *testing.T) {
	if initSQL == "" {
		t.Fatal("expected embedded initSQL to be non-empty")
	}

	expectedTables := []string{"users", "accounts", "account_access", "checking_records", "assets", "asset_values", "account_assets", "account_snapshots"}
	for _, table := range expectedTables {
		if !strings.Contains(initSQL, "CREATE TABLE IF NOT EXISTS "+table) {
			t.Errorf("expected initSQL to contain CREATE TABLE IF NOT EXISTS for %s", table)
		}
	}

	if !strings.Contains(initSQL, "uq_account_snapshots_account_date UNIQUE (account_id, snapshot_date)") {
		t.Error("expected initSQL to contain uq_account_snapshots_account_date constraint")
	}

	if !strings.Contains(initSQL, "uq_account_assets_account_asset UNIQUE (account_id, asset_id)") {
		t.Error("expected initSQL to contain uq_account_assets_account_asset constraint")
	}

	if !strings.Contains(initSQL, "uq_assets_user_label UNIQUE NULLS NOT DISTINCT (user_id, label)") {
		t.Error("expected initSQL to contain uq_assets_user_label constraint")
	}

	if !strings.Contains(initSQL, "uq_assets_user_lower_label ON assets (user_id, LOWER(label)) NULLS NOT DISTINCT") {
		t.Error("expected initSQL to contain uq_assets_user_lower_label index")
	}

	if !strings.Contains(initSQL, "uq_asset_values_asset_date UNIQUE (asset_id, value_date)") {
		t.Error("expected initSQL to contain uq_asset_values_asset_date constraint")
	}

	if !strings.Contains(initSQL, "label VARCHAR(100) NOT NULL") {
		t.Error("expected initSQL to contain label column definition in asset_values table")
	}

	if !strings.Contains(initSQL, "value_date DATE NOT NULL") {
		t.Error("expected initSQL to contain value_date column definition in asset_values table")
	}

	if !strings.Contains(initSQL, "admin_id UUID NOT NULL REFERENCES users(id)") {
		t.Error("expected initSQL to contain admin_id column definition in accounts table")
	}

	if !strings.Contains(initSQL, "name VARCHAR(100) NOT NULL") {
		t.Error("expected initSQL to contain name column definition in accounts table")
	}

	if !strings.Contains(initSQL, "access_level VARCHAR(20) NOT NULL") {
		t.Error("expected initSQL to contain access_level column definition in account_access table")
	}

	if !strings.Contains(initSQL, "totp_secret VARCHAR(64) NOT NULL") {
		t.Error("expected initSQL to contain totp_secret column definition in users table")
	}
}
