// Internal metrics for HTTP Endpoint Pool

package lsvmi

import (
	"bytes"
	"fmt"
	"strconv"
)

const (
	HTTP_ENDPOINT_STATS_SEND_BUFFER_COUNT_DELTA_METRIC        = "lsvmi_http_ep_send_buffer_count_delta"
	HTTP_ENDPOINT_STATS_SEND_BUFFER_BYTE_COUNT_DELTA_METRIC   = "lsvmi_http_ep_send_buffer_byte_count_delta"
	HTTP_ENDPOINT_STATS_SEND_BUFFER_ERROR_COUNT_DELTA_METRIC  = "lsvmi_http_ep_send_buffer_error_count_delta"
	HTTP_ENDPOINT_STATS_HEALTH_CHECK_COUNT_DELTA_METRIC       = "lsvmi_http_ep_healthcheck_count_delta"
	HTTP_ENDPOINT_STATS_HEALTH_CHECK_ERROR_COUNT_DELTA_METRIC = "lsvmi_http_ep_healthcheck_error_count_delta"

	HTTP_ENDPOINT_POOL_STATS_HEALTHY_ROTATE_COUNT_DELTA_METRIC      = "lsvmi_http_ep_pool_healthy_rotate_count_delta"
	HTTP_ENDPOINT_POOL_STATS_NO_HEALTHY_EP_ERROR_COUNT_DELTA_METRIC = "lsvmi_http_ep_pool_no_healthy_ep_error_count_delta"

	HTTP_ENDPOINT_URL_LABEL_NAME = "url"
)

var httpEndpointStatsMetricsNameMap = map[int]string{
	HTTP_ENDPOINT_STATS_SEND_BUFFER_COUNT:        HTTP_ENDPOINT_STATS_SEND_BUFFER_COUNT_DELTA_METRIC,
	HTTP_ENDPOINT_STATS_SEND_BUFFER_BYTE_COUNT:   HTTP_ENDPOINT_STATS_SEND_BUFFER_BYTE_COUNT_DELTA_METRIC,
	HTTP_ENDPOINT_STATS_SEND_BUFFER_ERROR_COUNT:  HTTP_ENDPOINT_STATS_SEND_BUFFER_ERROR_COUNT_DELTA_METRIC,
	HTTP_ENDPOINT_STATS_HEALTH_CHECK_COUNT:       HTTP_ENDPOINT_STATS_HEALTH_CHECK_COUNT_DELTA_METRIC,
	HTTP_ENDPOINT_STATS_HEALTH_CHECK_ERROR_COUNT: HTTP_ENDPOINT_STATS_HEALTH_CHECK_ERROR_COUNT_DELTA_METRIC,
}

var httpEndpointPoolStatsMetricsNameMap = map[int]string{
	HTTP_ENDPOINT_POOL_STATS_HEALTHY_ROTATE_COUNT:      HTTP_ENDPOINT_POOL_STATS_HEALTHY_ROTATE_COUNT_DELTA_METRIC,
	HTTP_ENDPOINT_POOL_STATS_NO_HEALTHY_EP_ERROR_COUNT: HTTP_ENDPOINT_POOL_STATS_NO_HEALTHY_EP_ERROR_COUNT_DELTA_METRIC,
}

type httpEndpointPoolStatsIndexMetricMap map[int][]byte

type HttpEndpointPoolInternalMetrics struct {
	// Internal metrics, for common values:
	internalMetrics *InternalMetrics
	// Dual buffer holding current, previous delta stats:
	stats [2]*HttpEndpointPoolStats
	// Which one is current:
	crtStatsIndx int
	// Cache the full metrics for each url# and stats index:
	httpEndpointMetricsCache map[string]httpEndpointPoolStatsIndexMetricMap
	// Cache the full metrics for pool stats:
	httpEndpointPoolMetricsCache httpEndpointPoolStatsIndexMetricMap
	// A buffer for the timestamp suffix:
	tsSuffixBuf *bytes.Buffer
}

func NewHttpEndpointPoolInternalMetrics(internalMetrics *InternalMetrics) *HttpEndpointPoolInternalMetrics {
	return &HttpEndpointPoolInternalMetrics{
		internalMetrics:              internalMetrics,
		httpEndpointMetricsCache:     make(map[string]httpEndpointPoolStatsIndexMetricMap),
		httpEndpointPoolMetricsCache: nil, // to force an update
		tsSuffixBuf:                  &bytes.Buffer{},
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
	buf *bytes.Buffer, fullCycle bool, tsSuffix []byte,
) int {
	crtStatsIndx := eppim.crtStatsIndx
	crtStats, prevStats := eppim.stats[crtStatsIndx], eppim.stats[1-crtStatsIndx]
	if fullCycle {
		prevStats = nil
	}

	if tsSuffix == nil {
		// This should happen only during unit testing:
		tsSuffix = eppim.internalMetrics.getTsSuffix()
	}

	metricsCount := 0

	var prevPoolStats []uint64
	crtPoolStats := crtStats.Stats
	if prevStats != nil {
		prevPoolStats = prevStats.Stats
	} else {
		prevPoolStats = nil
	}

	// For counter delta metrics, unless this is a full cycle, skip 0 values if
	// the previous scan value was also 0.
	indexMetricMap := eppim.httpEndpointPoolMetricsCache
	if indexMetricMap == nil {
		eppim.updatePoolMetricsCache()
		indexMetricMap = eppim.httpEndpointPoolMetricsCache
	}
	for index, metric := range indexMetricMap {
		crtVal := crtPoolStats[index]
		if crtVal != 0 || prevPoolStats == nil || crtVal != prevPoolStats[index] {
			buf.Write(metric)
			buf.WriteString(strconv.FormatUint(crtVal, 10))
			buf.Write(tsSuffix)
			metricsCount++
		}
	}

	for url, crtEpStats := range crtStats.EndpointStats {
		var prevEpStats HttpEndpointStats
		if prevStats != nil {
			prevEpStats = prevStats.EndpointStats[url]
		} else {
			prevEpStats = nil
		}

		indexMetricMap := eppim.httpEndpointMetricsCache[url]
		if indexMetricMap == nil {
			eppim.updateEPMetricsCache(url)
			indexMetricMap = eppim.httpEndpointMetricsCache[url]
		}
		for index, metric := range indexMetricMap {
			crtVal := crtEpStats[index]
			if crtVal != 0 || prevEpStats == nil || crtVal != prevEpStats[index] {
				buf.Write(metric)
				buf.WriteString(strconv.FormatUint(crtVal, 10))
				buf.Write(tsSuffix)
				metricsCount++
			}
		}
	}

	// Flip the buffers:
	eppim.crtStatsIndx = 1 - crtStatsIndx

	return metricsCount
}
