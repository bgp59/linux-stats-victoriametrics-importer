#! /usr/bin/env python3

"""
Generate definitions for net_snmp_parser.go

"""

import argparse
import os
import re

from testutils import go_module_root, procfs_proc_root_dir

default_net_snmp_file = os.path.join(procfs_proc_root_dir, "net", "snmp")

index_prefix = "NET_SNMP_"
map_variable_name = "netSnmpIndexMap"

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("net_snmp_file", default=default_net_snmp_file, nargs="?")
    args = parser.parse_args()

    proto_vars_list = []

    with open(args.net_snmp_file) as f:
        is_definition = True
        for line in f:
            if is_definition:
                words = line.split()
                words[0] = words[0][:-1]  # drop `:'
                proto_vars_list.append(words)
            is_definition = not is_definition

    # Index list and map:
    proto_list = []
    index_list = []
    index_map_by_proto = {}
    for proto_vars in proto_vars_list:
        proto, vars = proto_vars[0], proto_vars[1:]
        proto_list.append(proto)
        index_map_by_proto[proto] = []
        for var in vars:
            index = (
                index_prefix + proto + "_" + re.sub(r"([a-z])([A-Z])", r"\1_\2", var)
            ).upper()
            index_list.append(index)
            index_map_by_proto[proto].append((var, index))

    this_file_rel_path = os.path.relpath(os.path.abspath(__file__), go_module_root)
    net_snmp_file_rel_path = os.path.relpath(
        os.path.abspath(args.net_snmp_file), go_module_root
    )

    print(
        "\n"
        + "// Begin of automatically generated content:\n"
        + f"//  Script: {this_file_rel_path}\n"
        + f"//  Reference file: {net_snmp_file_rel_path}\n"
    )

    print("// Index definitions for parsed values:")
    print("const (")
    needs_iota = True
    for index in index_list:
        if needs_iota:
            print(f"  {index} = iota")
            needs_iota = False
        else:
            print(f"  {index}")
    print("\n" + "  // Must be last:\n" + f"  {index_prefix}NUM_VALUES")
    print(")")
    print()

    print("// Map net/snmp [PROTO][VARIABLE] pairs into parsed value indexes: ")
    print(f"var {map_variable_name} = map[string]map[string]int {{")
    for proto in proto_list:
        print(f'  "{proto}": {{')
        for var, index in index_map_by_proto[proto]:
            print(f'    "{var}": {index},')
        print("  },")
    print("}")
    print()

    print("// End of automatically generated content.\n")
