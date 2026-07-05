package radiusctl

import (
	"fmt"

	"github.com/spf13/cobra"
)

// nasCmd returns the `radiusctl nas` subcommand group.
func nasCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nas",
		Short: "Manage NAS devices",
	}
	cmd.AddCommand(
		nasListCmd(),
		nasCreateCmd(),
		nasUpdateCmd(),
		nasDeleteCmd(),
	)
	return cmd
}

func nasListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all NAS devices",
		RunE: func(cmd *cobra.Command, args []string) error {
			items, err := client.ListNAS(cmd.Context())
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(items)
				return nil
			}
			if len(items) == 0 {
				fmt.Println("no NAS devices")
				return nil
			}
			w := NewWriter()
			w.Row("ID", "NAME", "IP", "ENABLED", "DESCRIPTION")
			for _, n := range items {
				w.Row(n.ID, n.Name, n.IPAddress, yesNo(n.Enabled), n.Description)
			}
			w.Flush()
			return nil
		},
	}
}

func nasCreateCmd() *cobra.Command {
	var (
		name, ip, secret, desc string
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Register a new NAS device",
		RunE: func(cmd *cobra.Command, args []string) error {
			req := CreateNASRequest{
				Name:        name,
				IPAddress:   ip,
				Secret:      secret,
				Description: desc,
			}
			n, err := client.CreateNAS(cmd.Context(), req)
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(n)
				return nil
			}
			KV(
				[2]string{"id", n.ID},
				[2]string{"name", n.Name},
				[2]string{"ip_address", n.IPAddress},
				[2]string{"enabled", yesNo(n.Enabled)},
				[2]string{"description", n.Description},
			)
			return nil
		},
	}
	cmd.Flags().StringVarP(&name, "name", "n", "", "NAS name (required)")
	cmd.Flags().StringVarP(&ip, "ip", "i", "", "IP address (required)")
	cmd.Flags().StringVarP(&secret, "secret", "s", "", "RADIUS shared secret (required)")
	cmd.Flags().StringVarP(&desc, "description", "d", "", "Description")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("ip")
	cmd.MarkFlagRequired("secret")
	return cmd
}

func nasUpdateCmd() *cobra.Command {
	var (
		id, name, ip, secret, desc string
		enabled, disabled          bool
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an existing NAS device",
		RunE: func(cmd *cobra.Command, args []string) error {
			if enabled && disabled {
				return fmt.Errorf("--enabled and --disabled are mutually exclusive")
			}
			req := UpdateNASRequest{
				Name:        name,
				IPAddress:   ip,
				Secret:      secret,
				Description: desc,
			}
			if enabled || disabled {
				v := enabled // disabled=false → req.Enabled=false
				req.Enabled = &v
			}
			n, err := client.UpdateNAS(cmd.Context(), id, req)
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(n)
				return nil
			}
			KV(
				[2]string{"id", n.ID},
				[2]string{"name", n.Name},
				[2]string{"ip_address", n.IPAddress},
				[2]string{"enabled", yesNo(n.Enabled)},
				[2]string{"description", n.Description},
			)
			return nil
		},
	}
	cmd.Flags().StringVarP(&id, "id", "i", "", "NAS UUID (required)")
	cmd.Flags().StringVarP(&name, "name", "n", "", "NAS name")
	cmd.Flags().StringVar(&ip, "ip", "", "IP address")
	cmd.Flags().StringVarP(&secret, "secret", "s", "", "Shared secret")
	cmd.Flags().StringVarP(&desc, "description", "d", "", "Description")
	cmd.Flags().BoolVar(&enabled, "enable", false, "Enable the NAS")
	cmd.Flags().BoolVar(&disabled, "disable", false, "Disable the NAS")
	cmd.MarkFlagRequired("id")
	return cmd
}

func nasDeleteCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a NAS device",
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := client.DeleteNAS(cmd.Context(), id)
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(r)
				return nil
			}
			fmt.Printf("NAS %s deleted\n", r.ID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&id, "id", "i", "", "NAS UUID (required)")
	cmd.MarkFlagRequired("id")
	return cmd
}
