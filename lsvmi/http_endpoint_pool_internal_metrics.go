// Internal metrics for HTTP Endpoint Pool

package lsvmi

import (
	"bytes"
	"fmt"
	"strconv"
)

const (
	HTTP_ENDPOINT_STATS_SEND_BUFFER_DELTA_METRIC        = "lsvmi_http_ep_send_buffer_delta"
	HTTP_ENDPOINT_STATS_SEND_BUFFER_BYTE_DELTA_METRIC   = "lsvmi_http_ep_send_buffer_byte_delta"
	HTTP_ENDPOINT_STATS_SEND_BUFFER_ERROR_DELTA_METRIC  = "lsvmi_http_ep_send_buffer_error_delta"
	HTTP_ENDPOINT_STATS_HEALTH_CHECK_DELTA_METRIC       = "lsvmi_http_ep_healthcheck_delta"
	HTTP_ENDPOINT_STATS_HEALTH_CHECK_ERROR_DELTA_METRIC = "lsvmi_http_ep_healthcheck_error_delta"
	HTTP_ENDPOINT_STATS_STATE_METRIC                    = "lsvmi_http_ep_state"

	HTTP_ENDPOINT_POOL_STATS_HEALTHY_ROTATE_COUNT_METRIC      = "lsvmi_http_ep_pool_healthy_rotate_count"
	HTTP_ENDPOINT_POOL_STATS_NO_HEALTHY_EP_ERROR_DELTA_METRIC = "lsvmi_http_ep_pool_no_healthy_ep_error_delta"

	HTTP_ENDPOINT_URL_LABEL_NAME = "url"
)

var httpEndpointStatsDeltaMetricsNameMap = map[int]string{
	HTTP_ENDPOINT_STATS_SEND_BUFFER_COUNT:        HTTP_ENDPOINT_STATS_SEND_BUFFER_DELTA_METRIC,
	HTTP_ENDPOINT_STATS_SEND_BUFFER_BYTE_COUNT:   HTTP_ENDPOINT_STATS_SEND_BUFFER_BYTE_DELTA_METRIC,
	HTTP_ENDPOINT_STATS_SEND_BUFFER_ERROR_COUNT:  HTTP_ENDPOINT_STATS_SEND_BUFFER_ERROR_DELTA_METRIC,
	HTTP_ENDPOINT_STATS_HEALTH_CHECK_COUNT:       HTTP_ENDPOINT_STATS_HEALTH_CHECK_DELTA_METRIC,
	HTTP_ENDPOINT_STATS_HEALTH_CHECK_ERROR_COUNT: HTTP_ENDPOINT_STATS_HEALTH_CHECK_ERROR_DELTA_METRIC,
}

var httpEndpointStatsMetricsNameMap = map[int]string{
	HTTP_ENDPOINT_STATS_STATE: HTTP_ENDPOINT_STATS_STATE_METRIC,
}

var httpEndpointPoolStatsDeltaMetricsNameMap = map[int]string{
	HTTP_ENDPOINT_POOL_STATS_NO_HEALTHY_EP_ERROR_COUNT: HTTP_ENDPOINT_POOL_STATS_NO_HEALTHY_EP_ERROR_DELTA_METRIC,
}

var httpEndpointPoolStatsMetricsNameMap = map[int]string{
	HTTP_ENDPOINT_POOL_STATS_HEALTHY_ROTATE_COUNT: HTTP_ENDPOINT_POOL_STATS_HEALTHY_ROTATE_COUNT_METRIC,
}

type httpEndpointPoolStatsIndexMetricMap map[int][]byte

type HttpEndpointPoolInternalMetrics struct {
	// Internal metrics, for common values:
	internalMetrics *InternalMetrics
	// Dual storage for snapping the stats, used as current, previous, toggled
	// after every metrics generation:
	stats [2]*HttpEndpointPoolStats
	// The current index:
	currIndex int
	// Cache the full metrics for each url# and stats index:
	httpEndpointDeltaMetricsCache map[string]httpEndpointPoolStatsIndexMetricMap
	httpEndpointMetricsCache      map[string]httpEndpointPoolStatsIndexMetricMap
	// Cache the full metrics for pool stats:
	httpEndpointPoolDeltaMetricsCache httpEndpointPoolStatsIndexMetricMap
	httpEndpointPoolMetricsCache      httpEndpointPoolStatsIndexMetricMap
	// A buffer for the timestamp suffix:
	tsSuffixBuf *bytes.Buffer
}

func NewHttpEndpointPoolInternalMetrics(internalMetrics *InternalMetrics) *HttpEndpointPoolInternalMetrics {
	return &HttpEndpointPoolInternalMetrics{
		internalMetrics:               internalMetrics,
		httpEndpointDeltaMetricsCache: make(map[string]httpEndpointPoolStatsIndexMetricMap),
		httpEndpointMetricsCache:      make(map[string]httpEndpointPoolStatsIndexMetricMap),
		tsSuffixBuf:                   &bytes.Buffer{},
	}
}

