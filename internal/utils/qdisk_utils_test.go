package utils

import (
	"bytes"
	"fmt"
	"sort"
	"testing"
)

var testUtilsQdiscIfInfoUint32IndexNameMap = map[int]string{
	QDISC_PARENT:     "QDISC_PARENT",
	QDISC_HANDLE:     "QDISC_HANDLE",
	QDISC_PACKETS:    "QDISC_PACKETS",
	QDISC_DROPS:      "QDISC_DROPS",
	QDISC_REQUEUES:   "QDISC_REQUEUES",
	QDISC_OVERLIMITS: "QDISC_OVERLIMITS",
	QDISC_QLEN:       "QDISC_QLEN",
	QDISC_BACKLOG:    "QDISC_BACKLOG",
}

var testUtilsQdiscIfInfoUint64IndexNameMap = map[int]string{
	QDISC_BYTES:       "QDISC_BYTES",
	QDISC_GCFLOWS:     "QDISC_GCFLOWS",
	QDISC_THROTTLED:   "QDISC_THROTTLED",
	QDISC_FLOWSPLIMIT: "QDISC_FLOWSPLIMIT",
}

func TestUtilsQdiscInfo(t *testing.T) {
	t.Logf("QdiscAvail: %v", QdiscAvail)

	uQdiskInfo := NewUtilsQdiscInfo()
	err := uQdiskInfo.Get()
	if err != nil {
		t.Fatal(err)
	}

	buf := &bytes.Buffer{}

	ifNameList := make([]string, len(uQdiskInfo.Info))
	i := 0
	for ifName := range uQdiskInfo.Info {
		ifNameList[i] = ifName
		i++
	}
	sort.Strings(ifNameList)

	for _, ifName := range ifNameList {
		fmt.Fprintf(buf, "\n\nI/F Name: %s", ifName)
		uQdiskIfInfo := uQdiskInfo.Info[ifName]
		fmt.Fprintf(buf, "\n\tKind: %s", uQdiskIfInfo.Kind)
		for i := 0; i < QDISK_UINT32_NUM_STATS; i++ {
			fmt.Fprintf(
				buf,
				"\n\tUint32[%d (%s)]: %d",
				i, testUtilsQdiscIfInfoUint32IndexNameMap[i], uQdiskIfInfo.Uint32[i],
			)
		}
		for i := 0; i < QDISK_UINT64_NUM_STATS; i++ {
			fmt.Fprintf(
				buf,
				"\n\tUint64[%d (%s)]: %d",
				i, testUtilsQdiscIfInfoUint64IndexNameMap[i], uQdiskIfInfo.Uint64[i],
			)
		}
		fmt.Fprintf(buf, "\n")
	}

	t.Log(buf)
}
