package procfs

import (
	"bytes"
	"fmt"
	"path"
	"testing"
)

type NetDevTestCase struct {
	name        string
	procfsRoot  string
	primeNetDev *NetDev
	wantNetDev  *NetDev
	wantError   error
}

var netDevTestdataDir = path.Join(PROCFS_TESTDATA_ROOT, "net", "dev")

var netDevStatName = []string{
	"NET_DEV_RX_BYTES",
	"NET_DEV_RX_PACKETS",
	"NET_DEV_RX_ERRS",
	"NET_DEV_RX_DROP",
	"NET_DEV_RX_FIFO",
	"NET_DEV_RX_FRAME",
	"NET_DEV_RX_COMPRESSED",
	"NET_DEV_RX_MULTICAST",
	"NET_DEV_TX_BYTES",
	"NET_DEV_TX_PACKETS",
	"NET_DEV_TX_ERRS",
	"NET_DEV_TX_DROP",
	"NET_DEV_TX_FIFO",
	"NET_DEV_TX_COLLS",
	"NET_DEV_TX_CARRIER",
	"NET_DEV_TX_COMPRESSED",
}

var testNetDevHeader = []byte(`
Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
`)[1:]

var testNetDevNumLinesHeader = 2

func testNetDevParser(tc *NetDevTestCase, t *testing.T) {
	var netDev *NetDev

	if tc.primeNetDev != nil {
		netDev = tc.primeNetDev.Clone(true)
		if netDev.path == "" {
			netDev.path = path.Join(tc.procfsRoot, "net", "dev")
		}
	} else {
		netDev = NewNetDev(tc.procfsRoot)
	}

	err := netDev.Parse()
	if tc.wantError != nil {
		if err == nil || tc.wantError.Error() != err.Error() {
			t.Fatalf("want: %v error, got: %v", tc.wantError, err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}

	wantNetDev := tc.wantNetDev
	if wantNetDev == nil {
		return
	}

	diffBuf := &bytes.Buffer{}

	for i, wantDevStats := range wantNetDev.DevStats {
		gotDevStats := netDev.DevStats[i]
		if len(wantDevStats) != len(gotDevStats) {
			fmt.Fprintf(
				diffBuf,
				"\nlen(DevStats[%d]): want: %d, got: %d",
				i, len(wantDevStats), len(gotDevStats),
			)
		} else {
			for statIndex, wantStat := range wantDevStats {
				gotStat := gotDevStats[statIndex]
				if wantStat != gotStat {
					fmt.Fprintf(
						diffBuf,
						"\nDevStats[%d][%d (%s)]: want: %d, got: %d",
						i, statIndex, netDevStatName[statIndex], wantStat, gotStat,
					)
				}
			}
		}
	}

	for dev, wantIndex := range wantNetDev.DevStatsIndex {
		gotIndex, ok := netDev.DevStatsIndex[dev]
		if !ok {
			fmt.Fprintf(
				diffBuf,
				"\nDevStatsIndex[%s]: missing device", dev,
			)
		} else if wantIndex != gotIndex {
			fmt.Fprintf(
				diffBuf,
				"\nDevStatsIn dex[%s]: want: %d, got: %d",
				dev, wantIndex, gotIndex,
			)

		}
	}

	for dev := range netDev.DevStatsIndex {
		_, ok := wantNetDev.DevStatsIndex[dev]
		if !ok {
			fmt.Fprintf(
				diffBuf,
				"\nDevStatsIndex[%s]: unexpected device", dev,
			)
		}
	}

	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}
}

func TestNetDevParser(t *testing.T) {
	for _, tc := range []*NetDevTestCase{
		&NetDevTestCase{
			procfsRoot: path.Join(netDevTestdataDir, "field_mapping"),
			wantNetDev: &NetDev{
				DevStats: [][]uint64{
					{1000, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015},
					{2000, 2001, 2002, 2003, 2004, 2005, 2006, 2007, 2008, 2009, 2010, 2011, 2012, 2013, 2014, 2015},
				},
				DevStatsIndex: map[string]int{
					"lo":   0,
					"eth0": 1,
				},
			},
		},
		&NetDevTestCase{
			name:       "reuse",
			procfsRoot: path.Join(netDevTestdataDir, "field_mapping"),
			primeNetDev: &NetDev{
				DevStats: [][]uint64{
					make([]uint64, NET_DEV_NUM_STATS),
					make([]uint64, NET_DEV_NUM_STATS),
				},
				DevStatsIndex: map[string]int{
					"lo":   0,
					"eth0": 1,
				},
				validHeader:    testNetDevHeader,
				numLinesHeader: testNetDevNumLinesHeader,
			},
			wantNetDev: &NetDev{
				DevStats: [][]uint64{
					{1000, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015},
					{2000, 2001, 2002, 2003, 2004, 2005, 2006, 2007, 2008, 2009, 2010, 2011, 2012, 2013, 2014, 2015},
				},
				DevStatsIndex: map[string]int{
					"lo":   0,
					"eth0": 1,
				},
			},
		},
		&NetDevTestCase{
			name:       "remove_dev",
			procfsRoot: path.Join(netDevTestdataDir, "field_mapping"),
			primeNetDev: &NetDev{
				DevStats: [][]uint64{
					make([]uint64, NET_DEV_NUM_STATS),
					make([]uint64, NET_DEV_NUM_STATS),
					make([]uint64, NET_DEV_NUM_STATS),
				},
				DevStatsIndex: map[string]int{
					"lo":     0,
					"remove": 1,
					"eth0":   2,
				},
				validHeader:    testNetDevHeader,
				numLinesHeader: testNetDevNumLinesHeader,
			},
			wantNetDev: &NetDev{
				DevStats: [][]uint64{
					{1000, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015},
					{2000, 2001, 2002, 2003, 2004, 2005, 2006, 2007, 2008, 2009, 2010, 2011, 2012, 2013, 2014, 2015},
				},
				DevStatsIndex: map[string]int{
					"lo":   0,
					"eth0": 1,
				},
			},
		},
		&NetDevTestCase{
			procfsRoot: path.Join(netDevTestdataDir, "whitespaces"),
			wantNetDev: &NetDev{
				DevStats: [][]uint64{
					{1000, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015},
					{2000, 2001, 2002, 2003, 2004, 2005, 2006, 2007, 2008, 2009, 2010, 2011, 2012, 2013, 2014, 2015},
					{3000, 3001, 3002, 3003, 3004, 3005, 3006, 3007, 3008, 3009, 3010, 3011, 3012, 3013, 3014, 3015},
					{4000, 4001, 4002, 4003, 4004, 4005, 4006, 4007, 4008, 4009, 4010, 4011, 4012, 4013, 4014, 4015},
				},
				DevStatsIndex: map[string]int{
					"dev1": 0,
					"dev2": 1,
					"dev3": 2,
					"dev4": 3,
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
			func(t *testing.T) { testNetDevParser(tc, t) },
		)
	}
}
