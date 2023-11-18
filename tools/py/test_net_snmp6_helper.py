#! /usr/bin/env python3

"""
Generate the net/snmp file and print TestNetSnmpTestCase fields (Go sytnax) based
on captured proc/net/snmp

"""

import os
import sys

from testutils import lsvmi_testdata_procfs_root, procfs_testdata_root

reference_net_snmp6_file = os.path.join(lsvmi_testdata_procfs_root, "net", "snmp6")
output_net_snmp6_file = os.path.join(
    procfs_testdata_root, "net", "snmp6", "field_mapping", "net", "snmp6"
)

NET_SNMP6_NAME_CHECK_SEP = " "  # must match net_snmp6_parser.go
values_per_line = 4

if __name__ == "__main__":
    names = []
    values = []
    name_check_ref = []
    with open(reference_net_snmp6_file, "rt") as f:
        v = 1_000_0000_000_000
        for line in f:
            name, value = line.split()
            names.append(name)
            v += 1
            values.append(v)
    os.makedirs(os.path.dirname(output_net_snmp6_file), exist_ok=True)
    with open(output_net_snmp6_file, "wt") as f:
        for name, value in zip(names, values):
            print(f"{name:32s} {value}", file=f)
    print(f"{output_net_snmp6_file} generated", file=sys.stderr)
    print(
        "Cut and paste the following into the appropriate NetSnmp6TestCase\n",
        file=sys.stderr,
    )

    indent_lvl = 1

    # primeNetSnmp6 start:
    print("\t" * indent_lvl + "primeNetSnmp6: &NetSnmp6{")
    indent_lvl += 1

    # primeNetSnmp6 - Names:
    print("\t" * indent_lvl + "Names: []string{")
    indent_lvl += 1
    for k in range(0, len(names), values_per_line):
        print(
            "\t" * indent_lvl
            + ", ".join(map(lambda s: f'"{s}"', names[k : k + values_per_line]))
            + ","
        )
    indent_lvl -= 1
    print("\t" * indent_lvl + "},")

    # primeNetSnmp6 - Values:
    print("\t" * indent_lvl + f"Values: make([]uint64, {len(values)}),")

    # primeNetSnmp6 - nameCheckRef:
    print("\t" * indent_lvl + "nameCheckRef: []byte(")
    indent_lvl += 1

    for k in range(0, len(names), values_per_line):
        if k > 0:
            print(" +")
        print(
            "\t" * indent_lvl
            + '"'
            + NET_SNMP6_NAME_CHECK_SEP.join(names[k : k + values_per_line])
            + NET_SNMP6_NAME_CHECK_SEP
            + '"',
            end="",
        )
    indent_lvl -= 1
    print("),")

    # primeNetSnmp6 end:
    indent_lvl -= 1
    print("\t" * indent_lvl + "},")

    # wantNetSnmp6 start:
    print("\t" * indent_lvl + "wantNetSnmp6: &NetSnmp6{")
    indent_lvl += 1

    # primeNetSnmp6 - Names:
    print("\t" * indent_lvl + "Names: []string{")
    indent_lvl += 1
    for k in range(0, len(names), values_per_line):
        print(
            "\t" * indent_lvl
            + ", ".join(map(lambda s: f'"{s}"', names[k : k + values_per_line]))
            + ","
        )
    indent_lvl -= 1
    print("\t" * indent_lvl + "},")

    # primeNetSnmp6 - Values:
    print("\t" * indent_lvl + "Values: []uint64{")
    indent_lvl += 1
    for k in range(0, len(names), values_per_line):
        print(
            "\t" * indent_lvl
            + ", ".join(map(str, values[k : k + values_per_line]))
            + ","
        )
    indent_lvl -= 1
    print("\t" * indent_lvl + "},")

    # wantNetSnmp6 end:
    indent_lvl -= 1
    print("\t" * indent_lvl + "},")

    print()
