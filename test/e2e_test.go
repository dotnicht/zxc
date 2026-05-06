package test

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

const (
	grpcAddr     = "localhost:50051"
	migratorName = "zxc-migrator"
	workerName   = "zxc-worker"
	projectRoot  = ".."
	certsDir     = projectRoot + "/test/certs"
	rootUserID   = "00000000-0000-0000-0000-000000000001"
)

var (
	clientBinPath  string
	clientCfgRoot  string
	absCertsDir    string
	absProjectRoot string
	composeCmd     []string

	// sharedTenantName and sharedProfileID are set once by TestMain after a full
	// deploy cycle. Tests that need a deployed tenant with at least one profile
	// use these instead of running their own deploy.
	sharedTenantName string
	sharedProfileID  string
)

func log(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[%s] %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))
}

func TestMain(m *testing.M) {
	log("generating test TLS certificates in %s", certsDir)
	if err := generateCerts(certsDir); err != nil {
		fmt.Printf("generate certs failed: %v\n", err)
		os.Exit(1)
	}

	var err error
	composeCmd, err = dockerComposeCommand()
	if err != nil {
		fmt.Printf("resolve docker compose command failed: %v\n", err)
		os.Exit(1)
	}
	if os.Getenv("COVER") == "1" {
		// Append coverage override; paths are relative to projectRoot (runCompose sets cmd.Dir).
		composeCmd = append(composeCmd,
			"-f", "docker-compose.yml",
			"-f", "docker-compose.cover.yml",
		)
	}

	log("starting docker-compose stack")
	if out, err := runCompose(projectRoot, "up", "-d", "--build", "--remove-orphans"); err != nil {
		fmt.Printf("docker compose up failed:\n%s\n%v\n", out, err)
		os.Exit(1)
	}

	log("waiting for migrator container %q to finish", migratorName)
	if err := waitForMigrator(migratorName, 120*time.Second); err != nil {
		printComposeDiagnostics()
		fmt.Printf("migrator did not complete: %v\n", err)
		os.Exit(1)
	}

	log("waiting for gRPC endpoint %s", grpcAddr)
	if err := waitForGRPC(grpcAddr, 60*time.Second); err != nil {
		printComposeDiagnostics()
		fmt.Printf("gRPC server not ready: %v\n", err)
		os.Exit(1)
	}

	log("waiting for worker container %q to be running", workerName)
	if err := waitForContainer(workerName, 60*time.Second); err != nil {
		printComposeDiagnostics()
		fmt.Printf("worker not running: %v\n", err)
		os.Exit(1)
	}

	log("integration environment is ready; building shared fixture")
	tmpDir, err := os.MkdirTemp("", "zxc-client-e2e-*")
	if err != nil {
		fmt.Printf("create temp dir failed: %v\n", err)
		os.Exit(1)
	}
	clientCfgRoot = filepath.Join(tmpDir, "home")
	clientBinPath = filepath.Join(tmpDir, "zxc-client")
	absProjectRoot, err = filepath.Abs(projectRoot)
	if err != nil {
		fmt.Printf("resolve project root failed: %v\n", err)
		os.Exit(1)
	}
	absCertsDir, err = filepath.Abs(certsDir)
	if err != nil {
		fmt.Printf("resolve cert dir failed: %v\n", err)
		os.Exit(1)
	}

	log("building client binary at %s", clientBinPath)
	build := exec.Command("go", "build", "-o", clientBinPath, "./cmd/client")
	build.Dir = projectRoot
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Printf("build client failed:\n%s\n%v\n", out, err)
		os.Exit(1)
	}

	if err := writeClientConfig(rootUserID); err != nil {
		fmt.Printf("write initial client config failed: %v\n", err)
		os.Exit(1)
	}

	sharedTenantName, sharedProfileID = setupSharedFixture(tmpDir)

	log("starting tests")
	code := m.Run()
	// Graceful stop so instrumented binaries flush GOCOVERDIR before exit.
	runCompose(projectRoot, "stop")
	os.Exit(code)
}

func dockerComposeCommand() ([]string, error) {
	if _, err := exec.LookPath("docker-compose"); err == nil {
		return []string{"docker-compose"}, nil
	}
	if _, err := exec.LookPath("docker"); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "docker", "compose", "version")
		if err := cmd.Run(); err == nil {
			return []string{"docker", "compose"}, nil
		}
	}
	return nil, fmt.Errorf("neither docker-compose nor docker compose is available")
}

