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

	"github.com/google/uuid"
	"gorm.io/gorm"
	"zxc/internal/config"
	"zxc/internal/consts"
	"zxc/internal/deployer"
	"zxc/internal/models"
	"zxc/internal/queue"
	"zxc/internal/storage"
)

const maxDeployPayloadSize = 50 * 1024 * 1024

type DeployArgs struct {
	ReleaseID   uuid.UUID `json:"release_id"`
	ChangedByID uuid.UUID `json:"changed_by_id"`
}

func (DeployArgs) Kind() string { return "deploy" }

type DeployWorker struct {
	newTenant func(string) (*gorm.DB, error)
	rootDB    *gorm.DB
	cfg       *config.Config
}

func NewDeployWorker(newTenant func(string) (*gorm.DB, error), rootDB *gorm.DB, cfg *config.Config) *DeployWorker {
	return &DeployWorker{newTenant: newTenant, rootDB: rootDB, cfg: cfg}
}

func (w *DeployWorker) Work(ctx context.Context, job *queue.Job[DeployArgs]) error {
	id := job.Args.ReleaseID
	changedBy := job.Args.ChangedByID

	var route models.Route
	if err := w.rootDB.WithContext(ctx).Preload("Tenant").First(&route, "id = ?", id).Error; err != nil {
		return fmt.Errorf("load route for release %s: %w", id, err)
	}

	db, err := w.newTenant(route.Tenant.Database)
	if err != nil {
		return err
	}

	var release models.Release
	if err := db.WithContext(ctx).
		Preload("Target").
		Preload("Payload").
		Where("id = ? AND status = ?", id, models.ReleaseWait).
		First(&release).Error; err != nil {
		return fmt.Errorf("load release %s: %w", id, err)
	}

	if release.Target == nil {
		return fmt.Errorf("release %s has no target assigned", id)
	}
	if release.Payload == nil {
		return fmt.Errorf("release %s has no payload assigned", id)
	}

	now := time.Now()
	stale := now.Add(-15 * time.Minute)
	lockResult := db.Model(&models.Target{}).
		Where("id = ? AND (deploying = false OR deploying_at < ?)", release.Target.ID, stale).
		Updates(map[string]any{"deploying": true, "deploying_at": now})
	if lockResult.Error != nil {
		return fmt.Errorf("acquire deploy lock: %w", lockResult.Error)
	}
	if lockResult.RowsAffected == 0 {
		return queue.Snooze(10 * time.Second)
	}
	defer func() {
		if err := db.Model(&models.Target{}).Where("id = ?", release.Target.ID).
			Updates(map[string]any{"deploying": false, "deploying_at": nil}).Error; err != nil {
			slog.Error("failed to release deploy lock", "target_id", release.Target.ID, "error", err)
		}
	}()

	mc, bucket, err := storage.ClientFromConnectionString(route.Tenant.Storage)
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

	deployZip, err := injectConfig(scriptContent, release.Payload.Config, id.String(), w.cfg.Webhook, release.Target.Key)
	if err != nil {
		return fmt.Errorf("create deploy zip: %w", err)
	}

	releasePath := "releases/" + id.String() + ".zip"
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
		deployErr := fmt.Errorf("ssh deploy to %s: %w", release.Target.Address, err)
		if job.Attempt >= job.MaxAttempts {
			if updateErr := db.WithContext(ctx).Model(&models.Release{}).
				Where("id = ?", id).
				Update("status", models.ReleaseDead).Error; updateErr != nil {
				slog.Error("failed to mark release as dead", "release_id", id, "error", updateErr)
			}
		}
		return deployErr
	}

	if err := db.WithContext(ctx).Model(&models.Target{}).Where("id = ?", release.Target.ID).Update("status", models.TargetOnline).Error; err != nil {
		slog.Error("failed to update target status", "target_id", release.Target.ID, "error", err)
	}

	return db.WithContext(ctx).
		Model(&models.Release{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":        models.ReleaseDeployed,
			"changed_by_id": changedBy,
		}).Error
}

func injectConfig(zipContent []byte, config, releaseID, webhookURL, key string) ([]byte, error) {
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
			s = strings.ReplaceAll(s, consts.AUTH, releaseID)
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
