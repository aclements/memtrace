This repository contains a Pintool that logs the program counter and
address of all memory writes to a compact on-disk log, plus some
simple tools to examine that log.

The tools are written in Go, so the recommended way to fetch this
repository is

    go get github.com/aclements/memtrace

Building
--------

You'll need [Intel
Pin](https://software.intel.com/en-us/articles/pin-a-dynamic-binary-instrumentation-tool).

Set `PIN_ROOT` to the location of your Pin installation. E.g.,

    export PIN_ROOT=$HOME/opt/pin-2.14-71313-gcc.4.4.7-linux

To build, simply run `make`. This will generate a `.so` plug-in for
Pin.


Tracing
-------

To run a command and log its memory accesses, run

    $PIN_ROOT/pin.sh -t obj-intel64/memtrace.so -- CMD

where `CMD` is the command you want to trace. This will generate a
`memtrace.log` file.

Processing
----------

The `cmd` directory contains some simple tools for processing memory
logs. To build these a tool, change to its directory and run `go
build`.

`cmd/dump` prints the records from a trace, optionally narrowed to a
range of records. This is useful for seeing the details around a
particular time in program execution once another tool has been used
to find interesting points in execution.

`cmd/pcs` prints the program counters of all writes to a particular
address.

The `memtrace` directory contains a Go package for reading the log
format. You can build your own tools to process memory logs using this
package.
