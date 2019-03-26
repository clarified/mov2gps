//{{{  License
// Copyright 2016 KB Sriram
// Copyright 2018 A E Lawrence
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

package nb

import (
	"encoding/binary"
	"errors"
	"github.com/clarified/mov2gps/go/mov"
	"io"
	"log"
)

var ErrInvalidGPS = errors.New("Not a GPS block")
//{{{  SetDebug(debug bool) - to pass in debug flag)
var debug bool
func SetDebug(incomingDebug bool) {
	debug = incomingDebug
	log.SetFlags(log.Lshortfile)
	mov.SetDebug(incomingDebug)
}
//}}}
//{{{  GPSLog
// In recent nb firmware, there are two full records of GPRMC & GPGAA in
//  approximately NMEA ascii. But missing in earlier versions.
// The first two entries (skipping atom header) are because we now
// read the data directly, rather than the original mov.VisitAtoms

//{{{  NmeaPartLen,NmeOffset- count for variable length entries, offset
// The $GPRMC and $GPGAA entries are ','-separated ascii strings and
// may vary in length. The trailing entries after the checksums seem to be zero
// bytes. In each case we collect a generous NmeaSectionLength of bytes.
const	NmePartLen = 0x48
const	NmeOffset  = 0x80	// Offset between GGA and RMC
//}}}

type GPSLog struct {
	_		[4]byte	// atom type (normally free)
	_		uint32	// atom length - assume large enough
	Magic         [4]byte	// "GPS "
	_             [36]byte	// Nothing useful with later firmware?
	Hour          uint32
	Min           uint32
	Sec           uint32
	Year          uint32	// since 2000
	Mon           uint32
	Day           uint32	// not Ascii
	ReceiverSpec  byte	// Guess: A - valid, V for warning
	LatitudeSpec  byte	// N or S
	LongitudeSpec byte	// W or E
	_             byte
	Latitude      float32
	Longitude     float32
	Speed         float32
	Course	      float32
	// Above common for 312GW, but varies for others
	// In earlier versions, data below is all zeroed.
	// Can test by examining MagicRMC
	// Data beyond this point only seen in some firmware versions
	_		[12]byte	// Unknown
	// Next 14 bytes might a recent displayed date/ (summmer)time
	// Seems to lag by 2 or 2 or 3 seconds??
	_		[4]byte		// Ascii year
	_		[2]byte		// Ascii month 
	_		[2]byte		// Ascii day digits
	_		[2]byte		// Ascii digits hour adjusted for summer??
	_		[2]byte		// Ascii digits min ?
	_		[2]byte		// Ascii digits. Secs @ end of last file?
	_		[14]byte	// Unknown - zeroed in latest firmware?
	// Perhaps there may be variable width fields below here
	// as there are in $GPGGA, in which case we need to
	// parse on separator ","
	MagicRMC	[7]byte		// "$GPRMC,"
	RMCentries	[NmePartLen]byte	// collect whole body
	_		[NmeOffset - NmePartLen -7]byte
	//{{{  COMMENT Old approach
	//HourRMC		[2]byte		// Ascii string, hh
	//MinRMC		[2]byte		// Ascii, mm
	//SecRMC		[5]byte		// ASCII string, ss.ss
	//_		byte		// "," separator
	//Status		byte		// A for valid, V for warning
	//_		byte		// "," separator
	//LatitudeDegR	[2]byte		// ll degress
	//LatitudeMinR	[7]byte		// mm.mmmm minutes
	//_		byte		// "," separator
	//LatitudeSpecR	byte		// N or S
	//LongitudeDegR	[3]byte		// ddd degrees
	//LongitudeMinR	[7]byte		// mm.mmmm minutes
	//_		byte		// "," separator
	//LongitudeSpecR	byte		// W or E
	//_		byte		// "," separator
	//SpeedKnots	[3]byte		// k.kk (1.852 km/h)
	//_		byte		// "," separator
	//BearingR	[4]byte		// dd.dd degrees
	//_		byte		// "," separator
	//DayR		[2]byte		// dd, day of month
	//Month		[2]byte		// mm, digits
	//YearR		[2]byte		// yy, Ascii digits
	//_		[3]byte		// dummy magnetic variation
	//_		[4]byte		// Checksum
	//_		[63]byte	// Seems to be zeroed
	//}}}
	// Next GGA section
	MagicGGA	[7]byte		// "$GPGGA,"
	GGAentries	[NmePartLen]byte
	//{{{  COMMENT Old  approach
	//HourG		[2]byte		// Ascii string, hh. UTC
	//MinG		[2]byte		// Ascii, mm
	//SecG		[6]byte		// ASCII string, ss.sss
	//_		byte		// "," separator
	//LatitudeDegG	[2]byte		// ll degress
	//LatitudeMinG	[7]byte		// mm.mmmm minutes
	//_		byte		// "," separator
	//LatitudeSpecG	byte		// N or S
	//_		byte		// "," separator
	//LongitudeDegG	[3]byte		// ddd degrees
	//LongitudeMinG	[7]byte		// mm.mmmm minutes
	//_		byte		// "," separator
	//LongitudeSpecG	byte		// W or E
	//_		byte		// "," separator
	//StatusG		byte		// Ascii digit (1 normally)
	//_		byte		// "," separator
	//Trailing	[38]byte	// Variable width fields from now on
	//{{{  COMMENT Trailing Variable fields
	//// These fields are separated with ','
	//// NumSat    1 or 2 bytes, Ascii
	//// separator ','
	//// HDOP - Horizontal dilution of Precision x.xx ?
	//// separator ','
	//// Altitude in Ascii.  aa.a, or aaa.a and so on. Meters WGS 84.
	//// ellipsoid)
	////  "," separator
	////  "M" for metres, Altitude unit
	//// GeoidHeight	[4]byte	 n.nn - can be used for <geoidheight>
	//// separator ','
	////  "M" for metres, GeoidM units
	////  "," separator
	////  (blank field, unknown - maybe for DGPS update)
	////  "," separator
	//// [3]byte checksum
	////	And zeroes expected after this. The field is a long enough, we hope,
	////	to cover everything up to the checksum, even with ridiculous
	////      expansion of Altitudes etc.
	//// http://aprs.gids.nl/nmea/#gga may help
	//}}}
	//}}}
}
//}}}
//{{{  GPSInfo interface -- method to extract GPSLogs
type GPSInfo interface {
	GPSLogs() ([]GPSLog,*UserData,error)
}
//}}}
//{{{  type UserData - used to pass comment & format
type UserData struct {
	Inf	[]byte
	Fmt	[]byte
}
//}}}
//{{{  genRead interface : mov.ReadAtSeeker and io.Reader
type genRead interface {
	mov.ReadAtSeeker
	io.Reader
}
//}}}
//{{{  gpsRas struct -- Seems difficult to convert to simple alias
type gpsRas struct {
	ras genRead
}
//}}}
//{{{  NewInfo -- just builds a gpsRas struct around a "ras"
// ras is called with a type from os.Open, thus *os.File 
func NewInfo(ras genRead) GPSInfo {
	return &gpsRas{ras}
}

