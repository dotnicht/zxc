package test

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"
	"time"
)

// createExternalDB creates an empty postgres database and returns its localhost connection string
// (for test-side use). Use serverDSN() to get the Docker-internal equivalent to pass to the server.
func createExternalDB(t *testing.T, name string) string {
	t.Helper()
	admin, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		t.Fatalf("open admin connection: %v", err)
	}
	if _, err := admin.Exec(fmt.Sprintf(`CREATE DATABASE "%s"`, name)); err != nil {
		admin.Close()
		t.Fatalf("create external database %q: %v", name, err)
	}
	admin.Close()
	return fmt.Sprintf("postgres://postgres:postgres@localhost:5432/%s?sslmode=disable", name)
}

// serverDSN rewrites localhost → postgres so a DSN can be passed to the server running in Docker.
func serverDSN(conn string) string {
	return strings.ReplaceAll(conn, "@localhost:", "@postgres:")
}

// verifySchema checks that a table exists in the given schema using a test-side localhost connection.
func verifySchema(t *testing.T, conn, schema, table string) {
	t.Helper()
	db, err := sql.Open("postgres", conn)
	if err != nil {
		t.Fatalf("open connection %q: %v", conn, err)
	}
	defer db.Close()
	var exists bool
	if err := db.QueryRow(
		`SELECT EXISTS(
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = $1 AND table_name = $2
		)`, schema, table,
	).Scan(&exists); err != nil {
		t.Fatalf("check schema %q table %q: %v", schema, table, err)
	}
	if !exists {
		t.Fatalf("expected table %q in schema %q to exist (conn=%s)", table, schema, conn)
	}
}

func verifyOwnerSeeded(t *testing.T, conn string) {
	t.Helper()
	db, err := sql.Open("postgres", conn)
	if err != nil {
		t.Fatalf("open connection %q: %v", conn, err)
	}
	defer db.Close()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM main.users WHERE id = '00000000-0000-0000-0000-000000000001'`).Scan(&count); err != nil {
		t.Fatalf("count owner in main.users: %v", err)
	}
	if count == 0 {
		t.Fatalf("owner user not seeded in main schema (conn=%s)", conn)
	}
}

// tenantDBExists checks whether an auto-created database for a tenant exists.
func tenantDBExists(t *testing.T, dbName string) bool {
	t.Helper()
	admin, err := sql.Open("postgres", "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable")
	if err != nil {
		t.Fatalf("open admin connection: %v", err)
	}
	defer admin.Close()
	var exists bool
	if err := admin.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)`, dbName,
	).Scan(&exists); err != nil {
		t.Fatalf("check database existence %q: %v", dbName, err)
	}
	return exists
}

// tenantLocalConn returns a localhost-based connection string for a tenant's auto-created DB.
// Used by tests that need to connect directly (bypassing the server's Docker-internal hostname).
func tenantLocalConn(dbName, schema string) string {
	return fmt.Sprintf("postgres://postgres:postgres@localhost:5432/%s?search_path=%s&sslmode=disable",
		sanitizeTenantDBName(dbName), schema)
}

func tenantLocalJobsConn(name string) string {
	return fmt.Sprintf("postgres://postgres:postgres@localhost:5432/%s_jobs?sslmode=disable",
		sanitizeTenantDBName(name))
}

// queryTenantRow returns the stored connection string fields for a tenant.
type tenantRow struct {
	id, main, deploy, account, jobs, storage string
}

func queryTenantRow(t *testing.T, name string) tenantRow {
	t.Helper()
	db := rootDB(t)
	var r tenantRow
	err := db.QueryRow(
		`SELECT id::text, main, deploy, account, jobs, COALESCE(storage,'')
		 FROM tenants WHERE name = $1 AND deleted_at IS NULL`, name,
	).Scan(&r.id, &r.main, &r.deploy, &r.account, &r.jobs, &r.storage)
	if err != nil {
		t.Fatalf("query tenant row for %q: %v", name, err)
	}
	return r
}

// --- tests ---

