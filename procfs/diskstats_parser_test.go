package procfs

import (
	"bytes"
	"fmt"
	"path"
	"testing"
)

type DiskstatsTestCase struct {
	name                     string
	procfsRoot               string
	primeDiskstats           *Diskstats
	disableJiffiesToMillisec bool
	wantDiskstats            *Diskstats
	wantError                error
}

var diskstatsIndexName = []string{
	"DISKSTATS_MAJOR_NUM",
	"DISKSTATS_MINOR_NUM",
	"DISKSTATS_DEVICE",
	"DISKSTATS_NUM_READS_COMPLETED",
	"DISKSTATS_NUM_READS_MERGED",
	"DISKSTATS_NUM_READ_SECTORS",
	"DISKSTATS_READ_MILLISEC",
	"DISKSTATS_NUM_WRITES_COMPLETED",
	"DISKSTATS_NUM_WRITES_MERGED",
	"DISKSTATS_NUM_WRITE_SECTORS",
	"DISKSTATS_WRITE_MILLISEC",
	"DISKSTATS_NUM_IO_IN_PROGRESS",
	"DISKSTATS_IO_MILLISEC",
	"DISKSTATS_IO_WEIGTHED_MILLISEC",
	"DISKSTATS_NUM_DISCARDS_COMPLETED",
	"DISKSTATS_NUM_DISCARDS_MERGED",
	"DISKSTATS_NUM_DISCARD_SECTORS",
	"DISKSTATS_DISCARD_MILLISEC",
	"DISKSTATS_NUM_FLUSH_REQUESTS",
	"DISKSTATS_FLUSH_MILLISEC",
}

var diskstatsTestdataDir = path.Join(PROCFS_TESTDATA_ROOT, "diskstats")

func testDiskstatsParser(tc *DiskstatsTestCase, t *testing.T) {
	var diskstats *Diskstats

	if tc.primeDiskstats != nil {
		diskstats = tc.primeDiskstats.Clone(true)
		if diskstats.path == "" {
			diskstats.path = path.Join(tc.procfsRoot, "diskstats")
		}
	} else {
		diskstats = NewDiskstats(tc.procfsRoot)
		if tc.disableJiffiesToMillisec {
			diskstats.jiffiesToMillisec = 0
		}
	}

	err := diskstats.Parse()
	if tc.wantError != nil {
		if err == nil || tc.wantError.Error() != err.Error() {
			t.Fatalf("want: %v error, got: %v", tc.wantError, err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}

	wantDiskstats := tc.wantDiskstats
	if wantDiskstats == nil {
		return
	}

	diffBuf := &bytes.Buffer{}
	for dev, wantDevStats := range wantDiskstats.DevStats {
		gotDevStats := diskstats.DevStats[dev]
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
							"\nDevStats[%s][%d (%s)]: want: %d, got: %d",
							dev, i, diskstatsIndexName[i], wantStat, gotStat,
						)
					}
				}
			}
		}
	}
	for dev := range diskstats.DevStats {
		if wantDiskstats.DevStats[dev] == nil {
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

func TestDiskstatsParser(t *testing.T) {
	for _, tc := range []*DiskstatsTestCase{
		&DiskstatsTestCase{
			procfsRoot:               path.Join(diskstatsTestdataDir, "field_mapping"),
			disableJiffiesToMillisec: true,
			wantDiskstats: &Diskstats{
				DevStats: map[string][]uint32{
					"disk0": []uint32{1000, 1001, 0, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015, 1016, 1017, 1018, 1019},
					"disk1": []uint32{2000, 2001, 0, 2003, 2004, 2005, 2006, 2007, 2008, 2009, 2010, 2011, 2012, 2013, 2014, 2015, 2016, 2017, 2018, 2019},
				},
			},
		},
		&DiskstatsTestCase{
			name:       "reuse",
			procfsRoot: path.Join(diskstatsTestdataDir, "field_mapping"),
			primeDiskstats: &Diskstats{
				DevStats: map[string][]uint32{
					"disk0": make([]uint32, 20),
					"disk1": make([]uint32, 20),
				},
				devScanNum: map[string]int{
					"disk0": 42,
					"disk1": 42,
				},
				scanNum:           42,
				jiffiesToMillisec: 0,
				fieldsInJiffies:   diskstatsFieldsInJiffies,
			},
			wantDiskstats: &Diskstats{
				DevStats: map[string][]uint32{
					"disk0": []uint32{1000, 1001, 0, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015, 1016, 1017, 1018, 1019},
					"disk1": []uint32{2000, 2001, 0, 2003, 2004, 2005, 2006, 2007, 2008, 2009, 2010, 2011, 2012, 2013, 2014, 2015, 2016, 2017, 2018, 2019},
				},
			},
		},
		&DiskstatsTestCase{
			name:       "remove_dev",
			procfsRoot: path.Join(diskstatsTestdataDir, "field_mapping"),
			primeDiskstats: &Diskstats{
				DevStats: map[string][]uint32{
					"disk0":   make([]uint32, 20),
					"disk1":   make([]uint32, 20),
					"removed": make([]uint32, 20),
				},
				devScanNum: map[string]int{
					"disk0":   42,
					"disk1":   42,
					"removed": 42,
				},
				scanNum:           42,
				jiffiesToMillisec: 0,
				fieldsInJiffies:   diskstatsFieldsInJiffies,
			},
			wantDiskstats: &Diskstats{
				DevStats: map[string][]uint32{
					"disk0": []uint32{1000, 1001, 0, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 1012, 1013, 1014, 1015, 1016, 1017, 1018, 1019},
					"disk1": []uint32{2000, 2001, 0, 2003, 2004, 2005, 2006, 2007, 2008, 2009, 2010, 2011, 2012, 2013, 2014, 2015, 2016, 2017, 2018, 2019},
				},
			},
		},
		&DiskstatsTestCase{
			name:       "jiffies",
			procfsRoot: path.Join(diskstatsTestdataDir, "field_mapping"),
			primeDiskstats: &Diskstats{
				DevStats: map[string][]uint32{
					"disk0": make([]uint32, 20),
					"disk1": make([]uint32, 20),
				},
				devScanNum: map[string]int{
					"disk0": 42,
					"disk1": 42,
				},
				scanNum:           42,
				jiffiesToMillisec: 10,
				fieldsInJiffies:   diskstatsFieldsInJiffies,
			},
			wantDiskstats: &Diskstats{
				DevStats: map[string][]uint32{
					"disk0": []uint32{1000, 1001, 0, 1003, 1004, 1005, 1006, 1007, 1008, 1009, 1010, 1011, 10120, 1013, 1014, 1015, 1016, 1017, 1018, 1019},
					"disk1": []uint32{2000, 2001, 0, 2003, 2004, 2005, 2006, 2007, 2008, 2009, 2010, 2011, 20120, 2013, 2014, 2015, 2016, 2017, 2018, 2019},
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
			func(t *testing.T) { testDiskstatsParser(tc, t) },
		)
	}
}
