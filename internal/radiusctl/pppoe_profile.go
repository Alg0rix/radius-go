package radiusctl

import (
	"fmt"

	"github.com/spf13/cobra"
)

func pppoeProfileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pppoe-profile",
		Aliases: []string{"pppoe"},
		Short:   "Manage PPPoE profiles",
	}
	cmd.AddCommand(
		pppoeProfileListCmd(),
		pppoeProfileCreateCmd(),
		pppoeProfileGetCmd(),
		pppoeProfileUpdateCmd(),
		pppoeProfileDeleteCmd(),
	)
	return cmd
}

func pppoeProfileListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all PPPoE profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			items, err := client.ListPPPoEProfiles(cmd.Context())
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(items)
				return nil
			}
			if len(items) == 0 {
				fmt.Println("no pppoe profiles")
				return nil
			}
			w := NewWriter()
			w.Row("ID", "NAME", "ENABLED", "POOL", "MTU", "RATE-LIMIT")
			for _, p := range items {
				w.Row(p.ID, p.Name, yesNo(p.Enabled), p.FramedIPPool, itoa(p.MTU), strOr(p.RateLimit, "-"))
			}
			w.Flush()
			return nil
		},
	}
}

func pppoeProfileCreateCmd() *cobra.Command {
	var (
		name, description, pool, netmask, primaryDNS, secondaryDNS, rateLimit string
		compression                                                            bool
		mtu, mru, keepalive, bwUp, bwDown, sessTimeout, idleTimeout            int
		maxOctets                                                              int64
	)
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a PPPoE profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			req := CreatePPPoEProfileRequest{
				Name:              name,
				Description:       description,
				FramedIPPool:      pool,
				FramedIPNetmask:   netmask,
				PrimaryDNS:        primaryDNS,
				SecondaryDNS:      secondaryDNS,
				PPPCompression:    compression,
				MTU:               mtu,
				MRU:               mru,
				KeepaliveInterval: keepalive,
				RateLimit:         rateLimit,
				BandwidthMaxUp:    bwUp,
				BandwidthMaxDown:  bwDown,
				SessionTimeout:    sessTimeout,
				IdleTimeout:       idleTimeout,
				MaxTotalOctets:    maxOctets,
			}
			p, err := client.CreatePPPoEProfile(cmd.Context(), req)
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(p)
				return nil
			}
			printPPPoEProfile(p)
			return nil
		},
	}
	cmd.Flags().StringVarP(&name, "name", "n", "", "Profile name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Description")
	cmd.Flags().StringVar(&pool, "pool", "", "Framed IP pool name")
	cmd.Flags().StringVar(&netmask, "netmask", "", "Framed IP netmask")
	cmd.Flags().StringVar(&primaryDNS, "primary-dns", "", "Primary DNS server")
	cmd.Flags().StringVar(&secondaryDNS, "secondary-dns", "", "Secondary DNS server")
	cmd.Flags().BoolVar(&compression, "compression", false, "Enable PPP compression")
	cmd.Flags().IntVar(&mtu, "mtu", 0, "MTU")
	cmd.Flags().IntVar(&mru, "mru", 0, "MRU")
	cmd.Flags().IntVar(&keepalive, "keepalive", 0, "Keepalive interval (seconds)")
	cmd.Flags().StringVar(&rateLimit, "rate-limit", "", "MikroTik rate limit string")
	cmd.Flags().IntVar(&bwUp, "bw-up", 0, "Max upload bandwidth (kbps)")
	cmd.Flags().IntVar(&bwDown, "bw-down", 0, "Max download bandwidth (kbps)")
	cmd.Flags().IntVar(&sessTimeout, "session-timeout", 0, "Session timeout (seconds)")
	cmd.Flags().IntVar(&idleTimeout, "idle-timeout", 0, "Idle timeout (seconds)")
	cmd.Flags().Int64Var(&maxOctets, "max-octets", 0, "Max total octets")
	cmd.MarkFlagRequired("name")
	return cmd
}

func pppoeProfileGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get",
		Short: "Get a PPPoE profile by ID",
		RunE: func(cmd *cobra.Command, args []string) error {
			id := cmd.Flag("id").Value.String()
			if id == "" {
				return fmt.Errorf("--id is required")
			}
			items, err := client.ListPPPoEProfiles(cmd.Context())
			if err != nil {
				return err
			}
			for _, p := range items {
				if p.ID == id {
					if jsonOut {
						PrintJSON(p)
						return nil
					}
					printPPPoEProfile(p)
					return nil
				}
			}
			return fmt.Errorf("pppoe profile not found")
		},
	}
}

