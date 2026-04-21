package request

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"zxc/internal/db"
	"zxc/internal/events"
	"zxc/internal/jobs"
	"zxc/internal/models"
	"zxc/internal/workflow"
)

type Handler struct {
	rootDB *gorm.DB
	store  *workflow.Store
}

func NewHandler(rootDB *gorm.DB, store *workflow.Store) *Handler {
	return &Handler{rootDB: rootDB, store: store}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	auth := r.Header.Get("Authorization")
	releaseIDStr := strings.TrimPrefix(auth, "Bearer ")
	if releaseIDStr == auth || releaseIDStr == "" {
		http.Error(w, "Authorization: Bearer <release-id> header is required", http.StatusBadRequest)
		return
	}

	releaseID, err := uuid.Parse(releaseIDStr)
	if err != nil {
		http.Error(w, "release header must be a valid UUID", http.StatusBadRequest)
		return
	}

	var route models.Route
	if err := h.rootDB.Preload("Tenant").First(&route, "id = ?", releaseID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			http.Error(w, "release not found", http.StatusNotFound)
			return
		}
		http.Error(w, "failed to resolve release", http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		http.Error(w, "request body is required", http.StatusBadRequest)
		return
	}

	if !json.Valid(body) {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	tenantDB, err := db.NewConnection(route.Tenant.Database)
	if err != nil {
		http.Error(w, "failed to connect to tenant database", http.StatusInternalServerError)
		return
	}
	defer func() {
		if sqlDB, err := tenantDB.DB(); err == nil {
			sqlDB.Close()
		}
	}()

	record := &models.Request{
		ReleaseID: releaseID,
		Data:      json.RawMessage(body),
	}

	if err := tenantDB.Create(record).Error; err != nil {
		http.Error(w, "failed to save request", http.StatusInternalServerError)
		return
	}
	releaseKey := releaseID.String()
	requestKey := record.ID.String()
	if err := tenantDB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		if err := h.store.RecordEvent(r.Context(), tx, events.WebhookReceived{
			ReleaseID: releaseID,
			RequestID: record.ID,
			Body:      json.RawMessage(body),
		}); err != nil {
			return err
		}
		if err := h.store.EnqueueCommand(r.Context(), tx, workflow.CommandInput{
			Kind:          "release_mark_alive",
			AggregateType: "release",
			AggregateID:   releaseID,
			Payload: jobs.ReleaseMarkAliveArgs{
				TenantID:  route.TenantID,
				ReleaseID: releaseID,
				Body:      json.RawMessage(body),
			},
			DedupeKey: "release-mark-alive:" + releaseKey,
		}); err != nil {
			return err
		}
		return h.store.EnqueueCommand(r.Context(), tx, workflow.CommandInput{
			Kind:          "account_from_request",
			AggregateType: "request",
			AggregateID:   record.ID,
			Payload: jobs.AccountFromRequestArgs{
				TenantID:  route.TenantID,
				RequestID: record.ID,
				ReleaseID: releaseID,
			},
			DedupeKey: "request-account:" + requestKey,
		})
	}); err != nil {
		cleanupErr := tenantDB.Unscoped().Delete(&models.Request{}, "id = ?", record.ID).Error
		http.Error(w, errors.Join(err, cleanupErr).Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
