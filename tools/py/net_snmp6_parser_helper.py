#! /usr/bin/env python3

"""
Generate definitions for net_snmp6_parser.go

"""

import argparse
import os
import re

from testutils import go_module_root, lsvmi_testdata_procfs_root

default_net_snmp6_file = os.path.join(lsvmi_testdata_procfs_root, "net", "snmp6")

index_prefix = "NET_SNMP6_"

map_variable_name = "netSnmp6IndexMap"

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("net_snmp6_file", default=default_net_snmp6_file, nargs="?")
    args = parser.parse_args()

    proto_vars_list = []

    with open(args.net_snmp6_file) as f:
        variables = [
            line.split()[0]
            for line in f
        ]

    # Index list and map:
    index_list = []
    index_map = {}
    for var in variables:
        # UDPLite -> UDPlite:
        index = re.sub('Lite', 'lite', var)
        # CamelCase -> CAMEL_CASE:
        index = re.sub(r"([a-z6])([A-Z])", r"\1_\2", index).upper()
        # Known exceptions:
        index = re.sub(r"(ECT\d*|CE)(PKTS)", r"\1_\2", index)
        index = re.sub(r"(MLDV2REPORTS)", r"MLD_V2_REPORTS", index)
        # Prefix:
        index = index_prefix + index
        index_list.append(index)
        index_map[var] = index

    this_file_rel_path = os.path.relpath(os.path.abspath(__file__), go_module_root)
    net_snmp6_file_rel_path = os.path.relpath(
        os.path.abspath(args.net_snmp6_file), go_module_root
    )

    print(
        "\n"
        + "// Begin of automatically generated content:\n"
        + f"//  Script: {this_file_rel_path}\n"
        + f"//  Reference file: {net_snmp6_file_rel_path}\n"
    )

    print("// Index definitions for parsed values:")
    print("const (")
    needs_iota = True
    for index in index_list:
        if needs_iota:
            print(f"\t{index} = iota")
            needs_iota = False
        else:
            print(f"\t{index}")
    print(
        "\n"
        + "\t// Must be last:\n"
        + f"\t{index_prefix}NUM_VALUES"
    )
    print(")")
    print()

    print("// Map net/snmp6 VARIABLE into parsed value index: ")
    print(f"var {map_variable_name} = map[string]int {{")
    for var in variables:
        print(f'\t"{var}": {index_map[var]},')
    print("}")
    print()

    print("// End of automatically generated content.\n")
