This repository contains a Pintool that logs the program counter and
address of all memory writes to a compact on-disk log, plus some
simple tools to examine that log.

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

There is a Go package in the `memtrace` directory for reading the log
format. You can build your own tools to process the log format using
this package, or use some of the simple tools included in this
repository.

`dump.go` prints the records from a trace, optionally narrowed to a
range of records. This is useful for seeing the details around a
particular time in program execution once another tool has been used
to find interesting points in execution.

`pcs.go` prints the program counters of all writes to a particular
address.
