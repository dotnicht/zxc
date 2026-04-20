package middleware

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"zxc/internal/models"
)

func TestMetadataUUID(t *testing.T) {
	validID := uuid.New()

	testCases := []struct {
		name    string
		md      metadata.MD
		key     string
		wantErr codes.Code
		wantID  uuid.UUID
	}{
		{
			name:    "missing value",
			md:      metadata.MD{},
			key:     "x-user-id",
			wantErr: codes.Unauthenticated,
		},
		{
			name:    "invalid uuid",
			md:      metadata.Pairs("x-user-id", "nope"),
			key:     "x-user-id",
			wantErr: codes.InvalidArgument,
		},
		{
			name:   "valid uuid",
			md:     metadata.Pairs("x-user-id", validID.String()),
			key:    "x-user-id",
			wantID: validID,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := metadataUUID(tc.md, tc.key)
			if tc.wantErr != codes.OK {
				if status.Code(err) != tc.wantErr {
					t.Fatalf("status=%v want %v", status.Code(err), tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("metadataUUID returned error: %v", err)
			}
			if got != tc.wantID {
				t.Fatalf("id=%v want %v", got, tc.wantID)
			}
		})
	}
}

func TestContextRoundTrip(t *testing.T) {
	ctx := context.Background()
	user := &models.User{ID: uuid.New(), Name: "alice"}
	tenant := &models.Tenant{ID: uuid.New(), Name: "tenant-a"}

	ctx = contextWithUser(ctx, user)
	ctx = contextWithTenant(ctx, tenant)

	gotUser, ok := UserFromContext(ctx)
	if !ok || gotUser != user {
		t.Fatalf("UserFromContext mismatch: ok=%v user=%v", ok, gotUser)
	}

	gotTenant, ok := TenantFromContext(ctx, tenant.ID)
	if !ok || gotTenant != tenant {
		t.Fatalf("TenantFromContext mismatch: ok=%v tenant=%v", ok, gotTenant)
	}

	if _, ok := TenantFromContext(ctx, uuid.New()); ok {
		t.Fatalf("TenantFromContext unexpectedly matched wrong tenant id")
	}
}
