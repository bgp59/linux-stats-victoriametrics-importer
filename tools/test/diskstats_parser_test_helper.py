#! /usr/bin/env python3

"""
Generate the net/snmp file and print DiskstatsTestCase fields (Go syntax)

"""

import os
import sys

from testutils import procfs_testdata_root

output_diskstats_file = os.path.join(
    procfs_testdata_root, "diskstats", "field_mapping", "diskstats"
)

DISKSTATS_VALUE_FIELDS_NUM = 17

diskstats_num_devices = 2

if __name__ == "__main__":
    dev_info_map = {}
    for k in range(diskstats_num_devices):
        major, minor = k, k
        dev_info_map[(major, minor)] = {
            "name": f"disk{k}",
            "stats": [
                (k + 1) * 1000 + i + 1 for i in range(DISKSTATS_VALUE_FIELDS_NUM)
            ],
        }
    os.makedirs(os.path.dirname(output_diskstats_file), exist_ok=True)
    with open(output_diskstats_file, "wt") as f:
        for (major, minor), dev_info in dev_info_map.items():
            print(f"{major:8d}", f"{minor:8d}", dev_info["name"], file=f, end=" ")
            print(" ".join(map(str, dev_info["stats"])), file=f)
    print(f"{output_diskstats_file} generated", file=sys.stderr)

    print(
        "Cut and paste the following into the appropriate DiskstatsTestCase\n",
        file=sys.stderr,
    )

    indent_lvl = 1

    # primeDiskstats start:
    scan_num = 42

    print("\t" * indent_lvl + "primeDiskstats: &Diskstats{")
    indent_lvl += 1

    # primeDiskstats - DiskstatsDevInfo:
    print("\t" * indent_lvl + "DevInfoMap: map[string]*DiskstatsDevInfo{")
    indent_lvl += 1
    for (major, minor), dev_info in dev_info_map.items():
        print("\t" * indent_lvl + f'"{major}:{minor}": &DiskstatsDevInfo{{')
        indent_lvl += 1
        print("\t" * indent_lvl + f'Name: "{dev_info["name"]}",')
        print(
            "\t" * indent_lvl + f"Stats: make([]uint32, {DISKSTATS_VALUE_FIELDS_NUM}),"
        )
        print("\t" * indent_lvl + f"scanNum: {scan_num},")
        indent_lvl -= 1
        print("\t" * indent_lvl + "},")
    indent_lvl -= 1
    print("\t" * indent_lvl + "},")

    # primeDiskstats - scanNum:
    print("\t" * indent_lvl + f"scanNum: {scan_num},")

    # primeDiskstats - jiffies handling:
    print("\t" * indent_lvl + "jiffiesToMillisec: 0,")
    print("\t" * indent_lvl + "fieldsInJiffies: diskstatsFieldsInJiffies,")

    # primeDiskstats end:
    indent_lvl -= 1
    print("\t" * indent_lvl + "},")

    # wantDiskstats start:
    print("\t" * indent_lvl + "wantDiskstats: &Diskstats{")
    indent_lvl += 1

    # wantDiskstats - DiskstatsDevInfo:
    print("\t" * indent_lvl + "DevInfoMap: map[string]*DiskstatsDevInfo{")
    indent_lvl += 1
    for (major, minor), dev_info in dev_info_map.items():
        print("\t" * indent_lvl + f'"{major}:{minor}": &DiskstatsDevInfo{{')
        indent_lvl += 1
        print("\t" * indent_lvl + f'Name: "{dev_info["name"]}",')
        print(
            "\t" * indent_lvl
            + "Stats: []uint32{"
            + ", ".join(map(str, dev_info["stats"]))
            + "},"
        )
        indent_lvl -= 1
        print("\t" * indent_lvl + "},")
    indent_lvl -= 1
    print("\t" * indent_lvl + "},")

    # wantDiskstats end:
    indent_lvl -= 1
    print("\t" * indent_lvl + "},")
