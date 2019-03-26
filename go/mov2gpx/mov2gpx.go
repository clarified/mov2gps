//{{{  license
// Copyright 2016 KB Sriram
// Copyright 2018,2019 A E Lawrence
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//}}}

package main

//{{{  imports

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"github.com/clarified/mov2gps/go/nb"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
	"strconv"
)
//}}}
const version = "1"
//{{{  flags
// Should add a -V version flag sometime
// Need flag to avoid GGA if unreliable
var (
	overwrite = flag.Bool("w", false, "Overwrite any existing gpx file")
	verbose   = flag.Bool("v", false, 
				"Report Firmware details,etc, if present")
	debug     = flag.Bool("debug", false, "tracing to stderr")
	Odir      = flag.String("O","", 
	"Destination directory for gpx file(s) or '-' for stdout\n Default: MOV file directory")
	gpxVersion = flag.Int("g",1,
			"gpx version: 0 or 1 for 1.0 or 1.1")
	noNMEA = flag.Bool("x", false, "Do not use NMEA GGA records")
	verFlag = flag.Bool("V", false, "Display version")
	rubbish = flag.Bool("clean", true, 
		"Remove dubious points at sea with lat/long = 0/0")
)
//}}}
//{{{  usage
func usage() {
	fmt.Fprintln(os.Stderr, "usage: mov2gpx [flags] [path ...]")
	flag.PrintDefaults()
	os.Exit(2)
}
//}}}

//{{{  main
func main() {
	flag.Usage = usage
	flag.Parse()
	if *verFlag {
		fmt.Println("mov2gpx version " + version)
		return
	}
	if *gpxVersion > 1 {
		fmt.Printf(
		"*Only gpx 1.0 or 1.1 supported*.\n Set -g 0 for 1.0. Otherwise 1.1\n\n")
		usage()
	}

	// Try to give clicky-pointy types a clue:
	if flag.NArg() == 0 {
		usage()
	}

	nb.SetDebug(*debug)
	for i := 0; i < flag.NArg(); i++ {
		if err := process(flag.Arg(i)); err != nil {
			log.Fatal(err)
		}
	}
}
//}}}

//{{{  process(movePath) error
func process(movPath string) error {
	//{{{  Sort out files
	//{{{  Set gpxPath to output path
	ext := filepath.Ext(movPath)
	// Maybe should probe the "magic" field in the mov file, rather than rely
	// on the extension?
	if !strings.EqualFold(ext, ".mov") {
		return errors.New(fmt.Sprintf("%v: Does not end with .MOV", movPath))
	}

	var stdout bool
	var gpxPath string

	switch *Odir {
	case ""  : gpxPath = movPath[:len(movPath)-len(ext)]
	case "-" : stdout = true
	default  : root := filepath.Base(movPath)
		gpxPath = filepath.Join(*Odir,root[:len(root) - len(ext)])
	}

	if !stdout {
		gpxPath = gpxPath + ".gpx"
		if *verbose && *Odir != "" {
			fmt.Fprintf(os.Stderr,fmt.Sprintf("Writing to %s\n", gpxPath))
		}
	}
	//}}}

	//{{{  Check for overwrite
	if !*overwrite {
		if _, err := os.Stat(gpxPath); err == nil {
			return errors.New(fmt.Sprintf("%s: already exists. Use -w to overwrite.", gpxPath))
		}
	}
	//}}}

	movFile, err := os.Open(movPath)
	if err != nil {
		return err
	}
	defer movFile.Close()

	var out *bufio.Writer

	switch {
	case stdout : out = bufio.NewWriter(os.Stdout)
	default     :
		gpxFile, err := os.Create(gpxPath)
		if err != nil {
			return err
		}
		defer gpxFile.Close()
		out = bufio.NewWriter(gpxFile)
	}
	defer out.Flush()
	//}}}

	gpsLogs, udata, err := nb.NewInfo(movFile).GPSLogs()
	if err != nil {
		return err
	}
	//{{{  Handle any comments or information from udta
	if *verbose || *debug {
		fmt.Fprint(os.Stderr,fmt.Sprintf("%s\n\tComment: %s\t Format/firmware: %s\n",
						movPath,(*udata).Inf,(*udata).Fmt))
	}
	//}}}

	writeHeader(out)
	for _, gpsLog := range gpsLogs {
		if e := writePoint(out, &gpsLog); e != nil {
			return e
		}
	}
	writeFooter(out)
	return nil
}
//}}}
//{{{  writePoint(w,gpsLog) error
// https://en.wikipedia.org/wiki/GPS_Exchange_Format
// & the gpx xsd schemas.

