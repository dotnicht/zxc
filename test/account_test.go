package test

import (
	"fmt"
	"testing"
	"time"
)

// sharedDeploy runs one deploy for a given tenant and waits for a profile to appear.
// Returns the first profile ID. Fails the test if no profile appears within 90s.
func sharedDeploy(t *testing.T, ts int64, idx int) (tenantName, profileID string) {
	t.Helper()
	_, _, targetID, payloadID, tenantName := setupTenantWithDeps(t, ts, idx)

	releaseAdd := parseKVOutput(t, runTenantClient(t, tenantName,
		"release", "add", "--target", targetID, "--payload", payloadID,
	))
	runTenantClient(t, tenantName, "release", "deploy", "--id", releaseAdd["id"])

	adb := tenantAccountDB(t, tenantName)
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		_ = adb.QueryRow(`SELECT id::text FROM profiles WHERE deleted_at IS NULL LIMIT 1`).Scan(&profileID)
		if profileID != "" {
			return tenantName, profileID
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatal("no profile appeared in account DB within timeout")
	return
}

func TestAccountList(t *testing.T) {
	ts := time.Now().UnixNano()
	tenantName, _ := sharedDeploy(t, ts, 0)

	out := runTenantClient(t, tenantName, "account", "list")
	id := firstDataID(t, out)
	if id == "" {
		t.Fatal("expected at least one account in list")
	}
}

func TestAccountGet(t *testing.T) {
	ts := time.Now().UnixNano()
	tenantName, profileID := sharedDeploy(t, ts, 1)

	got := parseKVOutput(t, runTenantClient(t, tenantName, "account", "get", "--id", profileID))
	if got["id"] != profileID {
		t.Fatalf("get returned wrong id: %q", got["id"])
	}
	if got["name"] == "" {
		t.Fatal("expected non-empty name")
	}

	adb := tenantAccountDB(t, tenantName)
	var dbName string
	if err := adb.QueryRow(`SELECT name FROM profiles WHERE id = $1`, profileID).Scan(&dbName); err != nil {
		t.Fatalf("query profile name: %v", err)
	}
	if dbName != got["name"] {
		t.Fatalf("name mismatch: client=%q db=%q", got["name"], dbName)
	}
}

func TestAccountDisable(t *testing.T) {
	ts := time.Now().UnixNano()
	tenantName, profileID := sharedDeploy(t, ts, 2)

	got := parseKVOutput(t, runTenantClient(t, tenantName, "account", "disable", "--id", profileID))
	if got["status"] != "disabled" {
		t.Fatalf("expected status=disabled, got %q", got["status"])
	}

	adb := tenantAccountDB(t, tenantName)
	var dbStatus string
	if err := adb.QueryRow(`SELECT status FROM profiles WHERE id = $1`, profileID).Scan(&dbStatus); err != nil {
		t.Fatalf("query profile status: %v", err)
	}
	if dbStatus != "disabled" {
		t.Fatalf("DB status after disable: %q", dbStatus)
	}

	_ = fmt.Sprintf("disabled account %s", profileID)
}
