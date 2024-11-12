#! /usr/bin/env python3

"""
A naive tool for building metrics TOC style metrics list out of *_metrics.md files

Marko (https://marko-py.readthedocs.io/en/latest/) seems like an overkill for now.
"""

import argparse
import os
import re
import sys
from typing import List, Tuple

this_dir = os.path.dirname(os.path.abspath(__file__))
root_dir = os.path.dirname(os.path.dirname(this_dir))
docs_dir = os.path.join(root_dir, "docs")

MD_INDENT = " " * 2


def build_default_file_list(md_dir: str = docs_dir) -> List[str]:
    flist = []
    for fname in os.listdir(md_dir):
        if not re.search(r"""_metrics.md$""", fname):
            continue
        fpath = os.path.join(md_dir, fname)
        if os.path.isfile(fpath):
            flist.append(fpath)
            continue
    return sorted(flist)


def extract_title_metrics_toc(fpath: str) -> Tuple[str, List[str]]:
    in_toc = False
    fname = os.path.basename(fpath)
    title, metrics_toc = None, []
    with open(fpath) as f:
        for line in f:
            line = line.strip()
            m = re.match(r"""#\s+""", line)
            if m:
                title = f"[{line[m.end():]}]({fname})"
                continue
            if re.match(r"""\<!--\s*TOC""", line):
                in_toc = True
                continue
            if re.match(r"""\<!--\s*/TOC""", line):
                break
            if in_toc and re.match(
                r"""-\s+\[(?P<metric>[^\]]*)\]\(#(?P=metric)\)""", line
            ):
                line = re.sub(r"""\(#""", f"({fname}#", line)
                metrics_toc.append(line)
    return title, metrics_toc


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "-o",
        "--out-dir",
        default=docs_dir,
        help="""Output dir to create the .md files, use - to print to stdout.
             Default: %(default)s""",
    )
    parser.add_argument(
        "md_files",
        nargs="*",
        help=f"""Metrics .md files, default to {docs_dir}*_metrics.md""",
    )
    args = parser.parse_args()
    out_dir = args.out_dir
    md_files = args.md_files or build_default_file_list()

    warning = f"""
<!--
Do NOT edit this file by hand, it was automatically generated by
    {os.path.relpath(os.path.abspath(__file__), root_dir)}
from:
"""
    for fpath in md_files:
        warning += f"    {os.path.relpath(os.path.abspath(fpath), root_dir)}\n"
    warning += "-->\n"

    if out_dir != "-":
        metrics_by_gen_file = os.path.join(out_dir, "metrics_by_generator.md")
        of = open(metrics_by_gen_file, "wt")
    else:
        metrics_by_gen_file = None
        of = sys.stdout

    print("# All Metrics By Generator", file=of)
    print(warning, file=of)

    all_metrics = []
    for fpath in md_files:
        title, metrics = extract_title_metrics_toc(fpath)
        all_metrics.extend(metrics)
        print(f"- {title}", file=of)
        for metric in metrics:
            print(f"{MD_INDENT}{metric}", file=of)

    if metrics_by_gen_file is not None:
        of.close()
        print(f"{metrics_by_gen_file} created", file=sys.stderr)

    if out_dir != "-":
        metrics_alpha_file = os.path.join(out_dir, "metrics_alphabetically.md")
        of = open(metrics_alpha_file, "wt")
    else:
        metrics_alpha_file = None
        of = sys.stdout

    print("# All Metrics In Alphabetical Order", file=of)
    print(warning, file=of)
    for metric in sorted(all_metrics, key=lambda s: s.lower()):
        print(metric, file=of)
    if metrics_alpha_file is not None:
        of.close()
        print(f"{metrics_alpha_file} created", file=sys.stderr)
