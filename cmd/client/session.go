package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"zxc/api/session"
)

func sessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage sessions",
	}
	cmd.AddCommand(sessionAddCmd())
	cmd.AddCommand(sessionGetCmd())
	cmd.AddCommand(sessionListCmd())
	cmd.AddCommand(sessionSearchCmd())
	cmd.AddCommand(sessionUpdateCmd())
	cmd.AddCommand(sessionDeleteCmd())
	return cmd
}

func sessionAddCmd() *cobra.Command {
	var accountID, status string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a session",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.session.Create(authContext, &session.CreateRequest{
				TenantId:  tenantID,
				AccountId: accountID,
				Status:    status,
			})
			if err != nil {
				return err
			}
			printSession(resp.Session)
			return nil
		},
	}
	cmd.Flags().StringVar(&accountID, "account", "", "account ID")
	_ = cmd.MarkFlagRequired("account")
	cmd.Flags().StringVar(&status, "status", "", "session status: online, offline, sync")
	_ = cmd.MarkFlagRequired("status")
	return cmd
}

func sessionGetCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a session by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.session.Get(authContext, &session.GetRequest{TenantId: tenantID, Id: id})
			if err != nil {
				return err
			}
			printSession(resp.Session)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "session ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func sessionListCmd() *cobra.Command {
	var page, size int32
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.session.List(authContext, &session.ListRequest{TenantId: tenantID, Page: page, PageSize: size})
			if err != nil {
				return err
			}
			w := newTabWriter()
			fmt.Fprintf(w, "ID\tACCOUNT_ID\tSTATUS\tCREATED\n")
			for _, record := range resp.Sessions {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", record.Id, record.AccountId, record.Status, record.CreatedAt)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().Int32Var(&page, "page", 1, "page number")
	cmd.Flags().Int32Var(&size, "size", 20, "page size")
	return cmd
}

func sessionSearchCmd() *cobra.Command {
	var query string
	var page, size int32
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.session.Search(authContext, &session.SearchRequest{TenantId: tenantID, Query: query, Page: page, PageSize: size})
			if err != nil {
				return err
			}
			w := newTabWriter()
			fmt.Fprintf(w, "ID\tACCOUNT_ID\tSTATUS\tCREATED\n")
			for _, record := range resp.Sessions {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", record.Id, record.AccountId, record.Status, record.CreatedAt)
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

func sessionUpdateCmd() *cobra.Command {
	var id, accountID, status string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a session",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.session.Update(authContext, &session.UpdateRequest{
				TenantId:  tenantID,
				Id:        id,
				AccountId: accountID,
				Status:    status,
			})
			if err != nil {
				return err
			}
			printSession(resp.Session)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "session ID")
	_ = cmd.MarkFlagRequired("id")
	cmd.Flags().StringVar(&accountID, "account", "", "account ID")
	_ = cmd.MarkFlagRequired("account")
	cmd.Flags().StringVar(&status, "status", "", "session status: online, offline, sync")
	_ = cmd.MarkFlagRequired("status")
	return cmd
}

func sessionDeleteCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a session",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			if _, err := st.session.Delete(authContext, &session.DeleteRequest{TenantId: tenantID, Id: id}); err != nil {
				return err
			}
			fmt.Println("deleted")
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "session ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func printSession(record *session.Session) {
	printKV([][2]string{
		{"id", record.Id},
		{"account_id", record.AccountId},
		{"status", record.Status},
		{"created_at", record.CreatedAt},
		{"updated_at", record.UpdatedAt},
	})
}
