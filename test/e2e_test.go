package test

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"zxc/api/payload"
	"zxc/api/release"
	"zxc/api/target"
	"zxc/api/tenant"
	"zxc/api/user"
)

const (
	grpcAddr     = "localhost:50051"
	migratorName = "zxc-migrator"
	workerName   = "zxc-worker"
	projectRoot  = ".."
	certsDir     = projectRoot + "/test/certs"
	rootUserID   = "00000000-0000-0000-0000-000000000001"
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
	os.Exit(m.Run())
}

func newIntConn(t *testing.T) *grpc.ClientConn {
	t.Helper()
	creds, err := clientTLSCreds(certsDir)
	if err != nil {
		t.Fatalf("load TLS creds: %v", err)
	}
	conn, err := grpc.NewClient(grpcAddr,
		grpc.WithTransportCredentials(creds),
		grpc.WithDisableServiceConfig(),
	)
	if err != nil {
		t.Fatalf("dial gRPC: %v", err)
	}
	return conn
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

func rootAuthCtx(ctx context.Context) context.Context {
	return metadata.AppendToOutgoingContext(ctx, "x-user-id", rootUserID)
}

func tenantAuthCtx(ctx context.Context, tenantID, userID string) context.Context {
	return metadata.AppendToOutgoingContext(ctx,
		"x-tenant-id", tenantID,
		"x-user-id", userID,
	)
}

func setupTenantWithDeps(t *testing.T, ctx context.Context, conn *grpc.ClientConn, ts int64, idx int) (tenantID, ownerID, targetID, payloadID string) {
	t.Helper()
	tenantClient := tenant.NewTenantServiceClient(conn)
	userClient := user.NewUserServiceClient(conn)
	targetClient := target.NewTargetServiceClient(conn)
	plClient := payload.NewPayloadServiceClient(conn)

	tenantName := fmt.Sprintf("inttenant%d_%d", ts, idx)
	t.Logf("creating tenant %q", tenantName)
	tResp, err := tenantClient.Create(rootAuthCtx(ctx), &tenant.CreateRequest{Name: tenantName})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	tenantID = tResp.Tenant.Id
	t.Logf("tenant created: id=%s", tenantID)

	t.Logf("listing users for tenant %s", tenantID)
	authContext := tenantAuthCtx(ctx, tenantID, rootUserID)
	uResp, err := userClient.List(authContext, &user.ListRequest{TenantId: tenantID, PageSize: 10})
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if uResp.Total == 0 {
		t.Fatalf("no users in tenant")
	}
	ownerID = uResp.Users[0].Id
	t.Logf("tenant owner resolved: user_id=%s", ownerID)
	authContext = tenantAuthCtx(ctx, tenantID, ownerID)

	t.Log("loading SSH key fixture")
	sshKey, err := os.ReadFile(projectRoot + "/test/fixtures/id_ed25519")
	if err != nil {
		t.Fatalf("read id_ed25519: %v", err)
	}
	t.Log("creating deploy target")
	tgResp, err := targetClient.Create(authContext, &target.CreateRequest{
		TenantId: tenantID,
		OwnerId:  ownerID,
		Address:  "zxc-target",
		User:     "deploy",
		Key:      string(sshKey),
	})
	if err != nil {
		t.Fatalf("create target: %v", err)
	}
	targetID = tgResp.Target.Id
	t.Logf("target created: id=%s address=%s", targetID, tgResp.Target.Address)

	zipContent := buildFixtureZip(t)
	t.Logf("creating payload (%d bytes)", len(zipContent))
	pResp, err := plClient.Create(authContext, &payload.CreateRequest{
		TenantId: tenantID,
		OwnerId:  ownerID,
		Content:  zipContent,
		Name:     "payload.zip",
		Config:   "script.conf",
		Start:    "bash ~/script.sh",
	})
	if err != nil {
		t.Fatalf("create payload: %v", err)
	}
	payloadID = pResp.Payload.Id
	t.Logf("payload created: id=%s", payloadID)

	return
}

func TestE2E(t *testing.T) {
	started := time.Now()
	t.Log("opening mTLS gRPC connection")
	conn := newIntConn(t)
	defer conn.Close()
	ctx := context.Background()
	ts := time.Now().UnixNano()

	tenantID, ownerID, targetID, payloadID := setupTenantWithDeps(t, ctx, conn, ts, 0)
	t.Logf("fixture setup complete: tenant=%s owner=%s target=%s payload=%s", tenantID, ownerID, targetID, payloadID)

	releaseClient := release.NewReleaseServiceClient(conn)
	authContext := tenantAuthCtx(ctx, tenantID, ownerID)

	t.Log("creating release")
	createResp, err := releaseClient.Create(authContext, &release.CreateRequest{
		TenantId:  tenantID,
		OwnerId:   ownerID,
		TargetId:  targetID,
		PayloadId: payloadID,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	releaseID := createResp.Release.Id
	t.Logf("release created: id=%s status=%s", releaseID, createResp.Release.Status)

	t.Logf("triggering deploy for release %s", releaseID)
	deployResp, err := releaseClient.Deploy(authContext, &release.DeployRequest{
		TenantId: tenantID,
		Id:       releaseID,
		UserId:   ownerID,
	})
	if err != nil {
		t.Fatalf("Deploy: %v", err)
	}
	if deployResp.Release.Status != "wait" {
		t.Fatalf("expected 'wait', got %q", deployResp.Release.Status)
	}
	t.Logf("deploy accepted: release status=%s", deployResp.Release.Status)

	pollFor := func(target string, timeout time.Duration) string {
		t.Helper()
		t.Logf("polling for release status %q with timeout %s", target, timeout)
		deadline := time.Now().Add(timeout)
		var last string
		var prev string
		for time.Now().Before(deadline) {
			getResp, err := releaseClient.Get(authContext, &release.GetRequest{TenantId: tenantID, Id: releaseID})
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			last = getResp.Release.Status
			if last != prev {
				t.Logf("release %s status changed: %q", releaseID, last)
				prev = last
			}
			if last == target {
				t.Logf("release %s reached %q", releaseID, target)
				return last
			}
			time.Sleep(2 * time.Second)
		}
		t.Logf("timeout waiting for %q, last status=%q", target, last)
		return last
	}

	if s := pollFor("deployed", 90*time.Second); s != "deployed" {
		t.Fatalf("release did not reach 'deployed' within 90s, last status: %q", s)
	}
	if s := pollFor("alive", 60*time.Second); s != "alive" {
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
