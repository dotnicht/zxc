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
	cmd.AddCommand(sessionGetCmd())
	cmd.AddCommand(sessionListCmd())
	cmd.AddCommand(sessionStartCmd())
	cmd.AddCommand(sessionStopCmd())
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
			authContext, _, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.session.Get(authContext, &session.GetRequest{Id: parseUUID(id)})
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
			authContext, _, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.session.List(authContext, &session.ListRequest{Page: page, PageSize: size})
			if err != nil {
				return err
			}
			w := newTabWriter()
			fmt.Fprintf(w, "ID\tACCOUNT_ID\tSTATUS\tCREATED\n")
			for _, record := range resp.Sessions {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", formatUUID(record.Id), formatUUID(record.AccountId), record.Status, record.CreatedAt)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().Int32Var(&page, "page", 1, "page number")
	cmd.Flags().Int32Var(&size, "size", 20, "page size")
	return cmd
}

func sessionStartCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start a session",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.session.Start(authContext, &session.StartRequest{Id: parseUUID(id)})
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

func sessionStopCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop a session",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.session.Stop(authContext, &session.StopRequest{Id: parseUUID(id)})
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

func printSession(record *session.Session) {
	printKV([][2]string{
		{"id", formatUUID(record.Id)},
		{"account_id", formatUUID(record.AccountId)},
		{"status", record.Status},
		{"created_at", record.CreatedAt},
		{"updated_at", record.UpdatedAt},
	})
}
