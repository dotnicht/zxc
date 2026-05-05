package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"zxc/api/payload"
)

func payloadCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "payload",
		Short: "Manage payloads",
	}
	cmd.AddCommand(payloadAddCmd())
	cmd.AddCommand(payloadGetCmd())
	cmd.AddCommand(payloadListCmd())
	cmd.AddCommand(payloadUpdateCmd())
	cmd.AddCommand(payloadDeleteCmd())
	return cmd
}

func payloadAddCmd() *cobra.Command {
	var file, config, start, stop string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a payload",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, userID, err := tenantOwnerCtx(ctx)
			if err != nil {
				return err
			}
			content, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("read file %s: %w", file, err)
			}
			resp, err := st.payload.Create(authContext, &payload.CreateRequest{
				OwnerId: parseUUID(userID),
				Content: content,
				Name:    filepath.Base(file),
				Config:  config,
				Start:   start,
				Stop:    stop,
			})
			if err != nil {
				return err
			}
			printPayload(resp.Payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&file, "file", "", "path to file to upload")
	_ = cmd.MarkFlagRequired("file")
	cmd.Flags().StringVar(&config, "config", "", "name of the config file inside the zip (must contain {ZXC_URL} and {ZXC_AUTH})")
	_ = cmd.MarkFlagRequired("config")
	cmd.Flags().StringVar(&start, "start", "", "start command")
	_ = cmd.MarkFlagRequired("start")
	cmd.Flags().StringVar(&stop, "stop", "", "stop command")
	_ = cmd.MarkFlagRequired("stop")
	return cmd
}

func payloadGetCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a payload by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.payload.Get(authContext, &payload.GetRequest{Id: parseUUID(id)})
			if err != nil {
				return err
			}
			printPayload(resp.Payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "payload ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func payloadListCmd() *cobra.Command {
	var page, size int32
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List payloads",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.payload.List(authContext, &payload.ListRequest{Page: page, PageSize: size})
			if err != nil {
				return err
			}
			w := newTabWriter()
			fmt.Fprintf(w, "ID\tPATH\tCREATED\n")
			for _, p := range resp.Payloads {
				fmt.Fprintf(w, "%s\t%s\t%s\n", formatUUID(p.Id), p.Path, p.CreatedAt)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().Int32Var(&page, "page", 1, "page number")
	cmd.Flags().Int32Var(&size, "size", 20, "page size")
	return cmd
}

func payloadUpdateCmd() *cobra.Command {
	var id, path, config, start, stop string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a payload",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.payload.Update(authContext, &payload.UpdateRequest{
				Id:     parseUUID(id),
				Path:   path,
				Config: config,
				Start:  start,
				Stop:   stop,
			})
			if err != nil {
				return err
			}
			printPayload(resp.Payload)
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "payload ID")
	_ = cmd.MarkFlagRequired("id")
	cmd.Flags().StringVar(&path, "path", "", "payload path")
	cmd.Flags().StringVar(&config, "config", "", "config file template")
	cmd.Flags().StringVar(&start, "start", "", "start command")
	cmd.Flags().StringVar(&stop, "stop", "", "stop command")
	return cmd
}

func payloadDeleteCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a payload",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			_, err = st.payload.Delete(authContext, &payload.DeleteRequest{Id: parseUUID(id)})
			if err != nil {
				return err
			}
			fmt.Println("deleted")
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "payload ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func printPayload(p *payload.Payload) {
	printKV([][2]string{
		{"id", formatUUID(p.Id)},
		{"path", p.Path},
		{"owner_id", formatUUID(p.OwnerId)},
		{"config", p.Config},
		{"start", p.Start},
		{"stop", p.Stop},
		{"created_at", p.CreatedAt},
		{"updated_at", p.UpdatedAt},
	})
}
