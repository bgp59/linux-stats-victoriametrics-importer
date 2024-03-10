// Internal metrics for the lsvmi Go process

package lsvmi

import (
	"bytes"
	"fmt"
	"runtime"
	"strconv"
)

const (
	GO_NUM_GOROUTINE_METRIC           = "lsvmi_go_num_goroutine"
	GO_MEM_SYS_BYTES_METRIC           = "lsvmi_go_mem_sys_bytes"
	GO_MEM_HEAP_BYTES_METRIC          = "lsvmi_go_mem_heap_bytes"
	GO_MEM_HEAP_SYS_BYTES_METRIC      = "lsvmi_go_mem_heap_sys_bytes"
	GO_MEM_MALLOCS_DELTA_METRIC       = "lsvmi_go_mem_malloc_delta"
	GO_MEM_FREE_DELTA_METRIC          = "lsvmi_go_mem_free_delta"
	GO_MEM_IN_USE_OBJECT_COUNT_METRIC = "lsvmi_go_mem_in_use_object_count"
	GO_MEM_NUM_GC_DELTA_METRIC        = "lsvmi_go_mem_gc_delta"
)

const (
	// The order in the metrics cache:
	GO_NUM_GOROUTINE_METRIC_INDEX = iota
	GO_MEM_SYS_BYTES_METRIC_INDEX
	GO_MEM_HEAP_BYTES_METRIC_INDEX
	GO_MEM_HEAP_SYS_BYTES_METRIC_INDEX
	GO_MEM_MALLOCS_DELTA_METRIC_INDEX
	GO_MEM_FREE_DELTA_METRIC_INDEX
	GO_MEM_IN_USE_OBJECT_COUNT_METRIC_INDEX
	GO_MEM_NUM_GC_DELTA_METRIC_INDEX

	// Must be last:
	GO_INTERNAL_METRICS_NUM
)

var goInternalMetricsNameMap = map[int]string{
	GO_NUM_GOROUTINE_METRIC_INDEX:           GO_NUM_GOROUTINE_METRIC,
	GO_MEM_SYS_BYTES_METRIC_INDEX:           GO_MEM_SYS_BYTES_METRIC,
	GO_MEM_HEAP_BYTES_METRIC_INDEX:          GO_MEM_HEAP_BYTES_METRIC,
	GO_MEM_HEAP_SYS_BYTES_METRIC_INDEX:      GO_MEM_HEAP_SYS_BYTES_METRIC,
	GO_MEM_MALLOCS_DELTA_METRIC_INDEX:       GO_MEM_MALLOCS_DELTA_METRIC,
	GO_MEM_FREE_DELTA_METRIC_INDEX:          GO_MEM_FREE_DELTA_METRIC,
	GO_MEM_IN_USE_OBJECT_COUNT_METRIC_INDEX: GO_MEM_IN_USE_OBJECT_COUNT_METRIC,
	GO_MEM_NUM_GC_DELTA_METRIC_INDEX:        GO_MEM_NUM_GC_DELTA_METRIC,
}

type GoInternalMetrics struct {
	// Internal metrics, for common values:
	internalMetrics *InternalMetrics
	// Snap data:
	goVersion    string
	numGoRoutine int
	// Dual storage for snapping Go runtime data, used as current, previous,
	// toggled after every metrics generation:
	memStats [2]*runtime.MemStats
	// The current index:
	crtIndex int
	// Metrics cache:
	metricsCache map[int][]byte
}

func NewGoInternalMetrics(internalMetrics *InternalMetrics) *GoInternalMetrics {
	gim := &GoInternalMetrics{
		goVersion:       runtime.Version(),
		internalMetrics: internalMetrics,
	}
	gim.memStats[0] = &runtime.MemStats{}
	gim.memStats[1] = &runtime.MemStats{}
	return gim
}

func (gim *GoInternalMetrics) SnapStats() {
	if gim.memStats[gim.crtIndex] == nil {
		gim.memStats[gim.crtIndex] = &runtime.MemStats{}
	}
	runtime.ReadMemStats(gim.memStats[gim.crtIndex])
	gim.numGoRoutine = runtime.NumGoroutine()
}

func (gim *GoInternalMetrics) updateMetricsCache() {
	instance, hostname := GlobalInstance, GlobalHostname
	if gim.internalMetrics.instance != "" {
		instance = gim.internalMetrics.instance
	}
	if gim.internalMetrics.hostname != "" {
		hostname = gim.internalMetrics.hostname
	}

	gim.metricsCache = make(map[int][]byte)

	for index, name := range goInternalMetricsNameMap {
		gim.metricsCache[index] = []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
		))
	}
}

func (gim *GoInternalMetrics) generateMetrics(
	buf *bytes.Buffer, tsSuffix []byte,
) int {

	if tsSuffix == nil {
		// This should happen only during unit testing:
		tsSuffix = gim.internalMetrics.getTsSuffix()
	}

	metricsCache := gim.metricsCache
	if metricsCache == nil {
		gim.updateMetricsCache()
		metricsCache = gim.metricsCache
	}

	crtMemStats, prevMemStats := gim.memStats[gim.crtIndex], gim.memStats[1-gim.crtIndex]

	metricsCount := 0

	buf.Write(metricsCache[GO_NUM_GOROUTINE_METRIC_INDEX])
	buf.WriteString(strconv.Itoa(gim.numGoRoutine))
	buf.Write(tsSuffix)
	metricsCount++

	buf.Write(metricsCache[GO_MEM_SYS_BYTES_METRIC_INDEX])
	buf.WriteString(strconv.FormatUint(crtMemStats.Sys, 10))
	buf.Write(tsSuffix)
	metricsCount++

	buf.Write(metricsCache[GO_MEM_HEAP_BYTES_METRIC_INDEX])
	buf.WriteString(strconv.FormatUint(crtMemStats.HeapAlloc, 10))
	buf.Write(tsSuffix)
	metricsCount++

	buf.Write(metricsCache[GO_MEM_HEAP_SYS_BYTES_METRIC_INDEX])
	buf.WriteString(strconv.FormatUint(crtMemStats.HeapSys, 10))
	buf.Write(tsSuffix)
	metricsCount++

	buf.Write(metricsCache[GO_MEM_IN_USE_OBJECT_COUNT_METRIC_INDEX])
	buf.WriteString(strconv.FormatUint(crtMemStats.Mallocs-crtMemStats.Frees, 10))
	buf.Write(tsSuffix)
	metricsCount++

	// Note that deltas below work even at the 1st pass because prevMemStats has
	// been primed w/ 0 when GoInternalMetrics was created:
	buf.Write(metricsCache[GO_MEM_MALLOCS_DELTA_METRIC_INDEX])
	buf.WriteString(strconv.FormatUint(crtMemStats.Mallocs-prevMemStats.Mallocs, 10))
	buf.Write(tsSuffix)
	metricsCount++

	buf.Write(metricsCache[GO_MEM_FREE_DELTA_METRIC_INDEX])
	buf.WriteString(strconv.FormatUint(crtMemStats.Frees-prevMemStats.Frees, 10))
	buf.Write(tsSuffix)
	metricsCount++

	buf.Write(metricsCache[GO_MEM_NUM_GC_DELTA_METRIC_INDEX])
	buf.WriteString(strconv.FormatUint(uint64(crtMemStats.NumGC-prevMemStats.NumGC), 10))
	buf.Write(tsSuffix)
	metricsCount++

	// Flip the stats storage:
	gim.crtIndex = 1 - gim.crtIndex

	return metricsCount
}
