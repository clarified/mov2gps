mov2gpx
=======

Some video cameras with gps embed the position information in the video
files they generate: in addition to displaying the coordinates in video
frames, they are recorded directly in the file.

mov2gpx aims to extract such information from Quicktime MOV video files.

Many dashcams produce their video as MOV files: it seems likely that
cycle and bodyworn cameras may do something similar. Perhaps some
drone cameras take the same approach.

Camera manufacturers usually provide their own elaborate program for viewing
the video, and other functions including extracting the gps tracks.
But often only a very limited number of proprietary, closed source, operating
systems are supported. 

On linux, there is a rich set of programs for viewing and manipulating video,
so there is not much need for the proprietary programs, except for this
problem: there seemed to be only one program, sggps, which could
extract the gps information.

sggps worked with a Nextbase 312GW dashcam with the original firmware.
But with more recent firmware, sggps failed. After failing to contact the
sggps author, mov2gpx was forked from sggps.

mov2gps was developed to work on the 312GW camera, but there is good reason
to expect it to work with other cameras using similar chipsets. It has been
tested on video from a Street Guardian SG9665GC for which sggps was written.
That video reported *Novatek-96650*. Other Nextbase cameras turn out to
use a different variety of MOV video: version 1 of mov2gpx does not work
for those. However, version 2 will cater for most of those models.

mov2gps is written in portable **go**, so should work on any operating system
with **go** support. The releases subdirectory contains a few simple tar or zip
archives containing the binary executables pre-compiled for several operating
systems.

mov2gpx is intended to be lightweight, simple and fast; the time taken
to write the gpx output will almost always be dominated by the the speed
of the media, often flash. **go** by default produces rather large statically
linked files, so the usual mov2gpx binaries are typically of the order of 2MB.
The size can be reduced a little by stripping debug symbols and the like:
refer to the **go** documentation.

Sources are folded
------------------

Note that the source files are *folded*. If you modify the code, please use a
proper folding editor with support for indented folds such as vim with the
[Kent folding extensions](
https://www.vim.org/scripts/script.php?script_id=416).  The directory
`kent_folding` contains a link to the extensions.  Go folding support is
planned, but a small patch is provided to add **go** if you download an old
version.

Many folding editors are seriously deficient without proper indentation.
Indentation is especially important in **go** source code.

Viewing the sources without proper folding support is possible, but
is likely to be unpleasant and confusing. Much of the structure is
embedded in the folding including indented folds.


