// Scheduler metrics:

package lsvmi

import (
	"bytes"
	"fmt"
	"strconv"
)

const (
	COMPRESSOR_STATS_READ_COUNT_DELTA_METRIC          = "lsvmi_compressor_read_count_delta"
	COMPRESSOR_STATS_READ_BYTE_COUNT_DELTA_METRIC     = "lsvmi_compressor_read_byte_count_delta"
	COMPRESSOR_STATS_SEND_COUNT_DELTA_METRIC          = "lsvmi_compressor_send_count_delta"
	COMPRESSOR_STATS_SEND_BYTE_COUNT_DELTA_METRIC     = "lsvmi_compressor_send_byte_count_delta"
	COMPRESSOR_STATS_TIMEOUT_FLUSH_COUNT_DELTA_METRIC = "lsvmi_compressor_tout_flush_count_delta"
	COMPRESSOR_STATS_SEND_ERROR_COUNT_DELTA_METRIC    = "lsvmi_compressor_send_error_count_delta"
	COMPRESSOR_STATS_WRITE_ERROR_COUNT_DELTA_METRIC   = "lsvmi_compressor_write_error_count_delta"
	COMPRESSOR_STATS_COMPRESSION_FACTOR_METRIC        = "lsvmi_compressor_compression_factor"

	COMPRESSOR_ID_LABEL_NAME = "compressor"
)

var compressorStatsUint64MetricsNameMap = map[int]string{
	COMPRESSOR_STATS_READ_COUNT:          COMPRESSOR_STATS_READ_COUNT_DELTA_METRIC,
	COMPRESSOR_STATS_READ_BYTE_COUNT:     COMPRESSOR_STATS_READ_BYTE_COUNT_DELTA_METRIC,
	COMPRESSOR_STATS_SEND_COUNT:          COMPRESSOR_STATS_SEND_COUNT_DELTA_METRIC,
	COMPRESSOR_STATS_SEND_BYTE_COUNT:     COMPRESSOR_STATS_SEND_BYTE_COUNT_DELTA_METRIC,
	COMPRESSOR_STATS_TIMEOUT_FLUSH_COUNT: COMPRESSOR_STATS_TIMEOUT_FLUSH_COUNT_DELTA_METRIC,
	COMPRESSOR_STATS_SEND_ERROR_COUNT:    COMPRESSOR_STATS_SEND_ERROR_COUNT_DELTA_METRIC,
	COMPRESSOR_STATS_WRITE_ERROR_COUNT:   COMPRESSOR_STATS_WRITE_ERROR_COUNT_DELTA_METRIC,
}

var compressorStatsFloat64MetricsNameMap = map[int]string{
	COMPRESSOR_STATS_COMPRESSION_FACTOR: COMPRESSOR_STATS_COMPRESSION_FACTOR_METRIC,
}

type compressorPoolStatsIndexMetricMap map[int][]byte

type CompressorPoolInternalMetrics struct {
	// Internal metrics, for common values:
	internalMetrics *InternalMetrics
	// Dual buffer holding current, previous delta stats:
	stats [2]CompressorPoolStats
	// Which one is current:
	crtStatsIndx int
	// Cache the full metrics for each compressor# and stats index:
	uint64MetricsCache  map[string]compressorPoolStatsIndexMetricMap
	float64MetricsCache map[string]compressorPoolStatsIndexMetricMap
	// A buffer for the timestamp suffix:
	tsSuffixBuf *bytes.Buffer
}

func NewCompressorPoolInternalMetrics(internalMetrics *InternalMetrics) *CompressorPoolInternalMetrics {
	return &CompressorPoolInternalMetrics{
		internalMetrics:     internalMetrics,
		uint64MetricsCache:  make(map[string]compressorPoolStatsIndexMetricMap),
		float64MetricsCache: make(map[string]compressorPoolStatsIndexMetricMap),
		tsSuffixBuf:         &bytes.Buffer{},
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
	for index, name := range compressorStatsUint64MetricsNameMap {
		metric := fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			COMPRESSOR_ID_LABEL_NAME, compressorId,
		)
		indexMetricMap[index] = []byte(metric)
	}
	cpim.uint64MetricsCache[compressorId] = indexMetricMap

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
	buf *bytes.Buffer, fullCycle bool, tsSuffix []byte,
) int {
	crtStatsIndx := cpim.crtStatsIndx
	crtStats, prevStats := cpim.stats[crtStatsIndx], cpim.stats[1-crtStatsIndx]
	if fullCycle {
		prevStats = nil
	}

	if tsSuffix == nil {
		// This should happen only during unit testing:
		tsSuffix = cpim.internalMetrics.getTsSuffix()
	}

	// For counter delta metrics, unless this is a full cycle, skip 0 values if
	// the previous scan value was also 0.

	var prevCompressorStats *CompressorStats = nil
	metricsCount := 0
	for compressorId, crtCompressorStats := range crtStats {
		if prevStats != nil {
			prevCompressorStats = prevStats[compressorId]
		}

		uint64IndexMetricMap := cpim.uint64MetricsCache[compressorId]
		if uint64IndexMetricMap == nil {
			cpim.updateMetricsCache(compressorId)
			uint64IndexMetricMap = cpim.uint64MetricsCache[compressorId]
		}
		for index, metric := range uint64IndexMetricMap {
			crtVal := crtCompressorStats.Uint64Stats[index]
			if crtVal != 0 || prevCompressorStats == nil || crtVal != prevCompressorStats.Uint64Stats[index] {
				buf.Write(metric)
				buf.WriteString(strconv.FormatUint(crtVal, 10))
				buf.Write(tsSuffix)
				metricsCount++
			}
		}

		for index, metric := range cpim.float64MetricsCache[compressorId] {
			crtVal := crtCompressorStats.Float64Stats[index]
			if prevCompressorStats == nil || crtVal != prevCompressorStats.Float64Stats[index] {
				buf.Write(metric)
				buf.WriteString(strconv.FormatFloat(crtVal, 'f', 3, 64))
				buf.Write(tsSuffix)
				metricsCount++
			}
		}
	}

	// Flip the buffers:
	cpim.crtStatsIndx = 1 - crtStatsIndx

	return metricsCount
}
