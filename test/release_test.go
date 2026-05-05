package test

import (
	"testing"
	"time"
)

func TestReleaseAdd(t *testing.T) {
	ts := time.Now().UnixNano()
	tenantID, _, targetID, payloadID, tenantName := setupTenantWithDeps(t, ts, 0)
	_ = tenantID

	out := parseKVOutput(t, runTenantClient(t, tenantName,
		"release", "add",
		"--target", targetID,
		"--payload", payloadID,
	))
	id := out["id"]
	if id == "" {
		t.Fatal("expected release id in response")
	}

	db := tenantDeployDB(t, tenantName)
	var status string
	if err := db.QueryRow(`SELECT status FROM releases WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&status); err != nil {
		t.Fatalf("release not found in deploy DB: %v", err)
	}
	if status != "unknown" {
		t.Fatalf("expected status=unknown got %q", status)
	}
}

func TestReleaseGet(t *testing.T) {
	ts := time.Now().UnixNano()
	_, _, targetID, payloadID, tenantName := setupTenantWithDeps(t, ts, 1)

	created := parseKVOutput(t, runTenantClient(t, tenantName,
		"release", "add", "--target", targetID, "--payload", payloadID,
	))
	id := created["id"]

	got := parseKVOutput(t, runTenantClient(t, tenantName, "release", "get", "--id", id))
	if got["id"] != id {
		t.Fatalf("get returned wrong id: %q", got["id"])
	}
	if got["status"] == "" {
		t.Fatal("expected status in response")
	}
}

func TestReleaseDeploy(t *testing.T) {
	ts := time.Now().UnixNano()
	tenantID, _, targetID, payloadID, tenantName := setupTenantWithDeps(t, ts, 2)
	_ = tenantID

	releaseAdd := parseKVOutput(t, runTenantClient(t, tenantName,
		"release", "add", "--target", targetID, "--payload", payloadID,
	))
	releaseID := releaseAdd["id"]

	deployResp := parseKVOutput(t, runTenantClient(t, tenantName,
		"release", "deploy", "--id", releaseID,
	))
	if deployResp["status"] != "wait" {
		t.Fatalf("expected status=wait after deploy trigger, got %q", deployResp["status"])
	}

	statusRank := func(s string) int {
		switch s {
		case "wait":
			return 1
		case "deployed":
			return 2
		case "dead":
			return -1
		default:
			return 0
		}
	}

	deadline := time.Now().Add(120 * time.Second)
	var last string
	for time.Now().Before(deadline) {
		r := parseKVOutput(t, runTenantClient(t, tenantName, "release", "get", "--id", releaseID))
		last = r["status"]
		if statusRank(last) >= statusRank("deployed") {
			break
		}
		time.Sleep(2 * time.Second)
	}
	if statusRank(last) < statusRank("deployed") {
		t.Fatalf("release did not reach deployed within timeout, last=%q", last)
	}

	db := tenantDeployDB(t, tenantName)
	var dbStatus string
	if err := db.QueryRow(`SELECT status FROM releases WHERE id = $1`, releaseID).Scan(&dbStatus); err != nil {
		t.Fatalf("query release status: %v", err)
	}
	if dbStatus != "deployed" {
		t.Fatalf("expected DB status=deployed got %q", dbStatus)
	}
}

func TestReleaseList(t *testing.T) {
	ts := time.Now().UnixNano()
	_, _, targetID, payloadID, tenantName := setupTenantWithDeps(t, ts, 3)

	r1 := parseKVOutput(t, runTenantClient(t, tenantName,
		"release", "add", "--target", targetID, "--payload", payloadID,
	))
	r2 := parseKVOutput(t, runTenantClient(t, tenantName,
		"release", "add", "--target", targetID, "--payload", payloadID,
	))

	out := runTenantClient(t, tenantName, "release", "list")
	for _, id := range []string{r1["id"], r2["id"]} {
		found := false
		for _, line := range splitLines(out) {
			if containsField(line, id) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("release %s not found in list output:\n%s", id, out)
		}
	}
}

func splitLines(s string) []string {
	var lines []string
	for _, l := range splitNewlines(s) {
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

func splitNewlines(s string) []string {
	var lines []string
	cur := ""
	for _, c := range s {
		if c == '\n' {
			lines = append(lines, cur)
			cur = ""
		} else {
			cur += string(c)
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	return lines
}

func containsField(line, val string) bool {
	for _, f := range splitFields(line) {
		if f == val {
			return true
		}
	}
	return false
}

func splitFields(s string) []string {
	var fields []string
	cur := ""
	for _, c := range s {
		if c == ' ' || c == '\t' {
			if cur != "" {
				fields = append(fields, cur)
				cur = ""
			}
		} else {
			cur += string(c)
		}
	}
	if cur != "" {
		fields = append(fields, cur)
	}
	return fields
}

