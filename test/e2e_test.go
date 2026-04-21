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
	workerAName  = "zxc-worker-a"
	workerBName  = "zxc-worker-b"
	projectRoot  = ".."
	certsDir     = projectRoot + "/test/certs"
	rootUserID   = "00000000-0000-0000-0000-000000000001"
	workerAID    = "00000000-0000-0000-0000-000000000201"
	workerBID    = "00000000-0000-0000-0000-000000000202"
)

var (
	clientBinPath  string
	clientCfgRoot  string
	absCertsDir    string
	absProjectRoot string
	composeCmd     []string
)

func logStep(format string, args ...any) {
	fmt.Printf("[%s] %s\n", time.Now().Format(time.RFC3339), fmt.Sprintf(format, args...))
}

func TestMain(m *testing.M) {
	logStep("generating test TLS certificates in %s", certsDir)
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

	logStep("stopping any previous docker-compose stack")
	runCompose(projectRoot, "down", "-v", "--remove-orphans")

	logStep("starting docker-compose stack")
	if out, err := runCompose(projectRoot, "up", "-d", "--build"); err != nil {
		fmt.Printf("docker compose up failed:\n%s\n%v\n", out, err)
		os.Exit(1)
	}

	logStep("waiting for migrator container %q to finish", migratorName)
	if err := waitForMigrator(migratorName, 120*time.Second); err != nil {
		printComposeDiagnostics()
		fmt.Printf("migrator did not complete: %v\n", err)
		os.Exit(1)
	}

	logStep("waiting for gRPC endpoint %s", grpcAddr)
	if err := waitForGRPC(grpcAddr, 60*time.Second); err != nil {
		printComposeDiagnostics()
		fmt.Printf("gRPC server not ready: %v\n", err)
		os.Exit(1)
	}

	logStep("waiting for worker container %q to be running", workerAName)
	if err := waitForContainer(workerAName, 60*time.Second); err != nil {
		printComposeDiagnostics()
		fmt.Printf("worker not running: %v\n", err)
		os.Exit(1)
	}
	logStep("waiting for worker container %q to be running", workerBName)
	if err := waitForContainer(workerBName, 60*time.Second); err != nil {
		printComposeDiagnostics()
		fmt.Printf("worker not running: %v\n", err)
		os.Exit(1)
	}

	logStep("integration environment is ready; starting tests")
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

	logStep("building client binary at %s", clientBinPath)
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

	code := m.Run()
	runCompose(projectRoot, "down", "-v", "--remove-orphans")
	_ = os.RemoveAll(tmpDir)
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
	t.Log("building payload fixture zip")
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
	cmd := exec.Command(clientBinPath, args...)
	cmd.Dir = projectRoot
	cmd.Env = append(os.Environ(),
		"HOME="+clientCfgRoot,
		"XDG_CONFIG_HOME="+filepath.Join(clientCfgRoot, ".config"),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("client %s failed:\n%s\n%v", strings.Join(args, " "), out, err)
	}
	return string(out)
}

func runTenantClient(t *testing.T, tenantName string, args ...string) string {
	t.Helper()
	rootArgs := append([]string{"--tenant", tenantName}, args...)
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
	t.Logf("creating tenant %q", tenantName)
	tenantAdd := parseKVOutput(t, runClient(t, "tenant", "add", "--name", tenantName))
	tenantID = tenantAdd["id"]
	t.Logf("tenant created: id=%s", tenantID)

	t.Logf("listing users for tenant %s", tenantID)
	ownerID = firstDataID(t, runTenantClient(t, tenantName, "user", "list", "--size", "10"))
	t.Logf("tenant owner resolved: userid=%s", ownerID)

	t.Log("loading SSH key fixture")
	t.Log("creating deploy target")
	targetAdd := parseKVOutput(t, runClient(t,
		"--tenant", tenantName,
		"target", "add",
		"--address", "zxc-target",
		"--user", "deploy",
		"--key", filepath.Join(absProjectRoot, "test/fixtures/id_ed25519"),
	))
	targetID = targetAdd["id"]
	t.Logf("target created: id=%s address=%s", targetID, targetAdd["address"])

	zipContent := buildFixtureZip(t)
	tmpZip := filepath.Join(t.TempDir(), "payload.zip")
	if err := os.WriteFile(tmpZip, zipContent, 0o644); err != nil {
		t.Fatalf("write payload fixture zip: %v", err)
	}
	t.Logf("creating payload (%d bytes)", len(zipContent))
	payloadAdd := parseKVOutput(t, runClient(t,
		"--tenant", tenantName,
		"payload", "add",
		"--file", tmpZip,
		"--config", "script.conf",
		"--start", "bash ~/script.sh",
		"--stop", "true",
	))
	payloadID = payloadAdd["id"]
	t.Logf("payload created: id=%s", payloadID)

	return
}