func (eppim *HttpEndpointPoolInternalMetrics) updatePoolMetricsCache() {
	instance, hostname := GlobalInstance, GlobalHostname
	if eppim.internalMetrics.instance != "" {
		instance = eppim.internalMetrics.instance
	}
	if eppim.internalMetrics.hostname != "" {
		hostname = eppim.internalMetrics.hostname
	}
	eppim.httpEndpointPoolDeltaMetricsCache = make(httpEndpointPoolStatsIndexMetricMap)
	for index, name := range httpEndpointPoolStatsDeltaMetricsNameMap {
		eppim.httpEndpointPoolDeltaMetricsCache[index] = []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
		))
	}
	eppim.httpEndpointPoolMetricsCache = make(httpEndpointPoolStatsIndexMetricMap)
	for index, name := range httpEndpointPoolStatsMetricsNameMap {
		eppim.httpEndpointPoolMetricsCache[index] = []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
		))
	}
}

func (eppim *HttpEndpointPoolInternalMetrics) updateEPMetricsCache(url string) {
	instance, hostname := GlobalInstance, GlobalHostname
	if eppim.internalMetrics.instance != "" {
		instance = eppim.internalMetrics.instance
	}
	if eppim.internalMetrics.hostname != "" {
		hostname = eppim.internalMetrics.hostname
	}

	indexMetricMap := make(httpEndpointPoolStatsIndexMetricMap)
	for index, name := range httpEndpointStatsDeltaMetricsNameMap {
		metric := fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			HTTP_ENDPOINT_URL_LABEL_NAME, url,
		)
		indexMetricMap[index] = []byte(metric)
	}
	eppim.httpEndpointDeltaMetricsCache[url] = indexMetricMap
	indexMetricMap = make(httpEndpointPoolStatsIndexMetricMap)
	for index, name := range httpEndpointStatsMetricsNameMap {
		metric := fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			HTTP_ENDPOINT_URL_LABEL_NAME, url,
		)
		indexMetricMap[index] = []byte(metric)
	}
	eppim.httpEndpointMetricsCache[url] = indexMetricMap
}

func (eppim *HttpEndpointPoolInternalMetrics) generateMetrics(
	buf *bytes.Buffer, tsSuffix []byte,
) int {

	if tsSuffix == nil {
		// This should happen only during unit testing:
		tsSuffix = eppim.internalMetrics.getTsSuffix()
	}

	metricsCount := 0

	currEPPoolStats, prevEPPoolStats := eppim.stats[eppim.currIndex], eppim.stats[1-eppim.currIndex]

	var prevPoolStats []uint64
	currPoolStats := currEPPoolStats.PoolStats
	if prevEPPoolStats != nil {
		prevPoolStats = prevEPPoolStats.PoolStats
	}

	indexMetricMap := eppim.httpEndpointPoolDeltaMetricsCache
	if indexMetricMap == nil {
		// N.B. This will update all the other metrics caches!
		eppim.updatePoolMetricsCache()
		indexMetricMap = eppim.httpEndpointPoolDeltaMetricsCache
	}
	for index, metric := range indexMetricMap {
		val := currPoolStats[index]
		if prevPoolStats != nil {
			val -= prevPoolStats[index]
		}
		buf.Write(metric)
		buf.WriteString(strconv.FormatUint(val, 10))
		buf.Write(tsSuffix)
		metricsCount++
	}

	indexMetricMap = eppim.httpEndpointPoolMetricsCache
	for index, metric := range indexMetricMap {
		val := currPoolStats[index]
		buf.Write(metric)
		buf.WriteString(strconv.FormatUint(val, 10))
		buf.Write(tsSuffix)
		metricsCount++
	}

	var prevEPStats HttpEndpointStats
	for url, currEPStats := range currEPPoolStats.EndpointStats {
		if prevEPPoolStats != nil {
			prevEPStats = prevEPPoolStats.EndpointStats[url]
		} else {
			prevEPStats = nil
		}
		indexMetricMap := eppim.httpEndpointDeltaMetricsCache[url]
		if indexMetricMap == nil {
			// N.B. This will update all the other metrics cache for this URL!
			eppim.updateEPMetricsCache(url)
			indexMetricMap = eppim.httpEndpointDeltaMetricsCache[url]
		}
		for index, metric := range indexMetricMap {
			val := currEPStats[index]
			if prevEPStats != nil {
				val -= prevEPStats[index]
			}
			buf.Write(metric)
			buf.WriteString(strconv.FormatUint(val, 10))
			buf.Write(tsSuffix)
			metricsCount++
		}
		indexMetricMap = eppim.httpEndpointMetricsCache[url]
		for index, metric := range indexMetricMap {
			val := currEPStats[index]
			buf.Write(metric)
			buf.WriteString(strconv.FormatUint(val, 10))
			buf.Write(tsSuffix)
			metricsCount++
		}
	}

	// Flip the stats storage:
	eppim.currIndex = 1 - eppim.currIndex

	return metricsCount
}
