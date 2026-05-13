package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/heatxsink/hmuzik/internal/m3u"
	"github.com/spf13/cobra"
)

var playlistsCmd = &cobra.Command{
	Use:   "playlists",
	Short: "Convert all cmus playlists to extended m3u in.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmusConfigPlaylistPath := fmt.Sprintf("%s/.config/cmus/playlists/", os.Getenv("HOME"))
		outputPath := args[0]
		scrubPrefix := args[1]
		fmt.Println("Searching:", cmusConfigPlaylistPath)
		err := filepath.Walk(cmusConfigPlaylistPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.IsDir() {
				return nil
			}
			if err := m3u.CreateFromCmusPlaylist(path, outputPath, scrubPrefix); err != nil {
				return err
			}
			fmt.Println(filepath.Base(m3u.Filename(path, outputPath)))
			return nil
		})
		if err != nil {
			return err
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(playlistsCmd)
}
