package test

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestUserCreate(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("usrcreate%d", time.Now().UnixNano())
	runClient(t, "tenant", "add", "--name", name)

	out := parseKV(t, runTenant(t, name, "user", "add", "--name", "alice"))
	id := out["id"]
	if id == "" {
		t.Fatal("expected id in response")
	}
	if out["name"] != "alice" {
		t.Fatalf("name mismatch: got %q", out["name"])
	}

	db := tenantMainDB(t, name)
	var got string
	if err := db.QueryRow(`SELECT id::text FROM users WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&got); err != nil {
		t.Fatalf("user not found in main DB: %v", err)
	}
}

func TestUserGet(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("usrget%d", time.Now().UnixNano())
	runClient(t, "tenant", "add", "--name", name)

	created := parseKV(t, runTenant(t, name, "user", "add", "--name", "bob"))
	id := created["id"]

	got := parseKV(t, runTenant(t, name, "user", "get", "--id", id))
	if got["id"] != id {
		t.Fatalf("get returned wrong id: got %q want %q", got["id"], id)
	}
	if got["name"] != "bob" {
		t.Fatalf("name mismatch: got %q", got["name"])
	}
}

func TestUserUpdate(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("usrupd%d", time.Now().UnixNano())
	runClient(t, "tenant", "add", "--name", name)

	created := parseKV(t, runTenant(t, name, "user", "add", "--name", "before"))
	id := created["id"]

	updated := parseKV(t, runTenant(t, name, "user", "update", "--id", id, "--name", "after"))
	if updated["name"] != "after" {
		t.Fatalf("updated name mismatch: got %q", updated["name"])
	}

	db := tenantMainDB(t, name)
	var dbName string
	if err := db.QueryRow(`SELECT name FROM users WHERE id = $1`, id).Scan(&dbName); err != nil {
		t.Fatalf("query user: %v", err)
	}
	if dbName != "after" {
		t.Fatalf("DB name after update: %q", dbName)
	}
}

func TestUserDelete(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("usrdel%d", time.Now().UnixNano())
	runClient(t, "tenant", "add", "--name", name)

	created := parseKV(t, runTenant(t, name, "user", "add", "--name", "todelete"))
	id := created["id"]

	out := runTenant(t, name, "user", "delete", "--id", id)
	if !strings.Contains(out, "deleted") {
		t.Fatalf("expected 'deleted' in output, got: %q", out)
	}

	db := tenantMainDB(t, name)
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&count); err != nil {
		t.Fatalf("query user: %v", err)
	}
	if count != 0 {
		t.Fatal("user still exists after delete")
	}
}

func TestUserList(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("utest%d", time.Now().UnixNano())
	runClient(t, "tenant", "add", "--name", name)

	out := runTenant(t, name, "user", "list", "--size", "10")
	ownerID := firstID(t, out)
	if ownerID == "" {
		t.Fatal("expected at least one user after tenant creation")
	}

	db := tenantMainDB(t, name)
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM users WHERE id = $1`, ownerID).Scan(&count); err != nil {
		t.Fatalf("query user in tenant main DB: %v", err)
	}
	if count == 0 {
		t.Fatalf("user %s not found in tenant main DB", ownerID)
	}
}