//}}}
//{{{ Method GPSLogs - delivers gps data to top level 
// Method GPSLogs -- this is use to "deliver" results to top level command
// Since UserData so small, pointer not really needed, but good practice.
func (sgi *gpsRas) GPSLogs() ([]GPSLog,*UserData, error) {

	var udata UserData
	sa := &sampleAccumulator{}
	if err := mov.VisitAtoms(sa, sgi.ras); err != nil {
		return nil,nil, err
	}
	// Above sets position of sgi.ras to end.
	gpsLogs := make([]GPSLog, len(sa.audioOffsets))

	//{{{  Read the gps atom 64k bytes following each offset
	// and fail if it doesn't exist.
	// These atoms in the mdat are  of type "free", but we
	// don't match that.
	// With early firmware, they are 32K long, but with later version
	// they have grown to 64K. At most the  first 336 bytes contains non-zero
	// characters: the GPS info.
	// Maybe writing to 32/64K blocks helps with writing flash quickly?

	//{{{  Stategy: read the GPS data directly 
	// The original code reused mov.VisitAtoms, but that required
	// an assumption about the size of the free atoms containing
	// the gps information. And assumed that this size did not
	// vary with firmware version. So it would be necessary to read this size
	// before calling mov.VisitAtoms. This because there is no list: there is
	// no atom following the "free" atom. But if we need to read part
	// of the data stream, we might just as well read everything in one
	// go, which is what we now do.
	//}}}

	for i, offset := range sa.audioOffsets {
		goff := int64(offset + 0x10000)	
		_,err := sgi.ras.Seek(goff,io.SeekStart)
		if err != nil {
			return nil,nil,err
		}
		if err = binary.Read(sgi.ras, binary.LittleEndian, &gpsLogs[i]); 
			err != nil {
				return nil,nil, err
		}

	}
	//}}}
	udata.Inf = sa.format
	udata.Fmt = sa.comment
	return gpsLogs,&udata,nil
	//{{{  Original returned a copy of the slice
	//return gpsLogs[:],udata,nil  -- unclear why the original version
	//	returned a copy. No measured change to performance.
	//	We are already returning a slice, so essentially by reference
	//	so no redundant copying of the underlying array. Not clear
	//	what just copying the (pointer,len,cap) achieved here.
	//	Perhaps missing something?
	//}}}

}
//}}}

//{{{  TrimTrailingZeros
func TrimTrailingZeros(t []byte) []byte {
	// Could use bytes.TrimRight, but probably less efficient.
	// Below seems to be safe on all zeros which is a bit surprising
	// Unclear semantics of lazy && ...
	var i int
	for i = len(t) - 1; i >= 0 && t[i] == 0 ; i--  {
		}
	return t[:i+1]
	}
