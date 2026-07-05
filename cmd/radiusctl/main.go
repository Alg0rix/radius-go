package main

import (
	"fmt"
	"os"

	"github.com/Alg0rix/radius-go/internal/radiusctl"
	"github.com/spf13/cobra"
)

func main() {
	root := radiusctl.RootCmd()
	flags := radiusctl.BindGlobals(root)

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if flags.Secret == "" {
			return fmt.Errorf("secret is required: set --secret or RADIUS_SECRET env")
		}
		radiusctl.SetClient(radiusctl.NewClient(flags.Server, flags.Secret))
		radiusctl.SetJSONOut(flags.JSON)
		return nil
	}

	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
