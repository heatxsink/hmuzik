# hmuzik

## Description

I grew tired of making directories, and moving files around. I then remembered Brian Hatfield's Step 3: Automate everything.

Thus...

I required a process that can be thrown at a directory of well tagged audio files, and have it organize/create a '%Artist%/%Album%' directory structure.

I've also added the ability to generate a m3u playlist from a cmus playlist.


## Assumptions

1. I have only tested this on Linux, this should work on macOS (famous last words).
1. Using Go 1.22.0.
1. Contents of '--source' path has music files that are well tagged.


## Example Usage 
    
    $ hmuzik organize -s /home/ngranado/Music/lz -d /home/ngranado/Music/lz/organized 
    
    $ hmuzik cmus2m3u /home/ngranado/Music/Playlists/ambient-skool.txt /home/ngranado 
    

## Collaboration

Pull requests are welcome, so are github issues.


## License

MIT License

Copyright (c) 2024 Nick Granado

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
