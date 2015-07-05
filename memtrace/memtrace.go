// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package memtrace

import (
	"bytes"
	"encoding/binary"
	"io"
)

type Trace struct {
	r           io.ReadSeeker
	recno       int
	hdr         blockHeader
	blockEnd    int // recno past end of block
	blockData   []byte
	blockReader *bytes.Reader
	rec         Record
}

func NewTrace(r io.ReadSeeker) *Trace {
	return &Trace{r: r}
}

type Record struct {
	N      int
	PC, EA uint64
}

type blockHeader struct {
	Bytes, Recs uint64
}

func (t *Trace) readBlockHeader() error {
	if err := binary.Read(t.r, binary.LittleEndian, &t.hdr); err != nil {
		return err
	}
	t.blockEnd = t.recno + int(t.hdr.Recs)
	return nil
}

func (t *Trace) readBlockContent() error {
	size := int(t.hdr.Bytes - 16)
	if size > cap(t.blockData) {
		t.blockData = make([]byte, 0, size*2)
	}
	t.blockData = t.blockData[:size]

	if _, err := io.ReadFull(t.r, t.blockData); err != nil {
		return err
	}
	t.blockReader = bytes.NewReader(t.blockData)
	t.rec = Record{}
	return nil
}

func (t *Trace) readRecord() error {
	deltaPC, err := binary.ReadVarint(t.blockReader)
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return err
	}
	deltaEA, err := binary.ReadVarint(t.blockReader)
	if err != nil {
		if err == io.EOF {
			err = io.ErrUnexpectedEOF
		}
		return err
	}

	t.rec.N = t.recno
	t.rec.PC += uint64(deltaPC)
	t.rec.EA += uint64(deltaEA)
	t.recno++
	return nil
}

func (t *Trace) Seek(recno int) error {
	if _, err := t.r.Seek(0, 0); err != nil {
		return err
	}
	t.recno = 0
	for {
		if err := t.readBlockHeader(); err != nil {
			return err
		}
		if t.blockEnd > recno {
			break
		}
		// Skip block.
		if _, err := t.r.Seek(int64(t.hdr.Bytes-16), 1); err != nil {
			return err
		}
		t.recno += int(t.hdr.Recs)
	}
	// Load this block.
	if err := t.readBlockContent(); err != nil {
		return err
	}
	// Seek within the block.
	for t.recno < recno {
		if err := t.readRecord(); err != nil {
			return err
		}
	}
	return nil
}

func (t *Trace) ReadRecords(recs []Record) (int, error) {
	for i := range recs {
		if t.recno == t.blockEnd {
			// Read in next block.
			if err := t.readBlockHeader(); err != nil {
				return i, err
			}
			if err := t.readBlockContent(); err != nil {
				return i, err
			}
		}
		if err := t.readRecord(); err != nil {
			return i, err
		}
		recs[i] = t.rec
	}
	return len(recs), nil
}

// func (t *Trace) ReadBlock() ([]record, error) {
// 	var hdr blockHeader
// 	if err := binary.Read(t.r, binary.LittleEndian, &hdr); err != nil {
// 		return nil, err
// 	}

// 	// Read block
// 	block := make([]byte, hdr.Bytes-16)
// 	if _, err := io.ReadFull(t.r, block); err != nil {
// 		return nil, err
// 	}

// 	// Decode block
// 	blockReader := bytes.NewReader(block)
// 	out := make([]record, hdr.Recs)
// 	var pc, ea uint64
// 	for i := range out {
// 		deltaPC, err := binary.ReadVarint(blockReader)
// 		if err != nil {
// 			if err == io.EOF {
// 				err = io.ErrUnexpectedEOF
// 			}
// 			return nil, err
// 		}
// 		deltaEA, err := binary.ReadVarint(blockReader)
// 		if err != nil {
// 			if err == io.EOF {
// 				err = io.ErrUnexpectedEOF
// 			}
// 			return nil, err
// 		}

// 		pc += uint64(deltaPC)
// 		ea += uint64(deltaEA)
// 		out[i].pc = pc
// 		out[i].ea = ea
// 	}

// 	t.rec += int(hdr.Recs)
// 	return out, nil
// }
