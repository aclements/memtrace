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

	"./memtrace"
)

func main() {
	log.SetPrefix("pcs: ")
	log.SetFlags(0)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: pcs memtrace ea\n")
		os.Exit(2)
	}
	flag.Parse()
	if flag.NArg() != 2 {
		flag.Usage()
	}
	memtraceName, eaStr := flag.Arg(0), flag.Arg(1)
	ea, err := strconv.ParseUint(eaStr, 0, 64)
	if err != nil {
		flag.Usage()
	}

	traceFile, err := os.Open(memtraceName)
	if err != nil {
		log.Fatal(err)
	}
	defer traceFile.Close()

	trace := memtrace.NewTrace(traceFile)

	var recs [1024]memtrace.Record
	for {
		if _, err := trace.ReadRecords(recs[:]); err != nil {
			if err == io.EOF {
				break
			}
			log.Fatal(err)
		}

		for _, rec := range recs {
			if rec.EA == ea {
				fmt.Printf("%#x @%d\n", rec.PC, rec.N)
			}
		}
	}
}
