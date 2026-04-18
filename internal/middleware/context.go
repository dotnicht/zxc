package middleware

import (
	"context"

	"github.com/google/uuid"
	"zxc/internal/models"
)

type userKey struct{}
type tenantKey struct{}

type tenantEntry struct {
	id     uuid.UUID
	tenant *models.Tenant
}

func contextWithUser(ctx context.Context, u *models.User) context.Context {
	return context.WithValue(ctx, userKey{}, u)
}

func contextWithTenant(ctx context.Context, t *models.Tenant) context.Context {
	return context.WithValue(ctx, tenantKey{}, &tenantEntry{id: t.ID, tenant: t})
}

func UserFromContext(ctx context.Context) (*models.User, bool) {
	u, ok := ctx.Value(userKey{}).(*models.User)
	return u, ok
}

func TenantFromContext(ctx context.Context, tenantID uuid.UUID) (*models.Tenant, bool) {
	e, ok := ctx.Value(tenantKey{}).(*tenantEntry)
	if !ok || e == nil || e.id != tenantID {
		return nil, false
	}
	return e.tenant, true
}
