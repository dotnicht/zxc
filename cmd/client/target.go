package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"zxc/api/target"
)

func targetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "target",
		Short: "Manage targets",
	}
	cmd.AddCommand(targetAddCmd())
	cmd.AddCommand(targetGetCmd())
	cmd.AddCommand(targetListCmd())
	cmd.AddCommand(targetSearchCmd())
	cmd.AddCommand(targetUpdateCmd())
	cmd.AddCommand(targetDeleteCmd())
	return cmd
}

func targetAddCmd() *cobra.Command {
	var address, user, key string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a target",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, userID, err := tenantOwnerCtx(ctx)
			if err != nil {
				return err
			}
			keyContent, err := os.ReadFile(key)
			if err != nil {
				return fmt.Errorf("read key %s: %w", key, err)
			}
			resp, err := st.target.Create(authContext, &target.CreateRequest{
				TenantId: tenantID,
				OwnerId:  userID,
				Address:  address,
				User:     user,
				Key:      string(keyContent),
			})
			if err != nil {
				return err
			}
			printTarget(resp.Target)
			return nil
		},
	}
	cmd.Flags().StringVar(&address, "address", "", "target address")
	_ = cmd.MarkFlagRequired("address")
	cmd.Flags().StringVar(&user, "user", "", "ssh user")
	_ = cmd.MarkFlagRequired("user")
	cmd.Flags().StringVar(&key, "key", "", "path to ssh private key file")
	_ = cmd.MarkFlagRequired("key")
	return cmd
}

func targetGetCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a target by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.target.Get(authContext, &target.GetRequest{TenantId: tenantID, Id: id})
			if err != nil {
				return err
			}
			printTarget(resp.Target)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "target ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func targetListCmd() *cobra.Command {
	var page, size int32
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List targets",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.target.List(authContext, &target.ListRequest{TenantId: tenantID, Page: page, PageSize: size})
			if err != nil {
				return err
			}
			w := newTabWriter()
			fmt.Fprintf(w, "ID\tADDRESS\tSTATUS\tCREATED\n")
			for _, t := range resp.Targets {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.Id, t.Address, t.Status, t.CreatedAt)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().Int32Var(&page, "page", 1, "page number")
	cmd.Flags().Int32Var(&size, "size", 20, "page size")
	return cmd
}

func targetSearchCmd() *cobra.Command {
	var query string
	var page, size int32
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search targets",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.target.Search(authContext, &target.SearchRequest{TenantId: tenantID, Query: query, Page: page, PageSize: size})
			if err != nil {
				return err
			}
			w := newTabWriter()
			fmt.Fprintf(w, "ID\tADDRESS\tSTATUS\tCREATED\n")
			for _, t := range resp.Targets {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", t.Id, t.Address, t.Status, t.CreatedAt)
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

func targetUpdateCmd() *cobra.Command {
	var id, address, user, key string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a target",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			var keyContent string
			if key != "" {
				kb, err := os.ReadFile(key)
				if err != nil {
					return fmt.Errorf("read key %s: %w", key, err)
				}
				keyContent = string(kb)
			}
			resp, err := st.target.Update(authContext, &target.UpdateRequest{TenantId: tenantID, Id: id, Address: address, User: user, Key: keyContent})
			if err != nil {
				return err
			}
			printTarget(resp.Target)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "target ID")
	_ = cmd.MarkFlagRequired("id")
	cmd.Flags().StringVar(&address, "address", "", "target address")
	_ = cmd.MarkFlagRequired("address")
	cmd.Flags().StringVar(&user, "user", "", "ssh user")
	cmd.Flags().StringVar(&key, "key", "", "path to ssh private key file")
	return cmd
}

func printTarget(t *target.Target) {
	printKV([][2]string{
		{"id", t.Id},
		{"address", t.Address},
		{"user", t.User},
		{"status", t.Status},
		{"owner_id", t.OwnerId},
		{"created_at", t.CreatedAt},
		{"updated_at", t.UpdatedAt},
	})
}

func targetDeleteCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a target",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, tenantID, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			_, err = st.target.Delete(authContext, &target.DeleteRequest{TenantId: tenantID, Id: id})
			if err != nil {
				return err
			}
			fmt.Println("deleted")
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "target ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}
