package cmd

import (
	"fmt"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version, build, and VCS information.",
	RunE: func(cmd *cobra.Command, args []string) error {
		info, ok := debug.ReadBuildInfo()
		if !ok {
			return fmt.Errorf("build info not available")
		}
		version := info.Main.Version
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
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
