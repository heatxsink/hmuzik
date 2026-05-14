package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dhowden/tag"
	"github.com/spf13/cobra"
)

var (
	sourcePathOption      string
	destinationPathOption string
	dryRunFlagOption      bool
)

func isAudioFile(filename string) bool {
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".flac", ".mp3", ".m4a", ".aiff":
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

var organizeCmd = &cobra.Command{
	Use:   "organize",
	Short: "Organize a directory of disparate music files into Artist/Album/.",
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
			if err := os.MkdirAll(d, 0755); err != nil {
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

func init() {
	organizeCmd.Flags().StringVarP(&sourcePathOption, "source", "s", "", "source path with music files")
	organizeCmd.Flags().StringVarP(&destinationPathOption, "destination", "d", "", "destination path for organized music")
	organizeCmd.Flags().BoolVarP(&dryRunFlagOption, "dryrun", "r", false, "print planned moves without performing them")
	_ = organizeCmd.MarkFlagRequired("source")
	_ = organizeCmd.MarkFlagRequired("destination")
	rootCmd.AddCommand(organizeCmd)
}
