package request

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func mockGormDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	t.Helper()

	sqlDB, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherRegexp))
	if err != nil {
		t.Fatalf("sqlmock.New: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(postgres.New(postgres.Config{
		Conn:                 sqlDB,
		PreferSimpleProtocol: true,
	}), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	return db, mock
}

func TestCreateRejectsNonPost(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodGet, "/webhooks", nil)
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d want %d", rec.Code, http.StatusMethodNotAllowed)
	}
}

func TestCreateRejectsMissingAuthorization(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodPost, "/webhooks", strings.NewReader(`{"ok":true}`))
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateRejectsInvalidReleaseID(t *testing.T) {
	h := &Handler{}
	req := httptest.NewRequest(http.MethodPost, "/webhooks", strings.NewReader(`{"ok":true}`))
	req.Header.Set("Authorization", "Bearer not-a-uuid")
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestCreateReturnsNotFoundWhenRouteMissing(t *testing.T) {
	rootDB, mock := mockGormDB(t)
	h := &Handler{rootDB: rootDB}
	releaseID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "routes" WHERE id = $1 ORDER BY "routes"."id" LIMIT $2`)).
		WithArgs(releaseID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id"}))

	req := httptest.NewRequest(http.MethodPost, "/webhooks", strings.NewReader(`{"ok":true}`))
	req.Header.Set("Authorization", "Bearer "+releaseID.String())
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want %d", rec.Code, http.StatusNotFound)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestCreateRejectsEmptyBodyAfterRouteLookup(t *testing.T) {
	rootDB, mock := mockGormDB(t)
	h := &Handler{rootDB: rootDB}
	releaseID := uuid.New()
	tenantID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "routes" WHERE id = $1 ORDER BY "routes"."id" LIMIT $2`)).
		WithArgs(releaseID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id"}).AddRow(releaseID, tenantID))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" WHERE "tenants"."id" = $1 AND "tenants"."deleted_at" IS NULL`)).
		WithArgs(tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "database", "storage", "owner_id"}).AddRow(tenantID, "tenant-a", "postgres://example", "", uuid.New()))

	req := httptest.NewRequest(http.MethodPost, "/webhooks", strings.NewReader(""))
	req.Header.Set("Authorization", "Bearer "+releaseID.String())
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want %d", rec.Code, http.StatusBadRequest)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}

func TestCreateRejectsInvalidJSONAfterRouteLookup(t *testing.T) {
	rootDB, mock := mockGormDB(t)
	h := &Handler{rootDB: rootDB}
	releaseID := uuid.New()
	tenantID := uuid.New()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "routes" WHERE id = $1 ORDER BY "routes"."id" LIMIT $2`)).
		WithArgs(releaseID, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "tenant_id"}).AddRow(releaseID, tenantID))
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "tenants" WHERE "tenants"."id" = $1 AND "tenants"."deleted_at" IS NULL`)).
		WithArgs(tenantID).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "database", "storage", "owner_id"}).AddRow(tenantID, "tenant-a", "postgres://example", "", uuid.New()))

	req := httptest.NewRequest(http.MethodPost, "/webhooks", strings.NewReader("{oops"))
	req.Header.Set("Authorization", "Bearer "+releaseID.String())
	rec := httptest.NewRecorder()

	h.Create(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d want %d", rec.Code, http.StatusBadRequest)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sql expectations: %v", err)
	}
}
