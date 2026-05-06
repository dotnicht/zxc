package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"zxc/api/tenant"
)

func tenantCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tenant",
		Short: "Manage tenants",
	}
	cmd.AddCommand(tenantAddCmd())
	cmd.AddCommand(tenantGetCmd())
	cmd.AddCommand(tenantListCmd())
	return cmd
}

func tenantAddCmd() *cobra.Command {
	var name, database, deploy, account, jobs, storage string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, err := rootAuthCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.tenant.Create(authContext, &tenant.CreateRequest{
				Name:     name,
				Database: database,
				Deploy:   deploy,
				Account:  account,
				Jobs:     jobs,
				Storage:  storage,
			})
			if err != nil {
				return err
			}
			printTenant(resp.Tenant)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "tenant name")
	_ = cmd.MarkFlagRequired("name")
	cmd.Flags().StringVar(&database, "database", "", "external connection string for main schema (overrides auto-provisioning)")
	cmd.Flags().StringVar(&deploy, "deploy", "", "external connection string for deploy schema")
	cmd.Flags().StringVar(&account, "account", "", "external connection string for account schema")
	cmd.Flags().StringVar(&jobs, "jobs", "", "external connection string for jobs database")
	cmd.Flags().StringVar(&storage, "storage", "", "external storage connection string")
	return cmd
}

func tenantGetCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a tenant by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, err := rootAuthCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.tenant.Get(authContext, &tenant.GetRequest{Id: parseUUID(id)})
			if err != nil {
				return err
			}
			printTenant(resp.Tenant)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "tenant ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func tenantListCmd() *cobra.Command {
	var page, size int32
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tenants",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, err := rootAuthCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.tenant.List(authContext, &tenant.ListRequest{Page: page, PageSize: size})
			if err != nil {
				return err
			}
			w := newTabWriter()
			fmt.Fprintf(w, "ID\tNAME\tCREATED\n")
			for _, t := range resp.Tenants {
				fmt.Fprintf(w, "%s\t%s\t%s\n", formatUUID(t.Id), t.Name, t.CreatedAt)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().Int32Var(&page, "page", 1, "page number")
	cmd.Flags().Int32Var(&size, "size", 20, "page size")
	return cmd
}

func printTenant(t *tenant.Tenant) {
	printKV([][2]string{
		{"id", formatUUID(t.Id)},
		{"name", t.Name},
		{"database", t.Database},
		{"storage", t.Storage},
		{"owner_id", formatUUID(t.OwnerId)},
		{"created_at", t.CreatedAt},
		{"updated_at", t.UpdatedAt},
	})
}
