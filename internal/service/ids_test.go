package service

import (
	"testing"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestParseUUID(t *testing.T) {
	id := uuid.New()

	got, err := parseUUID(id.String(), "tenant_id")
	if err != nil {
		t.Fatalf("parseUUID returned error: %v", err)
	}
	if got != id {
		t.Fatalf("id=%v want %v", got, id)
	}

	_, err = parseUUID("bad", "tenant_id")
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status=%v want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestRequireAuthenticatedUser(t *testing.T) {
	authUserID := uuid.New()

	if err := requireAuthenticatedUser("", authUserID, "owner_id"); err != nil {
		t.Fatalf("empty raw should be accepted: %v", err)
	}

	if err := requireAuthenticatedUser(authUserID.String(), authUserID, "owner_id"); err != nil {
		t.Fatalf("matching user id should be accepted: %v", err)
	}

	err := requireAuthenticatedUser(uuid.New().String(), authUserID, "owner_id")
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status=%v want %v", status.Code(err), codes.PermissionDenied)
	}
}
