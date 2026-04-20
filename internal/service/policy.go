package service

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"zxc/internal/authz"
	"zxc/internal/middleware"
	"zxc/internal/models"
)

func authorizeAction(ctx context.Context, action string, tenant *models.Tenant, resource authz.Resource, related authz.Related) (authz.Decision, error) {
	engine, err := authz.Default()
	if err != nil {
		return authz.Decision{}, status.Errorf(codes.Internal, "failed to load authorization policy: %v", err)
	}

	user, ok := middleware.UserFromContext(ctx)
	if !ok || user == nil {
		return authz.Decision{}, status.Error(codes.Unauthenticated, "authenticated user is required")
	}

	input := authz.Input{
		Subject: authz.Subject{
			ID:     user.ID.String(),
			IsRoot: tenant == nil,
		},
		Action:   action,
		Resource: resource,
		Related:  related,
	}
	if tenant != nil {
		input.Tenant = authz.Tenant{
			ID:      tenant.ID.String(),
			OwnerID: tenant.OwnerID.String(),
		}
	}

	decision, err := engine.Evaluate(ctx, input)
	if err != nil {
		return authz.Decision{}, status.Errorf(codes.Internal, "authorization policy evaluation failed: %v", err)
	}
	if !decision.Allow {
		if decision.Reason == "" {
			decision.Reason = "policy denied"
		}
		return decision, status.Error(codes.PermissionDenied, decision.Reason)
	}
	return decision, nil
}
