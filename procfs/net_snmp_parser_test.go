package procfs

import (
	"bytes"
	"fmt"
	"path"
	"testing"
)

type NetSnmpTestCase struct {
	name         string
	procfsRoot   string
	primeNetSnmp *NetSnmp
	wantNetSnmp  *NetSnmp
	wantError    error
}

var netSnmpTestdataDir = path.Join(TESTDATA_PROCFS_ROOT, "net", "snmp")

func testNetSnmpParser(tc *NetSnmpTestCase, t *testing.T) {
	var netSnmp *NetSnmp

	wantNetSnmp := tc.wantNetSnmp

	// Sanity check:
	if len(wantNetSnmp.Names) != len(wantNetSnmp.Values) {
		t.Fatalf(
			"len(wantNetSnmp.Names): %d != %d len(wantNetSnmp.Values)",
			len(wantNetSnmp.Names), len(wantNetSnmp.Values),
		)
	}

	if tc.primeNetSnmp != nil {
		netSnmp = tc.primeNetSnmp.Clone(true)
		if netSnmp.path == "" {
			netSnmp.path = path.Join(tc.procfsRoot, "net", "snmp")
		}
	} else {
		netSnmp = NewNetSnmp(tc.procfsRoot)
	}

	err := netSnmp.Parse()
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

	if len(wantNetSnmp.Names) != len(netSnmp.Names) {
		fmt.Fprintf(
			diffBuf,
			"\nlen(Names): want: %d, got: %d",
			len(wantNetSnmp.Names), len(netSnmp.Names),
		)
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}
	for i, wantName := range wantNetSnmp.Names {
		gotName := netSnmp.Names[i]
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

	if len(wantNetSnmp.Values) != len(netSnmp.Values) {
		fmt.Fprintf(
			diffBuf,
			"\nlen(Values): want: %d, got: %d",
			len(wantNetSnmp.Values), len(netSnmp.Values),
		)
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}

	for i, wantValue := range wantNetSnmp.Values {
		gotValue := netSnmp.Values[i]
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

func TestNetSnmpParser(t *testing.T) {
	for _, tc := range []*NetSnmpTestCase{
		{
			procfsRoot: path.Join(netSnmpTestdataDir, "field_mapping"),
			wantNetSnmp: &NetSnmp{
				Names: []string{
					"ipForwarding", "ipDefaultTTL", "ipInReceives", "ipInHdrErrors", "ipInAddrErrors", "ipForwDatagrams", "ipInUnknownProtos", "ipInDiscards", "ipInDelivers", "ipOutRequests", "ipOutDiscards", "ipOutNoRoutes", "ipReasmTimeout", "ipReasmReqds", "ipReasmOKs", "ipReasmFails", "ipFragOKs", "ipFragFails", "ipFragCreates",
					"icmpInMsgs", "icmpInErrors", "icmpInCsumErrors", "icmpInDestUnreachs", "icmpInTimeExcds", "icmpInParmProbs", "icmpInSrcQuenchs", "icmpInRedirects", "icmpInEchos", "icmpInEchoReps", "icmpInTimestamps", "icmpInTimestampReps", "icmpInAddrMasks", "icmpInAddrMaskReps", "icmpOutMsgs", "icmpOutErrors", "icmpOutDestUnreachs", "icmpOutTimeExcds", "icmpOutParmProbs", "icmpOutSrcQuenchs", "icmpOutRedirects", "icmpOutEchos", "icmpOutEchoReps", "icmpOutTimestamps", "icmpOutTimestampReps", "icmpOutAddrMasks", "icmpOutAddrMaskReps",
					"icmpmsgInType3", "icmpmsgOutType3",
					"tcpRtoAlgorithm", "tcpRtoMin", "tcpRtoMax", "tcpMaxConn", "tcpActiveOpens", "tcpPassiveOpens", "tcpAttemptFails", "tcpEstabResets", "tcpCurrEstab", "tcpInSegs", "tcpOutSegs", "tcpRetransSegs", "tcpInErrs", "tcpOutRsts", "tcpInCsumErrors",
					"udpInDatagrams", "udpNoPorts", "udpInErrors", "udpOutDatagrams", "udpRcvbufErrors", "udpSndbufErrors", "udpInCsumErrors", "udpIgnoredMulti", "udpMemErrors",
					"udpliteInDatagrams", "udpliteNoPorts", "udpliteInErrors", "udpliteOutDatagrams", "udpliteRcvbufErrors", "udpliteSndbufErrors", "udpliteInCsumErrors", "udpliteIgnoredMulti", "udpliteMemErrors",
				},
				Values: []int64{
					1000, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015, 1016, 1017, 1018,
					3000, 3001, 3002, 3003, 3004, 3005, 3006, 3007, 3008, 3009, 3010, 3011, 3012, 3013, 3014, 3015, 3016, 3017, 3018, 3019, 3020, 3021, 3022, 3023, 3024, 3025, 3026,
					5000, 5001,
					7000, 7001, 7002, -7003, 7004, 7005, 7006, 7007, 7008, 7009, 7010, 7011, 7012, 7013, 7014,
					9000, 9001, 9002, 9003, 9004, 9005, 9006, 9007, 9008,
					11000, 11001, 11002, 11003, 11004, 11005, 11006, 11007, 11008,
				},
			},
		},
		{
			name:       "reuse",
			procfsRoot: path.Join(netSnmpTestdataDir, "field_mapping"),
			primeNetSnmp: &NetSnmp{
				Names: []string{
					"ipForwarding", "ipDefaultTTL", "ipInReceives", "ipInHdrErrors", "ipInAddrErrors", "ipForwDatagrams", "ipInUnknownProtos", "ipInDiscards", "ipInDelivers", "ipOutRequests", "ipOutDiscards", "ipOutNoRoutes", "ipReasmTimeout", "ipReasmReqds", "ipReasmOKs", "ipReasmFails", "ipFragOKs", "ipFragFails", "ipFragCreates",
					"icmpInMsgs", "icmpInErrors", "icmpInCsumErrors", "icmpInDestUnreachs", "icmpInTimeExcds", "icmpInParmProbs", "icmpInSrcQuenchs", "icmpInRedirects", "icmpInEchos", "icmpInEchoReps", "icmpInTimestamps", "icmpInTimestampReps", "icmpInAddrMasks", "icmpInAddrMaskReps", "icmpOutMsgs", "icmpOutErrors", "icmpOutDestUnreachs", "icmpOutTimeExcds", "icmpOutParmProbs", "icmpOutSrcQuenchs", "icmpOutRedirects", "icmpOutEchos", "icmpOutEchoReps", "icmpOutTimestamps", "icmpOutTimestampReps", "icmpOutAddrMasks", "icmpOutAddrMaskReps",
					"icmpmsgInType3", "icmpmsgOutType3",
					"tcpRtoAlgorithm", "tcpRtoMin", "tcpRtoMax", "tcpMaxConn", "tcpActiveOpens", "tcpPassiveOpens", "tcpAttemptFails", "tcpEstabResets", "tcpCurrEstab", "tcpInSegs", "tcpOutSegs", "tcpRetransSegs", "tcpInErrs", "tcpOutRsts", "tcpInCsumErrors",
					"udpInDatagrams", "udpNoPorts", "udpInErrors", "udpOutDatagrams", "udpRcvbufErrors", "udpSndbufErrors", "udpInCsumErrors", "udpIgnoredMulti", "udpMemErrors",
					"udpliteInDatagrams", "udpliteNoPorts", "udpliteInErrors", "udpliteOutDatagrams", "udpliteRcvbufErrors", "udpliteSndbufErrors", "udpliteInCsumErrors", "udpliteIgnoredMulti", "udpliteMemErrors",
				},
				Values: make([]int64, 19+27+2+15+9+9),
				lineInfo: []*NetSnmpLineInfo{
					{[]byte("Ip:"), 19},
					{[]byte("Icmp:"), 27},
					{[]byte("IcmpMsg:"), 2},
					{[]byte("Tcp:"), 15},
					{[]byte("Udp:"), 9},
					{[]byte("UdpLite:"), 9},
				},
			},
			wantNetSnmp: &NetSnmp{
				Names: []string{
					"ipForwarding", "ipDefaultTTL", "ipInReceives", "ipInHdrErrors", "ipInAddrErrors", "ipForwDatagrams", "ipInUnknownProtos", "ipInDiscards", "ipInDelivers", "ipOutRequests", "ipOutDiscards", "ipOutNoRoutes", "ipReasmTimeout", "ipReasmReqds", "ipReasmOKs", "ipReasmFails", "ipFragOKs", "ipFragFails", "ipFragCreates",
					"icmpInMsgs", "icmpInErrors", "icmpInCsumErrors", "icmpInDestUnreachs", "icmpInTimeExcds", "icmpInParmProbs", "icmpInSrcQuenchs", "icmpInRedirects", "icmpInEchos", "icmpInEchoReps", "icmpInTimestamps", "icmpInTimestampReps", "icmpInAddrMasks", "icmpInAddrMaskReps", "icmpOutMsgs", "icmpOutErrors", "icmpOutDestUnreachs", "icmpOutTimeExcds", "icmpOutParmProbs", "icmpOutSrcQuenchs", "icmpOutRedirects", "icmpOutEchos", "icmpOutEchoReps", "icmpOutTimestamps", "icmpOutTimestampReps", "icmpOutAddrMasks", "icmpOutAddrMaskReps",
					"icmpmsgInType3", "icmpmsgOutType3",
					"tcpRtoAlgorithm", "tcpRtoMin", "tcpRtoMax", "tcpMaxConn", "tcpActiveOpens", "tcpPassiveOpens", "tcpAttemptFails", "tcpEstabResets", "tcpCurrEstab", "tcpInSegs", "tcpOutSegs", "tcpRetransSegs", "tcpInErrs", "tcpOutRsts", "tcpInCsumErrors",
					"udpInDatagrams", "udpNoPorts", "udpInErrors", "udpOutDatagrams", "udpRcvbufErrors", "udpSndbufErrors", "udpInCsumErrors", "udpIgnoredMulti", "udpMemErrors",
					"udpliteInDatagrams", "udpliteNoPorts", "udpliteInErrors", "udpliteOutDatagrams", "udpliteRcvbufErrors", "udpliteSndbufErrors", "udpliteInCsumErrors", "udpliteIgnoredMulti", "udpliteMemErrors",
				},
				Values: []int64{
					1000, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015, 1016, 1017, 1018,
					3000, 3001, 3002, 3003, 3004, 3005, 3006, 3007, 3008, 3009, 3010, 3011, 3012, 3013, 3014, 3015, 3016, 3017, 3018, 3019, 3020, 3021, 3022, 3023, 3024, 3025, 3026,
					5000, 5001,
					7000, 7001, 7002, -7003, 7004, 7005, 7006, 7007, 7008, 7009, 7010, 7011, 7012, 7013, 7014,
					9000, 9001, 9002, 9003, 9004, 9005, 9006, 9007, 9008,
					11000, 11001, 11002, 11003, 11004, 11005, 11006, 11007, 11008,
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
			func(t *testing.T) { testNetSnmpParser(tc, t) },
		)
	}
}
