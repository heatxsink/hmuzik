package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/heatxsink/hmuzik/internal/device"
	"github.com/heatxsink/hmuzik/internal/mirror"
	"github.com/spf13/cobra"
)

// defaultMediaRoot is where udisks mounts removable media on Linux.
const defaultMediaRoot = "/run/media"

// ejectTimeout bounds unmount plus power-off; flushing a large write-back
// cache to a slow device can take a while.
const ejectTimeout = 2 * time.Minute

var (
	syncDeviceOption string
	syncDeleteOption bool
	syncDryRunOption bool
	syncEjectOption  bool
)

// syncTarget is one library folder and where it lands on the device.
type syncTarget struct {
	name    string
	source  string
	dest    string
	exclude []string
}

// syncTargets returns the music and playlists targets for a library under
// home and a device mounted at device.
func syncTargets(home, device string) (music, playlists syncTarget) {
	music = syncTarget{
		name:   "music",
		source: filepath.Join(home, "Music", "Artists"),
		dest:   filepath.Join(device, "Music", "Artists"),
		// Album art stays off the device, as cmus-ipod-sync did.
		exclude: []string{"cover.jpg"},
	}
	playlists = syncTarget{
		name:   "playlists",
		source: filepath.Join(home, "Music", "Playlists"),
		dest:   filepath.Join(device, "Playlists"),
	}
	return music, playlists
}

// withOverrides replaces t's source and dest with any non-empty override.
func (t syncTarget) withOverrides(source, dest string) syncTarget {
	if source != "" {
		t.source = source
	}
	if dest != "" {
		t.dest = dest
	}
	return t
}

// devicePath is the device mount point: --device, or the udisks default.
func devicePath() string {
	if syncDeviceOption != "" {
		return syncDeviceOption
	}
	return filepath.Join(defaultMediaRoot, os.Getenv("USER"), "IPOD")
}

func defaultSyncTargets() (music, playlists syncTarget) {
	return syncTargets(os.Getenv("HOME"), devicePath())
}

// runSync mirrors each target in order. A target that fails doesn't stop the
// next one; all errors are returned together. With --eject, the device is
// ejected only if every target succeeded.
func runSync(ctx context.Context, targets ...syncTarget) error {
	prefix := ""
	if syncDryRunOption {
		prefix = "[dryrun] "
	}
	var (
		errs    []error
		changed bool
	)
	for _, t := range targets {
		fmt.Printf("Syncing %s: %s -> %s\n", t.name, t.source, t.dest)
		start := time.Now()
		res, err := mirror.Mirror(t.source, t.dest, mirror.Options{
			Delete:  syncDeleteOption,
			DryRun:  syncDryRunOption,
			Exclude: t.exclude,
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
			errs = append(errs, fmt.Errorf("sync %s: %w", t.name, err))
		}
		changed = changed || res.Copied+res.Deleted > 0
	}
	if err := errors.Join(errs...); err != nil {
		if syncEjectOption && !syncDryRunOption {
			fmt.Println("Not ejecting: sync had errors. Device left mounted.")
		}
		return err
	}
	return finishSync(ctx, changed)
}

// finishSync ejects the device when asked, or reminds that it is still
// mounted.
func finishSync(ctx context.Context, changed bool) error {
	dev := devicePath()
	switch {
	case syncDryRunOption:
		if syncEjectOption {
			fmt.Printf("[dryrun] eject %s\n", dev)
		}
		return nil
	case syncEjectOption:
		ctx, cancel := context.WithTimeout(ctx, ejectTimeout)
		defer cancel()
		fmt.Printf("Ejecting %s...\n", dev)
		if err := device.Eject(ctx, dev); err != nil {
			return fmt.Errorf("eject: %w", err)
		}
		fmt.Println("Ejected; safe to unplug.")
	case changed:
		fmt.Println("Device left mounted.")
	}
	if changed {
		fmt.Println("Update the Rockbox database to pick up the changes.")
	}
	return nil
}

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Copy new and changed music and playlists to a mounted iPod.",
	Long: `Copy new and changed files to a mounted iPod: music first, then playlists.
Use the music or playlists subcommand to sync just one.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		music, playlists := defaultSyncTargets()
		return runSync(cmd.Context(), music, playlists)
	},
}

var (
	syncMusicSourceOption string
	syncMusicDestOption   string
)

var syncMusicCmd = &cobra.Command{
	Use:   "music",
	Short: "Copy new and changed library files to a mounted iPod.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		music, _ := defaultSyncTargets()
		return runSync(cmd.Context(), music.withOverrides(syncMusicSourceOption, syncMusicDestOption))
	},
}

var (
	syncPlaylistsSourceOption string
	syncPlaylistsDestOption   string
)

var syncPlaylistsCmd = &cobra.Command{
	Use:   "playlists",
	Short: "Copy new and changed playlists to a mounted iPod.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		_, playlists := defaultSyncTargets()
		return runSync(cmd.Context(), playlists.withOverrides(syncPlaylistsSourceOption, syncPlaylistsDestOption))
	},
}

func init() {
	pf := syncCmd.PersistentFlags()
	pf.StringVar(&syncDeviceOption, "device", "", "mount point of the device; must already exist (default /run/media/$USER/IPOD)")
	pf.BoolVar(&syncDeleteOption, "delete", false, "remove destination files that are no longer in the source")
	pf.BoolVarP(&syncDryRunOption, "dryrun", "r", false, "print what would change without writing")
	pf.BoolVar(&syncEjectOption, "eject", false, "unmount and power off the device after a successful sync")

	syncMusicCmd.Flags().StringVarP(&syncMusicSourceOption, "source", "s", "", "music library root (default $HOME/Music/Artists)")
	syncMusicCmd.Flags().StringVarP(&syncMusicDestOption, "dest", "d", "", "destination folder; overrides --device (default <device>/Music/Artists)")
	syncPlaylistsCmd.Flags().StringVarP(&syncPlaylistsSourceOption, "source", "s", "", "playlist folder (default $HOME/Music/Playlists)")
	syncPlaylistsCmd.Flags().StringVarP(&syncPlaylistsDestOption, "dest", "d", "", "destination folder; overrides --device (default <device>/Playlists)")

	syncCmd.AddCommand(syncMusicCmd, syncPlaylistsCmd)
	rootCmd.AddCommand(syncCmd)
}