func TestE2E(t *testing.T) {
	started := time.Now()
	ts := time.Now().UnixNano()

	tenantAID, ownerAID, targetAID, payloadAID, tenantAName := setupTenantWithDeps(t, ts, 0)
	tenantBID, ownerBID, targetBID, payloadBID, tenantBName := setupTenantWithDeps(t, ts, 1)
	tenantCID, ownerCID, targetCID, payloadCID, tenantCName := setupTenantWithDeps(t, ts, 2)
	t.Logf("fixture setup complete: tenantA=%s owner=%s target=%s payload=%s", tenantAID, ownerAID, targetAID, payloadAID)
	t.Logf("fixture setup complete: tenantB=%s owner=%s target=%s payload=%s", tenantBID, ownerBID, targetBID, payloadBID)
	t.Logf("fixture setup complete: tenantC=%s owner=%s target=%s payload=%s", tenantCID, ownerCID, targetCID, payloadCID)

	t.Log("registering root workers")
	parseKVOutput(t, runClient(t, "worker", "add", "--id", workerAID, "--name", "worker-a"))
	parseKVOutput(t, runClient(t, "worker", "add", "--id", workerBID, "--name", "worker-b"))

	t.Log("assigning tenant A to worker A")
	runClient(t, "worker", "assign", "--worker-id", workerAID, "--tenant-name", tenantAName)
	t.Log("assigning tenant B to worker B")
	runClient(t, "worker", "assign", "--worker-id", workerBID, "--tenant-name", tenantBName)

	createRelease := func(tenantName, targetID, payloadID string) string {
		t.Helper()
		t.Logf("creating release for tenant %s", tenantName)
		releaseAdd := parseKVOutput(t, runTenantClient(t, tenantName,
			"release", "add",
			"--target", targetID,
			"--payload", payloadID,
		))
		releaseID := releaseAdd["id"]
		t.Logf("release created for tenant %s: id=%s status=%s", tenantName, releaseID, releaseAdd["status"])
		return releaseID
	}

	deployRelease := func(tenantName, releaseID string) {
		t.Helper()
		t.Logf("triggering deploy for release %s in tenant %s", releaseID, tenantName)
		deployResp := parseKVOutput(t, runTenantClient(t, tenantName,
			"release", "deploy",
			"--id", releaseID,
		))
		if deployResp["status"] != "wait" {
			t.Fatalf("expected 'wait', got %q", deployResp["status"])
		}
	}

	releaseAID := createRelease(tenantAName, targetAID, payloadAID)
	releaseBID := createRelease(tenantBName, targetBID, payloadBID)
	releaseCID := createRelease(tenantCName, targetCID, payloadCID)

	deployRelease(tenantAName, releaseAID)
	deployRelease(tenantBName, releaseBID)
	deployRelease(tenantCName, releaseCID)

	statusRank := func(status string) int {
		switch status {
		case "unknown":
			return 0
		case "wait":
			return 1
		case "deployed":
			return 2
		case "alive":
			return 3
		case "dead":
			return -1
		default:
			return -2
		}
	}

	pollForAtLeast := func(tenantName, releaseID, target string, timeout time.Duration) string {
		t.Helper()
		t.Logf("polling for tenant %s release %s status at least %q with timeout %s", tenantName, releaseID, target, timeout)
		deadline := time.Now().Add(timeout)
		var last string
		var prev string
		for time.Now().Before(deadline) {
			getResp := parseKVOutput(t, runTenantClient(t, tenantName,
				"release", "get",
				"--id", releaseID,
			))
			last = getResp["status"]
			if last != prev {
				t.Logf("release %s status changed: %q", releaseID, last)
				prev = last
			}
			if statusRank(last) >= statusRank(target) {
				t.Logf("release %s reached %q or later state %q", releaseID, target, last)
				return last
			}
			time.Sleep(2 * time.Second)
		}
		t.Logf("timeout waiting for %q, last status=%q", target, last)
		return last
	}

	if s := pollForAtLeast(tenantAName, releaseAID, "deployed", 90*time.Second); statusRank(s) < statusRank("deployed") {
		t.Fatalf("tenant A release did not reach 'deployed' within 90s, last status: %q", s)
	}
	if s := pollForAtLeast(tenantAName, releaseAID, "alive", 60*time.Second); s != "alive" {
		t.Fatalf("tenant A release did not reach 'alive' within 60s, last status: %q", s)
	}
	if s := pollForAtLeast(tenantBName, releaseBID, "deployed", 90*time.Second); statusRank(s) < statusRank("deployed") {
		t.Fatalf("tenant B release did not reach 'deployed' within 90s, last status: %q", s)
	}
	if s := pollForAtLeast(tenantBName, releaseBID, "alive", 60*time.Second); s != "alive" {
		t.Fatalf("tenant B release did not reach 'alive' within 60s, last status: %q", s)
	}

	if s := pollForAtLeast(tenantCName, releaseCID, "wait", 15*time.Second); s != "wait" {
		t.Fatalf("expected unassigned tenant C release to remain 'wait', got %q", s)
	}

	requests, accounts := waitForWebhookAccounts(t, tenantAName, releaseAID, 35*time.Second)
	if requests < 2 {
		t.Fatalf("expected repeated webhook requests for tenant A release %s, got %d", releaseAID, requests)
	}
	if accounts < 1 {
		t.Fatalf("expected at least one account for tenant A release %s, got %d", releaseAID, accounts)
	}

	requests, accounts = waitForWebhookAccounts(t, tenantBName, releaseBID, 35*time.Second)
	if requests < 2 {
		t.Fatalf("expected repeated webhook requests for tenant B release %s, got %d", releaseBID, requests)
	}
	if accounts < 1 {
		t.Fatalf("expected at least one account for tenant B release %s, got %d", releaseBID, accounts)
	}

	waitForWorkerLog(t, workerAName, tenantAID, 30*time.Second)
	waitForWorkerLog(t, workerBName, tenantBID, 30*time.Second)
	assertWorkerLogsDoNotContain(t, workerAName, tenantBID)
	assertWorkerLogsDoNotContain(t, workerAName, tenantCID)
	assertWorkerLogsDoNotContain(t, workerBName, tenantAID)
	assertWorkerLogsDoNotContain(t, workerBName, tenantCID)

	t.Logf("end-to-end deploy completed successfully in %s", time.Since(started).Round(time.Millisecond))
}

