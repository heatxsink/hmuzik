# hmuzik

## Description

I grew tired of making directories, and moving files around. I then remembered Brian Hatfield's Step 3: Automate everything.

Thus...

I required a process that can be thrown at a directory of well tagged audio files, and have it organize/create a '%Artist%/%Album%' directory structure.

I've also added the ability to generate a m3u playlist from a cmus playlist.


## Assumptions

1. I have only tested this on Linux, this should work on macOS (famous last words).
1. Using Go 1.26.
1. Contents of '--source' path has music files that are well tagged.


## Example Usage

Organize a directory of audio files into `%Artist%/%Album%/`:

    $ hmuzik organize -s /home/ngranado/Music/lz -d /home/ngranado/Music/lz/organized

Convert a single cmus playlist to extended m3u:

    $ hmuzik m3u /home/ngranado/.config/cmus/playlists/ambient-skool /home/ngranado

Convert every cmus playlist under `~/.config/cmus/playlists/` to extended m3u:

    $ hmuzik playlists /home/ngranado/Music/Playlists /home/ngranado

Generate a "Recently Added" cmus playlist, grouped by album and ordered by disc/track:

    $ hmuzik recently-added

By default this scans `$HOME/Music/Artists` and writes a `Recently Added`
playlist covering the last 14 days into `$HOME/.config/cmus/playlists/`.
Albums float to the top by most recent ctime; tracks within an album keep
their (disc, track) order. Override with `-s/--source`, `-o/--output`, and
`-d/--days`. Pass `-w/--windows 1,7,30` instead of `-d` to write one
`recently added (NNd)` playlist per window. Use `-r/--dryrun` to print counts
without writing.

Copy new and changed library files to a mounted Rockbox iPod:

    $ hmuzik sync -r --delete
    $ hmuzik sync --delete

By default this mirrors `$HOME/Music/Artists` onto
`/run/media/$USER/IPOD/Music/Artists`, which must already exist (the command
fails rather than writing anywhere if the iPod isn't mounted). A file is copied
when it is missing or its size or modification time differs, so after editing
tags let the modification time change (don't use `metaflac --preserve-modtime`)
or the edit won't be picked up. Copies go to a temp file that is synced and
renamed into place, so an unplug mid-copy never leaves a truncated track.
If the iPod is unplugged mid-run, the disk fills, or it remounts
read-only, the sync stops instead of writing elsewhere or piling up errors.
`--delete` removes files that are no longer in the library; paths match
case-insensitively, as on FAT. It refuses to run against the device root
(anything containing `.rockbox` or `iPod_Control`). Hidden files are ignored on both sides. The
device is never ejected. Update the Rockbox database afterwards to pick up the
changes. The summary line reports elapsed time and throughput. Override paths
with `-s/--source` and `-d/--dest`; `-r/--dryrun` prints the plan without
writing.


## Layout

    cmd/                       cobra subcommands
    internal/m3u/              m3u playlist generation
    internal/mirror/           incremental library-to-device copy
    internal/recentlyadded/    recently-added playlist builder
    main.go                    entry point


## Collaboration

Pull requests are welcome, so are github issues.


## License

Copyright 2026 Nick Granado <ngranado@gmail.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
