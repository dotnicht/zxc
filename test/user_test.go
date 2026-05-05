package test

import (
	"fmt"
	"testing"
	"time"
)

func TestUserList(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("utest%d", time.Now().UnixNano())
	runClient(t, "tenant", "add", "--name", name)

	out := runTenantClient(t, name, "user", "list", "--size", "10")
	ownerID := firstDataID(t, out)
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
