package request

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"gorm.io/gorm"
	"zxc/internal/db"
	"zxc/internal/jobs"
	"zxc/internal/models"
	"zxc/internal/workflow"
)

type Handler struct {
	secret []byte
	rootDB *gorm.DB
	cache  *db.Cache
	store  *workflow.Store
}

func NewHandler(secret []byte, rootDB *gorm.DB, cache *db.Cache, store *workflow.Store) *Handler {
	return &Handler{secret: secret, rootDB: rootDB, cache: cache, store: store}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	auth := r.Header.Get("Authorization")
	tokenStr := strings.TrimPrefix(auth, "Bearer ")
	if tokenStr == auth || tokenStr == "" {
		http.Error(w, "Authorization: Bearer <token> header is required", http.StatusBadRequest)
		return
	}

	token, err := jwt.Parse([]byte(tokenStr), jwt.WithKey(jwa.HS256(), h.secret))
	if err != nil {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	var releaseIDVal, tenantIDVal string
	if err := token.Get("release_id", &releaseIDVal); err != nil {
		http.Error(w, "missing release_id in token", http.StatusBadRequest)
		return
	}
	if err := token.Get("tenant_id", &tenantIDVal); err != nil {
		http.Error(w, "missing tenant_id in token", http.StatusBadRequest)
		return
	}

	releaseID, err := uuid.Parse(releaseIDVal)
	if err != nil {
		http.Error(w, "invalid release_id in token", http.StatusBadRequest)
		return
	}
	tenantID, err := uuid.Parse(tenantIDVal)
	if err != nil {
		http.Error(w, "invalid tenant_id in token", http.StatusBadRequest)
		return
	}

	var tenant models.Tenant
	if err := h.rootDB.WithContext(r.Context()).First(&tenant, "id = ?", tenantID).Error; err != nil {
		http.Error(w, "tenant not found", http.StatusNotFound)
		return
	}

	tenantDB, err := h.cache.Get(tenant.Database)
	if err != nil {
		http.Error(w, "failed to connect to tenant database", http.StatusInternalServerError)
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

	record := &models.Request{
		ReleaseID: releaseID,
		Data:      json.RawMessage(body),
	}

	if err := tenantDB.Create(record).Error; err != nil {
		http.Error(w, "failed to save request", http.StatusInternalServerError)
		return
	}
	requestKey := record.ID.String()
	if err := tenantDB.WithContext(r.Context()).Transaction(func(tx *gorm.DB) error {
		return h.store.EnqueueCommand(r.Context(), tx, workflow.CommandInput{
			Kind:          "account_from_request",
			AggregateType: "request",
			AggregateID:   record.ID,
			Payload: jobs.AccountFromRequestArgs{
				TenantID:  tenantID,
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
