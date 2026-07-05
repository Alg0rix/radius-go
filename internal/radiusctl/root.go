package radiusctl

import (
	"os"

	"github.com/spf13/cobra"
)

// RootCmd returns the cobra root command for radiusctl. subcommands defined
// in nas.go / subscriber.go / session.go / voucher.go / status.go attach
// here. Global flags (--server, --secret, --json) are exposed and bound to
// the shared client state in Execute().
func RootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "radiusctl",
		Short: "Manage a radius-go server from the command line",
		Long: `radiusctl is an HTTP client for the radius-go management API.

Configure the target server with --server and --secret, or with the
RADIUS_SERVER and RADIUS_SECRET environment variables.

Examples:
  radiusctl status
  radiusctl nas list
  radiusctl subscriber create --username alice --password secret
  radiusctl session disconnect --username alice
  radiusctl voucher package list
  radiusctl pppoe-profile list`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		statusCmd(),
		nasCmd(),
		subscriberCmd(),
		sessionCmd(),
		voucherCmd(),
		pppoeProfileCmd(),
	)
	return root
}

// globalFlags holds the parsed values of --server / --secret / --json.
// Bound in Execute(); fields are exported so the main package can read them
// to build the client.
type globalFlags struct {
	Server string
	Secret string
	JSON   bool
}

// BindGlobals registers global flags on root and returns a pointer to the
// struct they bind to. The values are read in Execute() to build the client.
func BindGlobals(root *cobra.Command) *globalFlags {
	g := &globalFlags{}
	p := root.PersistentFlags()
	p.StringVar(&g.Server, "server", envOrDefault("RADIUS_SERVER", "http://localhost:8083"), "API server URL")
	p.StringVar(&g.Secret, "secret", envOrDefault("RADIUS_SECRET", ""), "Internal secret (Authorization: Bearer)")
	p.BoolVar(&g.JSON, "json", false, "Output raw JSON instead of tables")
	return g
}

// envOrDefault returns env value if set, otherwise def.
func envOrDefault(env, def string) string {
	if v, ok := os.LookupEnv(env); ok {
		return v
	}
	return def
}
