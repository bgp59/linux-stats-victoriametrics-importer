#! /usr/bin/env python3

# Generate Markdown table with parser benchmark results.

import argparse
import os
import platform
import re
import sys

this_dir = os.path.dirname(os.path.abspath(__file__))
project_root_dir = os.path.dirname(this_dir)

# goos: darwin
# goarch: amd64
# pkg: github.com/bgp59/linux-stats-victoriametrics-importer/benchmarks
# cpu: Intel(R) Core(TM) i7-8750H CPU @ 2.20GHz
# BenchmarkSoftirqsParserIO-12      	   73254	     16590 ns/op	     136 B/op	       3 allocs/op
# BenchmarkSoftirqsParser-12        	   64801	     18180 ns/op	     200 B/op	      13 allocs/op
# BenchmarkSoftirqsParserProm-12    	   40520	     30109 ns/op	   14992 B/op	      42 allocs/op
# PASS
# ok  	github.com/bgp59/linux-stats-victoriametrics-importer/benchmarks	4.616s

BENCHMARK_NAME = 0
BENCHMARK_RUN_COUNT = 1
BENCHMARK_TIME_PER_OP = 2
BENCHMARK_TIME_PER_OP_UNIT = 3
BENCHMARK_BYTES_PER_OP = 4
BENCHMARK_BYTES_PER_OP_UNIT = 5
BENCHMARK_ALLOCS_PER_OP = 6
BENCHMARK_ALLOCS_PER_OP_UNIT = 7
BENCHMARK_NUM_FIELDS = 8

NUMERIC_FIELDS = [
    BENCHMARK_RUN_COUNT,
    BENCHMARK_TIME_PER_OP,
    BENCHMARK_BYTES_PER_OP,
    BENCHMARK_ALLOCS_PER_OP,
]
RESULT_FIELDS = [BENCHMARK_NAME] + NUMERIC_FIELDS

EXPECTED_TIME_PER_OP_UNIT = "ns/op"

CONDITION_FIELDS = {
    "goos",
    "goarch",
    "cpu",
}

DEFAULT_COLUMN_ALIGN = object()

CONDITIONS_HEADERS = ["Cond", "Value"]
CONDITIONS_COLUMN_ALIGN = DEFAULT_COLUMN_ALIGN

RESULTS_HEADERS = ["Parser", "Benchmark", "Run#", "ns/op", "Bytes/op", "Allocs/op"]
RESULTS_COLUMN_ALIGN = [":---", ":---", "---:", "---:", "---:", "---:"]


def markdown_table_row(row, column_align=None):
    md_row = "| " + " | ".join(row) + " |"
    if column_align is DEFAULT_COLUMN_ALIGN:
        column_align = ["---"] * len(row)
    if column_align is not None:
        md_row += "\n| " + " | ".join(column_align) + " |"
    return md_row


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("in_file")
    parser.add_argument(
        "out_file",
        default=os.path.join(
            project_root_dir, "docs", f"parser-bench-{platform.platform().lower()}.md"
        ),
        nargs="?",
    )
    args = parser.parse_args()
    in_file = args.in_file
    out_file = args.out_file

    conditions = {}
    results_by_parser = {}
    numeric_fields = set(NUMERIC_FIELDS)

    with open(in_file) as in_f:
        for line in in_f:
            words = line.split()
            if len(words) < 2:
                continue
            if words[0].endswith(":"):
                cond_key = words[0][:-1].lower()
                if cond_key in CONDITION_FIELDS:
                    conditions[cond_key.upper()] = " ".join(words[1:])
            elif (
                len(words) != BENCHMARK_NUM_FIELDS
                or words[BENCHMARK_TIME_PER_OP_UNIT] != EXPECTED_TIME_PER_OP_UNIT
            ):
                continue
            name = words[BENCHMARK_NAME]
            m = re.match(r"Benchmark(.*Parser)", name)
            if m is None:
                continue
            parser = m.group(1)
            result = [
                int(words[i]) if i in numeric_fields else words[i]
                for i in RESULT_FIELDS
            ]
            if parser not in results_by_parser:
                results_by_parser[parser] = [result]
            else:
                results_by_parser[parser].append(result)

    sort_key_index = RESULT_FIELDS.index(BENCHMARK_TIME_PER_OP)
    for parser in results_by_parser:
        results_by_parser[parser] = sorted(
            results_by_parser[parser], key=lambda r: r[sort_key_index]
        )

    if out_file == "-":
        out_f = sys.stdout
    else:
        out_f = open(out_file, "wt")

    print("# Parser Benchmark Results", file=out_f)
    print(file=out_f)

    print("## Conditions", file=out_f)
    print(file=out_f)
    print(
        markdown_table_row(CONDITIONS_HEADERS, column_align=CONDITIONS_COLUMN_ALIGN),
        file=out_f,
    )
    for cond in sorted(conditions):
        print(markdown_table_row([cond, conditions[cond]]), file=out_f)
    print(file=out_f)

    print("## Results", file=out_f)
    print(file=out_f)

    print(
        markdown_table_row(RESULTS_HEADERS, column_align=RESULTS_COLUMN_ALIGN),
        file=out_f,
    )
    for parser in sorted(results_by_parser):
        result_row = [parser] + [""] * len(RESULT_FIELDS)
        for result in results_by_parser[parser]:
            for i, col in enumerate(result):
                if result_row[i + 1]:
                    result_row[i + 1] += "<br>"
                result_row[i + 1] += str(col)
        print(markdown_table_row(result_row), file=out_f)
    print(file=out_f)

    print(
        """Notes:

  1. `IO` suffix designates the benchmark for reading the file into a buffer
  2. `Prom` suffix designates the benchmark for the official [prometheus/procfs](https://github.com/prometheus/procfs) parsers
  3. No suffix designates the benchmark for the custom parsers
""",
        file=out_f,
    )

    if out_file != "-":
        out_f.close()
        print(f"{out_file} created", file=sys.stderr)
