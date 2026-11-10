package database

import (
	"strings"
	"testing"
)

func TestInitSQLEmbedded(t *testing.T) {
	if initSQL == "" {
		t.Fatal("expected embedded initSQL to be non-empty")
	}

	expectedTables := []string{"users", "accounts", "account_access", "checking_records"}
	for _, table := range expectedTables {
		if !strings.Contains(initSQL, "CREATE TABLE IF NOT EXISTS "+table) {
			t.Errorf("expected initSQL to contain CREATE TABLE IF NOT EXISTS for %s", table)
		}
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
}
