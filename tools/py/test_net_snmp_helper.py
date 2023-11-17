#! /usr/bin/env python3

"""
Generate the net/snmp file and print TestNetSnmpTestCase fields (Go sytnax) based
on captured proc/net/snmp

"""

import os
import sys

from testutils import (
    lsvmi_testdata_procfs_root,
    procfs_testdata_root,
)

reference_net_snmp_file = os.path.join(lsvmi_testdata_procfs_root, "net", "snmp")
output_net_snmp_file = os.path.join(procfs_testdata_root, "net", "snmp", "field_mapping", "net", "snmp")

net_snmp_signed_value_names = set([
    "TcpMaxConn"
])



if __name__ == '__main__':
    heading_by_line = []
    names_by_line = []
    values_by_line = []
    net_snmp_line_info = []
    with open(reference_net_snmp_file) as f:
        line_num = 0
        for line in f:
            line_num += 1
            if line_num % 2 == 1:
                heading_by_line.append(line.strip())
                words = line.split()
                prefix, stats = words[0], words[1:]
                proto = prefix[:-1]
                names = [
                    proto + stat
                    for stat in stats
                ]
                num_vals = len(names)
                values = [
                    1000*line_num + k
                    for k in range(num_vals)         
                ]
                for i in range(num_vals):
                    if names[i] in net_snmp_signed_value_names:
                        values[i] = -values[i]
                names_by_line.append(names)
                values_by_line.append(values)
                net_snmp_line_info.append((prefix, num_vals))
    
    os.makedirs(os.path.dirname(output_net_snmp_file), exist_ok=True)
    with open(output_net_snmp_file, "wt") as f:
        for i in range(len(heading_by_line)):
            print(heading_by_line[i], file=f)
            print(
                " ".join(map(str, 
                    [net_snmp_line_info[i][0]] + values_by_line[i],
                )),
                file=f
            )
    print(f"{output_net_snmp_file} generated", file=sys.stderr)
    print("Cut and paste the following into the appropriate NetSnmpTestCase\n", file=sys.stderr)

    indent_lvl = 1

    # primeNetSnmp start:
    print(
        "\t"*indent_lvl + "primeNetSnmp: &NetSnmp{"
    )

    # primeNetSnmp - Names:
    indent_lvl += 1
    print(
        "\t"*indent_lvl + "Names: []string{"
    )
    indent_lvl += 1
    for names in names_by_line:
        print(
            "\t"*indent_lvl +
            ", ".join(map(lambda s: f'"{s}"', names)) + ","
        )
    indent_lvl -= 1
    print(
        "\t"*indent_lvl + "},"
    )

    # primeNetSnmp - Values:
    print(
        "\t"*indent_lvl + "Values: make([]int64, " + "+".join(map(lambda v: str(len(v)),  values_by_line)) + "),"
    )

    # primeNetSnmp - lineInfo:   
    print(
        "\t"*indent_lvl + "lineInfo: []*NetSnmpLineInfo{"
    )
    indent_lvl += 1
    for line_info in net_snmp_line_info:
        print(
            "\t"*indent_lvl + 
            '{[]byte("' + line_info[0] + '"), ' + str(line_info[1]) + "},"
        )
    indent_lvl -= 1
    print(
        "\t"*indent_lvl + "},"
    )

    # primeNetSnmp end:
    indent_lvl -= 1
    print(
        "\t"*indent_lvl + "},"
    )

    # wantNetSnmp start:
    print(
        "\t"*indent_lvl + "wantNetSnmp: &NetSnmp{"
    )

    # wantNetSnmp - Names:
    indent_lvl += 1
    print(
        "\t"*indent_lvl + "Names: []string{"
    )
    indent_lvl += 1
    for names in names_by_line:
        print(
            "\t"*indent_lvl +
            ", ".join(map(lambda s: f'"{s}"', names)) + ","
        )
    indent_lvl -= 1
    print(
        "\t"*indent_lvl + "},"
    )

    # wantNetSnmp - Values:
    print(
          "\t"*indent_lvl + "Values: []int64{"
    )
    indent_lvl += 1
    for values in values_by_line:
        print(
            "\t"*indent_lvl + 
            ", ".join(map(str, values)) + ","
        )
    indent_lvl -= 1
    print(
        "\t"*indent_lvl + "}"
    )

    # wantNetSnmp end:
    indent_lvl -= 1
    print(
        "\t"*indent_lvl + "},"
    )

    print()

