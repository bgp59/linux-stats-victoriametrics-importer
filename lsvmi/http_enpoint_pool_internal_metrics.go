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
	// Storage for snapping stats:
	stats *HttpEndpointPoolStats
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

func (eppim *HttpEndpointPoolInternalMetrics) updatePoolMetricsCache() httpEndpointPoolStatsIndexMetricMap {
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
	return eppim.httpEndpointPoolMetricsCache
}

func (eppim *HttpEndpointPoolInternalMetrics) updateEPMetricsCache(url string) httpEndpointPoolStatsIndexMetricMap {
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
	return indexMetricMap
}

func (eppim *HttpEndpointPoolInternalMetrics) generateMetrics(
	buf *bytes.Buffer, tsSuffix []byte,
) int {

	if tsSuffix == nil {
		// This should happen only during unit testing:
		tsSuffix = eppim.internalMetrics.getTsSuffix()
	}

	metricsCount := 0

	indexMetricMap := eppim.httpEndpointPoolMetricsCache
	if indexMetricMap == nil {
		indexMetricMap = eppim.updatePoolMetricsCache()
	}
	poolStats := eppim.stats.Stats
	for index, metric := range indexMetricMap {
		buf.Write(metric)
		buf.WriteString(strconv.FormatUint(poolStats[index], 10))
		buf.Write(tsSuffix)
		metricsCount++
	}

	for url, epStats := range eppim.stats.EndpointStats {
		indexMetricMap := eppim.httpEndpointMetricsCache[url]
		if indexMetricMap == nil {
			indexMetricMap = eppim.updateEPMetricsCache(url)
		}
		for index, metric := range indexMetricMap {
			buf.Write(metric)
			buf.WriteString(strconv.FormatUint(epStats[index], 10))
			buf.Write(tsSuffix)
			metricsCount++
		}
	}

	return metricsCount
}
