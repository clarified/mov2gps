//{{{  license
// Copyright 2016 KB Sriram
// Copyright 2018 AE Lawrence
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

package mov

//{{{  imports 
import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"log"
)
//}}}

//{{{  SetDebug(debug bool)   -pass in debug flag
var debug bool
func SetDebug(incomingdebug bool) {
	debug = incomingdebug
	log.SetFlags(log.Lshortfile)
}
//}}}
var ErrAtomTooLarge = errors.New("Sorry, can't handle enormous atom")
//{{{  type Visitor interface- Visit methods
// The Visit method is invoked for each Atom encountered by VisitAtoms.
type Visitor interface {
	Visit([]string, *io.SectionReader) error
}
//}}}
//{{{  type ReadAtSeeker interface 
type ReadAtSeeker interface {
	io.ReaderAt
	io.Seeker
}
//}}}

//{{{  seekCur  
func seekCur(s io.Seeker) (int64, error) {
	return s.Seek(0, io.SeekCurrent)
}
//}}}
//{{{  seekEnd  
func seekEnd(s io.Seeker) (int64, error) {
	return s.Seek(0, io.SeekEnd)
}
//}}}

//{{{  VisitorFunc type - converts functions to Visit methods
// The VisitorFunc type which generates Visitors from plain functions.
// If f is a function with the appropriate signature,
// VisitorFunc(f) is a Visitor object that calls f.
type VisitorFunc func([]string, *io.SectionReader) error

func (f VisitorFunc) Visit(path []string, sr *io.SectionReader) error {
	return f(path, sr)
}
//}}}

//{{{  nextAtom(sr)  -- reads from stream sr
//{{{  Overview
// Takes sr (io.SectionReader) which has some current offset
// and returns the atom type (string) and a new Sectionreader
// pointing to the contents of this atom, with limit at the end.
// It is called repeatedly with the *same* sr, so it updates
// the sr offset to point to the next (outer) atom.
// So then next call examines the next outer atom, while
// the returned SectionReader can be used to examine the contents
// of this atom.
//}}}

func nextAtom(sr *io.SectionReader) (string, *io.SectionReader, error) {
	var asz uint32		// Usually the size of this atom
	var sz uint64		// For very large atoms
	//{{{  Get size into asz unless run off end
	// try to read length into asz. io.EOF if run off end
	if err := binary.Read(sr, binary.BigEndian, &asz); err != nil {
		return "", nil, err
	}
	//}}}
	atyp := make([]byte, 4) // Atom type
	if _, err := io.ReadFull(sr, atyp); err != nil {
		return "", nil, err
	}
	//{{{  Set sz to the size: there are 3 cases depending on asz
	switch asz {
	// asz =0 means "to the end of the file". Should only happen when sr points
	// at a bare file stream at the start of a scan so sr.Size() points to whole
	// file.
	case 0  : sz = uint64(sr.Size()) - 8 //Remaining size is whole - header

	case 1  :	//Enormous atom: read extended size
		if err := binary.Read(sr,binary.BigEndian,&sz); err !=nil {
		return "", nil, err
		}
		if sz > math.MaxInt64 {
			return "",nil, ErrAtomTooLarge
		}
		sz = sz - 16  // 4 for "size", 4 for type, 8 for extended 

	default : sz = uint64(asz) - 8 // remaining after size & type header 
	}
	//}}}

	cur, err := seekCur(sr)		// cur now points to the body of this atom
	if err != nil {
		return "", nil, err
	}
	if  debug {
		log.Printf("cur = %x, body length = %X, atom type is %v \n", 
		   cur, sz, string(atyp))
	}
	sr.Seek(int64(sz),io.SeekCurrent) // Update sr to point to next outer atom

	// Return type and SectionReader for this atom contents
	return string(atyp), io.NewSectionReader(sr, cur, int64(sz)), nil
}
//}}}
//{{{  visitAtomList
// sr points to an "outer" atom in a file or within another atom.
// It enters that atom to read the header, and update the sr position
// to point to the next "outer" atom, but also calls v.visit on
// the contents of this "outer" atom.
// If this is a relevant container atom it recurses to process the contained
// atoms.
// The root is a path through the parent atoms.

func visitAtomList(root []string, v Visitor, sr *io.SectionReader) error {
	if debug {
		log.Printf("Visiting at %v\n",root)
	}
	for {
		ctype, csr, err := nextAtom(sr)
		if err != nil {
			if err == io.EOF {
				return nil
			} else {
				return err
			}
		}
		err = v.Visit(append(root, ctype), csr) // process contents
		if err != nil {
			return err
		}

		//{{{  Explore relevant container atoms
		// Container atoms not included: meta
		switch ctype {
		// case "moov", "trak", "mdia", "minf", "stbl", "dinf":
		// Add udta below to see whether firmware version is there....
		// nb udta has @fmt & @inf embedded atoms.
		//   '@fmt -> Nextbase, @inf (old :just model, new: firmware version)
		case "moov", "trak", "mdia", "minf", "stbl", "dinf", "udta":
			err = visitAtomList(append(root, ctype), v, csr)
			if err != nil {
				return err
			}
		}
		//}}}
	}
}

//}}}
//{{{  VisitAtoms (v,rs) applies v.Visit to the atoms as they appear in rs
// VisitAtoms does a DFS visit of atoms in the provided mov file.
// That is, it visits the contained atoms inside outer atoms
// as they are encountered sequentially in the file stream.
// It has the side effect of Seeking to the end of rs.
//{{{  Graph
// As a graph:
//       atom ----> atom ----> atom ----> atom --->(top level, outer atoms)
//        |                     |           |
//      a --> a ...	     a --> a --> a  |
//      |                          |     |  |
//     ...			  ...	... |			    |
//                                      a -> a -> a -> a     
//     ...                        ...       ...
// In this view, it is a DFS.
//}}}

func VisitAtoms(v Visitor, rs ReadAtSeeker) error {
	len, err := seekEnd(rs)
	if err != nil {
		return err
	}
	return visitAtomList(make([]string, 0), v, io.NewSectionReader(rs, 0, len))
}
//}}}

