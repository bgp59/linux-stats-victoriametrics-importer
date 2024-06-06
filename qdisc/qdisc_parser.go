// qdisc parser a-la tc -s show qdisc

// Based on https://github.com/ema/qdisc, adapted for reusable objects, w/ data
// presented as slices. The latter allows for metrics generation in a loop.

package qdisc

import (
	"sync"
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
	// How often to refresh the Interface index -> name cache:
	IF_INDEX_TO_NAME_CACHE_REFRESH_INTERVAL = 60 * time.Second
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

type QdiscStats struct {
	// Map info by (ifIndex, handle), since it is unique:
	Info map[QdiscInfoKey]*QdiscInfo
	// Scan number used to identify out of scope handles; incremented w/ every
	// call, handles that have scan#(I/F) != scan#, will be removed:
	scanNum int
	// Interface index -> name cache, refreshed periodically or every time there
	// is a miss:
	ifIndexToNameCache            map[uint32]string
	ifIndexToNameCacheLastRefresh time.Time
	ifIndexToNameCacheLock        *sync.Mutex
}

func NewQdiscStats() *QdiscStats {
	return &QdiscStats{
		Info:                   make(map[QdiscInfoKey]*QdiscInfo),
		ifIndexToNameCacheLock: &sync.Mutex{},
	}
}