func pppoeProfileUpdateCmd() *cobra.Command {
	var (
		id, name, description, pool, netmask, primaryDNS, secondaryDNS, rateLimit string
		compression                                                                  bool
		mtu, mru, keepalive, bwUp, bwDown, sessTimeout, idleTimeout                  int
		maxOctets                                                                    int64
		enabled, disabled                                                            bool
	)
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a PPPoE profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}
			if enabled && disabled {
				return fmt.Errorf("--enable and --disable are mutually exclusive")
			}
			req := UpdatePPPoEProfileRequest{}
			if cmd.Flags().Changed("name") {
				req.Name = &name
			}
			if cmd.Flags().Changed("description") {
				req.Description = &description
			}
			if cmd.Flags().Changed("pool") {
				req.FramedIPPool = &pool
			}
			if cmd.Flags().Changed("netmask") {
				req.FramedIPNetmask = &netmask
			}
			if cmd.Flags().Changed("primary-dns") {
				req.PrimaryDNS = &primaryDNS
			}
			if cmd.Flags().Changed("secondary-dns") {
				req.SecondaryDNS = &secondaryDNS
			}
			if cmd.Flags().Changed("compression") {
				req.PPPCompression = &compression
			}
			if cmd.Flags().Changed("mtu") {
				req.MTU = &mtu
			}
			if cmd.Flags().Changed("mru") {
				req.MRU = &mru
			}
			if cmd.Flags().Changed("keepalive") {
				req.KeepaliveInterval = &keepalive
			}
			if cmd.Flags().Changed("rate-limit") {
				req.RateLimit = &rateLimit
			}
			if cmd.Flags().Changed("bw-up") {
				req.BandwidthMaxUp = &bwUp
			}
			if cmd.Flags().Changed("bw-down") {
				req.BandwidthMaxDown = &bwDown
			}
			if cmd.Flags().Changed("session-timeout") {
				req.SessionTimeout = &sessTimeout
			}
			if cmd.Flags().Changed("idle-timeout") {
				req.IdleTimeout = &idleTimeout
			}
			if cmd.Flags().Changed("max-octets") {
				req.MaxTotalOctets = &maxOctets
			}
			if enabled || disabled {
				v := enabled
				req.Enabled = &v
			}
			p, err := client.UpdatePPPoEProfile(cmd.Context(), id, req)
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(p)
				return nil
			}
			printPPPoEProfile(p)
			return nil
		},
	}
	cmd.Flags().StringVarP(&id, "id", "i", "", "Profile UUID (required)")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Profile name")
	cmd.Flags().StringVar(&description, "description", "", "Description")
	cmd.Flags().StringVar(&pool, "pool", "", "Framed IP pool name")
	cmd.Flags().StringVar(&netmask, "netmask", "", "Framed IP netmask")
	cmd.Flags().StringVar(&primaryDNS, "primary-dns", "", "Primary DNS server")
	cmd.Flags().StringVar(&secondaryDNS, "secondary-dns", "", "Secondary DNS server")
	cmd.Flags().BoolVar(&compression, "compression", false, "Enable PPP compression")
	cmd.Flags().IntVar(&mtu, "mtu", 0, "MTU")
	cmd.Flags().IntVar(&mru, "mru", 0, "MRU")
	cmd.Flags().IntVar(&keepalive, "keepalive", 0, "Keepalive interval (seconds)")
	cmd.Flags().StringVar(&rateLimit, "rate-limit", "", "MikroTik rate limit string")
	cmd.Flags().IntVar(&bwUp, "bw-up", 0, "Max upload bandwidth (kbps)")
	cmd.Flags().IntVar(&bwDown, "bw-down", 0, "Max download bandwidth (kbps)")
	cmd.Flags().IntVar(&sessTimeout, "session-timeout", 0, "Session timeout (seconds)")
	cmd.Flags().IntVar(&idleTimeout, "idle-timeout", 0, "Idle timeout (seconds)")
	cmd.Flags().Int64Var(&maxOctets, "max-octets", 0, "Max total octets")
	cmd.Flags().BoolVar(&enabled, "enable", false, "Enable the profile")
	cmd.Flags().BoolVar(&disabled, "disable", false, "Disable the profile")
	return cmd
}

func pppoeProfileDeleteCmd() *cobra.Command {
	var id string
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a PPPoE profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == "" {
				return fmt.Errorf("--id is required")
			}
			r, err := client.DeletePPPoEProfile(cmd.Context(), id)
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(r)
				return nil
			}
			fmt.Printf("pppoe profile %s deleted\n", r.ID)
			return nil
		},
	}
	cmd.Flags().StringVarP(&id, "id", "i", "", "Profile UUID (required)")
	return cmd
}

func printPPPoEProfile(p PPPoEProfile) {
	KV(
		[2]string{"id", p.ID},
		[2]string{"name", p.Name},
		[2]string{"description", p.Description},
		[2]string{"enabled", yesNo(p.Enabled)},
		[2]string{"framed_ip_pool", p.FramedIPPool},
		[2]string{"framed_ip_netmask", p.FramedIPNetmask},
		[2]string{"primary_dns", p.PrimaryDNS},
		[2]string{"secondary_dns", p.SecondaryDNS},
		[2]string{"ppp_compression", yesNo(p.PPPCompression)},
		[2]string{"mtu", itoa(p.MTU)},
		[2]string{"mru", itoa(p.MRU)},
		[2]string{"keepalive_interval", itoa(p.KeepaliveInterval)},
		[2]string{"rate_limit", strOr(p.RateLimit, "-")},
		[2]string{"bandwidth_max_up", itoa(p.BandwidthMaxUp)},
		[2]string{"bandwidth_max_down", itoa(p.BandwidthMaxDown)},
		[2]string{"session_timeout", itoa(p.SessionTimeout)},
		[2]string{"idle_timeout", itoa(p.IdleTimeout)},
		[2]string{"max_total_octets", fmt.Sprintf("%d", p.MaxTotalOctets)},
	)
}