// TestTenantRegisterDefault: no connection string overrides.
// Server auto-creates <name> and <name>_jobs databases, runs all migrations,
// seeds the owner user.
func TestTenantRegisterDefault(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("treg%d", time.Now().UnixNano())

	out := parseKVOutput(t, runClient(t, "tenant", "add", "--name", name))
	if out["id"] == "" {
		t.Fatal("expected id in response")
	}
	if out["name"] != name {
		t.Fatalf("name mismatch: got %q", out["name"])
	}

	row := queryTenantRow(t, name)

	// Verify stored connection strings reference the right DB/schema (hostname may differ).
	if !strings.Contains(row.main, sanitizeTenantDBName(name)) || !strings.Contains(row.main, "main") {
		t.Errorf("main conn string looks wrong: %s", row.main)
	}
	if !strings.Contains(row.deploy, sanitizeTenantDBName(name)) || !strings.Contains(row.deploy, "deploy") {
		t.Errorf("deploy conn string looks wrong: %s", row.deploy)
	}
	if !strings.Contains(row.account, sanitizeTenantDBName(name)) || !strings.Contains(row.account, "account") {
		t.Errorf("account conn string looks wrong: %s", row.account)
	}
	if !strings.Contains(row.jobs, sanitizeTenantDBName(name)+"_jobs") {
		t.Errorf("jobs conn string looks wrong: %s", row.jobs)
	}

	// Verify schemas via localhost connections (server uses Docker-internal hostname).
	verifySchema(t, tenantLocalConn(name, "main"), "main", "users")
	verifySchema(t, tenantLocalConn(name, "deploy"), "deploy", "targets")
	verifySchema(t, tenantLocalConn(name, "account"), "account", "profiles")
	verifyOwnerSeeded(t, tenantLocalConn(name, "main"))
}

