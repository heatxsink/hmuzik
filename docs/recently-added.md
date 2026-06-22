# recently-added

## Problem

The current `cmus-recently-added` shell script generates five cmus playlists
(`recently added (01d|07d|14d|30d|90d)`) by walking `$HOME/Music/Artists` with
`fd`, filtering on filesystem `ctime`, and sorting the results purely by
`stat -c %Z` (descending). The output is correct in the "what was added
recently" sense but completely wrong musically: tracks from the same album end
up scattered across the playlist because each file's ctime is independent. The
listening experience is jumbled — you hear track 7 of album A, then track 2 of
album B, then track 1 of album A.

We also walk the tree five times (once per window) and spawn a `stat` per file
via `--exec`, which is brute-force and slow on a large library.

## Goal

Generate the same five playlists, but **group tracks by album** and **order
within each album by disc/track number**. Albums themselves should still be
ordered by recency so the most recently added album appears at the top.

## Design

### Bucketing semantics

- Walk `$MUSIC_PATH` **once**. For each audio file, capture its `ctime` and
  read its tags (`AlbumArtist` || `Artist`, `Album`, `Disc`, `Track`).
- Group files into albums keyed by `(album_artist, album)`. Files missing tags
  or with empty album metadata fall back to a synthetic single-track "album"
  keyed by their file path so they're still included but never merged
  incorrectly with real albums.
- For each album, compute `album_ctime = max(ctime of its files)`. This is the
  recency anchor — adding a single new track to an old album bumps the whole
  album to the top, matching the spirit of "recently added".
- For each window `W ∈ {1d, 7d, 14d, 30d, 90d}`:
  - An album is **included** if `album_ctime` is within `W`.
  - **Only tracks whose own `ctime` is within `W`** are emitted. (Decision
    point — see Open Questions. Alternative: emit *all* tracks of any
    included album.)
  - Within the album, tracks sort by `(disc_number ASC, track_number ASC,
    filename ASC)` as a stable tiebreaker.
  - Albums sort by `album_ctime DESC`.

### Output format

Same as today: plain newline-delimited absolute paths, one per file, written
to `$CMUS_PLAYLIST_PATH/recently added (NNd)`. cmus reads these directly. The
existing `hmuzik playlists` subcommand already converts cmus playlists to
extended `.m3u`, so this new generator slots cleanly in front of it.

### CLI shape

New cobra subcommand under `main.go`:

```
hmuzik recently-added \
  --source $HOME/Music/Artists \
  --output $HOME/.config/cmus/playlists \
  [--windows 1,7,14,30,90] \
  [--dryrun]
```

- `--source` and `--output` reuse the existing persistent flag pattern where
  it makes sense; otherwise local flags on the subcommand. Note: the two
  `MarkFlagRequired` calls in `main.go:171-172` currently apply globally and
  will need to be moved to the subcommands that actually need them (`organize`)
  so this new command can declare its own required flags. That's a small
  pre-req cleanup, not new scope.
- `--windows` is a comma-separated list of day counts, defaulting to the
  five values above. Makes the script's hardcoded constants configurable
  without ceremony.
- `--dryrun` prints what would be written and skips disk writes, matching
  the existing flag's semantics.

### Implementation sketch

New package `recentlyadded/` keeps `main.go` thin and mirrors the layout of
the existing `m3u/` package. Public surface ~3 functions:

```go
type Track struct {
    Path        string
    Ctime       time.Time
    AlbumArtist string
    Album       string
    Disc        int
    Track       int
}

type Album struct {
    Key        string // album_artist + "\x00" + album
    LatestCtime time.Time
    Tracks     []*Track
}

// Scan walks root once, returns all audio files with parsed tags + ctime.
func Scan(root string) ([]*Track, error)

// GroupByAlbum buckets tracks into albums and sorts each album's tracks.
func GroupByAlbum(tracks []*Track) []*Album

// Playlist returns the ordered file paths for window W given the grouped albums.
func Playlist(albums []*Album, window time.Duration, now time.Time) []string
```

`Scan` uses `filepath.WalkDir` (cheaper than `Walk` — no per-entry `Lstat`),
reuses `isAudioFile` from `main.go` (lift it into a shared internal helper or
duplicate the four-line list — "a little copying is better than a little
dependency"), and reads tags via `github.com/dhowden/tag` exactly like
`organize` does today. `ctime` comes from `syscall.Stat_t.Ctim` on Linux —
which is what the bash `stat -c %Z` already returns, so behavior matches.
This is Linux-only; the README already says so.

`Playlist` writes via `os.WriteFile` with a `\n`-joined string. No template
needed — the cmus format is just paths.

### Performance

The bash script does roughly:

- 5 × `fd` walks of the tree
- 5 × N `stat` exec spawns
- 5 × sort + cut pipelines

The Go version does:

- 1 walk
- N tag reads (file opens — the expensive part)
- In-memory grouping + 5 sorts of an already-small album slice

Tag reads dominate. To stay honest about this: tag reads only happen for
files modified within the largest window (90d) — earlier files get skipped
after the `ctime` check, so the cost is bounded by recent activity, not
library size. On a cold cache the first run is still slower than the bash
script (which never opens files at all); subsequent runs benefit from the
page cache. Acceptable trade-off given the correctness win, and we can
add a tag-cache later if it actually hurts.

## Phases

Single PR — this is small enough to not warrant splitting.

1. Move the `--source`/`--destination` `MarkFlagRequired` calls in
   `main.go:171-172` off the root command and onto `organizeCmd` so the new
   subcommand can own its own flag contract.
2. Add `recentlyadded/` package with `Scan`, `GroupByAlbum`, `Playlist`,
   and a small `WriteCmusPlaylist(path string, paths []string) error`.
3. Add `recentlyAddedCmd` to `main.go`, wired into `rootCmd.AddCommand`.
4. Add unit tests for `GroupByAlbum` (tag fallback, missing album, disc/track
   sort) and `Playlist` (window filtering, recency ordering) using
   table-driven tests with synthetic `Track` slices — no filesystem fixtures
   needed for these layers.
5. Update `README.md` Example Usage section.
6. Once verified, delete `cmus-recently-added` from the repo root — it's
   superseded.

## Open Questions

1. **Partial-album semantics.** If you re-tagged track 5 of a 12-track album
   yesterday, should the 1-day playlist contain only track 5, or all 12 tracks
   of the album because the album is "active"? The design above goes with
   "only tracks in the window, but sorted within their album". The alternative
   ("whole album if any track qualifies") is arguably more listenable but
   inflates the 1d playlist significantly after any bulk re-tag. Default to
   the conservative option; add `--whole-album` later if it bites.

2. **AlbumArtist vs Artist fallback.** `organize` already prefers
   `AlbumArtist` then `Artist`. Same precedence here — keeps compilation
   albums grouped correctly. No question, just calling it out for review.

3. **Tag-less files.** The current bash script includes everything. The Go
   version will too, but tag-less files become single-track pseudo-albums
   keyed by file path. They'll appear in the playlist ordered purely by their
   own ctime, which is the same as today's behavior for those specific files.