func runCompose(dir string, args ...string) ([]byte, error) {
	if len(composeCmd) == 0 {
		return nil, fmt.Errorf("docker compose command is not initialized")
	}
	cmdArgs := append(append([]string{}, composeCmd[1:]...), args...)
	cmd := exec.Command(composeCmd[0], cmdArgs...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

func printComposeDiagnostics() {
	if out, err := runCompose(projectRoot, "ps"); err == nil {
		fmt.Printf("docker compose ps:\n%s\n", out)
	}
	if out, err := runCompose(projectRoot, "logs", "--tail=200"); err == nil {
		fmt.Printf("docker compose logs --tail=200:\n%s\n", out)
	}
}

func buildFixtureZip(t *testing.T) []byte {
	t.Helper()
	log("building payload fixture zip")
	scriptContent, _ := os.ReadFile(projectRoot + "/test/fixtures/script.sh")
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	sh, _ := w.Create("script.sh")
	sh.Write(scriptContent)
	conf, _ := w.Create("script.conf")
	conf.Write([]byte("{ZXC_URL}\n{ZXC_AUTH}\n"))
	w.Close()
	return buf.Bytes()
}

func writeClientConfig(userID string) error {
	cfgDir := filepath.Join(clientConfigBaseDir(), "zxc")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return err
	}
	body := fmt.Sprintf(`address = "localhost:50051"
userid = %q
timeout = "60s"

[tls]
ca = %q
cert = %q
key = %q
`, userID,
		filepath.Join(absCertsDir, "ca.crt"),
		filepath.Join(absCertsDir, "client.crt"),
		filepath.Join(absCertsDir, "client.key"),
	)
	return os.WriteFile(filepath.Join(cfgDir, "client.toml"), []byte(body), 0o644)
}

func clientConfigBaseDir() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(clientCfgRoot, "Library", "Application Support")
	case "windows":
		return filepath.Join(clientCfgRoot, "AppData", "Roaming")
	default:
		return filepath.Join(clientCfgRoot, ".config")
	}
}

func runClient(t *testing.T, args ...string) string {
	t.Helper()
	env := append(os.Environ(),
		"HOME="+clientCfgRoot,
		"XDG_CONFIG_HOME="+filepath.Join(clientCfgRoot, ".config"),
	)
	// Retry on transient gRPC Unavailable errors (connection refused / EOF under load)
	var (
		out []byte
		err error
	)
	for attempt := 0; attempt < 5; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt*2) * time.Second)
		}
		cmd := exec.Command(clientBinPath, args...)
		cmd.Dir = projectRoot
		cmd.Env = env
		out, err = cmd.CombinedOutput()
		if err == nil {
			return string(out)
		}
		s := string(out)
		// Only retry transient transport errors, not server-side errors like AlreadyExists or too-many-clients
		if !strings.Contains(s, "code = Unavailable") && !strings.Contains(s, "connection refused") {
			break
		}
	}
	t.Fatalf("client %s failed:\n%s\n%v", strings.Join(args, " "), out, err)
	return ""
}

// clientCmd returns a configured exec.Cmd without running it, for tests that
// need to assert on failure exit codes or error output.
func clientCmd(t *testing.T, args ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(clientBinPath, args...)
	cmd.Dir = projectRoot
	cmd.Env = append(os.Environ(),
		"HOME="+clientCfgRoot,
		"XDG_CONFIG_HOME="+filepath.Join(clientCfgRoot, ".config"),
	)
	return cmd
}

func runTenantClient(t *testing.T, name string, args ...string) string {
	t.Helper()
	rootArgs := append([]string{"--tenant", name}, args...)
	return runClient(t, rootArgs...)
}

func parseKVOutput(t *testing.T, out string) map[string]string {
	t.Helper()
	parsed := make(map[string]string)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		key := fields[0]
		value := strings.Join(fields[1:], " ")
		parsed[key] = value
	}
	return parsed
}

func firstDataID(t *testing.T, out string) string {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected table output with data rows, got:\n%s", out)
	}
	fields := strings.Fields(lines[1])
	if len(fields) == 0 {
		t.Fatalf("failed to parse first data row from:\n%s", out)
	}
	return fields[0]
}

