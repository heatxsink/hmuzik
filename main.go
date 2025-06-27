package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/tag"
	"github.com/heatxsink/hmuzik/m3u"
	"github.com/spf13/cobra"
)

var (
	name                  = "hmuzik"
	dryRunFlagOption      bool
	sourcePathOption      string
	destinationPathOption string
)

func isAudioFile(filename string) bool {
	normalize := strings.ToLower(filename)
	if strings.HasSuffix(normalize, ".flac") {
		return true
	} else if strings.HasSuffix(normalize, ".mp3") {
		return true
	} else if strings.HasSuffix(normalize, ".m4a") {
		return true
	} else if strings.HasSuffix(normalize, ".aiff") {
		return true
	}
	return false
}

func normalize(s string) string {
	n := strings.ReplaceAll(s, ".", "")
	n = strings.ReplaceAll(n, "/", "")
	n = strings.ReplaceAll(n, "#", "")
	n = strings.ReplaceAll(n, ":", "")
	n = strings.ReplaceAll(n, "?", "")
	n = strings.ReplaceAll(n, "!", "")
	n = strings.ReplaceAll(n, "'", "")
	n = strings.ReplaceAll(n, "\"", "")
	n = strings.ReplaceAll(n, "|", "")
	n = strings.ReplaceAll(n, ">", "")
	n = strings.ReplaceAll(n, "<", "")
	n = strings.ReplaceAll(n, "  ", " ")
	n = strings.TrimSpace(n)
	return n
}

func main() {
	rootCmd := &cobra.Command{
		Use:   name,
		Short: "",
	}
	playlistsCmd := &cobra.Command{
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
	m3uCmd := &cobra.Command{
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
	organizeCmd := &cobra.Command{
		Use:   "organize",
		Short: "Organize a directory path of desparate music files.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if sourcePathOption == "" {
				return fmt.Errorf("missing 'source' option")
			}
			if destinationPathOption == "" {
				return fmt.Errorf("missing 'destination' option")
			}
			fmt.Println("Searching:", sourcePathOption)
			err := filepath.Walk(sourcePathOption, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				if !isAudioFile(info.Name()) {
					return nil
				}
				if info.Size() == 0 {
					return nil
				}
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				defer f.Close()
				m, err := tag.ReadFrom(f)
				if err != nil {
					fmt.Println(err, "-->", info.Name())
					return nil
				}
				artist := "No Name"
				if strings.TrimSpace(m.AlbumArtist()) != "" {
					artist = normalize(m.AlbumArtist())
				} else if strings.TrimSpace(m.Artist()) != "" {
					artist = normalize(m.Artist())
				}
				album := "No Name"
				if strings.TrimSpace(m.Album()) != "" {
					album = normalize(m.Album())
				}
				d := fmt.Sprintf("%s/%s/%s", destinationPathOption, artist, album)
				if err := os.MkdirAll(d, 0777); err != nil {
					return err
				}
				destPath := fmt.Sprintf("%s/%s", d, info.Name())
				fmt.Println("source:", path)
				fmt.Println("\t ->", destPath)
				if dryRunFlagOption {
					return nil
				}
				if err := os.Rename(path, destPath); err != nil {
					return err
				}
				return nil
			})
			if err != nil {
				return err
			}
			fmt.Println("done.")
			return nil
		},
	}
	rootCmd.AddCommand(organizeCmd, m3uCmd, playlistsCmd)
	rootCmd.PersistentFlags().BoolVarP(&dryRunFlagOption, "dryrun", "r", false, "dryrun option")
	rootCmd.PersistentFlags().StringVarP(&sourcePathOption, "source", "s", "", "source path with music files")
	rootCmd.PersistentFlags().StringVarP(&destinationPathOption, "destination", "d", "", "destination path for organized music")
	rootCmd.MarkFlagRequired("source")
	rootCmd.MarkFlagRequired("destination")
	rootCmd.Execute()
}
