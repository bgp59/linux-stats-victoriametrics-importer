// Qdisc parser of a sort

// See man tc (tc -s qdisc show)

package utils

// The info returned by Golang qdisc.Get will be repackaged as lists of
// unit32/64 to make it easier to generate metrics in a loop.

const (
	// uint32 indices:
	QDISC_PARENT = iota
	QDISC_HANDLE
	QDISC_PACKETS
	QDISC_DROPS
	QDISC_REQUEUES
	QDISC_OVERLIMITS
	QDISC_QLEN
	QDISC_BACKLOG

	// Must be last:
	QDISK_UINT32_NUM_STATS
)

const (
	// uint64 indices:
	QDISC_BYTES = iota
	QDISC_GCFLOWS
	QDISC_THROTTLED
	QDISC_FLOWSPLIMIT

	// Must be last:
	QDISK_UINT64_NUM_STATS
)

type UtilsQdiscIfInfo struct {
	Kind   string
	Uint32 [QDISK_UINT32_NUM_STATS]uint32
	Uint64 [QDISK_UINT64_NUM_STATS]uint64
	// Scan number used to identify out of scope interfaces:
	scanNum int
}

type UtilsQdiscInfo struct {
	// Map info by I/F name:
	Info map[string]*UtilsQdiscIfInfo
	// Scan number used to identify out of scope interfaces; incremented w/
	// every call, I/F's that have scan#(I/F) != scan#, will be removed:
	scanNum int
}

func NewUtilsQdiscInfo() *UtilsQdiscInfo {
	return &UtilsQdiscInfo{
		Info: make(map[string]*UtilsQdiscIfInfo),
	}
}
