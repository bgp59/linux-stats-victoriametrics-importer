package procfs

import (
	"bytes"
	"fmt"
	"path"
	"testing"
)

type NetSnmp6TestCase struct {
	name          string
	procfsRoot    string
	primeNetSnmp6 *NetSnmp6
	wantNetSnmp6  *NetSnmp6
	wantError     error
}

var netSnmp6TestdataDir = path.Join(PROCFS_TESTDATA_ROOT, "net", "snmp6")

func testNetSnmp6Parser(tc *NetSnmp6TestCase, t *testing.T) {
	var netSnmp6 *NetSnmp6

	wantNetSnmp6 := tc.wantNetSnmp6

	// Sanity check:
	if len(wantNetSnmp6.Names) != len(wantNetSnmp6.Values) {
		t.Fatalf(
			"len(wantNetSnmp6.Names): %d != %d len(wantNetSnmp6.Values)",
			len(wantNetSnmp6.Names), len(wantNetSnmp6.Values),
		)
	}

	if tc.primeNetSnmp6 != nil {
		netSnmp6 = tc.primeNetSnmp6.Clone(true)
		if netSnmp6.path == "" {
			netSnmp6.path = path.Join(tc.procfsRoot, "net", "snmp6")
		}
	} else {
		netSnmp6 = NewNetSnmp6(tc.procfsRoot)
	}

	err := netSnmp6.Parse()
	if tc.wantError != nil {
		if err == nil || tc.wantError.Error() != err.Error() {
			t.Fatalf("want: %v error, got: %v", tc.wantError, err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}

	diffBuf := &bytes.Buffer{}

	if len(wantNetSnmp6.Names) != len(netSnmp6.Names) {
		fmt.Fprintf(
			diffBuf,
			"\nlen(Names): want: %d, got: %d",
			len(wantNetSnmp6.Names), len(netSnmp6.Names),
		)
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}
	for i, wantName := range wantNetSnmp6.Names {
		gotName := netSnmp6.Names[i]
		if wantName != gotName {
			fmt.Fprintf(
				diffBuf,
				"\nNames[%d]: want: %q, got: %q",
				i, wantName, gotName,
			)
		}
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}

	if len(wantNetSnmp6.Values) != len(netSnmp6.Values) {
		fmt.Fprintf(
			diffBuf,
			"\nlen(Values): want: %d, got: %d",
			len(wantNetSnmp6.Values), len(netSnmp6.Values),
		)
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}

	for i, wantValue := range wantNetSnmp6.Values {
		gotValue := netSnmp6.Values[i]
		if wantValue != gotValue {
			fmt.Fprintf(
				diffBuf,
				"\nValues[i]: want: %d, got: %d",
				wantValue, gotValue,
			)
		}
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}
}

func TestNetSnmp6Parser(t *testing.T) {
	for _, tc := range []*NetSnmp6TestCase{
		&NetSnmp6TestCase{
			procfsRoot: path.Join(netSnmp6TestdataDir, "field_mapping"),
			wantNetSnmp6: &NetSnmp6{
				Names: []string{
					"Ip6InReceives", "Ip6InHdrErrors", "Ip6InTooBigErrors", "Ip6InNoRoutes",
					"Ip6InAddrErrors", "Ip6InUnknownProtos", "Ip6InTruncatedPkts", "Ip6InDiscards",
					"Ip6InDelivers", "Ip6OutForwDatagrams", "Ip6OutRequests", "Ip6OutDiscards",
					"Ip6OutNoRoutes", "Ip6ReasmTimeout", "Ip6ReasmReqds", "Ip6ReasmOKs",
					"Ip6ReasmFails", "Ip6FragOKs", "Ip6FragFails", "Ip6FragCreates",
					"Ip6InMcastPkts", "Ip6OutMcastPkts", "Ip6InOctets", "Ip6OutOctets",
					"Ip6InMcastOctets", "Ip6OutMcastOctets", "Ip6InBcastOctets", "Ip6OutBcastOctets",
					"Ip6InNoECTPkts", "Ip6InECT1Pkts", "Ip6InECT0Pkts", "Ip6InCEPkts",
					"Icmp6InMsgs", "Icmp6InErrors", "Icmp6OutMsgs", "Icmp6OutErrors",
					"Icmp6InCsumErrors", "Icmp6InDestUnreachs", "Icmp6InPktTooBigs", "Icmp6InTimeExcds",
					"Icmp6InParmProblems", "Icmp6InEchos", "Icmp6InEchoReplies", "Icmp6InGroupMembQueries",
					"Icmp6InGroupMembResponses", "Icmp6InGroupMembReductions", "Icmp6InRouterSolicits", "Icmp6InRouterAdvertisements",
					"Icmp6InNeighborSolicits", "Icmp6InNeighborAdvertisements", "Icmp6InRedirects", "Icmp6InMLDv2Reports",
					"Icmp6OutDestUnreachs", "Icmp6OutPktTooBigs", "Icmp6OutTimeExcds", "Icmp6OutParmProblems",
					"Icmp6OutEchos", "Icmp6OutEchoReplies", "Icmp6OutGroupMembQueries", "Icmp6OutGroupMembResponses",
					"Icmp6OutGroupMembReductions", "Icmp6OutRouterSolicits", "Icmp6OutRouterAdvertisements", "Icmp6OutNeighborSolicits",
					"Icmp6OutNeighborAdvertisements", "Icmp6OutRedirects", "Icmp6OutMLDv2Reports", "Icmp6OutType133",
					"Icmp6OutType135", "Icmp6OutType143", "Udp6InDatagrams", "Udp6NoPorts",
					"Udp6InErrors", "Udp6OutDatagrams", "Udp6RcvbufErrors", "Udp6SndbufErrors",
					"Udp6InCsumErrors", "Udp6IgnoredMulti", "Udp6MemErrors", "UdpLite6InDatagrams",
					"UdpLite6NoPorts", "UdpLite6InErrors", "UdpLite6OutDatagrams", "UdpLite6RcvbufErrors",
					"UdpLite6SndbufErrors", "UdpLite6InCsumErrors", "UdpLite6MemErrors",
				},
				Values: []uint64{
					10000000000001, 10000000000002, 10000000000003, 10000000000004,
					10000000000005, 10000000000006, 10000000000007, 10000000000008,
					10000000000009, 10000000000010, 10000000000011, 10000000000012,
					10000000000013, 10000000000014, 10000000000015, 10000000000016,
					10000000000017, 10000000000018, 10000000000019, 10000000000020,
					10000000000021, 10000000000022, 10000000000023, 10000000000024,
					10000000000025, 10000000000026, 10000000000027, 10000000000028,
					10000000000029, 10000000000030, 10000000000031, 10000000000032,
					10000000000033, 10000000000034, 10000000000035, 10000000000036,
					10000000000037, 10000000000038, 10000000000039, 10000000000040,
					10000000000041, 10000000000042, 10000000000043, 10000000000044,
					10000000000045, 10000000000046, 10000000000047, 10000000000048,
					10000000000049, 10000000000050, 10000000000051, 10000000000052,
					10000000000053, 10000000000054, 10000000000055, 10000000000056,
					10000000000057, 10000000000058, 10000000000059, 10000000000060,
					10000000000061, 10000000000062, 10000000000063, 10000000000064,
					10000000000065, 10000000000066, 10000000000067, 10000000000068,
					10000000000069, 10000000000070, 10000000000071, 10000000000072,
					10000000000073, 10000000000074, 10000000000075, 10000000000076,
					10000000000077, 10000000000078, 10000000000079, 10000000000080,
					10000000000081, 10000000000082, 10000000000083, 10000000000084,
					10000000000085, 10000000000086, 10000000000087,
				},
			},
		},
		&NetSnmp6TestCase{
			name:       "reuse",
			procfsRoot: path.Join(netSnmp6TestdataDir, "field_mapping"),
			primeNetSnmp6: &NetSnmp6{
				Names: []string{
					"Ip6InReceives", "Ip6InHdrErrors", "Ip6InTooBigErrors", "Ip6InNoRoutes",
					"Ip6InAddrErrors", "Ip6InUnknownProtos", "Ip6InTruncatedPkts", "Ip6InDiscards",
					"Ip6InDelivers", "Ip6OutForwDatagrams", "Ip6OutRequests", "Ip6OutDiscards",
					"Ip6OutNoRoutes", "Ip6ReasmTimeout", "Ip6ReasmReqds", "Ip6ReasmOKs",
					"Ip6ReasmFails", "Ip6FragOKs", "Ip6FragFails", "Ip6FragCreates",
					"Ip6InMcastPkts", "Ip6OutMcastPkts", "Ip6InOctets", "Ip6OutOctets",
					"Ip6InMcastOctets", "Ip6OutMcastOctets", "Ip6InBcastOctets", "Ip6OutBcastOctets",
					"Ip6InNoECTPkts", "Ip6InECT1Pkts", "Ip6InECT0Pkts", "Ip6InCEPkts",
					"Icmp6InMsgs", "Icmp6InErrors", "Icmp6OutMsgs", "Icmp6OutErrors",
					"Icmp6InCsumErrors", "Icmp6InDestUnreachs", "Icmp6InPktTooBigs", "Icmp6InTimeExcds",
					"Icmp6InParmProblems", "Icmp6InEchos", "Icmp6InEchoReplies", "Icmp6InGroupMembQueries",
					"Icmp6InGroupMembResponses", "Icmp6InGroupMembReductions", "Icmp6InRouterSolicits", "Icmp6InRouterAdvertisements",
					"Icmp6InNeighborSolicits", "Icmp6InNeighborAdvertisements", "Icmp6InRedirects", "Icmp6InMLDv2Reports",
					"Icmp6OutDestUnreachs", "Icmp6OutPktTooBigs", "Icmp6OutTimeExcds", "Icmp6OutParmProblems",
					"Icmp6OutEchos", "Icmp6OutEchoReplies", "Icmp6OutGroupMembQueries", "Icmp6OutGroupMembResponses",
					"Icmp6OutGroupMembReductions", "Icmp6OutRouterSolicits", "Icmp6OutRouterAdvertisements", "Icmp6OutNeighborSolicits",
					"Icmp6OutNeighborAdvertisements", "Icmp6OutRedirects", "Icmp6OutMLDv2Reports", "Icmp6OutType133",
					"Icmp6OutType135", "Icmp6OutType143", "Udp6InDatagrams", "Udp6NoPorts",
					"Udp6InErrors", "Udp6OutDatagrams", "Udp6RcvbufErrors", "Udp6SndbufErrors",
					"Udp6InCsumErrors", "Udp6IgnoredMulti", "Udp6MemErrors", "UdpLite6InDatagrams",
					"UdpLite6NoPorts", "UdpLite6InErrors", "UdpLite6OutDatagrams", "UdpLite6RcvbufErrors",
					"UdpLite6SndbufErrors", "UdpLite6InCsumErrors", "UdpLite6MemErrors",
				},
				Values: make([]uint64, 87),
				nameCheckRef: []byte(
					"Ip6InReceives Ip6InHdrErrors Ip6InTooBigErrors Ip6InNoRoutes " +
						"Ip6InAddrErrors Ip6InUnknownProtos Ip6InTruncatedPkts Ip6InDiscards " +
						"Ip6InDelivers Ip6OutForwDatagrams Ip6OutRequests Ip6OutDiscards " +
						"Ip6OutNoRoutes Ip6ReasmTimeout Ip6ReasmReqds Ip6ReasmOKs " +
						"Ip6ReasmFails Ip6FragOKs Ip6FragFails Ip6FragCreates " +
						"Ip6InMcastPkts Ip6OutMcastPkts Ip6InOctets Ip6OutOctets " +
						"Ip6InMcastOctets Ip6OutMcastOctets Ip6InBcastOctets Ip6OutBcastOctets " +
						"Ip6InNoECTPkts Ip6InECT1Pkts Ip6InECT0Pkts Ip6InCEPkts " +
						"Icmp6InMsgs Icmp6InErrors Icmp6OutMsgs Icmp6OutErrors " +
						"Icmp6InCsumErrors Icmp6InDestUnreachs Icmp6InPktTooBigs Icmp6InTimeExcds " +
						"Icmp6InParmProblems Icmp6InEchos Icmp6InEchoReplies Icmp6InGroupMembQueries " +
						"Icmp6InGroupMembResponses Icmp6InGroupMembReductions Icmp6InRouterSolicits Icmp6InRouterAdvertisements " +
						"Icmp6InNeighborSolicits Icmp6InNeighborAdvertisements Icmp6InRedirects Icmp6InMLDv2Reports " +
						"Icmp6OutDestUnreachs Icmp6OutPktTooBigs Icmp6OutTimeExcds Icmp6OutParmProblems " +
						"Icmp6OutEchos Icmp6OutEchoReplies Icmp6OutGroupMembQueries Icmp6OutGroupMembResponses " +
						"Icmp6OutGroupMembReductions Icmp6OutRouterSolicits Icmp6OutRouterAdvertisements Icmp6OutNeighborSolicits " +
						"Icmp6OutNeighborAdvertisements Icmp6OutRedirects Icmp6OutMLDv2Reports Icmp6OutType133 " +
						"Icmp6OutType135 Icmp6OutType143 Udp6InDatagrams Udp6NoPorts " +
						"Udp6InErrors Udp6OutDatagrams Udp6RcvbufErrors Udp6SndbufErrors " +
						"Udp6InCsumErrors Udp6IgnoredMulti Udp6MemErrors UdpLite6InDatagrams " +
						"UdpLite6NoPorts UdpLite6InErrors UdpLite6OutDatagrams UdpLite6RcvbufErrors " +
						"UdpLite6SndbufErrors UdpLite6InCsumErrors UdpLite6MemErrors "),
			},
			wantNetSnmp6: &NetSnmp6{
				Names: []string{
					"Ip6InReceives", "Ip6InHdrErrors", "Ip6InTooBigErrors", "Ip6InNoRoutes",
					"Ip6InAddrErrors", "Ip6InUnknownProtos", "Ip6InTruncatedPkts", "Ip6InDiscards",
					"Ip6InDelivers", "Ip6OutForwDatagrams", "Ip6OutRequests", "Ip6OutDiscards",
					"Ip6OutNoRoutes", "Ip6ReasmTimeout", "Ip6ReasmReqds", "Ip6ReasmOKs",
					"Ip6ReasmFails", "Ip6FragOKs", "Ip6FragFails", "Ip6FragCreates",
					"Ip6InMcastPkts", "Ip6OutMcastPkts", "Ip6InOctets", "Ip6OutOctets",
					"Ip6InMcastOctets", "Ip6OutMcastOctets", "Ip6InBcastOctets", "Ip6OutBcastOctets",
					"Ip6InNoECTPkts", "Ip6InECT1Pkts", "Ip6InECT0Pkts", "Ip6InCEPkts",
					"Icmp6InMsgs", "Icmp6InErrors", "Icmp6OutMsgs", "Icmp6OutErrors",
					"Icmp6InCsumErrors", "Icmp6InDestUnreachs", "Icmp6InPktTooBigs", "Icmp6InTimeExcds",
					"Icmp6InParmProblems", "Icmp6InEchos", "Icmp6InEchoReplies", "Icmp6InGroupMembQueries",
					"Icmp6InGroupMembResponses", "Icmp6InGroupMembReductions", "Icmp6InRouterSolicits", "Icmp6InRouterAdvertisements",
					"Icmp6InNeighborSolicits", "Icmp6InNeighborAdvertisements", "Icmp6InRedirects", "Icmp6InMLDv2Reports",
					"Icmp6OutDestUnreachs", "Icmp6OutPktTooBigs", "Icmp6OutTimeExcds", "Icmp6OutParmProblems",
					"Icmp6OutEchos", "Icmp6OutEchoReplies", "Icmp6OutGroupMembQueries", "Icmp6OutGroupMembResponses",
					"Icmp6OutGroupMembReductions", "Icmp6OutRouterSolicits", "Icmp6OutRouterAdvertisements", "Icmp6OutNeighborSolicits",
					"Icmp6OutNeighborAdvertisements", "Icmp6OutRedirects", "Icmp6OutMLDv2Reports", "Icmp6OutType133",
					"Icmp6OutType135", "Icmp6OutType143", "Udp6InDatagrams", "Udp6NoPorts",
					"Udp6InErrors", "Udp6OutDatagrams", "Udp6RcvbufErrors", "Udp6SndbufErrors",
					"Udp6InCsumErrors", "Udp6IgnoredMulti", "Udp6MemErrors", "UdpLite6InDatagrams",
					"UdpLite6NoPorts", "UdpLite6InErrors", "UdpLite6OutDatagrams", "UdpLite6RcvbufErrors",
					"UdpLite6SndbufErrors", "UdpLite6InCsumErrors", "UdpLite6MemErrors",
				},
				Values: []uint64{
					10000000000001, 10000000000002, 10000000000003, 10000000000004,
					10000000000005, 10000000000006, 10000000000007, 10000000000008,
					10000000000009, 10000000000010, 10000000000011, 10000000000012,
					10000000000013, 10000000000014, 10000000000015, 10000000000016,
					10000000000017, 10000000000018, 10000000000019, 10000000000020,
					10000000000021, 10000000000022, 10000000000023, 10000000000024,
					10000000000025, 10000000000026, 10000000000027, 10000000000028,
					10000000000029, 10000000000030, 10000000000031, 10000000000032,
					10000000000033, 10000000000034, 10000000000035, 10000000000036,
					10000000000037, 10000000000038, 10000000000039, 10000000000040,
					10000000000041, 10000000000042, 10000000000043, 10000000000044,
					10000000000045, 10000000000046, 10000000000047, 10000000000048,
					10000000000049, 10000000000050, 10000000000051, 10000000000052,
					10000000000053, 10000000000054, 10000000000055, 10000000000056,
					10000000000057, 10000000000058, 10000000000059, 10000000000060,
					10000000000061, 10000000000062, 10000000000063, 10000000000064,
					10000000000065, 10000000000066, 10000000000067, 10000000000068,
					10000000000069, 10000000000070, 10000000000071, 10000000000072,
					10000000000073, 10000000000074, 10000000000075, 10000000000076,
					10000000000077, 10000000000078, 10000000000079, 10000000000080,
					10000000000081, 10000000000082, 10000000000083, 10000000000084,
					10000000000085, 10000000000086, 10000000000087,
				},
			},
		},
	} {
		var name string
		if tc.name != "" {
			name = fmt.Sprintf("name=%s,procfsRoot=%s", tc.name, tc.procfsRoot)
		} else {
			name = fmt.Sprintf("procfsRoot=%s", tc.procfsRoot)
		}
		t.Run(
			name,
			func(t *testing.T) { testNetSnmp6Parser(tc, t) },
		)
	}
}