func setupTenantWithDeps(t *testing.T, ts int64, idx int) (tenantID, ownerID, targetID, payloadID, tenantName string) {
	t.Helper()
	tenantName = fmt.Sprintf("inttenant%d_%d", ts, idx)
	name := tenantName
	log("creating tenant %q", name)
	tenantAdd := parseKVOutput(t, runClient(t, "tenant", "add", "--name", name))
	tenantID = tenantAdd["id"]
	log("tenant created: id=%s", tenantID)

	log("listing users for tenant %s", tenantID)
	ownerID = firstDataID(t, runTenantClient(t, name, "user", "list", "--size", "10"))
	log("tenant owner resolved: userid=%s", ownerID)

	log("creating deploy target")
	targetAdd := parseKVOutput(t, runClient(t,
		"--tenant", name,
		"target", "add",
		"--address", "zxc-target",
		"--user", "deploy",
		"--key", filepath.Join(absProjectRoot, "test/fixtures/id_ed25519"),
	))
	targetID = targetAdd["id"]
	log("target created: id=%s address=%s", targetID, targetAdd["address"])

	zipContent := buildFixtureZip(t)
	tmpZip := filepath.Join(t.TempDir(), "payload.zip")
	if err := os.WriteFile(tmpZip, zipContent, 0o644); err != nil {
		t.Fatalf("write payload fixture zip: %v", err)
	}
	log("creating payload (%d bytes)", len(zipContent))
	payloadAdd := parseKVOutput(t, runClient(t,
		"--tenant", name,
		"payload", "add",
		"--file", tmpZip,
		"--config", "script.conf",
		"--start", "bash ~/script.sh",
		"--stop", "true",
	))
	payloadID = payloadAdd["id"]
	log("payload created: id=%s", payloadID)

	return
}

// setupSharedFixture creates one deployed tenant and waits for a profile, then
// returns the tenant name and profile ID for use by all profile-dependent tests.
func setupSharedFixture(tmpDir string) (tenantName, profileID string) {
	ts := time.Now().UnixNano()
	tenantName = fmt.Sprintf("shared%d", ts)
	log("creating shared fixture tenant %q", tenantName)

	// We need a *testing.T to call helpers; use a fake one via a helper test runner.
	// Instead, inline the minimal setup directly.
	addOut, err := runClientDirect("tenant", "add", "--name", tenantName)
	if err != nil {
		fmt.Printf("shared fixture: create tenant failed: %v\n%s\n", err, addOut)
		os.Exit(1)
	}

	// Resolve owner
	userOut, err := runClientDirect("--tenant", tenantName, "user", "list", "--size", "10")
	if err != nil {
		fmt.Printf("shared fixture: list users failed: %v\n%s\n", err, userOut)
		os.Exit(1)
	}
	ownerID := firstDataIDRaw(string(userOut))

	// Add target
	keyPath := filepath.Join(absProjectRoot, "test/fixtures/id_ed25519")
	targetOut, err := runClientDirect("--tenant", tenantName,
		"target", "add", "--address", "zxc-target", "--user", "deploy", "--key", keyPath)
	if err != nil {
		fmt.Printf("shared fixture: add target failed: %v\n%s\n", err, targetOut)
		os.Exit(1)
	}
	targetID := parseKVRaw(string(targetOut))["id"]

	// Build and upload payload
	scriptContent, _ := os.ReadFile(projectRoot + "/test/fixtures/script.sh")
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	sh, _ := w.Create("script.sh")
	sh.Write(scriptContent)
	conf, _ := w.Create("script.conf")
	conf.Write([]byte("{ZXC_URL}\n{ZXC_AUTH}\n"))
	w.Close()
	zipBytes := buf.Bytes()

	zipPath := filepath.Join(tmpDir, "shared_payload.zip")
	if err := os.WriteFile(zipPath, zipBytes, 0o644); err != nil {
		fmt.Printf("shared fixture: write zip: %v\n", err)
		os.Exit(1)
	}
	payloadOut, err := runClientDirect("--tenant", tenantName,
		"payload", "add", "--file", zipPath,
		"--config", "script.conf", "--start", "bash ~/script.sh", "--stop", "true")
	if err != nil {
		fmt.Printf("shared fixture: add payload failed: %v\n%s\n", err, payloadOut)
		os.Exit(1)
	}
	payloadID := parseKVRaw(string(payloadOut))["id"]

	// Create and deploy release
	releaseOut, err := runClientDirect("--tenant", tenantName,
		"release", "add", "--target", targetID, "--payload", payloadID)
	if err != nil {
		fmt.Printf("shared fixture: add release failed: %v\n%s\n", err, releaseOut)
		os.Exit(1)
	}
	releaseID := parseKVRaw(string(releaseOut))["id"]

	if _, err := runClientDirect("--tenant", tenantName, "release", "deploy", "--id", releaseID); err != nil {
		fmt.Printf("shared fixture: deploy release failed: %v\n", err)
		os.Exit(1)
	}

	log("shared fixture: tenant=%s owner=%s target=%s payload=%s release=%s", tenantName, ownerID, targetID, payloadID, releaseID)
	log("shared fixture: waiting for profile in account DB (up to 90s)")

	adb, err := sql.Open("postgres", accountConn(tenantName))
	if err != nil {
		fmt.Printf("shared fixture: open account DB: %v\n", err)
		os.Exit(1)
	}
	defer adb.Close()

	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) {
		_ = adb.QueryRow(`SELECT id::text FROM profiles WHERE deleted_at IS NULL LIMIT 1`).Scan(&profileID)
		if profileID != "" {
			log("shared fixture ready: tenant=%s profile=%s", tenantName, profileID)
			return tenantName, profileID
		}
		time.Sleep(2 * time.Second)
	}

	fmt.Printf("shared fixture: no profile appeared within 90s\n")
	os.Exit(1)
	return
}

