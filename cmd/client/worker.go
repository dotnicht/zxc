package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"zxc/api/worker"
)

func workerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "worker",
		Short: "Manage workers",
	}
	cmd.AddCommand(workerAddCmd())
	cmd.AddCommand(workerGetCmd())
	cmd.AddCommand(workerListCmd())
	cmd.AddCommand(workerSearchCmd())
	cmd.AddCommand(workerUpdateCmd())
	cmd.AddCommand(workerDeleteCmd())
	cmd.AddCommand(workerAssignCmd())
	cmd.AddCommand(workerUnassignCmd())
	cmd.AddCommand(workerTenantsCmd())
	cmd.AddCommand(workerWorkersForTenantCmd())
	return cmd
}

func workerAddCmd() *cobra.Command {
	var id, name string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, err := rootAuthCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.worker.Create(authContext, &worker.CreateRequest{Id: id, Name: name})
			if err != nil {
				return err
			}
			printWorker(resp.Worker)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "worker ID (optional)")
	cmd.Flags().StringVar(&name, "name", "", "worker name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func workerGetCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a worker by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, err := rootAuthCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.worker.Get(authContext, &worker.GetRequest{Id: id})
			if err != nil {
				return err
			}
			printWorker(resp.Worker)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "worker ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func workerListCmd() *cobra.Command {
	var page, size int32
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List workers",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, err := rootAuthCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.worker.List(authContext, &worker.ListRequest{Page: page, PageSize: size})
			if err != nil {
				return err
			}
			w := newTabWriter()
			fmt.Fprintf(w, "ID\tNAME\tCREATED\n")
			for _, item := range resp.Workers {
				fmt.Fprintf(w, "%s\t%s\t%s\n", item.Id, item.Name, item.CreatedAt)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().Int32Var(&page, "page", 1, "page number")
	cmd.Flags().Int32Var(&size, "size", 20, "page size")
	return cmd
}

func workerSearchCmd() *cobra.Command {
	var query string
	var page, size int32
	cmd := &cobra.Command{
		Use:   "search",
		Short: "Search workers",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, err := rootAuthCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.worker.Search(authContext, &worker.SearchRequest{Query: query, Page: page, PageSize: size})
			if err != nil {
				return err
			}
			w := newTabWriter()
			fmt.Fprintf(w, "ID\tNAME\tCREATED\n")
			for _, item := range resp.Workers {
				fmt.Fprintf(w, "%s\t%s\t%s\n", item.Id, item.Name, item.CreatedAt)
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

func workerUpdateCmd() *cobra.Command {
	var id, name string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, err := rootAuthCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.worker.Update(authContext, &worker.UpdateRequest{Id: id, Name: name})
			if err != nil {
				return err
			}
			printWorker(resp.Worker)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "worker ID")
	_ = cmd.MarkFlagRequired("id")
	cmd.Flags().StringVar(&name, "name", "", "worker name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func workerDeleteCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, err := rootAuthCtx(ctx)
			if err != nil {
				return err
			}
			if _, err := st.worker.Delete(authContext, &worker.DeleteRequest{Id: id}); err != nil {
				return err
			}
			fmt.Println("deleted")
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "worker ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func workerAssignCmd() *cobra.Command {
	var workerID, tenantName string
	cmd := &cobra.Command{
		Use:   "assign",
		Short: "Assign a tenant to a worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, err := rootAuthCtx(ctx)
			if err != nil {
				return err
			}
			tenantID, err := resolveTenant(ctx, tenantName)
			if err != nil {
				return err
			}
			if _, err := st.worker.AssignTenant(authContext, &worker.AssignTenantRequest{WorkerId: workerID, TenantId: tenantID}); err != nil {
				return err
			}
			fmt.Println("assigned")
			return nil
		},
	}
	cmd.Flags().StringVar(&workerID, "worker-id", "", "worker ID")
	_ = cmd.MarkFlagRequired("worker-id")
	cmd.Flags().StringVar(&tenantName, "tenant-name", "", "tenant name")
	_ = cmd.MarkFlagRequired("tenant-name")
	return cmd
}

func workerUnassignCmd() *cobra.Command {
	var workerID, tenantName string
	cmd := &cobra.Command{
		Use:   "unassign",
		Short: "Remove a tenant assignment from a worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, err := rootAuthCtx(ctx)
			if err != nil {
				return err
			}
			tenantID, err := resolveTenant(ctx, tenantName)
			if err != nil {
				return err
			}
			if _, err := st.worker.UnassignTenant(authContext, &worker.UnassignTenantRequest{WorkerId: workerID, TenantId: tenantID}); err != nil {
				return err
			}
			fmt.Println("unassigned")
			return nil
		},
	}
	cmd.Flags().StringVar(&workerID, "worker-id", "", "worker ID")
	_ = cmd.MarkFlagRequired("worker-id")
	cmd.Flags().StringVar(&tenantName, "tenant-name", "", "tenant name")
	_ = cmd.MarkFlagRequired("tenant-name")
	return cmd
}

func workerTenantsCmd() *cobra.Command {
	var workerID string
	var page, size int32
	cmd := &cobra.Command{
		Use:   "tenants",
		Short: "List tenants assigned to a worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, err := rootAuthCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.worker.ListTenants(authContext, &worker.ListTenantsRequest{WorkerId: workerID, Page: page, PageSize: size})
			if err != nil {
				return err
			}
			w := newTabWriter()
			fmt.Fprintf(w, "ID\tNAME\tCREATED\n")
			for _, item := range resp.Tenants {
				fmt.Fprintf(w, "%s\t%s\t%s\n", item.Id, item.Name, item.CreatedAt)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&workerID, "worker-id", "", "worker ID")
	_ = cmd.MarkFlagRequired("worker-id")
	cmd.Flags().Int32Var(&page, "page", 1, "page number")
	cmd.Flags().Int32Var(&size, "size", 20, "page size")
	return cmd
}

func workerWorkersForTenantCmd() *cobra.Command {
	var tenantName string
	var page, size int32
	cmd := &cobra.Command{
		Use:   "workers-for-tenant",
		Short: "List workers assigned to a tenant",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, err := rootAuthCtx(ctx)
			if err != nil {
				return err
			}
			tenantID, err := resolveTenant(ctx, tenantName)
			if err != nil {
				return err
			}
			resp, err := st.worker.ListWorkersForTenant(authContext, &worker.ListWorkersForTenantRequest{TenantId: tenantID, Page: page, PageSize: size})
			if err != nil {
				return err
			}
			w := newTabWriter()
			fmt.Fprintf(w, "ID\tNAME\tCREATED\n")
			for _, item := range resp.Workers {
				fmt.Fprintf(w, "%s\t%s\t%s\n", item.Id, item.Name, item.CreatedAt)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().StringVar(&tenantName, "tenant-name", "", "tenant name")
	_ = cmd.MarkFlagRequired("tenant-name")
	cmd.Flags().Int32Var(&page, "page", 1, "page number")
	cmd.Flags().Int32Var(&size, "size", 20, "page size")
	return cmd
}

func printWorker(record *worker.Worker) {
	printKV([][2]string{
		{"id", record.Id},
		{"name", record.Name},
		{"created_at", record.CreatedAt},
		{"updated_at", record.UpdatedAt},
	})
}
