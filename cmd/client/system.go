package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"zxc/api/system"
)

func systemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "system",
		Short: "Manage systems",
	}
	cmd.AddCommand(systemAddCmd())
	cmd.AddCommand(systemGetCmd())
	cmd.AddCommand(systemListCmd())
	cmd.AddCommand(systemUpdateCmd())
	cmd.AddCommand(systemDeleteCmd())
	return cmd
}

func systemAddCmd() *cobra.Command {
	var name, sync string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a system",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.system.Create(authContext, &system.CreateRequest{Name: name, Sync: sync})
			if err != nil {
				return err
			}
			printSystem(resp.System)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "system name")
	_ = cmd.MarkFlagRequired("name")
	cmd.Flags().StringVar(&sync, "sync", "generator", "plugin binary name")
	return cmd
}

func systemGetCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a system by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.system.Get(authContext, &system.GetRequest{Id: parseUUID(id)})
			if err != nil {
				return err
			}
			printSystem(resp.System)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "system ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func systemListCmd() *cobra.Command {
	var page, size int32
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List systems",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.system.List(authContext, &system.ListRequest{Page: page, PageSize: size})
			if err != nil {
				return err
			}
			w := writer()
			fmt.Fprintf(w, "ID\tNAME\tCREATED\n")
			for _, s := range resp.Systems {
				fmt.Fprintf(w, "%s\t%s\t%s\n", formatUUID(s.Id), s.Name, s.CreatedAt)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().Int32Var(&page, "page", 1, "page number")
	cmd.Flags().Int32Var(&size, "size", 20, "page size")
	return cmd
}

func systemUpdateCmd() *cobra.Command {
	var id, name, sync string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a system",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.system.Update(authContext, &system.UpdateRequest{Id: parseUUID(id), Name: name, Sync: sync})
			if err != nil {
				return err
			}
			printSystem(resp.System)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "system ID")
	_ = cmd.MarkFlagRequired("id")
	cmd.Flags().StringVar(&name, "name", "", "new system name")
	_ = cmd.MarkFlagRequired("name")
	cmd.Flags().StringVar(&sync, "sync", "", "plugin binary name")
	return cmd
}

func systemDeleteCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a system",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			_, err = st.system.Delete(authContext, &system.DeleteRequest{Id: parseUUID(id)})
			if err != nil {
				return err
			}
			fmt.Println("deleted")
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "system ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func printSystem(s *system.System) {
	printKV([][2]string{
		{"id", formatUUID(s.Id)},
		{"name", s.Name},
		{"sync", s.Sync},
		{"created_at", s.CreatedAt},
		{"updated_at", s.UpdatedAt},
	})
}
