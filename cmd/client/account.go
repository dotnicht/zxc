package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"zxc/api/account"
)

func accountCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Manage accounts",
	}
	cmd.AddCommand(accountAddCmd())
	cmd.AddCommand(accountGetCmd())
	cmd.AddCommand(accountListCmd())
	cmd.AddCommand(accountSearchCmd())
	cmd.AddCommand(accountUpdateCmd())
	cmd.AddCommand(accountDeleteCmd())
	return cmd
}

func accountAddCmd() *cobra.Command {
	var tenant, name string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create an account",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx, tenant)
			if err != nil {
				return err
			}
			resp, err := st.account.Create(authContext, &account.CreateRequest{TenantId: tenantID, Name: name})
			if err != nil {
				return err
			}
			printAccount(resp.Account)
			return nil
		},
	}
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant name")
	_ = cmd.MarkFlagRequired("tenant")
	cmd.Flags().StringVar(&name, "name", "", "account name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func accountGetCmd() *cobra.Command {
	var tenant, id string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get an account by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx, tenant)
			if err != nil {
				return err
			}
			resp, err := st.account.Get(authContext, &account.GetRequest{TenantId: tenantID, Id: id})
			if err != nil {
				return err
			}
			printAccount(resp.Account)
			return nil
		},
	}
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant name")
	_ = cmd.MarkFlagRequired("tenant")
	cmd.Flags().StringVar(&id, "id", "", "account ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func accountListCmd() *cobra.Command {
	var tenant string
	var page, size int32
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx, tenant)
			if err != nil {
				return err
			}
			resp, err := st.account.List(authContext, &account.ListRequest{TenantId: tenantID, Page: page, PageSize: size})
			if err != nil {
				return err
			}
			w := newTabWriter()
			fmt.Fprintf(w, "ID\tNAME\tCREATED\n")
			for _, a := range resp.Accounts {
				fmt.Fprintf(w, "%s\t%s\t%s\n", a.Id, a.Name, a.CreatedAt)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant name")
	_ = cmd.MarkFlagRequired("tenant")
	cmd.Flags().Int32Var(&page, "page", 1, "page number")
	cmd.Flags().Int32Var(&size, "size", 20, "page size")
	return cmd
}

func accountSearchCmd() *cobra.Command {
	var tenant, query string
	var page, size int32
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx, tenant)
			if err != nil {
				return err
			}
			resp, err := st.account.Search(authContext, &account.SearchRequest{TenantId: tenantID, Query: query, Page: page, PageSize: size})
			if err != nil {
				return err
			}
			w := newTabWriter()
			fmt.Fprintf(w, "ID\tNAME\tCREATED\n")
			for _, a := range resp.Accounts {
				fmt.Fprintf(w, "%s\t%s\t%s\n", a.Id, a.Name, a.CreatedAt)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant name")
	_ = cmd.MarkFlagRequired("tenant")
	cmd.Flags().StringVar(&query, "query", "", "search query")
	_ = cmd.MarkFlagRequired("query")
	cmd.Flags().Int32Var(&page, "page", 1, "page number")
	cmd.Flags().Int32Var(&size, "size", 20, "page size")
	return cmd
}

func accountUpdateCmd() *cobra.Command {
	var tenant, id, name string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an account",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx, tenant)
			if err != nil {
				return err
			}
			resp, err := st.account.Update(authContext, &account.UpdateRequest{TenantId: tenantID, Id: id, Name: name})
			if err != nil {
				return err
			}
			printAccount(resp.Account)
			return nil
		},
	}
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant name")
	_ = cmd.MarkFlagRequired("tenant")
	cmd.Flags().StringVar(&id, "id", "", "account ID")
	_ = cmd.MarkFlagRequired("id")
	cmd.Flags().StringVar(&name, "name", "", "new account name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func accountDeleteCmd() *cobra.Command {
	var tenant, id string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an account",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx, tenant)
			if err != nil {
				return err
			}
			if _, err := st.account.Delete(authContext, &account.DeleteRequest{TenantId: tenantID, Id: id}); err != nil {
				return err
			}
			fmt.Println("deleted")
			return nil
		},
	}
	cmd.Flags().StringVar(&tenant, "tenant", "", "tenant name")
	_ = cmd.MarkFlagRequired("tenant")
	cmd.Flags().StringVar(&id, "id", "", "account ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func printAccount(a *account.Account) {
	printKV([][2]string{
		{"id", a.Id},
		{"name", a.Name},
		{"created_at", a.CreatedAt},
		{"updated_at", a.UpdatedAt},
	})
}
