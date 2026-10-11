package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/heatxsink/hmuzik/internal/mirror"
	"github.com/spf13/cobra"
)

// defaultMediaRoot is where udisks mounts removable media on Linux.
const defaultMediaRoot = "/run/media"

var (
	syncSourceOption string
	syncDestOption   string
	syncDeleteOption bool
	syncDryRunOption bool
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Copy new and changed library files to a mounted iPod.",
	RunE: func(cmd *cobra.Command, args []string) error {
		home := os.Getenv("HOME")
		source := syncSourceOption
		if source == "" {
			source = filepath.Join(home, "Music", "Artists")
		}
		dest := syncDestOption
		if dest == "" {
			dest = filepath.Join(defaultMediaRoot, os.Getenv("USER"), "IPOD", "Music", "Artists")
		}
		prefix := ""
		if syncDryRunOption {
			prefix = "[dryrun] "
		}
		fmt.Printf("Syncing: %s -> %s\n", source, dest)
		start := time.Now()
		res, err := mirror.Mirror(source, dest, mirror.Options{
			Delete: syncDeleteOption,
			DryRun: syncDryRunOption,
			Report: func(a mirror.Action, rel string) {
				fmt.Printf("%s%s %s\n", prefix, a, rel)
			},
		})
		elapsed := time.Since(start)
		mb := float64(res.Bytes) / 1e6
		fmt.Printf("%sCopied %d files (%.1f MB), deleted %d, unchanged %d in %s",
			prefix, res.Copied, mb, res.Deleted, res.Skipped, elapsed.Round(time.Second))
		if !syncDryRunOption && res.Copied > 0 {
			fmt.Printf(" (%.1f MB/s)", mb/elapsed.Seconds())
		}
		fmt.Println(".")
		if err != nil {
			return err
		}
		if !syncDryRunOption && res.Copied+res.Deleted > 0 {
			fmt.Println("Device left mounted. Update the Rockbox database to pick up the changes.")
		}
		return nil
	},
}

func init() {
	syncCmd.Flags().StringVarP(&syncSourceOption, "source", "s", "", "music library root (default $HOME/Music/Artists)")
	syncCmd.Flags().StringVarP(&syncDestOption, "dest", "d", "", "destination on the mounted device; must already exist (default /run/media/$USER/IPOD/Music/Artists)")
	syncCmd.Flags().BoolVar(&syncDeleteOption, "delete", false, "remove destination files that are no longer in the source")
	syncCmd.Flags().BoolVarP(&syncDryRunOption, "dryrun", "r", false, "print what would change without writing")
	rootCmd.AddCommand(syncCmd)
}
