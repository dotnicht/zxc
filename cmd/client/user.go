package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"zxc/api/user"
)

func userCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage users",
	}
	cmd.AddCommand(userAddCmd())
	cmd.AddCommand(userGetCmd())
	cmd.AddCommand(userListCmd())
	cmd.AddCommand(userUpdateCmd())
	cmd.AddCommand(userDeleteCmd())
	return cmd
}

func userAddCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.user.Create(authContext, &user.CreateRequest{
				Name: name,
			})
			if err != nil {
				return err
			}
			u := resp.User
			printKV([][2]string{
				{"id", u.Id},
				{"name", u.Name},
				{"created_at", u.CreatedAt},
				{"updated_at", u.UpdatedAt},
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "user name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func userGetCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a user by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.user.Get(authContext, &user.GetRequest{Id: id})
			if err != nil {
				return err
			}
			u := resp.User
			printKV([][2]string{
				{"id", u.Id},
				{"name", u.Name},
				{"created_at", u.CreatedAt},
				{"updated_at", u.UpdatedAt},
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "user ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}

func userListCmd() *cobra.Command {
	var page, size int32
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List users",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.user.List(authContext, &user.ListRequest{Page: page, PageSize: size})
			if err != nil {
				return err
			}
			w := newTabWriter()
			fmt.Fprintf(w, "ID\tNAME\tCREATED\n")
			for _, u := range resp.Users {
				fmt.Fprintf(w, "%s\t%s\t%s\n", u.Id, u.Name, u.CreatedAt)
			}
			w.Flush()
			return nil
		},
	}
	cmd.Flags().Int32Var(&page, "page", 1, "page number")
	cmd.Flags().Int32Var(&size, "size", 20, "page size")
	return cmd
}

func userUpdateCmd() *cobra.Command {
	var id, name string
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			resp, err := st.user.Update(authContext, &user.UpdateRequest{Id: id, Name: name})
			if err != nil {
				return err
			}
			u := resp.User
			printKV([][2]string{
				{"id", u.Id},
				{"name", u.Name},
				{"created_at", u.CreatedAt},
				{"updated_at", u.UpdatedAt},
			})
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "user ID")
	_ = cmd.MarkFlagRequired("id")
	cmd.Flags().StringVar(&name, "name", "", "new user name")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func userDeleteCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := newCtx()
			defer cancel()
			authContext, _, err := tenantCtx(ctx)
			if err != nil {
				return err
			}
			_, err = st.user.Delete(authContext, &user.DeleteRequest{Id: id})
			if err != nil {
				return err
			}
			fmt.Println("deleted")
			return nil
		},
	}
	cmd.Flags().StringVar(&id, "id", "", "user ID")
	_ = cmd.MarkFlagRequired("id")
	return cmd
}
