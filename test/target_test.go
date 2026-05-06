package test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTargetGet(t *testing.T) {
	t.Parallel()
	ts := time.Now().UnixNano()
	name := fmt.Sprintf("tgtget%d", ts)
	runClient(t, "tenant", "add", "--name", name)

	created := parseKV(t, runTenant(t, name,
		"target", "add",
		"--address", "zxc-target",
		"--user", "deploy",
		"--key", filepath.Join(absProjectRoot, "test/fixtures/id_ed25519"),
	))
	id := created["id"]

	got := parseKV(t, runTenant(t, name, "target", "get", "--id", id))
	if got["id"] != id {
		t.Fatalf("get returned wrong id: got %q want %q", got["id"], id)
	}
	if got["address"] != "zxc-target" {
		t.Fatalf("address mismatch: got %q", got["address"])
	}
}

func TestTargetUpdate(t *testing.T) {
	t.Parallel()
	ts := time.Now().UnixNano()
	name := fmt.Sprintf("tgtupd%d", ts)
	runClient(t, "tenant", "add", "--name", name)

	created := parseKV(t, runTenant(t, name,
		"target", "add",
		"--address", "zxc-target",
		"--user", "deploy",
		"--key", filepath.Join(absProjectRoot, "test/fixtures/id_ed25519"),
	))
	id := created["id"]

	updated := parseKV(t, runTenant(t, name,
		"target", "update",
		"--id", id,
		"--address", "zxc-target2",
		"--user", "deploy",
	))
	if updated["address"] != "zxc-target2" {
		t.Fatalf("updated address mismatch: got %q", updated["address"])
	}

	db := tenantDeployDB(t, name)
	var dbAddr string
	if err := db.QueryRow(`SELECT address FROM targets WHERE id = $1`, id).Scan(&dbAddr); err != nil {
		t.Fatalf("query target: %v", err)
	}
	if dbAddr != "zxc-target2" {
		t.Fatalf("DB address after update: %q", dbAddr)
	}
}

func TestTargetDelete(t *testing.T) {
	t.Parallel()
	ts := time.Now().UnixNano()
	name := fmt.Sprintf("tgtdel%d", ts)
	runClient(t, "tenant", "add", "--name", name)

	created := parseKV(t, runTenant(t, name,
		"target", "add",
		"--address", "zxc-target",
		"--user", "deploy",
		"--key", filepath.Join(absProjectRoot, "test/fixtures/id_ed25519"),
	))
	id := created["id"]

	out := runTenant(t, name, "target", "delete", "--id", id)
	if !strings.Contains(out, "deleted") {
		t.Fatalf("expected 'deleted' in output, got: %q", out)
	}

	db := tenantDeployDB(t, name)
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM targets WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&count); err != nil {
		t.Fatalf("query target: %v", err)
	}
	if count != 0 {
		t.Fatal("target still exists after delete")
	}
}

func TestTargetAdd(t *testing.T) {
	t.Parallel()
	ts := time.Now().UnixNano()
	name := fmt.Sprintf("tgtadd%d", ts)
	runClient(t, "tenant", "add", "--name", name)

	out := parseKV(t, runTenant(t, name,
		"target", "add",
		"--address", "zxc-target",
		"--user", "deploy",
		"--key", filepath.Join(absProjectRoot, "test/fixtures/id_ed25519"),
	))
	id := out["id"]
	if id == "" {
		t.Fatal("expected target id in response")
	}

	db := tenantDeployDB(t, name)
	var got string
	if err := db.QueryRow(`SELECT id::text FROM targets WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&got); err != nil {
		t.Fatalf("target not found in deploy DB: %v", err)
	}
}

func TestTargetList(t *testing.T) {
	t.Parallel()
	ts := time.Now().UnixNano()
	name := fmt.Sprintf("tgtlist%d", ts)
	runClient(t, "tenant", "add", "--name", name)

	r1 := parseKV(t, runTenant(t, name,
		"target", "add", "--address", "zxc-target", "--user", "deploy",
		"--key", filepath.Join(absProjectRoot, "test/fixtures/id_ed25519"),
	))
	r2 := parseKV(t, runTenant(t, name,
		"target", "add", "--address", "zxc-target2", "--user", "deploy",
		"--key", filepath.Join(absProjectRoot, "test/fixtures/id_ed25519"),
	))

	out := runTenant(t, name, "target", "list")
	if !strings.Contains(out, r1["id"]) {
		t.Fatalf("list missing target %s", r1["id"])
	}
	if !strings.Contains(out, r2["id"]) {
		t.Fatalf("list missing target %s", r2["id"])
	}
}
