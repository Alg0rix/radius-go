package radiusctl

import (
	"github.com/spf13/cobra"
)

// statusCmd returns the `radiusctl status` command.
func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show RADIUS server status and counters",
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := client.GetStatus(cmd.Context())
			if err != nil {
				return err
			}
			if jsonOut {
				PrintJSON(s)
				return nil
			}
			KV(
				[2]string{"health", s.Health},
				[2]string{"started_at", s.StartedAt.String()},
				[2]string{"nas_count", itoa(s.NASCount)},
				[2]string{"subscriber_count", itoa(s.SubscriberCount)},
				[2]string{"active_sessions", itoa(s.ActiveSessions)},
				[2]string{"auth_requests", uitoa(s.AuthRequests)},
				[2]string{"auth_accepts", uitoa(s.AuthAccepts)},
				[2]string{"auth_rejects", uitoa(s.AuthRejects)},
				[2]string{"acct_requests", uitoa(s.AcctRequests)},
			)
			return nil
		},
	}
}
