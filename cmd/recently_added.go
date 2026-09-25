package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/heatxsink/hmuzik/internal/recentlyadded"
	"github.com/spf13/cobra"
)

var (
	recentlyAddedSourceOption  string
	recentlyAddedOutputOption  string
	recentlyAddedDaysOption    int
	recentlyAddedWindowsOption string
	recentlyAddedDryRunOption  bool
)

var recentlyAddedCmd = &cobra.Command{
	Use:   "recently-added",
	Short: "Generate a cmus \"Recently Added\" playlist, grouped by album.",
	RunE: func(cmd *cobra.Command, args []string) error {
		home := os.Getenv("HOME")
		source := recentlyAddedSourceOption
		if source == "" {
			source = filepath.Join(home, "Music", "Artists")
		}
		output := recentlyAddedOutputOption
		if output == "" {
			output = filepath.Join(home, ".config", "cmus", "playlists")
		}
		targets, err := recentlyAddedTargets(recentlyAddedDaysOption, recentlyAddedWindowsOption, cmd.Flags().Changed("windows"))
		if err != nil {
			return err
		}
		maxDays := targets[len(targets)-1].days
		now := time.Now()
		since := now.Add(-time.Duration(maxDays) * 24 * time.Hour)

		if !recentlyAddedDryRunOption {
			if err := os.MkdirAll(output, 0755); err != nil {
				return fmt.Errorf("mkdir %s: %w", output, err)
			}
		}
		fmt.Println("Scanning:", source)
		tracks, err := recentlyadded.Scan(source, since)
		if err != nil {
			return err
		}
		fmt.Printf("Found %d audio files within the last %d days.\n", len(tracks), maxDays)

		albums := recentlyadded.GroupByAlbum(tracks)
		fmt.Printf("Grouped into %d albums.\n", len(albums))

		for _, t := range targets {
			window := time.Duration(t.days) * 24 * time.Hour
			paths := recentlyadded.Playlist(albums, window, now)
			out := filepath.Join(output, t.name)
			if recentlyAddedDryRunOption {
				fmt.Printf("[dryrun] %s: %d tracks\n", out, len(paths))
				continue
			}
			if err := recentlyadded.WriteCmusPlaylist(out, paths); err != nil {
				return fmt.Errorf("write %s: %w", out, err)
			}
			fmt.Printf("%s: %d tracks\n", out, len(paths))
		}
		return nil
	},
}

type recentlyAddedTarget struct {
	days int
	name string
}

// recentlyAddedTargets returns the playlists to write, sorted by window: a
// single "Recently Added" playlist by default, or one "recently added (NNd)"
// playlist per window when --windows is given.
func recentlyAddedTargets(days int, windows string, useWindows bool) ([]recentlyAddedTarget, error) {
	if !useWindows {
		if days <= 0 {
			return nil, fmt.Errorf("invalid --days %d: must be > 0", days)
		}
		return []recentlyAddedTarget{{days: days, name: "Recently Added"}}, nil
	}
	ds, err := parseWindowDays(windows)
	if err != nil {
		return nil, err
	}
	targets := make([]recentlyAddedTarget, 0, len(ds))
	for _, d := range ds {
		targets = append(targets, recentlyAddedTarget{days: d, name: fmt.Sprintf("recently added (%02dd)", d)})
	}
	return targets, nil
}

func parseWindowDays(s string) ([]int, error) {
	parts := strings.Split(s, ",")
	days := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid window %q: %w", p, err)
		}
		if n <= 0 {
			return nil, fmt.Errorf("invalid window %d: must be > 0", n)
		}
		days = append(days, n)
	}
	if len(days) == 0 {
		return nil, fmt.Errorf("no windows specified")
	}
	slices.Sort(days)
	return slices.Compact(days), nil
}

func init() {
	recentlyAddedCmd.Flags().StringVarP(&recentlyAddedSourceOption, "source", "s", "", "music library root (default $HOME/Music/Artists)")
	recentlyAddedCmd.Flags().StringVarP(&recentlyAddedOutputOption, "output", "o", "", "cmus playlist directory (default $HOME/.config/cmus/playlists)")
	recentlyAddedCmd.Flags().IntVarP(&recentlyAddedDaysOption, "days", "d", 14, "write one \"Recently Added\" playlist covering this many days")
	recentlyAddedCmd.Flags().StringVarP(&recentlyAddedWindowsOption, "windows", "w", "", "comma-separated day windows; writes one \"recently added (NNd)\" playlist per window instead")
	recentlyAddedCmd.Flags().BoolVarP(&recentlyAddedDryRunOption, "dryrun", "r", false, "report counts without writing playlists")
	recentlyAddedCmd.MarkFlagsMutuallyExclusive("days", "windows")
	rootCmd.AddCommand(recentlyAddedCmd)
}
