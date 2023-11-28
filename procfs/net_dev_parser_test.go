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

var netDevIndexName = []string{
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

	diffBuf := &bytes.Buffer{}
	wantNetDev := tc.wantNetDev

	for dev, wantDevStats := range wantNetDev.DevStats {
		gotDevStats := netDev.DevStats[dev]
		if gotDevStats == nil {
			fmt.Fprintf(
				diffBuf,
				"\nDevStats[%s]: missing device", dev,
			)
		} else {
			if len(wantDevStats) != len(gotDevStats) {
				fmt.Fprintf(
					diffBuf,
					"\nlen(DevStats[%s]): want: %d, got: %d",
					dev, len(wantDevStats), len(gotDevStats),
				)
			} else {
				for i, wantStat := range wantDevStats {
					gotStat := gotDevStats[i]
					if wantStat != gotStat {
						fmt.Fprintf(
							diffBuf,
							"\nDevStats[%s (%d)][%s]: want: %d, got: %d",
							dev, i, netDevIndexName, wantStat, gotStat,
						)
					}
				}
			}
		}
	}

	for dev := range netDev.DevStats {
		if wantNetDev.DevStats[dev] == nil {
			fmt.Fprintf(
				diffBuf,
				"\nDevStats[%s]: unexpected device", dev,
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
				DevStats: map[string][]uint64{
					"lo":   {1000, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015},
					"eth0": {2000, 2001, 2002, 2003, 2004, 2005, 2006, 2007, 2008, 2009, 2010, 2011, 2012, 2013, 2014, 2015},
				},
			},
		},
		&NetDevTestCase{
			name:       "reuse",
			procfsRoot: path.Join(netDevTestdataDir, "field_mapping"),
			primeNetDev: &NetDev{
				DevStats: map[string][]uint64{
					"lo":   make([]uint64, NET_DEV_NUM_STATS),
					"eth0": make([]uint64, NET_DEV_NUM_STATS),
				},
				devScanNum: map[string]int{
					"lo":   1,
					"eth0": 1,
				},
				scanNum:     13,
				validHeader: testNetDevHeader,
			},
			wantNetDev: &NetDev{
				DevStats: map[string][]uint64{
					"lo":   {1000, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015},
					"eth0": {2000, 2001, 2002, 2003, 2004, 2005, 2006, 2007, 2008, 2009, 2010, 2011, 2012, 2013, 2014, 2015},
				},
			},
		},
		&NetDevTestCase{
			name:       "remove_dev",
			procfsRoot: path.Join(netDevTestdataDir, "field_mapping"),
			primeNetDev: &NetDev{
				DevStats: map[string][]uint64{
					"remove": make([]uint64, NET_DEV_NUM_STATS),
					"lo":     make([]uint64, NET_DEV_NUM_STATS),
					"eth0":   make([]uint64, NET_DEV_NUM_STATS),
				},
				devScanNum: map[string]int{
					"remove": 2,
					"lo":     2,
					"eth0":   2,
				},
				scanNum:     2,
				validHeader: testNetDevHeader,
			},
			wantNetDev: &NetDev{
				DevStats: map[string][]uint64{
					"lo":   {1000, 1001, 1002, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015},
					"eth0": {2000, 2001, 2002, 2003, 2004, 2005, 2006, 2007, 2008, 2009, 2010, 2011, 2012, 2013, 2014, 2015},
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
