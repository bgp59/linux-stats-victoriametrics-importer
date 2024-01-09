# procfs Parsers

## Rationale For Custom Parsers

This stats collector is intended to be very granular time-wise, with sampling intervals of 1 second or even hundreds of milliseconds. 

To ensure minimal %CPU utilization, the parsers have to be optimized for speed and minimum processing.

The tests in [benchmarks](../benchmarks) directory illustrate the performance gains v. the established [prometheus/procfs](https://github.com/prometheus/procfs).

The performance gains come at a cost though:
* the code is less modular/readable; e.g. long-ish loops with no function calls inside (TODO: explore Go [inline](https://pkg.go.dev/golang.org/x/tools/internal/refactor/inline) package, maybe once it matures)
* data presentation (i.e. parsed data) may be in raw format, for instance `[]byte` instead of `int`

## 


