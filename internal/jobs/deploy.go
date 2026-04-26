package jobs

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"gorm.io/gorm"
	"zxc/internal/config"
	"zxc/internal/consts"
	"zxc/internal/deployer"
	"zxc/internal/models"
	"zxc/internal/storage"
	"zxc/internal/workflow"
)

const maxDeployPayloadSize = 50 * 1024 * 1024

type DeployReleaseArgs struct {
	TenantID    uuid.UUID `json:"tenant_id"`
	ReleaseID   uuid.UUID `json:"release_id"`
	ChangedByID uuid.UUID `json:"changed_by_id"`
}

type DeployWorker struct {
	store     *workflow.Store
	newTenant func(string) (*gorm.DB, error)
	rootDB    *gorm.DB
	cfg       *config.Config
}

func NewDeployWorker(store *workflow.Store, newTenant func(string) (*gorm.DB, error), rootDB *gorm.DB, cfg *config.Config) *DeployWorker {
	return &DeployWorker{store: store, newTenant: newTenant, rootDB: rootDB, cfg: cfg}
}

func (w *DeployWorker) Work(ctx context.Context, job *workflow.Job[DeployReleaseArgs]) error {
	var tenant models.Tenant
	if err := w.rootDB.WithContext(ctx).First(&tenant, "id = ?", job.Args.TenantID).Error; err != nil {
		return fmt.Errorf("load tenant %s: %w", job.Args.TenantID, err)
	}

	db, err := w.newTenant(tenant.Database)
	if err != nil {
		return err
	}

	var release models.Release
	if err := db.WithContext(ctx).
		Preload("Target").
		Preload("Payload").
		First(&release, "id = ?", job.Args.ReleaseID).Error; err != nil {
		return fmt.Errorf("load release %s: %w", job.Args.ReleaseID, err)
	}

	if release.Status != models.ReleaseWait && release.Status != models.ReleaseAlive {
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
		return workflow.Snooze(10 * time.Second)
	}
	defer func() {
		if err := db.Model(&models.Target{}).Where("id = ?", release.Target.ID).
			Updates(map[string]any{"deploying": false, "deploying_at": nil}).Error; err != nil {
			slog.Error("failed to release deploy lock", "target_id", release.Target.ID, "error", err)
		}
	}()

	mc, bucket, err := storage.ClientFromConnectionString(tenant.Storage)
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

	deployZip, err := injectConfig(scriptContent, release.Payload.Config, release.ID, job.Args.TenantID, w.cfg.Webhook, w.cfg.Secret, release.Target.Key)
	if err != nil {
		return fmt.Errorf("create deploy zip: %w", err)
	}

	releasePath := "releases/" + release.ID.String() + ".zip"
	if err := mc.Upload(ctx, bucket, releasePath, bytes.NewReader(deployZip), int64(len(deployZip)), "application/zip"); err != nil {
		return fmt.Errorf("upload release zip: %w", err)
	}

	if err := deployer.Deploy(ctx, deployer.Request{
		Host:     release.Target.Address,
		User:     release.Target.User,
		Key:      []byte(release.Target.Key),
		Payload:  bytes.NewReader(deployZip),
		StopCmd:  release.Payload.Stop,
		StartCmd: release.Payload.Start,
	}); err != nil {
		if job.Attempt >= job.MaxAttempts {
			if err := w.markReleaseDead(ctx, db, job.Args); err != nil {
				slog.Error("failed to mark release dead after deploy failure", "release_id", job.Args.ReleaseID, "error", err)
			}
		}
		return fmt.Errorf("ssh deploy to %s: %w", release.Target.Address, err)
	}

	if err := db.WithContext(ctx).Model(&models.Target{}).
		Where("id = ?", release.Target.ID).
		Update("status", models.TargetOnline).Error; err != nil {
		slog.Error("failed to update target status", "target_id", release.Target.ID, "error", err)
	}

	result := db.WithContext(ctx).Model(&models.Release{}).
		Where("id = ? AND status = ?", release.ID, models.ReleaseWait).
		Updates(map[string]any{
			"status":        models.ReleaseDeployed,
			"changed_by_id": job.Args.ChangedByID,
		})
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected > 0 {
		if err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			return w.store.EnqueueCommand(ctx, tx, workflow.CommandInput{
				Kind:          "release_health_timeout",
				AggregateType: "release",
				AggregateID:   release.ID,
				Payload: ReleaseHealthTimeoutArgs{
					TenantID:  job.Args.TenantID,
					ReleaseID: release.ID,
				},
				RunAt:       time.Now().UTC().Add(2 * time.Minute),
				MaxAttempts: 1,
				DedupeKey:   "release-health:" + release.ID.String(),
			})
		}); err != nil {
			revertErr := db.WithContext(ctx).Model(&models.Release{}).
				Where("id = ? AND status = ?", release.ID, models.ReleaseDeployed).
				Updates(map[string]any{
					"status":        models.ReleaseWait,
					"changed_by_id": job.Args.ChangedByID,
				}).Error
			return errors.Join(err, revertErr)
		}
	}

	return nil
}

func (w *DeployWorker) markReleaseDead(ctx context.Context, db *gorm.DB, args DeployReleaseArgs) error {
	result := db.WithContext(ctx).Model(&models.Release{}).
		Where("id = ? AND status <> ?", args.ReleaseID, models.ReleaseDead).
		Updates(map[string]any{
			"status":        models.ReleaseDead,
			"changed_by_id": args.ChangedByID,
		})
	if result.Error != nil {
		return result.Error
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
