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
	sourcePathOption      string
	destinationPathOption string
)

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
				if !info.IsDir() {
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
					err = os.MkdirAll(d, 0777)
					if err != nil {
						return err
					}
					filename := strings.TrimPrefix(path, sourcePathOption)
					newPath := fmt.Sprintf("%s/%s", d, filename)
					fmt.Println(path, "\n\t->", newPath)
					err = os.Rename(path, newPath)
					if err != nil {
						return err
					}
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
	rootCmd.PersistentFlags().StringVarP(&sourcePathOption, "source", "s", "", "source path with music files")
	rootCmd.PersistentFlags().StringVarP(&destinationPathOption, "destination", "d", "", "destination path for organized music")
	rootCmd.MarkFlagRequired("source")
	rootCmd.MarkFlagRequired("destination")
	rootCmd.Execute()
}
