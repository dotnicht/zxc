package test

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
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

	logStep("stopping any previous docker-compose stack")
	down := exec.Command("docker-compose", "down", "-v", "--remove-orphans")
	down.Dir = projectRoot
	down.Run()

	logStep("starting docker-compose stack")
	up := exec.Command("docker-compose", "up", "-d", "--build")
	up.Dir = projectRoot
	if out, err := up.CombinedOutput(); err != nil {
		fmt.Printf("docker-compose up failed:\n%s\n%v\n", out, err)
		os.Exit(1)
	}

	logStep("waiting for migrator container %q to finish", migratorName)
	if err := waitForMigrator(migratorName, 120*time.Second); err != nil {
		fmt.Printf("migrator did not complete: %v\n", err)
		os.Exit(1)
	}

	logStep("waiting for gRPC endpoint %s", grpcAddr)
	if err := waitForGRPC(grpcAddr, 60*time.Second); err != nil {
		fmt.Printf("gRPC server not ready: %v\n", err)
		os.Exit(1)
	}

	logStep("waiting for worker container %q to be running", workerName)
	if err := waitForContainer(workerName, 60*time.Second); err != nil {
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
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
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
	ownerID = firstDataID(t, runClient(t, "user", "list", "--tenant", tenantName, "--size", "10"))
	t.Logf("tenant owner resolved: userid=%s", ownerID)

	t.Log("loading SSH key fixture")
	t.Log("creating deploy target")
	targetAdd := parseKVOutput(t, runClient(t,
		"target", "add",
		"--tenant", tenantName,
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
		"payload", "add",
		"--tenant", tenantName,
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

	tenantID, ownerID, targetID, payloadID, tenantName := setupTenantWithDeps(t, ts, 0)
	t.Logf("fixture setup complete: tenant=%s owner=%s target=%s payload=%s", tenantID, ownerID, targetID, payloadID)

	t.Log("creating release")
	releaseAdd := parseKVOutput(t, runClient(t,
		"release", "add",
		"--tenant", tenantName,
		"--target", targetID,
		"--payload", payloadID,
	))
	releaseID := releaseAdd["id"]
	t.Logf("release created: id=%s status=%s", releaseID, releaseAdd["status"])

	t.Logf("triggering deploy for release %s", releaseID)
	deployResp := parseKVOutput(t, runClient(t,
		"release", "deploy",
		"--tenant", tenantName,
		"--id", releaseID,
	))
	if deployResp["status"] != "wait" {
		t.Fatalf("expected 'wait', got %q", deployResp["status"])
	}
	t.Logf("deploy accepted: release status=%s", deployResp["status"])

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

	pollForAtLeast := func(target string, timeout time.Duration) string {
		t.Helper()
		t.Logf("polling for release status at least %q with timeout %s", target, timeout)
		deadline := time.Now().Add(timeout)
		var last string
		var prev string
		for time.Now().Before(deadline) {
			getResp := parseKVOutput(t, runClient(t,
				"release", "get",
				"--tenant", tenantName,
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

	if s := pollForAtLeast("deployed", 90*time.Second); statusRank(s) < statusRank("deployed") {
		t.Fatalf("release did not reach 'deployed' within 90s, last status: %q", s)
	}
	if s := pollForAtLeast("alive", 60*time.Second); s != "alive" {
		t.Fatalf("release did not reach 'alive' within 60s, last status: %q", s)
	}

	t.Logf("end-to-end deploy completed successfully in %s", time.Since(started).Round(time.Millisecond))
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
