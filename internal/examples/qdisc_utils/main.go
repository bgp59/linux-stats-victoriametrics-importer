package main

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/utils"
)

var testUtilsQdiscIfInfoUint32IndexNameMap = map[int]string{
	utils.QDISC_PARENT:     "QDISC_PARENT",
	utils.QDISC_HANDLE:     "QDISC_HANDLE",
	utils.QDISC_PACKETS:    "QDISC_PACKETS",
	utils.QDISC_DROPS:      "QDISC_DROPS",
	utils.QDISC_REQUEUES:   "QDISC_REQUEUES",
	utils.QDISC_OVERLIMITS: "QDISC_OVERLIMITS",
	utils.QDISC_QLEN:       "QDISC_QLEN",
	utils.QDISC_BACKLOG:    "QDISC_BACKLOG",
}

var testUtilsQdiscIfInfoUint64IndexNameMap = map[int]string{
	utils.QDISC_BYTES:       "QDISC_BYTES",
	utils.QDISC_GCFLOWS:     "QDISC_GCFLOWS",
	utils.QDISC_THROTTLED:   "QDISC_THROTTLED",
	utils.QDISC_FLOWSPLIMIT: "QDISC_FLOWSPLIMIT",
}

func main() {
	fmt.Printf("QdiscAvail: %v\n", utils.QdiscAvail)

	uQdiskInfo := utils.NewUtilsQdiscInfo()
	start := time.Now()
	err := uQdiskInfo.Get()
	callDuration := time.Since(start)
	if err != nil {
		fmt.Fprintf(os.Stderr, "uQdiskInfo.Get(): %v\n", err)
		return
	}
	fmt.Printf("Call duration: %s\n", callDuration)

	ifNameList := make([]string, len(uQdiskInfo.Info))
	i := 0
	for ifName := range uQdiskInfo.Info {
		ifNameList[i] = ifName
		i++
	}
	sort.Strings(ifNameList)

	for _, ifName := range ifNameList {
		fmt.Println()
		fmt.Printf("I/F Name: %s\n", ifName)
		uQdiskIfInfo := uQdiskInfo.Info[ifName]
		fmt.Printf("\tKind: %s\n", uQdiskIfInfo.Kind)
		fmt.Println()
		for i := 0; i < utils.QDISK_UINT32_NUM_STATS; i++ {
			fmt.Printf(
				"\tUint32[%d (%s)]: %d\n",
				i, testUtilsQdiscIfInfoUint32IndexNameMap[i], uQdiskIfInfo.Uint32[i],
			)
		}
		fmt.Println()
		for i := 0; i < utils.QDISK_UINT64_NUM_STATS; i++ {
			fmt.Printf(
				"\tUint64[%d (%s)]: %d\n",
				i, testUtilsQdiscIfInfoUint64IndexNameMap[i], uQdiskIfInfo.Uint64[i],
			)
		}
	}
	fmt.Println()
}
