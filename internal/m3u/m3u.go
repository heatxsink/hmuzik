package m3u

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"unicode"

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
	defer func() { _ = f.Close() }()
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

func CreateFromCmusPlaylist(cmusPlaylistPath string, outputPath string, prefix string, resolver *Resolver) error {
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
		if errors.Is(err, fs.ErrNotExist) {
			resolved, rerr := resolver.Resolve(line)
			if rerr != nil {
				fmt.Println(rerr, "-->", line)
				continue
			}
			fmt.Println("resolved", line, "-->", resolved)
			line = resolved
			f, err = os.Open(line)
		}
		if err != nil {
			fmt.Println(err)
			continue
		}
		track := &Track{
			Path:   strings.TrimPrefix(line, prefix),
			Info:   "",
			Length: -1,
		}
		m, terr := tag.ReadFrom(f)
		switch {
		case terr == nil:
			track.Info = stripControl(fmt.Sprintf("%s - %s", m.Artist(), m.Title()))
		case errors.Is(terr, tag.ErrNoTagsFound):
			fmt.Println(terr, "-->", f.Name())
			track.Info = strings.TrimSuffix(filepath.Base(line), filepath.Ext(line))
		default:
			fmt.Println(terr, "-->", f.Name())
			track.Info = strings.TrimSuffix(filepath.Base(line), filepath.Ext(line))
		}
		_ = f.Close()
		tracks = append(tracks, track)
	}
	pl := &Playlist{
		Title:  NormalizeForTitle(cmusPlaylistPath),
		Tracks: tracks,
	}
	return pl.ToFile(Filename(cmusPlaylistPath, outputPath))
}

// stripControl drops control characters from tag text so a malformed tag
// cannot break an #EXTINF line across multiple lines.
func stripControl(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}
