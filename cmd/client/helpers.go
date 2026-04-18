package main

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"zxc/api/tenant"
)

func newCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

func resolveTenant(ctx context.Context, client tenant.TenantServiceClient, name string) (string, error) {
	resp, err := client.Search(ctx, &tenant.SearchRequest{Query: name, Page: 1, PageSize: 10})
	if err != nil {
		return "", fmt.Errorf("searching tenants: %w", err)
	}
	for _, t := range resp.Tenants {
		if t.Name == name {
			return t.Id, nil
		}
	}
	return "", fmt.Errorf("tenant %q not found", name)
}

func tenantAndUser(ctx context.Context, tenantName string) (string, string, error) {
	if st.cfg.UserID == "" {
		return "", "", fmt.Errorf("user_id must be set in config")
	}
	tenantID, err := resolveTenant(ctx, st.tenant, tenantName)
	if err != nil {
		return "", "", err
	}
	return tenantID, st.cfg.UserID, nil
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
