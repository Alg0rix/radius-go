package radiusctl

import (
	"fmt"

	"github.com/spf13/cobra"
)

// subscriberCmd returns the `radiusctl subscriber` subcommand group.
func subscriberCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "subscriber",
		Aliases: []string{"user"},
		Short:   "Manage RADIUS subscribers (users)",
	}
	cmd.AddCommand(
		subscriberListCmd(),
		subscriberCreateCmd(),
		subscriberUpdateCmd(),
		subscriberDeleteCmd(),
	)
	return cmd
}

func subscriberListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all subscribers",
		RunE: func(cmd *cobra.Command, args []string) error {
			items, err := client.ListSubscribers(cmd.Context())
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(items)
				return nil
			}
			if len(items) == 0 {
				fmt.Println("no subscribers")
				return nil
			}
			w := NewWriter()
			w.Row("ID", "USERNAME", "ENABLED", "SERVICE", "GROUP", "RATE")
			for _, s := range items {
				w.Row(s.ID, s.Username, yesNo(s.Enabled), s.ServiceType, s.FramedIP+s.MikrotikGroup, s.RateLimit)
			}
			w.Flush()
			return nil
		},
	}
}

func subscriberCreateCmd() *cobra.Command {
	var (
		username, password, fullname, email, framedIP, group, rate, service string
		simulUse, sessTimeout, idleTimeout                                  int
		bwUp, bwDown, maxOctets                                             uint32
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new subscriber",
		RunE: func(cmd *cobra.Command, args []string) error {
			req := CreateSubscriberRequest{
				Username:         username,
				Password:         password,
				FullName:         fullname,
				Email:            email,
				SimultaneousUse:  simulUse,
				SessionTimeout:   sessTimeout,
				IdleTimeout:      idleTimeout,
				FramedIP:         framedIP,
				MikrotikGroup:    group,
				RateLimit:        rate,
				BandwidthMaxUp:   bwUp,
				BandwidthMaxDown: bwDown,
				MaxTotalOctets:   maxOctets,
				ServiceType:      service,
			}
			s, err := client.CreateSubscriber(cmd.Context(), req)
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(s)
				return nil
			}
			printSubscriber(s)
			return nil
		},
	}
	cmd.Flags().StringVarP(&username, "username", "u", "", "Username (required)")
	cmd.Flags().StringVarP(&password, "password", "p", "", "Password (required)")
	cmd.Flags().StringVarP(&fullname, "full-name", "n", "", "Full name")
	cmd.Flags().StringVarP(&email, "email", "e", "", "Email")
	cmd.Flags().StringVar(&framedIP, "framed-ip", "", "Framed IP address")
	cmd.Flags().StringVar(&group, "mikrotik-group", "", "MikroTik group")
	cmd.Flags().StringVar(&rate, "rate-limit", "", "MikroTik rate limit string")
	cmd.Flags().StringVar(&service, "service-type", "framed", "Service type: framed|login")
	cmd.Flags().IntVar(&simulUse, "simultaneous-use", 0, "Max simultaneous sessions")
	cmd.Flags().IntVar(&sessTimeout, "session-timeout", 0, "Session timeout (seconds)")
	cmd.Flags().IntVar(&idleTimeout, "idle-timeout", 0, "Idle timeout (seconds)")
	cmd.Flags().Uint32Var(&bwUp, "bw-up", 0, "Max upload bandwidth")
	cmd.Flags().Uint32Var(&bwDown, "bw-down", 0, "Max download bandwidth")
	cmd.Flags().Uint32Var(&maxOctets, "max-octets", 0, "Max total octets")
	cmd.MarkFlagRequired("username")
	cmd.MarkFlagRequired("password")
	return cmd
}