//{{{  RMC index offsets
// Unfortunately the RMC entries are sometimes truncated and
// corrupted, so we have the -x flag to avoid. And try to set if 
//  problems detected. The main reason for keeping is for debug
//  investigation of new dashcams/firmware

const (
	rUTC	= iota	// hour-min-sec without separators
	rValid	= iota	// A - valid, V - warning
	rLat	= iota	// Latitude
	rNorS	= iota	// North or South (N|S)
	rLong	= iota	// Longitude
	rEorW	= iota	// East or West (E|W)
	rSpeed	= iota	// Speed in knots (1.852 kM/hr)
	rCourse	= iota	// Course (degrees, true)
	rDate	= iota	// Date (ddmmyy)
	_	= iota	// Mag variation
	_	= iota	// (E|W) of variation
	_	= iota	// Checksum & trailing zeros
)
//}}}
//{{{  GGA index offsets

const (
	gUTC	= iota	// hour-min-sec without separators
	gLat	= iota	// Latitude
	gNorS	= iota	// North or South (N|S)
	gLong	= iota	// Longitude
	gEorW	= iota	// East or West (E|W)
	gQual	= iota	// 0 for no fix, 1 for gps, 2 for dgps
	gSat	= iota	// No of satelites in use
	gHdop	= iota	// Horizontal dilution of prec.
	gHeight	= iota	// Height
	gHUnit	= iota	// Height unit (Metres normally)
	gGeoid	= iota	// Geoid separation
	gGUnit	= iota	// Geo sep unit (Metres normally)
	_	= iota	// Differential relevance?
	_	= iota	// Checksum & trailing zeros
)
//}}}

