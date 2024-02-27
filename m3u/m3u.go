package m3u

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

type Track struct {
	Path   string
	Artist string
	Title  string
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
#EXTINF:{{ .Length }},{{ .Artist }} - {{ .Title }}
{{ .Path }}
{{- end }}
`

func NormalizeTitle(cmusPlaylistPath string) string {
	m3uPath := strings.TrimSuffix(cmusPlaylistPath, filepath.Ext(cmusPlaylistPath))
	m3uTitle := strings.ReplaceAll(filepath.Base(m3uPath), "-", " ")
	return strings.ReplaceAll(m3uTitle, "_", " ")
}

func GleanFilename(cmusPlaylistPath string) string {
	m3uPath := strings.TrimSuffix(cmusPlaylistPath, filepath.Ext(cmusPlaylistPath))
	return m3uPath + ".m3u"
}

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
