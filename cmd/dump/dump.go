// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	"github.com/aclements/memtrace/memtrace"
)

func main() {
	var err error

	log.SetPrefix("dump: ")
	log.SetFlags(0)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: dump memtrace [low [hi]]\n")
		os.Exit(2)
	}
	flag.Parse()
	if flag.NArg() < 1 || flag.NArg() > 3 {
		flag.Usage()
	}
	memtraceName := flag.Arg(0)
	low, hi := 0, int(^uint(0)>>1)
	if flag.NArg() >= 2 {
		lowStr := flag.Arg(1)
		low, err = strconv.Atoi(lowStr)
		if err != nil {
			flag.Usage()
		}
	}
	if flag.NArg() >= 3 {
		hiStr := flag.Arg(2)
		hi, err = strconv.Atoi(hiStr)
		if err != nil {
			flag.Usage()
		}
	}

	traceFile, err := os.Open(memtraceName)
	if err != nil {
		log.Fatal(err)
	}
	defer traceFile.Close()

	trace := memtrace.NewTrace(traceFile)
	if err := trace.Seek(low); err != nil {
		log.Fatal(err)
	}

	var recs [1]memtrace.Record
	for i := 0; i <= hi-low; i++ {
		if _, err := trace.ReadRecords(recs[:]); err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}
		fmt.Printf("%#x %#x @%d\n", recs[0].PC, recs[0].EA, recs[0].N)
	}
}
