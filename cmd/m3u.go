package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/heatxsink/hmuzik/internal/m3u"
	"github.com/spf13/cobra"
)

var m3uCmd = &cobra.Command{
	Use:   "m3u",
	Short: "Convert cmus playlist to extended m3u.",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cmusPlaylistPath := args[0]
		scrubPrefix := args[1]
		outputPath := filepath.Dir(cmusPlaylistPath)
		if err := m3u.CreateFromCmusPlaylist(cmusPlaylistPath, outputPath, scrubPrefix); err != nil {
			return err
		}
		filename := m3u.Filename(cmusPlaylistPath, outputPath)
		fmt.Println(filename, "has been created")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(m3uCmd)
}
