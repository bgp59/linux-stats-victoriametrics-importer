# procfs Parsers

## Rationale For Custom Parsers

This stats collector is intended to be very granular time-wise, with sampling intervals of 1 second or even hundreds of milliseconds. 

To ensure minimal %CPU utilization, the parsers have to be optimized for speed and minimum processing.

The tests in [benchmarks](../benchmarks) directory illustrate the performance gains v. the established [prometheus/procfs](https://github.com/prometheus/procfs); the `..._parser_test.go` files have section with benchmark results showing the performance of raw file read (the baseline), the one of this package parser and that of the Prometheus `procfs` package. For instance [diskstats_parser_test.go](../benchmarks/diskstats_parser_test.go#L41-L47) shows that the overhead of this parser is ~ 4 µsec whereas the Prometheus one is ~ 84 µsec.

The performance gains come at a cost though:
* the code is less modular/readable; e.g. long-ish loops with no function calls inside (TODO: explore Go [inline](https://pkg.go.dev/golang.org/x/tools/internal/refactor/inline) package, maybe once it matures)
* data presentation (i.e. parsed data) may be in raw format, for instance `[]byte` instead of `int`

## Implementation Principles

* implement reusable objects: e.g. [buffer pools](readfile_buf_pool.go#L26-L41) for reading files and specialized parser structures with  a `Parse()` method. The underlying storage for the parsed data is allocated once when the parser is created and the content is updated with every `Parse()` call to reflect the latest data.
* avoid function calls even if this leads to long loops
* read the `/proc` files in one call into a `[]byte` buffer and parse it in one pass. Avoid scanners, field splitters and format conversion package function calls. e.g. [net_dev_parser.go](net_dev_parser.go#L196-L210)
* avoid unnecessary conversion for numerical data that will not suffer further transformations inside the caller of the parser. The stats collector uses the [Prometheus exposition text format](https://github.com/prometheus/docs/blob/main/content/docs/instrumenting/exposition_formats.md#text-based-format) and its output is in the form of `[]byte` so it is more efficient to present stats data that will be used as-is in metrics or label values directly as `[]byte`.
e.g. [pid_status_parser.go](pid_status_parser.go#L99)
