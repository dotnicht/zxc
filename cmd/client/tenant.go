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
	var name string
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
			resp, err := st.tenant.Create(authContext, &tenant.CreateRequest{Name: name})
			if err != nil {
				return err
			}
			t := resp.Tenant
			printKV([][2]string{
				{"id", t.Id},
				{"name", t.Name},
				{"database", t.Database},
				{"storage", t.Storage},
				{"owner_id", t.OwnerId},
				{"created_at", t.CreatedAt},
				{"updated_at", t.UpdatedAt},
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "tenant name")
	_ = cmd.MarkFlagRequired("name")
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
			resp, err := st.tenant.Get(authContext, &tenant.GetRequest{Id: id})
			if err != nil {
				return err
			}
			t := resp.Tenant
			printKV([][2]string{
				{"id", t.Id},
				{"name", t.Name},
				{"database", t.Database},
				{"storage", t.Storage},
				{"owner_id", t.OwnerId},
				{"created_at", t.CreatedAt},
				{"updated_at", t.UpdatedAt},
			})
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
				fmt.Fprintf(w, "%s\t%s\t%s\n", t.Id, t.Name, t.CreatedAt)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().Int32Var(&page, "page", 1, "page number")
	cmd.Flags().Int32Var(&size, "size", 20, "page size")
	return cmd
}
