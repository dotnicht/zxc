package test

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestTenantCreate(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("ttest%d", time.Now().UnixNano())
	out := parseKVOutput(t, runClient(t, "tenant", "add", "--name", name))
	if out["id"] == "" {
		t.Fatal("expected id in response")
	}
	if out["name"] != name {
		t.Fatalf("expected name=%q got %q", name, out["name"])
	}

	db := rootDB(t)
	var got string
	if err := db.QueryRow(`SELECT name FROM tenants WHERE name = $1 AND deleted_at IS NULL`, name).Scan(&got); err != nil {
		t.Fatalf("tenant not found in root DB: %v", err)
	}
	if got != name {
		t.Fatalf("root DB name mismatch: %q", got)
	}
}

func TestTenantList(t *testing.T) {
	t.Parallel()
	ts := time.Now().UnixNano()
	n1 := fmt.Sprintf("tlist%da", ts)
	n2 := fmt.Sprintf("tlist%db", ts)
	runClient(t, "tenant", "add", "--name", n1)
	runClient(t, "tenant", "add", "--name", n2)

	out := runClient(t, "tenant", "list")
	if !strings.Contains(out, n1) {
		t.Fatalf("list output missing %q:\n%s", n1, out)
	}
	if !strings.Contains(out, n2) {
		t.Fatalf("list output missing %q:\n%s", n2, out)
	}
}

func TestTenantGet(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("tget%d", time.Now().UnixNano())
	created := parseKVOutput(t, runClient(t, "tenant", "add", "--name", name))
	id := created["id"]

	got := parseKVOutput(t, runClient(t, "tenant", "get", "--id", id))
	if got["name"] != name {
		t.Fatalf("expected name=%q got %q", name, got["name"])
	}
	if got["id"] != id {
		t.Fatalf("expected id=%q got %q", id, got["id"])
	}
}
