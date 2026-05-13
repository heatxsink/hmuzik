package m3u

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/dhowden/tag"
)

type Track struct {
	Path   string
	Info   string
	Length int
}

type Playlist struct {
	Title  string
	Tracks []*Track
}

var m3uTemplate = `#EXTM3U
#EXTENC:UTF-8
#PLAYLIST:{{ .Title }}
{{- range .Tracks }}
#EXTINF:{{ .Length }},{{ .Info }}
{{ .Path }}
{{- end }}
`

func (pl *Playlist) ToFile(path string) error {
	m3u := template.New("m3u")
	m3u, err := m3u.Parse(m3uTemplate)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return m3u.Execute(f, pl)
}

func NormalizeForTitle(cmusPlaylistPath string) string {
	m3uTitle := strings.ReplaceAll(filepath.Base(cmusPlaylistPath), "-", " ")
	return strings.ReplaceAll(m3uTitle, "_", " ")
}

func Filename(cmusPlaylistPath string, outputPath string) string {
	m3uBase := fmt.Sprintf("%s.m3u", filepath.Base(cmusPlaylistPath))
	return filepath.Join(outputPath, m3uBase)
}

func CreateFromCmusPlaylist(cmusPlaylistPath string, outputPath string, prefix string) error {
	data, err := os.ReadFile(cmusPlaylistPath)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	tracks := []*Track{}
	for _, line := range lines {
		if line == "" {
			continue
		}
		f, err := os.Open(line)
		if err != nil {
			fmt.Println(err)
			continue
		}
		defer f.Close()
		track := &Track{
			Path:   strings.TrimPrefix(line, prefix),
			Info:   "",
			Length: -1,
		}
		m, err := tag.ReadFrom(f)
		if err == tag.ErrNoTagsFound {
			fmt.Println(err, "-->", f.Name())
			track.Info = strings.TrimSuffix(filepath.Base(line), filepath.Ext(line))
		} else {
			track.Info = fmt.Sprintf("%s - %s", m.Artist(), m.Title())
		}
		tracks = append(tracks, track)
	}
	pl := &Playlist{
		Title:  NormalizeForTitle(cmusPlaylistPath),
		Tracks: tracks,
	}
	return pl.ToFile(Filename(cmusPlaylistPath, outputPath))
}
