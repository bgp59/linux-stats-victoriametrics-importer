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

DISKSTATS_NUM_FIELDS = 20
DISKSTATS_DEVICE_FIELD_NUM = 2

diskstats_num_devices = 2

if __name__ == "__main__":
    dev_stats = {}
    for dev_num in range(diskstats_num_devices):
        dev = f"disk{dev_num}"
        dev_stats[dev] = [
            (dev_num + 1) * 1000 + i if i != DISKSTATS_DEVICE_FIELD_NUM else 0
            for i in range(DISKSTATS_NUM_FIELDS)
        ]
    os.makedirs(os.path.dirname(output_diskstats_file), exist_ok=True)
    with open(output_diskstats_file, "wt") as f:
        for dev, stats in dev_stats.items():
            for i in range(DISKSTATS_DEVICE_FIELD_NUM):
                print(f"{stats[i]:8d} ", file=f, end="")
            print(f"{dev} ", file=f, end="")
            print(" ".join(map(str, stats[DISKSTATS_DEVICE_FIELD_NUM + 1 :])), file=f)
    print(f"{output_diskstats_file} generated", file=sys.stderr)

    print(
        "Cut and paste the following into the appropriate DiskstatsTestCase\n",
        file=sys.stderr,
    )

    indent_lvl = 1

    # primeDiskstats start:
    print("\t" * indent_lvl + "primeDiskstats: &Diskstats{")
    indent_lvl += 1

    # primeDiskstats - DevStats:
    print("\t" * indent_lvl + "DevStats: map[string][]uint32{")
    indent_lvl += 1
    for dev in dev_stats:
        print("\t" * indent_lvl + f'"{dev}": make([]uint32, {DISKSTATS_NUM_FIELDS}),')
    indent_lvl -= 1
    print("\t" * indent_lvl + "},")

    # primeDiskstats - devScanNum:
    scan_num = 42
    print("\t" * indent_lvl + "devScanNum: map[string]int{")
    indent_lvl += 1
    for dev in dev_stats:
        print("\t" * indent_lvl + f'"{dev}": {scan_num},')
    indent_lvl -= 1
    print("\t" * indent_lvl + "},")
    print("\t" * indent_lvl + f"scanNum: {scan_num},")

    # primeDiskstats - jiffies handling:
    print("\t" * indent_lvl + "jiffiesToMillisec: 0,")
    print("\t" * indent_lvl + "fieldsInJiffies: diskstatsFieldsInJiffies,")

    # primeDiskstats end:
    indent_lvl -= 1
    print("\t" * indent_lvl + "},")

    # wantDiskstats:
    print("\t" * indent_lvl + "wantDiskstats: &Diskstats{")
    indent_lvl += 1

    # wantDiskstats - DevStats:
    print("\t" * indent_lvl + "DevStats: map[string][]uint32{")
    indent_lvl += 1
    for dev, stats in dev_stats.items():
        print(
            "\t" * indent_lvl
            + f'"{dev}": '
            + "[]uint32{"
            + ", ".join(map(str, stats))
            + "},"
        )
    indent_lvl -= 1
    print("\t" * indent_lvl + "},")

    # wantDiskstats end:
    indent_lvl -= 1
    print("\t" * indent_lvl + "},")
