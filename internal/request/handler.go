package request

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"zxc/internal/db"
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

type createResponse struct {
	ID        string          `json:"id"`
	Data      json.RawMessage `json:"data"`
	CreatedAt string          `json:"created_at"`
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

	var existing models.Request
	if err := tenantDB.Where("release_id = ?", releaseID).First(&existing).Error; err == nil {
		existingID := existing.ID.String()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(createResponse{
			ID:        existingID,
			Data:      existing.Data,
			CreatedAt: existing.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
		return
	}

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
	tenantKey := route.TenantID.String()
	if err := h.store.RecordEvent(r.Context(), nil, workflow.EventInput{
		Kind:          "webhook_received",
		AggregateType: "release",
		AggregateID:   releaseKey,
		TenantID:      &route.TenantID,
		Payload: map[string]any{
			"release_id": releaseKey,
			"request_id": requestKey,
			"body":       json.RawMessage(body),
		},
	}); err != nil {
		http.Error(w, "failed to record event", http.StatusInternalServerError)
		return
	}

	if err := h.store.EnqueueCommand(r.Context(), nil, workflow.CommandInput{
		Kind:          "release_mark_alive",
		AggregateType: "release",
		AggregateID:   releaseKey,
		TenantID:      &route.TenantID,
		Payload: map[string]any{
			"tenant_id":  tenantKey,
			"release_id": releaseKey,
			"body":       json.RawMessage(body),
		},
		DedupeKey: "release-mark-alive:" + releaseKey,
	}); err != nil {
		http.Error(w, "failed to enqueue release update", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(createResponse{
		ID:        requestKey,
		Data:      record.Data,
		CreatedAt: record.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}
