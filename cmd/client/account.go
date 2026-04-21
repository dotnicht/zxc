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
	cmd.AddCommand(accountGetCmd())
	cmd.AddCommand(accountListCmd())
	cmd.AddCommand(accountSearchCmd())
	cmd.AddCommand(accountDisableCmd())
	return cmd
}

func accountGetCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get an account by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx)
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
	cmd.Flags().StringVar(&id, "id", "", "account ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func accountListCmd() *cobra.Command {
	var page, size int32
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.account.List(authContext, &account.ListRequest{TenantId: tenantID, Page: page, PageSize: size})
			if err != nil {
				return err
			}
			w := newTabWriter()
			fmt.Fprintf(w, "ID\tNAME\tSTATUS\tCREATED\n")
			for _, a := range resp.Accounts {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.Id, a.Name, a.Status, a.CreatedAt)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().Int32Var(&page, "page", 1, "page number")
	cmd.Flags().Int32Var(&size, "size", 20, "page size")
	return cmd
}

func accountSearchCmd() *cobra.Command {
	var query string
	var page, size int32
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search accounts",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.account.Search(authContext, &account.SearchRequest{TenantId: tenantID, Query: query, Page: page, PageSize: size})
			if err != nil {
				return err
			}
			w := newTabWriter()
			fmt.Fprintf(w, "ID\tNAME\tSTATUS\tCREATED\n")
			for _, a := range resp.Accounts {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", a.Id, a.Name, a.Status, a.CreatedAt)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&query, "query", "", "search query")
	_ = cmd.MarkFlagRequired("query")
	cmd.Flags().Int32Var(&page, "page", 1, "page number")
	cmd.Flags().Int32Var(&size, "size", 20, "page size")
	return cmd
}

func accountDisableCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "disable",
		Short: "Disable an account",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.account.Disable(authContext, &account.DisableRequest{TenantId: tenantID, Id: id})
			if err != nil {
				return err
			}
			printAccount(resp.Account)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "account ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func printAccount(a *account.Account) {
	printKV([][2]string{
		{"id", a.Id},
		{"name", a.Name},
		{"status", a.Status},
		{"created_at", a.CreatedAt},
		{"updated_at", a.UpdatedAt},
	})
}
