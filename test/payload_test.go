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
	t.Parallel()
	ts := time.Now().UnixNano()
	name := fmt.Sprintf("pldadd%d", ts)
	runClient(t, "tenant", "add", "--name", name)
	systemID := firstID(t, runTenant(t, name, "system", "list"))

	zipContent := buildZip(t)
	tmpZip := filepath.Join(t.TempDir(), "payload.zip")
	if err := os.WriteFile(tmpZip, zipContent, 0o644); err != nil {
		t.Fatalf("write payload zip: %v", err)
	}

	out := parseKV(t, runTenant(t, name,
		"payload", "add",
		"--file", tmpZip,
		"--config", "script.conf",
		"--start", "bash ~/script.sh",
		"--stop", "true",
		"--system", systemID,
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

func TestPayloadGet(t *testing.T) {
	t.Parallel()
	ts := time.Now().UnixNano()
	name := fmt.Sprintf("pldget%d", ts)
	runClient(t, "tenant", "add", "--name", name)
	systemID := firstID(t, runTenant(t, name, "system", "list"))

	zipContent := buildZip(t)
	tmpZip := filepath.Join(t.TempDir(), "payload.zip")
	if err := os.WriteFile(tmpZip, zipContent, 0o644); err != nil {
		t.Fatalf("write payload zip: %v", err)
	}

	created := parseKV(t, runTenant(t, name,
		"payload", "add",
		"--file", tmpZip,
		"--config", "script.conf",
		"--start", "bash ~/script.sh",
		"--stop", "true",
		"--system", systemID,
	))
	id := created["id"]

	got := parseKV(t, runTenant(t, name, "payload", "get", "--id", id))
	if got["id"] != id {
		t.Fatalf("get returned wrong id: got %q want %q", got["id"], id)
	}
	if got["config"] != "script.conf" {
		t.Fatalf("config mismatch: got %q", got["config"])
	}
}

func TestPayloadUpdate(t *testing.T) {
	t.Parallel()
	ts := time.Now().UnixNano()
	name := fmt.Sprintf("pldupd%d", ts)
	runClient(t, "tenant", "add", "--name", name)
	systemID := firstID(t, runTenant(t, name, "system", "list"))

	zipContent := buildZip(t)
	tmpZip := filepath.Join(t.TempDir(), "payload.zip")
	if err := os.WriteFile(tmpZip, zipContent, 0o644); err != nil {
		t.Fatalf("write payload zip: %v", err)
	}

	created := parseKV(t, runTenant(t, name,
		"payload", "add",
		"--file", tmpZip,
		"--config", "script.conf",
		"--start", "bash ~/script.sh",
		"--stop", "true",
		"--system", systemID,
	))
	id := created["id"]

	updated := parseKV(t, runTenant(t, name,
		"payload", "update",
		"--id", id,
		"--start", "bash ~/new.sh",
		"--stop", "false",
	))
	if updated["start"] != "bash ~/new.sh" {
		t.Fatalf("updated start mismatch: got %q", updated["start"])
	}

	db := tenantDeployDB(t, name)
	var dbStart string
	if err := db.QueryRow(`SELECT start FROM payloads WHERE id = $1`, id).Scan(&dbStart); err != nil {
		t.Fatalf("query payload: %v", err)
	}
	if dbStart != "bash ~/new.sh" {
		t.Fatalf("DB start after update: %q", dbStart)
	}
}

func TestPayloadDelete(t *testing.T) {
	t.Parallel()
	ts := time.Now().UnixNano()
	name := fmt.Sprintf("plddel%d", ts)
	runClient(t, "tenant", "add", "--name", name)
	systemID := firstID(t, runTenant(t, name, "system", "list"))

	zipContent := buildZip(t)
	tmpZip := filepath.Join(t.TempDir(), "payload.zip")
	if err := os.WriteFile(tmpZip, zipContent, 0o644); err != nil {
		t.Fatalf("write payload zip: %v", err)
	}

	created := parseKV(t, runTenant(t, name,
		"payload", "add",
		"--file", tmpZip,
		"--config", "script.conf",
		"--start", "bash ~/script.sh",
		"--stop", "true",
		"--system", systemID,
	))
	id := created["id"]

	out := runTenant(t, name, "payload", "delete", "--id", id)
	if !strings.Contains(out, "deleted") {
		t.Fatalf("expected 'deleted' in output, got: %q", out)
	}

	db := tenantDeployDB(t, name)
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM payloads WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&count); err != nil {
		t.Fatalf("query payload: %v", err)
	}
	if count != 0 {
		t.Fatal("payload still exists after delete")
	}
}

func TestPayloadList(t *testing.T) {
	t.Parallel()
	ts := time.Now().UnixNano()
	name := fmt.Sprintf("pldlist%d", ts)
	runClient(t, "tenant", "add", "--name", name)
	systemID := firstID(t, runTenant(t, name, "system", "list"))

	makeZip := func() string {
		zip := buildZip(t)
		p := filepath.Join(t.TempDir(), "p.zip")
		if err := os.WriteFile(p, zip, 0o644); err != nil {
			t.Fatalf("write zip: %v", err)
		}
		return p
	}

	r1 := parseKV(t, runTenant(t, name,
		"payload", "add", "--file", makeZip(), "--config", "script.conf",
		"--start", "bash ~/script.sh", "--stop", "true", "--system", systemID,
	))
	r2 := parseKV(t, runTenant(t, name,
		"payload", "add", "--file", makeZip(), "--config", "script.conf",
		"--start", "bash ~/script.sh", "--stop", "true", "--system", systemID,
	))

	out := runTenant(t, name, "payload", "list")
	if !strings.Contains(out, r1["id"]) {
		t.Fatalf("list missing payload %s", r1["id"])
	}
	if !strings.Contains(out, r2["id"]) {
		t.Fatalf("list missing payload %s", r2["id"])
	}
}
