package test

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestSystemAdd(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("sysadd%d", time.Now().UnixNano())
	runClient(t, "tenant", "add", "--name", name)

	out := parseKV(t, runTenant(t, name, "system", "add", "--name", "mysys"))
	id := out["id"]
	if id == "" {
		t.Fatal("expected id in response")
	}
	if out["name"] != "mysys" {
		t.Fatalf("name mismatch: got %q", out["name"])
	}

	db := tenantMainDB(t, name)
	var got string
	if err := db.QueryRow(`SELECT id::text FROM systems WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&got); err != nil {
		t.Fatalf("system not found in main DB: %v", err)
	}
}

func TestSystemGet(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("sysget%d", time.Now().UnixNano())
	runClient(t, "tenant", "add", "--name", name)

	created := parseKV(t, runTenant(t, name, "system", "add", "--name", "getsys"))
	id := created["id"]

	got := parseKV(t, runTenant(t, name, "system", "get", "--id", id))
	if got["id"] != id {
		t.Fatalf("get returned wrong id: got %q want %q", got["id"], id)
	}
	if got["name"] != "getsys" {
		t.Fatalf("name mismatch: got %q", got["name"])
	}
}

func TestSystemUpdate(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("sysupd%d", time.Now().UnixNano())
	runClient(t, "tenant", "add", "--name", name)

	created := parseKV(t, runTenant(t, name, "system", "add", "--name", "before"))
	id := created["id"]

	updated := parseKV(t, runTenant(t, name, "system", "update", "--id", id, "--name", "after"))
	if updated["name"] != "after" {
		t.Fatalf("updated name mismatch: got %q", updated["name"])
	}

	db := tenantMainDB(t, name)
	var dbName string
	if err := db.QueryRow(`SELECT name FROM systems WHERE id = $1`, id).Scan(&dbName); err != nil {
		t.Fatalf("query system: %v", err)
	}
	if dbName != "after" {
		t.Fatalf("DB name after update: %q", dbName)
	}
}

func TestSystemDelete(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("sysdel%d", time.Now().UnixNano())
	runClient(t, "tenant", "add", "--name", name)

	created := parseKV(t, runTenant(t, name, "system", "add", "--name", "delsys"))
	id := created["id"]

	out := runTenant(t, name, "system", "delete", "--id", id)
	if !strings.Contains(out, "deleted") {
		t.Fatalf("expected 'deleted' in output, got: %q", out)
	}

	db := tenantMainDB(t, name)
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM systems WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&count); err != nil {
		t.Fatalf("query system: %v", err)
	}
	if count != 0 {
		t.Fatal("system still exists after delete")
	}
}

func TestSystemList(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("syslist%d", time.Now().UnixNano())
	runClient(t, "tenant", "add", "--name", name)

	r1 := parseKV(t, runTenant(t, name, "system", "add", "--name", "s1"))
	r2 := parseKV(t, runTenant(t, name, "system", "add", "--name", "s2"))

	out := runTenant(t, name, "system", "list")
	if !strings.Contains(out, r1["id"]) {
		t.Fatalf("list missing system %s", r1["id"])
	}
	if !strings.Contains(out, r2["id"]) {
		t.Fatalf("list missing system %s", r2["id"])
	}
}
