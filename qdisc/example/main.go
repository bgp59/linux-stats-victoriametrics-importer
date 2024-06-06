package main

import (
	"fmt"
	"os"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/qdisc"
)

var qdiscInfoUint32IndexNameMap = map[int]string{
	qdisc.QDISC_PARENT:     "QDISC_PARENT",
	qdisc.QDISC_HANDLE:     "QDISC_HANDLE",
	qdisc.QDISC_PACKETS:    "QDISC_PACKETS",
	qdisc.QDISC_DROPS:      "QDISC_DROPS",
	qdisc.QDISC_REQUEUES:   "QDISC_REQUEUES",
	qdisc.QDISC_OVERLIMITS: "QDISC_OVERLIMITS",
	qdisc.QDISC_QLEN:       "QDISC_QLEN",
	qdisc.QDISC_BACKLOG:    "QDISC_BACKLOG",
}

var qdiscInfoUint64IndexNameMap = map[int]string{
	qdisc.QDISC_BYTES:       "QDISC_BYTES",
	qdisc.QDISC_GCFLOWS:     "QDISC_GCFLOWS",
	qdisc.QDISC_THROTTLED:   "QDISC_THROTTLED",
	qdisc.QDISC_FLOWSPLIMIT: "QDISC_FLOWSPLIMIT",
}

func main() {
	fmt.Printf("QdiscAvail: %v\n", qdisc.QdiscAvail)

	qs := qdisc.NewQdiscStats()

	for k := 1; k <= 2; k++ {
		start := time.Now()
		err := qs.Update()
		callDuration := time.Since(start)
		if err != nil {
			fmt.Fprintf(os.Stderr, "qs.Update(): %v\n", err)
			return
		}
		fmt.Printf("Call# %d duration: %s\n", k, callDuration)

		for _, qi := range qs.Info {
			fmt.Println()
			fmt.Printf("I/F Name: %s\n", qi.IfName)
			fmt.Printf("\tKind: %s\n", qi.Kind)
			fmt.Println()
			for i := 0; i < qdisc.QDISK_UINT32_NUM_STATS; i++ {
				fmt.Printf(
					"\tUint32[%d (%s)]: %d\n",
					i, qdiscInfoUint32IndexNameMap[i], qi.Uint32[i],
				)
			}
			fmt.Println()
			for i := 0; i < qdisc.QDISK_UINT64_NUM_STATS; i++ {
				fmt.Printf(
					"\tUint64[%d (%s)]: %d\n",
					i, qdiscInfoUint64IndexNameMap[i], qi.Uint64[i],
				)
			}
		}
		fmt.Println()
	}
}
