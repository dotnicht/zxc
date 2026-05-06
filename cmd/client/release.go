package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"zxc/api/release"
)

func releaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "release",
		Short: "Manage releases",
	}
	cmd.AddCommand(releaseAddCmd())
	cmd.AddCommand(releaseGetCmd())
	cmd.AddCommand(releaseListCmd())
	cmd.AddCommand(releaseDeployCmd())
	return cmd
}

func releaseAddCmd() *cobra.Command {
	var targetID, payloadID string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a release",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, userID, err := tenantOwnerCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.release.Create(authContext, &release.CreateRequest{
				OwnerId:   parseUUID(userID),
				TargetId:  parseUUID(targetID),
				PayloadId: parseUUID(payloadID),
			})
			if err != nil {
				return err
			}
			printRelease(resp.Release)
			return nil
		},
	}
	cmd.Flags().StringVar(&targetID, "target", "", "target ID")
	_ = cmd.MarkFlagRequired("target")
	cmd.Flags().StringVar(&payloadID, "payload", "", "payload ID")
	_ = cmd.MarkFlagRequired("payload")
	return cmd
}

func releaseGetCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a release by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.release.Get(authContext, &release.GetRequest{Id: parseUUID(id)})
			if err != nil {
				return err
			}
			printRelease(resp.Release)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "release ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func releaseListCmd() *cobra.Command {
	var page, size int32
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List releases",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.release.List(authContext, &release.ListRequest{Page: page, PageSize: size})
			if err != nil {
				return err
			}
			w := writer()
			fmt.Fprintf(w, "ID\tSTATUS\tTARGET_ID\tPAYLOAD_ID\tCREATED\n")
			for _, r := range resp.Releases {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", formatUUID(r.Id), r.Status, formatUUID(r.TargetId), formatUUID(r.PayloadId), r.CreatedAt)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().Int32Var(&page, "page", 1, "page number")
	cmd.Flags().Int32Var(&size, "size", 20, "page size")
	return cmd
}

func releaseDeployCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Trigger release deployment",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, userID, err := tenantOwnerCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.release.Deploy(authContext, &release.DeployRequest{
				Id:     parseUUID(id),
				UserId: parseUUID(userID),
			})
			if err != nil {
				return err
			}
			printRelease(resp.Release)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "release ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func printRelease(r *release.Release) {
	printKV([][2]string{
		{"id", formatUUID(r.Id)},
		{"status", r.Status},
		{"owner_id", formatUUID(r.OwnerId)},
		{"target_id", formatUUID(r.TargetId)},
		{"payload_id", formatUUID(r.PayloadId)},
		{"changed_by_id", formatUUID(r.ChangedById)},
		{"created_at", r.CreatedAt},
		{"updated_at", r.UpdatedAt},
	})
}
