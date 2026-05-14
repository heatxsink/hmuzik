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
		out := cmd.OutOrStdout()
		fmt.Fprintf(out, "hmuzik %s\n", version)
		fmt.Fprintf(out, "  go:       %s\n", info.GoVersion)
		for _, s := range info.Settings {
			switch s.Key {
			case "vcs.revision":
				fmt.Fprintf(out, "  revision: %s\n", s.Value)
			case "vcs.time":
				fmt.Fprintf(out, "  built:    %s\n", s.Value)
			case "vcs.modified":
				fmt.Fprintf(out, "  dirty:    %s\n", s.Value)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
