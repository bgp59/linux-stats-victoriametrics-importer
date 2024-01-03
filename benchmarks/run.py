#! /usr/bin/env python3

import argparse
import os
import sys
import subprocess

this_dir = os.path.dirname(os.path.abspath(__file__))

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument("benchmark", nargs=1, help="""Benchamrk pattern""")
    args = parser.parse_args()

    p = subprocess.run(
        ['go', 'test', '-benchmem', '-cpu', '1', '-bench', args.benchmark[0]],
        cwd=this_dir,
        check=True,
        capture_output=True,
        encoding='utf-8',
    )
    output, results_by_ns_per_op = [], {}
    for line in p.stdout.splitlines():
        words = line.split()
        if len(words) > 3 and words[3] == 'ns/op':
            results_by_ns_per_op[int(words[2])] = line
            continue
        if results_by_ns_per_op:
            output.extend(
                results_by_ns_per_op[k] for k in sorted(results_by_ns_per_op)
            )
            results_by_ns_per_op = None
        output.append(line)
    for line in output:
        print("// " + line)
