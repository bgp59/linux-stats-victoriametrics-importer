// Scheduler metrics:

package lsvmi

import (
	"bytes"
	"fmt"
	"strconv"
)

const (
	COMPRESSOR_STATS_READ_DELTA_METRIC          = "lsvmi_compressor_read_delta"
	COMPRESSOR_STATS_READ_BYTE_DELTA_METRIC     = "lsvmi_compressor_read_byte_delta"
	COMPRESSOR_STATS_SEND_DELTA_METRIC          = "lsvmi_compressor_send_delta"
	COMPRESSOR_STATS_SEND_BYTE_DELTA_METRIC     = "lsvmi_compressor_send_byte_delta"
	COMPRESSOR_STATS_TIMEOUT_FLUSH_DELTA_METRIC = "lsvmi_compressor_tout_flush_delta"
	COMPRESSOR_STATS_SEND_ERROR_DELTA_METRIC    = "lsvmi_compressor_send_error_delta"
	COMPRESSOR_STATS_WRITE_ERROR_DELTA_METRIC   = "lsvmi_compressor_write_error_delta"
	COMPRESSOR_STATS_COMPRESSION_FACTOR_METRIC  = "lsvmi_compressor_compression_factor"

	COMPRESSOR_ID_LABEL_NAME = "compressor"
)

var compressorStatsUint64DeltaMetricsNameMap = map[int]string{
	COMPRESSOR_STATS_READ_COUNT:          COMPRESSOR_STATS_READ_DELTA_METRIC,
	COMPRESSOR_STATS_READ_BYTE_COUNT:     COMPRESSOR_STATS_READ_BYTE_DELTA_METRIC,
	COMPRESSOR_STATS_SEND_COUNT:          COMPRESSOR_STATS_SEND_DELTA_METRIC,
	COMPRESSOR_STATS_SEND_BYTE_COUNT:     COMPRESSOR_STATS_SEND_BYTE_DELTA_METRIC,
	COMPRESSOR_STATS_TIMEOUT_FLUSH_COUNT: COMPRESSOR_STATS_TIMEOUT_FLUSH_DELTA_METRIC,
	COMPRESSOR_STATS_SEND_ERROR_COUNT:    COMPRESSOR_STATS_SEND_ERROR_DELTA_METRIC,
	COMPRESSOR_STATS_WRITE_ERROR_COUNT:   COMPRESSOR_STATS_WRITE_ERROR_DELTA_METRIC,
}

var compressorStatsFloat64MetricsNameMap = map[int]string{
	COMPRESSOR_STATS_COMPRESSION_FACTOR: COMPRESSOR_STATS_COMPRESSION_FACTOR_METRIC,
}

type compressorPoolStatsIndexMetricMap map[int][]byte

type CompressorPoolInternalMetrics struct {
	// Internal metrics, for common values:
	internalMetrics *InternalMetrics
	// Dual storage for snapping the stats, used as current, previous, toggled
	// after every metrics generation:
	stats [2]CompressorPoolStats
	// The current index:
	crtIndex int
	// Cache the full metrics for each compressor# and stats index:
	uint64DeltaMetricsCache map[string]compressorPoolStatsIndexMetricMap
	float64MetricsCache     map[string]compressorPoolStatsIndexMetricMap
}

func NewCompressorPoolInternalMetrics(internalMetrics *InternalMetrics) *CompressorPoolInternalMetrics {
	return &CompressorPoolInternalMetrics{
		internalMetrics:         internalMetrics,
		uint64DeltaMetricsCache: make(map[string]compressorPoolStatsIndexMetricMap),
		float64MetricsCache:     make(map[string]compressorPoolStatsIndexMetricMap),
	}
}

func (cpim *CompressorPoolInternalMetrics) updateMetricsCache(compressorId string) {
	instance, hostname := GlobalInstance, GlobalHostname
	if cpim.internalMetrics.instance != "" {
		instance = cpim.internalMetrics.instance
	}
	if cpim.internalMetrics.hostname != "" {
		hostname = cpim.internalMetrics.hostname
	}

	indexMetricMap := make(compressorPoolStatsIndexMetricMap)
	for index, name := range compressorStatsUint64DeltaMetricsNameMap {
		metric := fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			COMPRESSOR_ID_LABEL_NAME, compressorId,
		)
		indexMetricMap[index] = []byte(metric)
	}
	cpim.uint64DeltaMetricsCache[compressorId] = indexMetricMap

	indexMetricMap = make(compressorPoolStatsIndexMetricMap)
	for index, name := range compressorStatsFloat64MetricsNameMap {
		metric := fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			COMPRESSOR_ID_LABEL_NAME, compressorId,
		)
		indexMetricMap[index] = []byte(metric)
	}
	cpim.float64MetricsCache[compressorId] = indexMetricMap
}

func (cpim *CompressorPoolInternalMetrics) generateMetrics(
	buf *bytes.Buffer, tsSuffix []byte,
) int {
	if tsSuffix == nil {
		// This should happen only during unit testing:
		tsSuffix = cpim.internalMetrics.getTsSuffix()
	}

	metricsCount := 0
	crtStats, prevStats := cpim.stats[cpim.crtIndex], cpim.stats[1-cpim.crtIndex]

	var prevCompressorStats *CompressorStats
	for compressorId, crtCompressorStats := range crtStats {
		if prevStats != nil {
			prevCompressorStats = prevStats[compressorId]
		} else {
			prevCompressorStats = nil
		}
		uint64IndexMetricMap := cpim.uint64DeltaMetricsCache[compressorId]
		if uint64IndexMetricMap == nil {
			// N.B. the following will also update cpim.float64MetricsCache:
			cpim.updateMetricsCache(compressorId)
			uint64IndexMetricMap = cpim.uint64DeltaMetricsCache[compressorId]
		}
		for index, metric := range uint64IndexMetricMap {
			val := crtCompressorStats.Uint64Stats[index]
			if prevCompressorStats != nil {
				val -= prevCompressorStats.Uint64Stats[index]
			}
			buf.Write(metric)
			buf.WriteString(strconv.FormatUint(val, 10))
			buf.Write(tsSuffix)
			metricsCount++
		}
		for index, metric := range cpim.float64MetricsCache[compressorId] {
			val := crtCompressorStats.Float64Stats[index]
			buf.Write(metric)
			buf.WriteString(strconv.FormatFloat(val, 'f', 3, 64))
			buf.Write(tsSuffix)
			metricsCount++
		}
	}

	// Flip the stats storage:
	cpim.crtIndex = 1 - cpim.crtIndex

	return metricsCount
}
