// qdisc parser a-la tc -s show qdisc

// Based on https://github.com/ema/qdisc, adapted for reusable objects, w/ data
// presented as slices. The latter allows for metrics generation in a loop.

package qdisc

import (
	"time"
)

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

const (
	// maj:min split:
	QDISC_MAJ_NUM_BITS = 16
	QDISC_MIN_NUM_BITS = 32 - QDISC_MAJ_NUM_BITS

	// How often to refresh the Interface index -> name cache:
	QDISC_IF_INDEX_TO_NAME_CACHE_REFRESH_INTERVAL = 60 * time.Second
)

type QdiscInfoKey struct {
	IfIndex uint32
	Handle  uint32
}

type QdiscInfo struct {
	IfName string
	Kind   string
	Uint32 [QDISK_UINT32_NUM_STATS]uint32
	Uint64 [QDISK_UINT64_NUM_STATS]uint64
	// Scan number used to identify out of scope interfaces:
	scanNum int
}

// QdiscStats object will be used for metrics generation as a current, previous
// tandem. The following information will be shared by both members; normally
// the access should be protected by a lock but since only one member is used at
// a time, the lock can be skipped.
type QdiscStatsShared struct {
	// Scan number used to identify out of scope qdiscs; incremented w/ every
	// call, qdiscs that have scan#(I/F) != scan#, will be removed:
	scanNum *int

	// Interface index -> name cache, refreshed periodically or every time there
	// is a miss:
	ifIndexToNameCache            map[uint32]string
	ifIndexToNameCacheLastRefresh time.Time

	// Netlink connection and request, however since the package that defines it
	// may be OS specific, use an unspecified type here:
	netConn       any
	netReqMessage any
}

type QdiscStats struct {
	// Map info by (ifIndex, handle), since it is unique:
	Info map[QdiscInfoKey]*QdiscInfo

	// Shared info:
	shared *QdiscStatsShared
}

func NewQdiscStats() *QdiscStats {
	return &QdiscStats{
		Info: make(map[QdiscInfoKey]*QdiscInfo),
		shared: &QdiscStatsShared{
			ifIndexToNameCache: make(map[uint32]string),
			scanNum:            new(int),
		},
	}
}

func (qs *QdiscStats) Clone() *QdiscStats {
	newQs := &QdiscStats{
		Info:   make(map[QdiscInfoKey]*QdiscInfo),
		shared: qs.shared,
	}

	for qiKei, qi := range qs.Info {
		newQs.Info[qiKei] = &QdiscInfo{
			IfName:  qi.IfName,
			Kind:    qi.Kind,
			scanNum: qi.scanNum,
		}
	}

	return newQs
}
