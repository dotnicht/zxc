package test

import (
	"fmt"
	"testing"
)

func TestAccountList(t *testing.T) {
	t.Parallel()
	out := runTenant(t, sharedTenantName, "account", "list")
	id := firstID(t, out)
	if id == "" {
		t.Fatal("expected at least one account in list")
	}
}

func TestAccountGet(t *testing.T) {
	t.Parallel()
	got := parseKV(t, runTenant(t, sharedTenantName, "account", "get", "--id", sharedProfileID))
	if got["id"] != sharedProfileID {
		t.Fatalf("get returned wrong id: %q", got["id"])
	}
	if got["name"] == "" {
		t.Fatal("expected non-empty name")
	}

	adb := tenantAccountDB(t, sharedTenantName)
	var dbName string
	if err := adb.QueryRow(`SELECT name FROM profiles WHERE id = $1`, sharedProfileID).Scan(&dbName); err != nil {
		t.Fatalf("query profile name: %v", err)
	}
	if dbName != got["name"] {
		t.Fatalf("name mismatch: client=%q db=%q", got["name"], dbName)
	}
}

func TestAccountDisable(t *testing.T) {
	t.Parallel()
	// Insert a fresh profile so we don't mutate the shared one
	adb := tenantAccountDB(t, sharedTenantName)
	var profileID string
	if err := adb.QueryRow(
		`INSERT INTO profiles (name, status) VALUES ('disabletest', 'unknown') RETURNING id::text`,
	).Scan(&profileID); err != nil {
		t.Fatalf("insert profile: %v", err)
	}

	got := parseKV(t, runTenant(t, sharedTenantName, "account", "disable", "--id", profileID))
	if got["status"] != "disabled" {
		t.Fatalf("expected status=disabled, got %q", got["status"])
	}

	var dbStatus string
	if err := adb.QueryRow(`SELECT status FROM profiles WHERE id = $1`, profileID).Scan(&dbStatus); err != nil {
		t.Fatalf("query profile status: %v", err)
	}
	if dbStatus != "disabled" {
		t.Fatalf("DB status after disable: %q", dbStatus)
	}

	_ = fmt.Sprintf("disabled account %s", profileID)
}
