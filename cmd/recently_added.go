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
	recentlyAddedWindowsOption string
	recentlyAddedDryRunOption  bool
)

var recentlyAddedCmd = &cobra.Command{
	Use:   "recently-added",
	Short: "Generate cmus playlists of recently-added tracks, grouped by album.",
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
		days, err := parseWindowDays(recentlyAddedWindowsOption)
		if err != nil {
			return err
		}
		maxDays := slices.Max(days)
		now := time.Now()
		since := now.Add(-time.Duration(maxDays) * 24 * time.Hour)

		fmt.Println("Scanning:", source)
		tracks, err := recentlyadded.Scan(source, since)
		if err != nil {
			return err
		}
		fmt.Printf("Found %d audio files within the last %d days.\n", len(tracks), maxDays)

		albums := recentlyadded.GroupByAlbum(tracks)
		fmt.Printf("Grouped into %d albums.\n", len(albums))

		for _, d := range days {
			window := time.Duration(d) * 24 * time.Hour
			paths := recentlyadded.Playlist(albums, window, now)
			out := filepath.Join(output, fmt.Sprintf("recently added (%02dd)", d))
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
	recentlyAddedCmd.Flags().StringVarP(&recentlyAddedWindowsOption, "windows", "w", "1,7,14,30,90", "comma-separated day windows")
	recentlyAddedCmd.Flags().BoolVarP(&recentlyAddedDryRunOption, "dryrun", "r", false, "report counts without writing playlists")
	rootCmd.AddCommand(recentlyAddedCmd)
}
