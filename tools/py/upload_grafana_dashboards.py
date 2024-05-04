#! /usr/bin/env python3

"""
Upload Grafana LSVMI dashboards for safekeeping
"""

import argparse
import json
import os
import sys

import requests

this_dir = os.path.dirname(os.path.abspath(__file__))
root_dir = os.path.dirname(os.path.dirname(this_dir))

default_grafana_root_url = "http://localhost:3000"
default_grafana_user = "admin"
default_grafana_password = "lsvmi"

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "-r",
        "--root-url",
        default=default_grafana_root_url,
        help="""Grafana root URL, default: %(default)r.""",
    )
    parser.add_argument(
        "-u",
        "--user",
        default=default_grafana_user,
        help="""Grafana user, default: %(default)r.""",
    )
    parser.add_argument(
        "-p",
        "--password",
        default=default_grafana_password,
        help="""Grafana password, default: %(default)r.""",
    )
    parser.add_argument(
        "-f",
        "--folder",
        help="""Grafana folder""",
    )
    parser.add_argument(
        "dashfile",
        nargs="+",
    )
    args = parser.parse_args()
    root_url = args.root_url
    auth = (args.user, args.password)
    folder = args.folder

    folder_uid = None
    if folder is not None:
        r = requests.get(f"{root_url}/api/folders", auth=auth)
        r.raise_for_status()
        for f in r.json():
            if f.get("title") == folder:
                folder_uid = f.get("uid")
                break
        if folder_uid is None:
            raise RuntimeError(f"Cannot find {folder} folder")

    for dashfile in args.dashfile:
        with open(dashfile, "rt") as f:
            dashboard = json.load(f)
        post_data = {
            "dashboard": dashboard,
            "message": f"Uploaded from {dashfile}",
            "overwrite": True,
        }
        if folder_uid is not None:
            post_data["folderUid"] = folder_uid
        r = requests.post(f"{root_url}/api/dashboards/db", json=post_data, auth=auth)
        r.raise_for_status()
        print(
            f'Dashboard {dashboard.get("title")!r} uploaded to {folder if folder else "General"!r} folder',
            file=sys.stderr,
        )
