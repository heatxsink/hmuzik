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

Generate "recently added" cmus playlists, grouped by album and ordered by disc/track:

    $ hmuzik recently-added

By default this scans `$HOME/Music/Artists` and writes `recently added (NNd)`
playlists for the windows `1, 7, 14, 30, 90` days into
`$HOME/.config/cmus/playlists/`. Albums float to the top by most recent
ctime; tracks within an album keep their (disc, track) order. Override with
`-s/--source`, `-o/--output`, and `-w/--windows`. Use `-r/--dryrun` to print
counts without writing.


## Layout

    cmd/                       cobra subcommands
    internal/m3u/              m3u playlist generation
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
