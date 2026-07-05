package radiusctl

import (
	"fmt"

	"github.com/spf13/cobra"
)

// sessionCmd returns the `radiusctl session` subcommand group.
func sessionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "session",
		Short: "Manage active RADIUS sessions",
	}
	cmd.AddCommand(
		sessionListCmd(),
		sessionDisconnectCmd(),
		sessionCoACmd(),
		sessionCleanupCmd(),
		sessionReconcileCmd(),
	)
	return cmd
}

func sessionListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List active sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			items, err := client.ListSessions(cmd.Context())
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(items)
				return nil
			}
			if len(items) == 0 {
				fmt.Println("no active sessions")
				return nil
			}
			w := NewWriter()
			w.Row("ID", "USERNAME", "NAS IP", "FRAMED IP", "STATUS", "TIME", "IN", "OUT")
			for _, s := range items {
				w.Row(
					s.SessionID,
					s.Username,
					strOr(s.NASIP, "-"),
					strOr(s.FramedIP, "-"),
					s.SessionStatus,
					fmt.Sprintf("%ds", s.SessionTime),
					humanBytes(s.InputOctets),
					humanBytes(s.OutputOctets),
				)
			}
			w.Flush()
			return nil
		},
	}
}

func sessionDisconnectCmd() *cobra.Command {
	var username, reason string
	cmd := &cobra.Command{
		Use:   "disconnect",
		Short: "Send PoD disconnect for a user",
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := client.DisconnectUser(cmd.Context(), DisconnectRequest{
				Username: username,
				Reason:   reason,
			})
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(r)
				return nil
			}
			fmt.Printf("disconnected %v sessions for user %q\n", r["disconnected_count"], r["username"])
			return nil
		},
	}
	cmd.Flags().StringVarP(&username, "username", "u", "", "Username (required)")
	cmd.Flags().StringVarP(&reason, "reason", "r", "", "Disconnect reason")
	cmd.MarkFlagRequired("username")
	return cmd
}

func sessionCoACmd() *cobra.Command {
	var (
		username, rateLimit, group string
		bwUp, bwDown, maxOctets    uint32
	)
	cmd := &cobra.Command{
		Use:   "coa-change",
		Short: "Send CoA to change user profile on active sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			req := CoAChangeRequest{
				Username:  username,
				RateLimit: rateLimit,
				Group:     group,
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
			result, err := client.CoAChange(cmd.Context(), req)
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(result)
				return nil
			}
			KV(
				[2]string{"disconnected_count", itoa(result.DisconnectedCount)},
			)
			if len(result.FailedNAS) > 0 {
				fmt.Println("failed NAS:")
				for _, nas := range result.FailedNAS {
					fmt.Printf("  - %s\n", nas)
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&username, "username", "u", "", "Username (required)")
	cmd.Flags().StringVar(&rateLimit, "rate-limit", "", "New MikroTik rate limit")
	cmd.Flags().StringVar(&group, "mikrotik-group", "", "New MikroTik group")
	cmd.Flags().Uint32Var(&bwUp, "bw-up", 0, "New max upload bandwidth")
	cmd.Flags().Uint32Var(&bwDown, "bw-down", 0, "New max download bandwidth")
	cmd.Flags().Uint32Var(&maxOctets, "max-octets", 0, "New max total octets")
	cmd.MarkFlagRequired("username")
	return cmd
}

func sessionCleanupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cleanup",
		Short: "Clean up stale sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := client.CleanupSessions(cmd.Context())
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(r)
				return nil
			}
			KV(
				[2]string{"stale_cleaned", itoa(r.StaleSessionsCleaned)},
				[2]string{"active_kept", itoa(r.ActiveSessionsKept)},
				[2]string{"cleaned_at", r.CleanedAt.String()},
			)
			return nil
		},
	}
}

func sessionReconcileCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reconcile",
		Short: "Merge DB sessions into in-memory state",
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := client.ReconcileSessions(cmd.Context())
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(r)
				return nil
			}
			fmt.Printf("merged %d sessions\n", r["merged"])
			return nil
		},
	}
}
