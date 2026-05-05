package test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPayloadAdd(t *testing.T) {
	ts := time.Now().UnixNano()
	name := fmt.Sprintf("pldadd%d", ts)
	runClient(t, "tenant", "add", "--name", name)

	zipContent := buildFixtureZip(t)
	tmpZip := filepath.Join(t.TempDir(), "payload.zip")
	if err := os.WriteFile(tmpZip, zipContent, 0o644); err != nil {
		t.Fatalf("write payload zip: %v", err)
	}

	out := parseKVOutput(t, runTenantClient(t, name,
		"payload", "add",
		"--file", tmpZip,
		"--config", "script.conf",
		"--start", "bash ~/script.sh",
		"--stop", "true",
	))
	id := out["id"]
	if id == "" {
		t.Fatal("expected payload id in response")
	}

	db := tenantDeployDB(t, name)
	var got string
	if err := db.QueryRow(`SELECT id::text FROM payloads WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&got); err != nil {
		t.Fatalf("payload not found in deploy DB: %v", err)
	}
}

func TestPayloadList(t *testing.T) {
	ts := time.Now().UnixNano()
	name := fmt.Sprintf("pldlist%d", ts)
	runClient(t, "tenant", "add", "--name", name)

	makeZip := func() string {
		zip := buildFixtureZip(t)
		p := filepath.Join(t.TempDir(), "p.zip")
		if err := os.WriteFile(p, zip, 0o644); err != nil {
			t.Fatalf("write zip: %v", err)
		}
		return p
	}

	r1 := parseKVOutput(t, runTenantClient(t, name,
		"payload", "add", "--file", makeZip(), "--config", "script.conf",
		"--start", "bash ~/script.sh", "--stop", "true",
	))
	r2 := parseKVOutput(t, runTenantClient(t, name,
		"payload", "add", "--file", makeZip(), "--config", "script.conf",
		"--start", "bash ~/script.sh", "--stop", "true",
	))

	out := runTenantClient(t, name, "payload", "list")
	if !strings.Contains(out, r1["id"]) {
		t.Fatalf("list missing payload %s", r1["id"])
	}
	if !strings.Contains(out, r2["id"]) {
		t.Fatalf("list missing payload %s", r2["id"])
	}
}
