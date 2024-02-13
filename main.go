package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/tag"
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

func main() {
	rootCmd := &cobra.Command{
		Use:   name,
		Short: "",
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
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				defer f.Close()
				m, err := tag.ReadFrom(f)
				if err != nil {
					return err
				}
				artist := "No Name"
				if strings.TrimSpace(m.AlbumArtist()) != "" {
					artist = m.AlbumArtist()
				} else if strings.TrimSpace(m.Artist()) != "" {
					artist = m.Artist()
				}
				album := "No Name"
				if strings.TrimSpace(m.Album()) != "" {
					album = m.Album()
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
	rootCmd.AddCommand(organizeCmd)
	rootCmd.PersistentFlags().BoolVarP(&dryRunFlagOption, "dryrun", "r", false, "dryrun option")
	rootCmd.PersistentFlags().StringVarP(&sourcePathOption, "source", "s", "", "source path with music files")
	rootCmd.PersistentFlags().StringVarP(&destinationPathOption, "destination", "d", "", "destination path for organized music")
	rootCmd.MarkFlagRequired("source")
	rootCmd.MarkFlagRequired("destination")
	rootCmd.Execute()
}