// TestTenantRegisterExternalJobs: caller provides an existing jobs database.
// Server skips creating <name>_jobs, uses the provided one, runs go-workflows
// migrations against it. Main/deploy/account are still auto-provisioned.
func TestTenantRegisterExternalJobs(t *testing.T) {
	t.Parallel()
	ts := time.Now().UnixNano()
	name := fmt.Sprintf("trexjobs%d", ts)
	extJobs := createExternalDB(t, fmt.Sprintf("extjobs%d", ts))
	srvJobs := serverDSN(extJobs)

	out := parseKVOutput(t, runClient(t, "tenant", "add", "--name", name, "--jobs", srvJobs))
	if out["id"] == "" {
		t.Fatal("expected id in response")
	}

	row := queryTenantRow(t, name)

	// jobs connection stored as provided (server-facing DSN).
	if row.jobs != srvJobs {
		t.Errorf("jobs mismatch:\n got  %s\n want %s", row.jobs, srvJobs)
	}
	// main auto-provisioned — DB should exist.
	if !tenantDBExists(t, sanitizeTenantDBName(name)) {
		t.Errorf("auto-created DB for %q should exist", name)
	}

	// go-workflows migrations ran on the external jobs DB (connect via localhost).
	db, err := sql.Open("postgres", extJobs)
	if err != nil {
		t.Fatalf("open ext jobs db: %v", err)
	}
	defer db.Close()
	var exists bool
	if err := db.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM information_schema.tables WHERE table_name = 'instances')`,
	).Scan(&exists); err != nil {
		t.Fatalf("check instances table: %v", err)
	}
	if !exists {
		t.Fatal("go-workflows migrations not applied to external jobs DB")
	}

	// Main schema and owner still provisioned.
	verifySchema(t, tenantLocalConn(name, "main"), "main", "users")
	verifyOwnerSeeded(t, tenantLocalConn(name, "main"))
}

// TestTenantRegisterExternalDeployAccount: deploy and account use external
// connection strings; main and jobs are auto-provisioned.
func TestTenantRegisterExternalDeployAccount(t *testing.T) {
	t.Parallel()
	ts := time.Now().UnixNano()
	name := fmt.Sprintf("trexda%d", ts)
	extDB := createExternalDB(t, fmt.Sprintf("extda%d", ts))
	srvDeploy := serverDSN(extDB) + "&search_path=deploy"
	srvAccount := serverDSN(extDB) + "&search_path=account"
	localDeploy := extDB + "&search_path=deploy"
	localAccount := extDB + "&search_path=account"

	out := parseKVOutput(t, runClient(t, "tenant", "add", "--name", name,
		"--deploy", srvDeploy, "--account", srvAccount))
	if out["id"] == "" {
		t.Fatal("expected id in response")
	}

	row := queryTenantRow(t, name)

	if row.deploy != srvDeploy {
		t.Errorf("deploy mismatch:\n got  %s\n want %s", row.deploy, srvDeploy)
	}
	if row.account != srvAccount {
		t.Errorf("account mismatch:\n got  %s\n want %s", row.account, srvAccount)
	}
	// main auto-provisioned — DB should exist.
	if !tenantDBExists(t, sanitizeTenantDBName(name)) {
		t.Errorf("auto-created DB for %q should exist", name)
	}

	// Migrations applied to both external schemas (verify via localhost).
	verifySchema(t, localDeploy, "deploy", "targets")
	verifySchema(t, localAccount, "account", "profiles")

	// Main DB auto-created and owner seeded.
	verifySchema(t, tenantLocalConn(name, "main"), "main", "users")
	verifyOwnerSeeded(t, tenantLocalConn(name, "main"))
}

// TestTenantRegisterAllExternal: all four connection strings are provided.
// Server creates no databases itself; uses and migrates the external ones.
func TestTenantRegisterAllExternal(t *testing.T) {
	t.Parallel()
	ts := time.Now().UnixNano()
	name := fmt.Sprintf("trexall%d", ts)

	localMain := createExternalDB(t, fmt.Sprintf("extalldbs%d", ts))
	localJobs := createExternalDB(t, fmt.Sprintf("extallj%d", ts))
	srvMain := serverDSN(localMain) + "&search_path=main"
	srvDeploy := serverDSN(localMain) + "&search_path=deploy"
	srvAccount := serverDSN(localMain) + "&search_path=account"
	srvJobs := serverDSN(localJobs)
	localMainConn := localMain + "&search_path=main"
	localDeploy := localMain + "&search_path=deploy"
	localAccount := localMain + "&search_path=account"

	out := parseKVOutput(t, runClient(t, "tenant", "add", "--name", name,
		"--database", srvMain,
		"--deploy", srvDeploy,
		"--account", srvAccount,
		"--jobs", srvJobs,
	))
	if out["id"] == "" {
		t.Fatal("expected id in response")
	}

	row := queryTenantRow(t, name)

	if row.main != srvMain {
		t.Errorf("main mismatch:\n got  %s\n want %s", row.main, srvMain)
	}
	if row.deploy != srvDeploy {
		t.Errorf("deploy mismatch:\n got  %s\n want %s", row.deploy, srvDeploy)
	}
	if row.account != srvAccount {
		t.Errorf("account mismatch:\n got  %s\n want %s", row.account, srvAccount)
	}
	if row.jobs != srvJobs {
		t.Errorf("jobs mismatch:\n got  %s\n want %s", row.jobs, srvJobs)
	}

	// No auto-created database for this tenant name.
	if tenantDBExists(t, sanitizeTenantDBName(name)) {
		t.Errorf("server should not have auto-created database %q when all connections are external", name)
	}

	// All schemas and migrations applied on the external databases (verify via localhost).
	verifySchema(t, localMainConn, "main", "users")
	verifySchema(t, localDeploy, "deploy", "targets")
	verifySchema(t, localAccount, "account", "profiles")
	verifyOwnerSeeded(t, localMainConn)
}

// TestTenantRegisterDuplicate: registering the same name twice returns AlreadyExists.
func TestTenantRegisterDuplicate(t *testing.T) {
	t.Parallel()
	name := fmt.Sprintf("trdup%d", time.Now().UnixNano())
	runClient(t, "tenant", "add", "--name", name)

	cmd := clientCmd(t, "tenant", "add", "--name", name)
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("expected error on duplicate name, got success")
	}
	if !strings.Contains(string(out), "AlreadyExists") {
		t.Fatalf("expected AlreadyExists error, got:\n%s", out)
	}
}

// TestTenantRegisterInvalidName: various invalid names are rejected with InvalidArgument.
func TestTenantRegisterInvalidName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		desc string
	}{
		{"1startswithdigit", "starts with digit"},
		{"HasUppercase", "contains uppercase"},
		{"has-hyphen", "contains hyphen"},
		{"has space", "contains space"},
		{strings.Repeat("a", 64), "64 chars (over limit)"},
	}
	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			t.Parallel()
			cmd := clientCmd(t, "tenant", "add", "--name", tc.name)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("expected error for %q, got success", tc.name)
			}
			s := string(out)
			if !strings.Contains(s, "InvalidArgument") && !strings.Contains(s, "invalid") && !strings.Contains(s, "required") {
				t.Fatalf("expected validation error, got:\n%s", s)
			}
		})
	}
}
