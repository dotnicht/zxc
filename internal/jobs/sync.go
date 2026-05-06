package jobs

import (
	"context"
	"fmt"
	"math/rand"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	goplugin "github.com/hashicorp/go-plugin"
	"github.com/cschleiden/go-workflows/workflow"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"zxc/internal/models"
	genplugin "zxc/plugin/generator"
)

type SyncArgs struct {
	TenantID uuid.UUID
}

func Sync(ctx workflow.Context, args SyncArgs) error {
	for {
		if _, err := workflow.ExecuteActivity[any](ctx, workflow.DefaultActivityOptions, SyncActivity, args).Get(ctx); err != nil {
			return err
		}
		workflow.Sleep(ctx, 60*time.Second)
	}
}

type syncDeps struct {
	rootDB     *gorm.DB
	newMain    func(string) (*gorm.DB, error)
	newAccount func(string) (*gorm.DB, error)
	pluginsDir string
	mu         sync.Mutex
	cache      map[string]genplugin.Generator
}

var syncDep *syncDeps

func RegisterSync(rootDB *gorm.DB, connect func(string) (*gorm.DB, error), pluginsDir string) {
	syncDep = &syncDeps{
		rootDB:     rootDB,
		newMain:    connect,
		newAccount: connect,
		pluginsDir: pluginsDir,
		cache:      make(map[string]genplugin.Generator),
	}
}

func SyncActivity(ctx context.Context, args SyncArgs) error {
	var tenant models.Tenant
	if err := syncDep.rootDB.WithContext(ctx).First(&tenant, "id = ?", args.TenantID).Error; err != nil {
		return err
	}

	mainDB, err := syncDep.newMain(tenant.Main)
	if err != nil {
		return err
	}

	var sys models.System
	if err := mainDB.WithContext(ctx).Where("name = 'default' AND deleted_at IS NULL").First(&sys).Error; err != nil {
		return fmt.Errorf("default system not found: %w", err)
	}

	gen, err := syncDep.plugin(sys.Sync)
	if err != nil {
		return fmt.Errorf("load plugin %q: %w", sys.Sync, err)
	}

	accountDB, err := syncDep.newAccount(tenant.Account)
	if err != nil {
		return err
	}

	var profiles []models.Profile
	if err := accountDB.WithContext(ctx).Where("system_id = ? AND deleted_at IS NULL", sys.ID).Find(&profiles).Error; err != nil {
		return err
	}

	for _, profile := range profiles {
		if err := post(ctx, accountDB, profile, gen); err != nil {
			return err
		}
	}
	return nil
}

func (d *syncDeps) plugin(name string) (genplugin.Generator, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if gen, ok := d.cache[name]; ok {
		return gen, nil
	}
	client := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig: genplugin.Handshake,
		Plugins:         genplugin.PluginMap,
		Cmd:             exec.Command(filepath.Join(d.pluginsDir, name)),
	})
	rpc, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, err
	}
	raw, err := rpc.Dispense("generator")
	if err != nil {
		client.Kill()
		return nil, err
	}
	gen := raw.(genplugin.Generator)
	d.cache[name] = gen
	return gen, nil
}

func post(ctx context.Context, db *gorm.DB, profile models.Profile, gen genplugin.Generator) error {
	var talks []models.Talk
	if err := db.WithContext(ctx).Where("profile_id = ? AND deleted_at IS NULL", profile.ID).Find(&talks).Error; err != nil {
		return err
	}

	var talk models.Talk
	if len(talks) == 0 {
		talk = models.Talk{ProfileID: profile.ID}
		if err := db.WithContext(ctx).Create(&talk).Error; err != nil {
			return err
		}
	} else {
		talk = talks[rand.Intn(len(talks))]
	}

	var contacts []models.Contact
	if err := db.WithContext(ctx).Where("profile_id = ? AND deleted_at IS NULL", profile.ID).Find(&contacts).Error; err != nil {
		return err
	}

	var contact models.Contact
	if len(contacts) == 0 {
		contact = models.Contact{ProfileID: profile.ID, Name: "bot"}
		if err := db.WithContext(ctx).Create(&contact).Error; err != nil {
			return err
		}
	} else {
		contact = contacts[rand.Intn(len(contacts))]
	}

	text, err := gen.Post(ctx, profile.Name)
	if err != nil {
		return err
	}

	return db.WithContext(ctx).Create(&models.Post{
		TalkID:    talk.ID,
		ProfileID: profile.ID,
		ContactID: contact.ID,
		Text:      text,
	}).Error
}
