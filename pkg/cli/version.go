package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags "-X pantry/pkg/cli.Version=1.2.3".
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	//nolint:revive
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(Version)
	},
}
