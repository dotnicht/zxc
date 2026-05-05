package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"zxc/api/tenant"
)

func newCtx() (context.Context, context.CancelFunc) {
	timeout := 10 * time.Second
	if st.cfg != nil && st.cfg.Timeout != "" {
		if parsed, err := time.ParseDuration(st.cfg.Timeout); err == nil && parsed > 0 {
			timeout = parsed
		}
	}
	return context.WithTimeout(context.Background(), timeout)
}

func userID() (string, error) {
	if st.cfg.UserID == "" {
		return "", fmt.Errorf("userid must be set in config")
	}
	return st.cfg.UserID, nil
}

func authCtx(ctx context.Context, tenantID string) (context.Context, error) {
	userID, err := userID()
	if err != nil {
		return nil, err
	}
	return metadata.AppendToOutgoingContext(ctx,
		"x-tenant-id", tenantID,
		"x-user-id", userID,
	), nil
}

func rootAuthCtx(ctx context.Context) (context.Context, error) {
	userID, err := userID()
	if err != nil {
		return nil, err
	}
	return metadata.AppendToOutgoingContext(ctx, "x-user-id", userID), nil
}

func parseUUID(s string) []byte {
	id := uuid.MustParse(s)
	return id[:]
}

func formatUUID(b []byte) string {
	return uuid.UUID(b).String()
}

func resolveTenant(ctx context.Context, name string) (string, error) {
	authContext, err := rootAuthCtx(ctx)
	if err != nil {
		return "", err
	}
	resp, err := st.tenant.List(authContext, &tenant.ListRequest{Page: 1, PageSize: 100})
	if err != nil {
		return "", fmt.Errorf("listing tenants: %w", err)
	}
	for _, t := range resp.Tenants {
		if t.Name == name {
			return formatUUID(t.Id), nil
		}
	}
	return "", fmt.Errorf("tenant %q not found", name)
}

func effectiveTenantName() (string, error) {
	if tenantOverride != "" {
		return tenantOverride, nil
	}
	if st.cfg != nil && st.cfg.Tenant != "" {
		return st.cfg.Tenant, nil
	}
	return "", fmt.Errorf("tenant must be set via --tenant or client config")
}

func tenantCtx(ctx context.Context) (context.Context, string, error) {
	name, err := effectiveTenantName()
	if err != nil {
		return nil, "", err
	}
	tenantID, err := resolveTenant(ctx, name)
	if err != nil {
		return nil, "", err
	}
	authContext, err := authCtx(ctx, tenantID)
	if err != nil {
		return nil, "", err
	}
	return authContext, tenantID, nil
}

func tenantOwnerCtx(ctx context.Context) (context.Context, string, string, error) {
	authContext, tenantID, err := tenantCtx(ctx)
	if err != nil {
		return nil, "", "", err
	}
	userID, err := userID()
	if err != nil {
		return nil, "", "", err
	}
	return authContext, tenantID, userID, nil
}

func newTabWriter() *tabwriter.Writer {
	return tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
}

func printKV(pairs [][2]string) {
	w := newTabWriter()
	for _, p := range pairs {
		fmt.Fprintf(w, "%s\t%s\n", p[0], p[1])
	}
	w.Flush()
}

func die(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
