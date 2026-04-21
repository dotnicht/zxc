package jobs

import (
	"context"
	"fmt"

	"zxc/internal/authz"
	"zxc/internal/models"
)

func authorizeReleaseTransition(ctx context.Context, tenant *models.Tenant, from, to string) error {
	decision, err := authz.EvaluateDefault(ctx, authz.Input{
		Subject: authz.Subject{
			System: true,
		},
		Action: "release.transition",
		Tenant: authz.Tenant{
			ID:      tenant.ID,
			OwnerID: tenant.OwnerID,
		},
		Resource: authz.Resource{
			Type:       "release",
			Status:     from,
			NextStatus: to,
		},
	})
	if err != nil {
		return fmt.Errorf("authorization policy evaluation failed: %w", err)
	}
	if !decision.Allow {
		if decision.Reason == "" {
			decision.Reason = "policy denied"
		}
		return fmt.Errorf("release transition %s -> %s denied: %s", from, to, decision.Reason)
	}
	return nil
}