func runClientDirect(args ...string) ([]byte, error) {
	cmd := exec.Command(clientBinPath, args...)
	cmd.Dir = projectRoot
	cmd.Env = append(os.Environ(),
		"HOME="+clientCfgRoot,
		"XDG_CONFIG_HOME="+filepath.Join(clientCfgRoot, ".config"),
	)
	return cmd.CombinedOutput()
}

func parseKVRaw(out string) map[string]string {
	parsed := make(map[string]string)
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		parsed[fields[0]] = strings.Join(fields[1:], " ")
	}
	return parsed
}

func firstDataIDRaw(out string) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		return ""
	}
	fields := strings.Fields(lines[1])
	if len(fields) == 0 {
		return ""
	}
	return fields[0]
}

func TestE2E(t *testing.T) {
	t.Parallel()
	started := time.Now()
	ts := time.Now().UnixNano()

	tenantID, ownerID, targetID, payloadID, tenantName := setupTenantWithDeps(t, ts, 0)
	log("fixture setup complete: tenant=%s owner=%s target=%s payload=%s", tenantID, ownerID, targetID, payloadID)

	log("creating release for tenant %s", tenantName)
	releaseAdd := parseKVOutput(t, runTenantClient(t, tenantName,
		"release", "add",
		"--target", targetID,
		"--payload", payloadID,
	))
	releaseID := releaseAdd["id"]
	log("release created: id=%s status=%s", releaseID, releaseAdd["status"])

	log("triggering deploy for release %s", releaseID)
	deployResp := parseKVOutput(t, runTenantClient(t, tenantName,
		"release", "deploy",
		"--id", releaseID,
	))
	if deployResp["status"] != "wait" {
		t.Fatalf("expected 'wait', got %q", deployResp["status"])
	}

	statusRank := func(s string) int {
		switch s {
		case "unknown":
			return 0
		case "wait":
			return 1
		case "deployed":
			return 2
		case "dead":
			return -1
		default:
			return -2
		}
	}

	pollForAtLeast := func(id, target string, timeout time.Duration) string {
		t.Helper()
		log("polling release %s for status at least %q with timeout %s", id, target, timeout)
		deadline := time.Now().Add(timeout)
		var last, prev string
		for time.Now().Before(deadline) {
			getResp := parseKVOutput(t, runTenantClient(t, tenantName,
				"release", "get",
				"--id", id,
			))
			last = getResp["status"]
			if last != prev {
				log("release %s status changed: %q", id, last)
				prev = last
			}
			if statusRank(last) >= statusRank(target) {
				log("release %s reached %q or later state %q", id, target, last)
				return last
			}
			time.Sleep(2 * time.Second)
		}
		log("timeout waiting for %q, last status=%q", target, last)
		return last
	}

	if s := pollForAtLeast(releaseID, "deployed", 90*time.Second); statusRank(s) < statusRank("deployed") {
		t.Fatalf("release did not reach 'deployed' within 90s, last status: %q", s)
	}

	log("waiting for webhook requests and accounts to appear in tenant DB")
	requests, accounts := waitForWebhookAccounts(t, tenantName, releaseID, 60*time.Second)
	log("webhook result: requests=%d accounts=%d", requests, accounts)
	if requests < 2 {
		t.Fatalf("expected repeated webhook requests for release %s, got %d", releaseID, requests)
	}
	if accounts < 1 {
		t.Fatalf("expected at least one account for release %s, got %d", releaseID, accounts)
	}
	log("webhook pipeline verified: %d requests received, %d accounts created", requests, accounts)
	verifyAccountFromRequest(t, tenantName, releaseID)

	log("end-to-end deploy completed successfully in %s", time.Since(started).Round(time.Millisecond))
}

