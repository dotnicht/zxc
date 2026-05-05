package jobs

import (
	"context"
	"math/rand"
	"strings"
	"time"

	"github.com/cschleiden/go-workflows/workflow"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"zxc/internal/models"
)

var sentences = []string{
	"Hello, how are you doing today?",
	"Just checking in to see how things are going.",
	"Did you get a chance to look at the latest update?",
	"Let me know if you need anything from my end.",
	"I think we are making good progress here.",
	"Can we schedule a call to discuss this further?",
	"Looking forward to hearing your thoughts on this.",
	"Thanks for the quick response, really appreciate it.",
	"I will follow up with more details shortly.",
	"Everything seems to be on track so far.",
}

func randomText() string {
	n := rand.Intn(3) + 1
	picked := make([]string, n)
	for i := range picked {
		picked[i] = sentences[rand.Intn(len(sentences))]
	}
	return strings.Join(picked, " ")
}

type GenerateArgs struct {
	TenantID uuid.UUID
}

func Generate(ctx workflow.Context, args GenerateArgs) error {
	for {
		if _, err := workflow.ExecuteActivity[any](ctx, workflow.DefaultActivityOptions, GenerateActivity, args).Get(ctx); err != nil {
			return err
		}
		workflow.Sleep(ctx, 60*time.Second)
	}
}

type generateDeps struct {
	rootDB     *gorm.DB
	newAccount func(string) (*gorm.DB, error)
}

var generateDep *generateDeps

func RegisterGenerateDeps(rootDB *gorm.DB, newAccount func(string) (*gorm.DB, error)) {
	generateDep = &generateDeps{rootDB: rootDB, newAccount: newAccount}
}

func GenerateActivity(ctx context.Context, args GenerateArgs) error {
	var tenant models.Tenant
	if err := generateDep.rootDB.WithContext(ctx).First(&tenant, "id = ?", args.TenantID).Error; err != nil {
		return err
	}

	db, err := generateDep.newAccount(tenant.Account)
	if err != nil {
		return err
	}

	var profiles []models.Profile
	if err := db.WithContext(ctx).Where("deleted_at IS NULL").Find(&profiles).Error; err != nil {
		return err
	}

	for _, profile := range profiles {
		if err := generateForProfile(ctx, db, profile); err != nil {
			return err
		}
	}
	return nil
}

func generateForProfile(ctx context.Context, db *gorm.DB, profile models.Profile) error {
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

	post := models.Post{
		TalkID:    talk.ID,
		ProfileID: profile.ID,
		ContactID: contact.ID,
		Text:      randomText(),
	}
	return db.WithContext(ctx).Create(&post).Error
}