func waitForWebhookAccounts(t *testing.T, tenantName, releaseID string, timeout time.Duration) (requests, accounts int) {
	t.Helper()

	db, err := sql.Open("postgres", tenantDSN(tenantName))
	if err != nil {
		t.Fatalf("open tenant database: %v", err)
	}
	defer db.Close()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if err := db.QueryRow(`
			SELECT COUNT(*)::int FROM requests
			WHERE release_id = $1 AND deleted_at IS NULL
		`, releaseID).Scan(&requests); err != nil {
			t.Fatalf("count requests: %v", err)
		}
		if err := db.QueryRow(`SELECT COUNT(*)::int FROM accounts`).Scan(&accounts); err != nil {
			t.Fatalf("count accounts: %v", err)
		}
		if requests >= 2 && accounts >= 1 {
			return requests, accounts
		}
		time.Sleep(2 * time.Second)
	}

	return requests, accounts
}

func tenantDSN(tenantName string) string {
	return fmt.Sprintf("postgres://postgres:postgres@localhost:5432/%s?sslmode=disable", sanitizeTenantDBName(tenantName))
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

func waitForWorkerLog(t *testing.T, containerName, needle string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := exec.Command("docker", "logs", containerName).CombinedOutput()
		if err == nil && strings.Contains(string(out), needle) {
			return
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("worker %s logs did not contain %q within %s", containerName, needle, timeout)
}

func assertWorkerLogsDoNotContain(t *testing.T, containerName, needle string) {
	t.Helper()
	out, err := exec.Command("docker", "logs", containerName).CombinedOutput()
	if err != nil {
		t.Fatalf("read worker logs for %s: %v", containerName, err)
	}
	if strings.Contains(string(out), needle) {
		t.Fatalf("expected worker %s logs not to contain %q", containerName, needle)
	}
}