func waitForWebhookAccounts(t *testing.T, name, id string, timeout time.Duration) (requests, accounts int) {
	t.Helper()

	deployDB, err := sql.Open("postgres", deployConn(name))
	if err != nil {
		t.Fatalf("open deploy database: %v", err)
	}
	defer deployDB.Close()

	accountDB, err := sql.Open("postgres", accountConn(name))
	if err != nil {
		t.Fatalf("open account database: %v", err)
	}
	defer accountDB.Close()

	var prev string
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := deployDB.QueryRow(`
			SELECT COUNT(*)::int FROM requests
			WHERE release_id = $1 AND deleted_at IS NULL
		`, id).Scan(&requests); err != nil {
			t.Fatalf("count requests: %v", err)
		}
		if err := accountDB.QueryRow(`SELECT COUNT(*)::int FROM profiles`).Scan(&accounts); err != nil {
			t.Fatalf("count profiles: %v", err)
		}
		cur := fmt.Sprintf("requests=%d accounts=%d", requests, accounts)
		if cur != prev {
			log("webhook poll: %s", cur)
			prev = cur
		}
		if requests >= 2 && accounts >= 1 {
			return requests, accounts
		}
		time.Sleep(2 * time.Second)
	}

	return requests, accounts
}

func verifyAccountFromRequest(t *testing.T, name, id string) {
	t.Helper()

	deployDB, err := sql.Open("postgres", deployConn(name))
	if err != nil {
		t.Fatalf("open deploy database: %v", err)
	}
	defer deployDB.Close()

	accountDB, err := sql.Open("postgres", accountConn(name))
	if err != nil {
		t.Fatalf("open account database: %v", err)
	}
	defer accountDB.Close()

	var nodeName string
	if err := deployDB.QueryRow(`
		SELECT data->>'node_name'
		FROM requests
		WHERE release_id = $1 AND deleted_at IS NULL AND data->>'node_name' IS NOT NULL
		LIMIT 1
	`, id).Scan(&nodeName); err != nil {
		t.Fatalf("read node_name from request: %v", err)
	}
	if nodeName == "" {
		t.Fatal("node_name in request data is empty")
	}

	var count int
	if err := accountDB.QueryRow(`SELECT COUNT(*)::int FROM profiles WHERE name = $1`, nodeName).Scan(&count); err != nil {
		t.Fatalf("check profile by name: %v", err)
	}
	if count == 0 {
		t.Fatalf("no profile found with name %q derived from webhook request", nodeName)
	}
	log("account name %q matches node_name from webhook request data", nodeName)
}

func deployConn(name string) string {
	return fmt.Sprintf("postgres://postgres:postgres@localhost:5432/%s?sslmode=disable&search_path=deploy", sanitizeTenantDBName(name))
}

func accountConn(name string) string {
	return fmt.Sprintf("postgres://postgres:postgres@localhost:5432/%s?sslmode=disable&search_path=account", sanitizeTenantDBName(name))
}

func sanitizeTenantDBName(name string) string {
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			result.WriteRune(r)
		} else if r >= 'A' && r <= 'Z' {
			result.WriteRune(r + 32)
		}
	}
	return result.String()
}

func waitForMigrator(containerName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := exec.Command("docker", "inspect",
			"--format={{.State.Status}}:{{.State.ExitCode}}", containerName).Output()
		if err == nil {
			parts := strings.SplitN(strings.TrimSpace(string(out)), ":", 2)
			if len(parts) == 2 && parts[0] == "exited" && parts[1] == "0" {
				return nil
			}
			if len(parts) == 2 && parts[0] == "exited" && parts[1] != "0" {
				return fmt.Errorf("migrator exited with code %s", parts[1])
			}
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("migrator did not finish within %v", timeout)
}

func waitForGRPC(addr string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		c, err := net.DialTimeout("tcp", addr, 2*time.Second)
		if err == nil {
			c.Close()
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("gRPC at %s not ready within %v", addr, timeout)
}

func waitForContainer(containerName string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := exec.Command("docker", "inspect",
			"--format={{.State.Status}}", containerName).Output()
		if err == nil && strings.TrimSpace(string(out)) == "running" {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("container %s not running within %v", containerName, timeout)
}
