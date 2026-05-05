package test

import (
	"testing"
)

func TestSessionStartStop(t *testing.T) {
	t.Parallel()
	adb := tenantAccountDB(t, sharedTenantName)
	var sessionID string
	if err := adb.QueryRow(
		`INSERT INTO sessions (profile_id, status) VALUES ($1, 'offline') RETURNING id::text`,
		sharedProfileID,
	).Scan(&sessionID); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	started := parseKVOutput(t, runTenantClient(t, sharedTenantName, "session", "start", "--id", sessionID))
	if started["status"] != "online" {
		t.Fatalf("expected status=online after start, got %q", started["status"])
	}
	var dbStatus string
	if err := adb.QueryRow(`SELECT status FROM sessions WHERE id = $1`, sessionID).Scan(&dbStatus); err != nil {
		t.Fatalf("query session: %v", err)
	}
	if dbStatus != "online" {
		t.Fatalf("DB status after start: %q", dbStatus)
	}

	stopped := parseKVOutput(t, runTenantClient(t, sharedTenantName, "session", "stop", "--id", sessionID))
	if stopped["status"] != "offline" {
		t.Fatalf("expected status=offline after stop, got %q", stopped["status"])
	}
	if err := adb.QueryRow(`SELECT status FROM sessions WHERE id = $1`, sessionID).Scan(&dbStatus); err != nil {
		t.Fatalf("query session: %v", err)
	}
	if dbStatus != "offline" {
		t.Fatalf("DB status after stop: %q", dbStatus)
	}
}

func TestSessionList(t *testing.T) {
	t.Parallel()
	adb := tenantAccountDB(t, sharedTenantName)
	if _, err := adb.Exec(
		`INSERT INTO sessions (profile_id, status) VALUES ($1, 'offline')`, sharedProfileID,
	); err != nil {
		t.Fatalf("insert session: %v", err)
	}

	out := runTenantClient(t, sharedTenantName, "session", "list")
	if out == "" {
		t.Fatal("expected non-empty session list output")
	}
}
