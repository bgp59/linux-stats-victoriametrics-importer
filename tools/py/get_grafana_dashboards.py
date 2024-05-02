#! /usr/bin/env python3

"""
Retreive Grafana LSVMI dashboards for safekeeping
"""

import argparse
import json
import os
import re
import pprint
import sys

import requests


this_dir = os.path.dirname(os.path.abspath(__file__))
root_dir = os.path.dirname(os.path.dirname(this_dir))

default_grafana_root_url = "http://localhost:3000"
default_grafana_user = "admin"
default_grafana_password = "lsvmi"
default_grafana_folder = None
default_grafana_out_dir = os.path.join('grafana', 'dashboards')


def normalize_title(title: str) -> str:
    # camelCaseTitle -> camel_case_title
    normal_title = re.sub(r'([a-z])([A-Z])', r'\1_\2', title).lower()
    # non-standard chars -> _
    normal_title = re.sub(r'[^a-z_0-9]+', '_', normal_title)
    return normal_title
  

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "-r", "--root-url",
        default=default_grafana_root_url,
        help="""Grafana root URL, default: %(default)r."""
    )
    parser.add_argument(
        "-u", "--user",
        default=default_grafana_user,
        help="""Grafana user, default: %(default)r."""
    )
    parser.add_argument(
        "-p", "--password",
        default=default_grafana_password,
        help="""Grafana password, default: %(default)r."""
    )
    parser.add_argument(
        "-f", "--folder",
        default=default_grafana_folder,
        help="""Grafana folder, if not specified select all folders."""
    )
    parser.add_argument(
        "-t", "--title",
        action="append",
        help="""Grafana dashboard title(s), if not specified select select all dashboards."""
    )
    parser.add_argument(
        "-o", "--out-dir",
        default=default_grafana_out_dir,
        help="""Out dir, use `-' for stdout, default: %(default)r. A relative
             path is relative to the root location of the project."""
    )
    parser.add_argument(
        "-N", "--normalize-uid",
        action="store_true",
        help="""Normalize UID based on title."""
    )

    args = parser.parse_args()
    root_url = args.root_url
    auth = (args.user, args.password)
    folder = args.folder
    dashboard_titles = None if not args.title else set(args.title)

    r = requests.get(f'{root_url}/api/search?type=dash-db', auth=auth)
    r.raise_for_status()

    uid_list = [
        d.get("uid") for d in r.json()
        if (
            (dashboard_titles is None or d.get("title") in dashboard_titles)
            and 
            (folder is None or d.get("folderTitle", "") == folder)
        )
    ]
    
    out_dir = None if args.out_dir == "-" else args.out_dir
    if out_dir is not None:
        out_dir = os.path.join(root_dir, out_dir)
    for uid in uid_list:
        r = requests.get(f'{root_url}/api/dashboards/uid/{uid}', auth=auth)
        r.raise_for_status()
        dashboard_meta = r.json()
        dashboard, meta = dashboard_meta["dashboard"], dashboard_meta["meta"]
        norm_title, norm_folder  = normalize_title(dashboard["title"]), normalize_title(meta["folderTitle"])
        if out_dir is not None:
            fname = os.path.join(out_dir, norm_folder, norm_title + ".json")
            os.makedirs(os.path.dirname(fname), exist_ok=True)
            ofile = open(fname, "wt")
        else:
            ofile = sys.stdout
        if args.normalize_uid:
            org_uid, new_uid = dashboard["uid"], norm_title # f"{norm_folder}_{norm_title}"
            if org_uid != new_uid:
                dashboard["uid"] = new_uid
                print(
                    f'dashboard: {dashboard["title"]!r} folder: {meta["folderTitle"]!r}: uid: {org_uid!r} -> {new_uid!r}', file=sys.stderr)
        json.dump(dashboard, ofile, indent=2)
        print("", file=ofile)
        if out_dir is not None:
            ofile.close()
            print(f"{fname} created", file=sys.stderr)
