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
	// Storage for snapping stats:
	stats CompressorPoolStats
	// Cache the full metrics for each compressor# and stats index:
	uint64MetricsCache  map[string]compressorPoolStatsIndexMetricMap
	float64MetricsCache map[string]compressorPoolStatsIndexMetricMap
}

func NewCompressorPoolInternalMetrics(internalMetrics *InternalMetrics) *CompressorPoolInternalMetrics {
	return &CompressorPoolInternalMetrics{
		internalMetrics:     internalMetrics,
		uint64MetricsCache:  make(map[string]compressorPoolStatsIndexMetricMap),
		float64MetricsCache: make(map[string]compressorPoolStatsIndexMetricMap),
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
	buf *bytes.Buffer, tsSuffix []byte,
) int {
	if tsSuffix == nil {
		// This should happen only during unit testing:
		tsSuffix = cpim.internalMetrics.getTsSuffix()
	}

	metricsCount := 0
	for compressorId, compressorStats := range cpim.stats {

		uint64IndexMetricMap := cpim.uint64MetricsCache[compressorId]
		if uint64IndexMetricMap == nil {
			// N.B. the following will also update cpim.float64MetricsCache:
			cpim.updateMetricsCache(compressorId)
			uint64IndexMetricMap = cpim.uint64MetricsCache[compressorId]
		}
		for index, metric := range uint64IndexMetricMap {
			buf.Write(metric)
			buf.WriteString(strconv.FormatUint(compressorStats.Uint64Stats[index], 10))
			buf.Write(tsSuffix)
			metricsCount++
		}
		for index, metric := range cpim.float64MetricsCache[compressorId] {
			buf.Write(metric)
			buf.WriteString(strconv.FormatFloat(compressorStats.Float64Stats[index], 'f', 3, 64))
			buf.Write(tsSuffix)
			metricsCount++
		}
	}

	return metricsCount
}