func writePoint(w *bufio.Writer, gpsLog *nb.GPSLog) error {
// Not actually using error at present, so could remove?

	//{{{  Do nothing if a rubbish point
	// Real gps videos always seem to have "GPS ", even when rubbish points.
	// Testing for GPS is worthwhile when fed non gps video
	if string(gpsLog.Magic[:]) != "GPS " ||
		(*rubbish && gpsLog.Latitude == 0 && gpsLog.Longitude == 0 )||
		gpsLog.Mon == 0  {
		return nil
	}
	//}}}
	localNo := *noNMEA // Could leave at default
	//{{{  Set flag(s) if RMC/GGA records seen: if so slice around ","
	// RMC has speed, but we need conversion from knots, so no advantage
	// over direct value. Include in debug in case some dash cam
	// only includes RMC. Likewise parse gga even when noNMEA so that
	// debug can probe unknown firmware/dashcams & examine corrupt
	// entries.

	rmc := string(gpsLog.MagicRMC[:]) == "$GPRMC,"
	gga := string(gpsLog.MagicGGA[:]) == "$GPGGA,"

	// It is very likely that the rmc and gga flags are the same for all points in
	// a single file. If that is so, an obvious optimization is to 
	// set these at the outer "file" level.

	// Do the slice separation here (minimal parse)
	var ggaSlices [][]byte

	//{{{  COMMENT rmcSlices not used for now
	//if rmc && *debug {
	//        rmcSlices = bytes.Split(gpsLog.RMCentries[:],[]byte(`,`))
	//}
	//}}}
	if gga {
		ggaSlices = bytes.Split(gpsLog.GGAentries[:],[]byte(`,`))
		// Unfortunately some entries can be unexpectedly blank
		//  so we need various tedious checks later

		// RMC entries can be corrupt, so check GGA at least isn't truncated
		localNo = *noNMEA || len(ggaSlices) < 14
	}

	//}}}
	//{{{  Lat,Lon attributes
	// Could get from RMC or GAA, but still requires some reformatting, so little
	// advantage. But if there are any dash-cams that only provide RMC or GGA
	// this might need modifying.

	w.WriteString(fmt.Sprintf(`
      <trkpt lat="%.6f" lon="%.6f">`,
		toDD(gpsLog.LatitudeSpec, gpsLog.Latitude),
		toDD(gpsLog.LongitudeSpec, gpsLog.Longitude)))
	//}}}
	//{{{  SpeedCourse(first) -- anon so closure.
	var speedCourse func(bool) = func(first bool) {

		//{{{  Do nothing if first does not match gpxVersion
		// When first is true, only active for gpx1.0.
		//  If first is false, only active for gpx1.1
		// If anyone has set -g negative, they deserve what they get!
		if (!first && *gpxVersion == 0 ) || (first && *gpxVersion == 1 ) {
		      return
		}
		//}}}

		var prefix,speed,course,courseVal,ctag string
		var testCourse bool

		//{{{  open extension if 1.1 & set prefix while we are here
		if *gpxVersion == 1 {
		      w.WriteString(`
	<extensions>
	  <gpxtpx:TrackPointExtension>`)

		prefix = "gpxtpx:"
		}
		//}}}

		//{{{  build speed
		// speed comes in knots, convert to m/s
		const knotsToMpersec = 1852.0/3600.0	//Decimal to get float64

		//{{{  COMMENT Simple speed, but multiple string copies
		//     speed = `
		//<` + prefix + `speed>` +
		//     fmt.Sprintf("%.6f", (gpsLog.Speed)*knotsToMpersec) +  `</` +
		//     prefix + `speed>`
		//}}}
		//{{{  Builder:  marginally more efficient way to build speed
		stag := prefix + `speed`
		var sb strings.Builder

		sb.WriteString(`
	   <`)
		sb.WriteString(stag)
		sb.WriteString(`>`)
		sb.WriteString( fmt.Sprintf("%.6f", (gpsLog.Speed)*knotsToMpersec))
		sb.WriteString(`</`)
		sb.WriteString(stag)
		sb.WriteString(`>`)
		speed = sb.String()

		//}}}
		//}}}
		//{{{  course & testCourse
		// At low speeds, 0 values for course appear to mean unknown, so
		// skip such points
		// It turns out that RMC can be truncated, so rather than
		// do checks for validity, just collect from main log
		testCourse = gpsLog.Speed > 2 || gpsLog.Course > 0.00001
		if testCourse {
			courseVal = fmt.Sprintf("%.6f",gpsLog.Course)
		}

		// Probably not worth using a Builder for course
		ctag   = prefix + `course>`
		course = `
	   <` + ctag + courseVal + `</` + ctag

		//}}}

	      if *gpxVersion == 1 {
		      w.WriteString(speed)
		      if testCourse {
			      w.WriteString(course)
		      }
	      } else {
		      if testCourse {
			      w.WriteString(course)
		      } 
		      w.WriteString(speed)
	      }

	      //{{{  Close extension for 1.1
	      if *gpxVersion == 1 {
		      w.WriteString(`
	  </gpxtpx:TrackPointExtension>
       </extensions>`)
	      }
	      //}}}
	}
      //}}}

	//{{{  debug for RMC,GGA
	  // Buried in lang spec under Conversions: strings <-> byte slices only
	  // See subsection : Conversions to and from a string type

	if *debug {
		if rmc {
			log.Printf("RMC present: RMC  %s\n", gpsLog.RMCentries)
		}

		if gga {
			log.Printf("GGA present: ggaSlices = %s\n", ggaSlices)
		}
	}
	//}}}

       //{{{  <ele>
       // Height in metres (no idea what other units can occur in gHUnit)
       // gHeight can sometimes be empty which leads to a strictly invalid
       // gpx file. 
       if !localNo && gga {
		height := string(ggaSlices[gHeight])
		if height != "" && bytes.Equal(ggaSlices[gHUnit], []byte{'M'}) {
			w.WriteString(fmt.Sprintf(`
	<ele>%s</ele>`, height))
		}
	}
//}}}
       //{{{  <time>
       //{{{ Time should (must) be in UTC, formatted as ISO8601 
       // sggps did something strange here.
       // It assumed that the time read from the MOV was local
       // rather than UTC. Maybe some dashcams do that: after all
       // the displayed value in the video is local.
       // But Nextbase, at least, seem to use UTC in the gps record.

       // The gpx specfication requires that the time is UTC
       // and formatted according to ISO 8601. sggps conformed to
       // ISO 8601, but violated gpx by using local time in the gpx.
       // The original sggps code looked like:
       //	_, offset := time.Now().Zone()
       //	var neg byte
       //	if offset < 0 {
       //		neg = '-'
       //		offset = -offset
       //	} else {
       //		neg = '+'
       //	}
       //
       //	hOffset := offset / 3600
       //	sOffset := offset % 3600

       //}}}
       w.WriteString(fmt.Sprintf(`
        <time>%4d-%02d-%02dT%02d:%02d:%02dZ</time>`,
	2000+int(gpsLog.Year), int(gpsLog.Mon), int(gpsLog.Day),
	int(gpsLog.Hour), int(gpsLog.Min), int(gpsLog.Sec),
	))
//}}}
	speedCourse(true)
	if !localNo {
	       //{{{  <geoidheight>
		if gga {
			geoid := ggaSlices[gGeoid]
			if len(geoid) > 0 &&
				bytes.Equal(ggaSlices[gGUnit], []byte{'M'}) {
				w.WriteString(fmt.Sprintf(`
	<geoidheight>%s</geoidheight>`, geoid))
			}
		}
	       //}}}
	       //{{{  <sat>
		if gga && *gpxVersion == 0 {
			sat := ggaSlices[gSat]
			if len(sat) > 0 {
				w.WriteString(fmt.Sprintf(`
	<sat>%s</sat>`, sat))
			}
		}
	       //}}}
	       //{{{  <hdop>
	       if gga  {
		       hdop := ggaSlices[gHdop]
		       if len(hdop) > 0 {
			       w.WriteString(fmt.Sprintf(`
	<hdop>%s</hdop>`, ggaSlices[gHdop]))
			}
		}
		//}}}
	}
	speedCourse(false)

      //{{{  /trkpt
	w.WriteString(`
     </trkpt>`)
      //}}}

	return nil
}
//}}}
//{{{  toDD(spec,v) float32
// Input comes as decimal minutes
func toDD(spec byte, v float32) float32 {
	deg, frac := math.Modf(float64(v) / 100)
	result := float32(deg + frac/0.6)
	if spec == 'S' || spec == 'W' {
		result = -result
	}
	return result
}
//}}}
//{{{  writeHeader(w) error
func writeHeader(w *bufio.Writer) error {

	gver := strconv.Itoa(*gpxVersion)

	_, err := w.WriteString(`<?xml version="1.0" encoding="UTF-8" ?>
<gpx
 xmlns="http://www.topografix.com/GPX/1/`)
 
	_, err = w.WriteString(gver)
	_, err = w.WriteString( `"
 xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
 xsi:schemaLocation="http://www.topografix.com/GPX/1/`)
	_, err = w.WriteString(gver)
	_, err = w.WriteString( ` http://www.topografix.com/GPX/1/`)
	_, err = w.WriteString(gver)
	_, err = w.WriteString( `/gpx.xsd"
`)
	if *gpxVersion == 1 {
		_, err = w.WriteString(
		  ` xmlns:gpxtpx="http://www.garmin.com/xmlschemas/TrackPointExtension/v2"
`)
	}
	_, err = w.WriteString( ` version="1.`)
	_, err = w.WriteString(gver)
	_, err = w.WriteString(`"
 creator="mov2gpx">
  <trk>
    <trkseg>`)
	return err
}
//}}}
//{{{  writeFooter(w) error
func writeFooter(w *bufio.Writer) error {
	_, err := w.WriteString(`
    </trkseg>
  </trk>
</gpx>
`)
	return err
}
//}}}
