package jobs

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/cschleiden/go-workflows/workflow"
	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"gorm.io/gorm"
	"zxc/internal/config"
	"zxc/internal/consts"
	"zxc/internal/infra"
	"zxc/internal/models"
)

const maxDeployPayloadSize = 50 * 1024 * 1024

type DeployArgs struct {
	TenantID    uuid.UUID
	ReleaseID   uuid.UUID
	ChangedByID uuid.UUID
}

func Deploy(ctx workflow.Context, args DeployArgs) error {
	_, err := workflow.ExecuteActivity[any](ctx, workflow.DefaultActivityOptions, DeployActivity, args).Get(ctx)
	return err
}

type deployDeps struct {
	rootDB    *gorm.DB
	newTenant func(string) (*gorm.DB, error)
	cfg       *config.Config
}

var deployDep *deployDeps

func RegisterDeployDeps(rootDB *gorm.DB, newTenant func(string) (*gorm.DB, error), cfg *config.Config) {
	deployDep = &deployDeps{rootDB: rootDB, newTenant: newTenant, cfg: cfg}
}

func DeployActivity(ctx context.Context, args DeployArgs) error {
	var tenant models.Tenant
	if err := deployDep.rootDB.WithContext(ctx).First(&tenant, "id = ?", args.TenantID).Error; err != nil {
		return fmt.Errorf("load tenant %s: %w", args.TenantID, err)
	}

	db, err := deployDep.newTenant(tenant.Database)
	if err != nil {
		return err
	}

	var release models.Release
	if err := db.WithContext(ctx).
		Preload("Target").
		Preload("Payload").
		First(&release, "id = ?", args.ReleaseID).Error; err != nil {
		return fmt.Errorf("load release %s: %w", args.ReleaseID, err)
	}

	if release.Status != models.ReleaseWait {
		return nil
	}
	if release.Target == nil {
		return fmt.Errorf("release %s has no target assigned", release.ID)
	}
	if release.Payload == nil {
		return fmt.Errorf("release %s has no payload assigned", release.ID)
	}

	now := time.Now().UTC()
	stale := now.Add(-15 * time.Minute)
	lockResult := db.WithContext(ctx).Model(&models.Target{}).
		Where("id = ? AND (deploying = false OR deploying_at < ?)", release.Target.ID, stale).
		Updates(map[string]any{"deploying": true, "deploying_at": now})
	if lockResult.Error != nil {
		return fmt.Errorf("acquire deploy lock: %w", lockResult.Error)
	}
	if lockResult.RowsAffected == 0 {
		return fmt.Errorf("target is locked, will retry")
	}
	defer func() {
		if err := db.Model(&models.Target{}).Where("id = ?", release.Target.ID).
			Updates(map[string]any{"deploying": false, "deploying_at": nil}).Error; err != nil {
			slog.Error("failed to release deploy lock", "target_id", release.Target.ID, "error", err)
		}
	}()

	mc, bucket, err := infra.StorageClientFromConnectionString(tenant.Storage)
	if err != nil {
		return fmt.Errorf("storage client: %w", err)
	}

	scriptReader, err := mc.Download(ctx, bucket, release.Payload.Path)
	if err != nil {
		return fmt.Errorf("download payload script: %w", err)
	}
	defer scriptReader.Close()

	scriptContent, err := io.ReadAll(io.LimitReader(scriptReader, maxDeployPayloadSize))
	if err != nil {
		return fmt.Errorf("read payload script: %w", err)
	}

	deployZip, err := injectConfig(scriptContent, release.Payload.Config, release.ID, args.TenantID, deployDep.cfg.Webhook, deployDep.cfg.Secret, release.Target.Key)
	if err != nil {
		return fmt.Errorf("create deploy zip: %w", err)
	}

	releasePath := "releases/" + release.ID.String() + ".zip"
	if err := mc.Upload(ctx, bucket, releasePath, bytes.NewReader(deployZip), int64(len(deployZip)), "application/zip"); err != nil {
		return fmt.Errorf("upload release zip: %w", err)
	}

	if err := infra.Deploy(ctx, infra.SSHRequest{
		Host:     release.Target.Address,
		User:     release.Target.User,
		Key:      []byte(release.Target.Key),
		Payload:  bytes.NewReader(deployZip),
		StopCmd:  release.Payload.Stop,
		StartCmd: release.Payload.Start,
	}); err != nil {
		if markErr := db.WithContext(ctx).Model(&models.Release{}).
			Where("id = ? AND status <> ?", args.ReleaseID, models.ReleaseDead).
			Updates(map[string]any{"status": models.ReleaseDead, "changed_by_id": args.ChangedByID}).Error; markErr != nil {
			slog.Error("failed to mark release dead", "release_id", args.ReleaseID, "error", markErr)
		}
		return fmt.Errorf("ssh deploy to %s: %w", release.Target.Address, err)
	}

	if err := db.WithContext(ctx).Model(&models.Target{}).
		Where("id = ?", release.Target.ID).
		Update("status", models.TargetOnline).Error; err != nil {
		slog.Error("failed to update target status", "target_id", release.Target.ID, "error", err)
	}

	if err := db.WithContext(ctx).Model(&models.Release{}).
		Where("id = ? AND status = ?", release.ID, models.ReleaseWait).
		Updates(map[string]any{"status": models.ReleaseDeployed, "changed_by_id": args.ChangedByID}).Error; err != nil {
		return err
	}

	return nil
}

func injectConfig(zipContent []byte, config string, releaseID, tenantID uuid.UUID, webhookURL, secret, key string) ([]byte, error) {
	token, err := jwt.NewBuilder().
		Claim("release_id", releaseID.String()).
		Claim("tenant_id", tenantID.String()).
		Build()
	if err != nil {
		return nil, fmt.Errorf("build jwt: %w", err)
	}
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.HS256(), []byte(secret)))
	if err != nil {
		return nil, fmt.Errorf("sign jwt: %w", err)
	}

	r, err := zip.NewReader(bytes.NewReader(zipContent), int64(len(zipContent)))
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	for _, f := range r.File {
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return nil, err
		}

		if f.Name == config {
			s := strings.ReplaceAll(string(b), consts.URL, webhookURL)
			s = strings.ReplaceAll(s, consts.AUTH, string(signed))
			b = []byte(s)
		}

		e, err := w.Create(f.Name)
		if err != nil {
			return nil, err
		}
		if _, err := e.Write(b); err != nil {
			return nil, err
		}
	}

	if key != "" {
		kf, err := w.Create("key")
		if err != nil {
			return nil, err
		}
		if _, err := kf.Write([]byte(key)); err != nil {
			return nil, err
		}
	}

	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
