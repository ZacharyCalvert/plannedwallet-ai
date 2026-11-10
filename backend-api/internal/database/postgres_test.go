package database

import (
	"strings"
	"testing"
)

func TestInitSQLEmbedded(t *testing.T) {
	if initSQL == "" {
		t.Fatal("expected embedded initSQL to be non-empty")
	}

	expectedTables := []string{"users", "accounts", "checking_records"}
	for _, table := range expectedTables {
		if !strings.Contains(initSQL, "CREATE TABLE IF NOT EXISTS "+table) {
			t.Errorf("expected initSQL to contain CREATE TABLE IF NOT EXISTS for %s", table)
		}
	}

	if !strings.Contains(initSQL, "name VARCHAR(100) NOT NULL") {
		t.Error("expected initSQL to contain name column definition in accounts table")
	}
}
