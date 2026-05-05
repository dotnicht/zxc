package test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTargetAdd(t *testing.T) {
	ts := time.Now().UnixNano()
	name := fmt.Sprintf("tgtadd%d", ts)
	runClient(t, "tenant", "add", "--name", name)

	out := parseKVOutput(t, runTenantClient(t, name,
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
	ts := time.Now().UnixNano()
	name := fmt.Sprintf("tgtlist%d", ts)
	runClient(t, "tenant", "add", "--name", name)

	r1 := parseKVOutput(t, runTenantClient(t, name,
		"target", "add", "--address", "zxc-target", "--user", "deploy",
		"--key", filepath.Join(absProjectRoot, "test/fixtures/id_ed25519"),
	))
	r2 := parseKVOutput(t, runTenantClient(t, name,
		"target", "add", "--address", "zxc-target2", "--user", "deploy",
		"--key", filepath.Join(absProjectRoot, "test/fixtures/id_ed25519"),
	))

	out := runTenantClient(t, name, "target", "list")
	if !strings.Contains(out, r1["id"]) {
		t.Fatalf("list missing target %s", r1["id"])
	}
	if !strings.Contains(out, r2["id"]) {
		t.Fatalf("list missing target %s", r2["id"])
	}
}
