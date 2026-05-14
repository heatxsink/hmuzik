package cmd

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version, build, and VCS information.",
	Run: func(cmd *cobra.Command, args []string) {
		info, ok := debug.ReadBuildInfo()
		if !ok {
			fmt.Fprintln(os.Stderr, "build info not available")
			os.Exit(1)
		}
		version := strings.TrimSuffix(info.Main.Version, "+dirty")
		if version == "" {
			version = "(devel)"
		}
		var sha, ts string
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				if len(s.Value) >= 7 {
					sha = s.Value[:7]
				} else {
					sha = s.Value
				}
			case "vcs.time":
				ts = s.Value
			}
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), version, sha, ts)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