//}}}
//{{{  getUserData - for Comment and Format strings
func getUserData (sr *io.SectionReader) ([]byte,error) {
	// reads the (c)*** atom user data list string 
	// which has a 16 bit string count & lang code.
	// We assume that there is only one such string
	// although really there many be several in different
	// languages. Maybe modify later to cover that?
	var stringCount uint16	// Size of string, despite spec!
	var langCode	uint16
	//{{{  get stringCount
	if err := binary.Read(sr,binary.BigEndian,&stringCount) ; err != nil {
		return nil,err
	}
	//}}}
	//{{{  get langCode: not used yet, so could just Seek past
	if err := binary.Read(sr,binary.BigEndian,&langCode) ; err != nil {
		return nil,err
	}
	//}}}
	if stringCount == 0 {
		return nil,nil
	}
	target := make([]byte,stringCount)
	if _,err := io.ReadFull(sr,target); err != nil  {
		return nil,err
	}
	// Need to strip trailing zero bytes here
	return target,nil
}
//}}}
//{{{  sampleAccumulator struct
type sampleAccumulator struct {
	inSound		bool
	audioOffsets	[]uint32
	format		[]byte
	comment		[]byte
}
//}}}
//{{{  Method Visit for *sampleAccumulator
//{{{  frea atom - seems to be a Kodak special
// note parked here for now: frea atom
// Nextbase seem to use a Kodak special frea atom matching
// https://sno.phy.queensu.ca/~phil/exiftool/TagNames/Kodak.html
// https://metacpan.org/pod/distribution/Image-ExifTool/lib/Image/ExifTool/TagNames.pod#Kodak-frea-Tags
//{{{  COMMENT Kodak frea tags
// Tag ID	Tag Name	Writable
//'scra' 	PreviewImage 	no 	 
//'thma' 	ThumbnailImage 	no 	 
//'tima' 	Duration 	no	-- uint32 ?
//'ver ' 	KodakVersion 	no 	 
//}}}
//}}}

// const  copyright = 0xa9	// or just \xa9 - for udta atoms 

func (sa *sampleAccumulator) Visit(path []string, sr *io.SectionReader) error {
	last := len(path) - 1
	//{{{  Ensure that the path can be split. Otherwise not a target.
	if last < 1 {
		return nil
	}
	//}}}
	cur := path[last]
	switch {
	//{{{  Note when in sound track
	// the inside minf test may be redundant
	case  cur == "smhd" && inside(path[:last],"minf"):
		sa.inSound = true
		return nil
	//}}} 
	//{{{  record when exit from sound
	// Ought to make this more robust? Best would be when exit track, but
	// not so easy with this approach. Instead whenever enter a track?
	//case len(path) < 5 || cur == "vmhd":
	case  cur == "trak":
		sa.inSound = false
		return nil
	//}}}
	//{{{  Get the offsets when encounter the sound "stco"
	// Make this more robust as well....
	case sa.inSound && cur == "stco" && inside(path[:last], "stbl") :
		offsets, err := getAudioChunks(sr)
		sa.audioOffsets = offsets
		return err
	//}}}
	//{{{  format info 
	// Assume only one such udta atom is present, else we will
	// overwrite and only return the last.
	// The Nextbase entries are a bit dirty, so care needed. Trailing garbage
	// after counted string in one case. Many trailing zeros included in the
	// counted string in the other case.
	case cur == "\xa9fmt" && path[last-1] == "udta" :
		// need to set sa.Format from nil to value
		s,_ := getUserData(sr)
		sa.format = TrimTrailingZeros(s)
		return nil
	//}}}
	//{{{  Comment information
	case cur == "\xa9inf" && path[last-1] == "udta" :
		s,_ := getUserData(sr)
		sa.comment = TrimTrailingZeros(s)
		return nil
	//}}}
	default:
		return nil
	}
}
//}}}
//{{{  inside(path,target) : is target in path?
// Used to check for a target string in the path, looking from the end
// rather than the start. Used in parsing the atom heirarchy.
func inside(p []string, target string) bool {
	if  end := len(p) -1 ; end < 0 {
		return false		//empty p
	} else {
		//{{{  Look for immediate match, recurse otherwise
		if match := p[end] == target ; match {
			//{{{  Return true  immediately if a match
			return match
			//}}}
		} else {
			//{{{   Recurse if there is more
			switch {
				case end > 0 : return inside(p[:end - 1],target)
				default	     : return false
			}
			//}}}
		}
		//}}}
	}
}
//}}}
//{{{  getAudioChunks from stco atom
// Examine stco atom
func getAudioChunks(sr *io.SectionReader) ([]uint32, error) {
	//{{{  discard version/flags
	_,err := sr.Seek(4,io.SeekCurrent)
	if err != nil {
		return nil, err
	}
	//}}}
	var nent uint32
	//{{{  collect number of entries
	err = binary.Read(sr, binary.BigEndian, &nent)
	if err != nil {
		return nil, err
	}
	//}}}
	result := make([]uint32, nent)

	//{{{  Read chunk offset table
	for i := range result {
		err = binary.Read(sr, binary.BigEndian, &result[i])
		if err != nil {
			return nil, err
		}
	}
	//}}}
	if debug {
		log.Printf("chunk offsets: %x\n", result)
	}
	return result, nil
}
//}}}