func subscriberUpdateCmd() *cobra.Command {
	var (
		id, username, password, fullname, email, framedIP, group, rate, service string
		simulUse, sessTimeout, idleTimeout                                      int
		bwUp, bwDown, maxOctets                                                 uint32
		enabled, disabled                                                       bool
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an existing subscriber",
		RunE: func(cmd *cobra.Command, args []string) error {
			if enabled && disabled {
				return fmt.Errorf("--enable and --disable are mutually exclusive")
			}
			req := UpdateSubscriberRequest{
				Username:      username,
				Password:      password,
				FullName:      fullname,
				Email:         email,
				FramedIP:      framedIP,
				MikrotikGroup: group,
				RateLimit:     rate,
				ServiceType:   service,
			}
			if cmd.Flags().Changed("simultaneous-use") {
				req.SimultaneousUse = &simulUse
			}
			if cmd.Flags().Changed("session-timeout") {
				req.SessionTimeout = &sessTimeout
			}
			if cmd.Flags().Changed("idle-timeout") {
				req.IdleTimeout = &idleTimeout
			}
			if cmd.Flags().Changed("bw-up") {
				req.BandwidthMaxUp = &bwUp
			}
			if cmd.Flags().Changed("bw-down") {
				req.BandwidthMaxDown = &bwDown
			}
			if cmd.Flags().Changed("max-octets") {
				req.MaxTotalOctets = &maxOctets
			}
			if enabled || disabled {
				v := enabled
				req.Enabled = &v
			}
			s, err := client.UpdateSubscriber(cmd.Context(), id, req)
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(s)
				return nil
			}
			printSubscriber(s)
			return nil
		},
	}
	cmd.Flags().StringVarP(&id, "id", "i", "", "Subscriber UUID (required)")
	cmd.Flags().StringVarP(&username, "username", "u", "", "Username")
	cmd.Flags().StringVarP(&password, "password", "p", "", "Password")
	cmd.Flags().StringVarP(&fullname, "full-name", "n", "", "Full name")
	cmd.Flags().StringVarP(&email, "email", "e", "", "Email")
	cmd.Flags().StringVar(&framedIP, "framed-ip", "", "Framed IP address")
	cmd.Flags().StringVar(&group, "mikrotik-group", "", "MikroTik group")
	cmd.Flags().StringVar(&rate, "rate-limit", "", "MikroTik rate limit string")
	cmd.Flags().StringVar(&service, "service-type", "", "Service type: framed|login")
	cmd.Flags().IntVar(&simulUse, "simultaneous-use", 0, "Max simultaneous sessions")
	cmd.Flags().IntVar(&sessTimeout, "session-timeout", 0, "Session timeout (seconds)")
	cmd.Flags().IntVar(&idleTimeout, "idle-timeout", 0, "Idle timeout (seconds)")
	cmd.Flags().Uint32Var(&bwUp, "bw-up", 0, "Max upload bandwidth")
	cmd.Flags().Uint32Var(&bwDown, "bw-down", 0, "Max download bandwidth")
	cmd.Flags().Uint32Var(&maxOctets, "max-octets", 0, "Max total octets")
	cmd.Flags().BoolVar(&enabled, "enable", false, "Enable the subscriber")
	cmd.Flags().BoolVar(&disabled, "disable", false, "Disable the subscriber")
	cmd.MarkFlagRequired("id")
	return cmd
}

func subscriberDeleteCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a subscriber",
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := client.DeleteSubscriber(cmd.Context(), id)
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(r)
				return nil
			}
			fmt.Printf("subscriber %s deleted\n", r.ID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&id, "id", "i", "", "Subscriber UUID (required)")
	cmd.MarkFlagRequired("id")
	return cmd
}

// printSubscriber renders a single subscriber as key-value pairs.
func printSubscriber(s Subscriber) {
	KV(
		[2]string{"id", s.ID},
		[2]string{"username", s.Username},
		[2]string{"full_name", s.FullName},
		[2]string{"email", s.Email},
		[2]string{"enabled", yesNo(s.Enabled)},
		[2]string{"service_type", s.ServiceType},
		[2]string{"framed_ip", strOr(s.FramedIP, "-")},
		[2]string{"mikrotik_group", strOr(s.MikrotikGroup, "-")},
		[2]string{"rate_limit", strOr(s.RateLimit, "-")},
		[2]string{"simultaneous_use", itoa(s.SimultaneousUse)},
		[2]string{"session_timeout", itoa(s.SessionTimeout)},
		[2]string{"idle_timeout", itoa(s.IdleTimeout)},
		[2]string{"bandwidth_max_up", uitoa(uint64(s.BandwidthMaxUp))},
		[2]string{"bandwidth_max_down", uitoa(uint64(s.BandwidthMaxDown))},
		[2]string{"max_total_octets", uitoa(uint64(s.MaxTotalOctets))},
		[2]string{"is_voucher", yesNo(s.IsVoucher)},
		[2]string{"created_at", s.CreatedAt.String()},
	)
}
