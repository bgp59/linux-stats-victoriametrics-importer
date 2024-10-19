#! /usr/bin/env python3

# Run benchmark(s) for parsers and consolidate the results:

import argparse
import re
import subprocess
import sys

from tabulate import tabulate
from testutils import benchmarks_root_dir

# Indexes in result:
# BenchmarkDiskstatsParserIO   	   68028	     17242 ns/op	     152 B/op	       3 allocs/op
# BenchmarkDiskstatsParser     	   56817	     21376 ns/op	     336 B/op	      38 allocs/op
# BenchmarkDiskstatsParserProm 	   10000	    103585 ns/op	   14744 B/op	     176 allocs/op
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

RESULT_HEADERS = ["Name", "Run#", "ns/op", "Bytes/op", "Allocs/op"]

EXPECTED_TIME_PER_OP_UNIT = "ns/op"


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("parser", nargs="*", help="""Parser, e.g. StatParser""")
    args = parser.parse_args()

    results_by_parser = {}
    numeric_fields = set(NUMERIC_FIELDS)
    sort_key_index = RESULT_FIELDS.index(BENCHMARK_TIME_PER_OP)
    for parser in args.parser or [".*"]:
        p = subprocess.run(
            [
                "go",
                "test",
                "-benchmem",
                "-cpu",
                "1",
                "-bench",
                f"^Benchmark{parser}Parser",
            ],
            cwd=benchmarks_root_dir,
            check=False,
            capture_output=True,
            encoding="utf-8",
        )
        if p.returncode != 0:
            print(p.stdout)
            print(p.stderr, file=sys.stderr)
            print(f"exit code {p.returncode}", file=sys.stderr)
            sys.exit(p.returncode)

        for line in p.stdout.splitlines():
            words = line.split()
            if (
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

    for parser in sorted(results_by_parser):
        rows = sorted(results_by_parser[parser], key=lambda r: r[sort_key_index])
        print(f"{parser}:")
        print()
        print(tabulate(rows, headers=RESULT_HEADERS))
        print()
