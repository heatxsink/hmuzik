package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var (
	dryRunFlagOption bool
)

var rootCmd = &cobra.Command{
	Use:   "hmuzik",
	Short: "",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&dryRunFlagOption, "dryrun", "r", false, "dryrun option")
}
