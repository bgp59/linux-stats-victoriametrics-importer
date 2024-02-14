// Internal stats for http_endpoint_pool

package lsvmi

import "sync"

// Endpoint stats:
const (
	HTTP_ENDPOINT_STATS_SEND_BUFFER_COUNT = iota
	HTTP_ENDPOINT_STATS_SEND_BUFFER_BYTE_COUNT
	HTTP_ENDPOINT_STATS_SEND_BUFFER_ERROR_COUNT
	HTTP_ENDPOINT_STATS_HEALTH_CHECK_COUNT
	HTTP_ENDPOINT_STATS_HEALTH_CHECK_ERROR_COUNT
	// Must be last:
	HTTP_ENDPOINT_STATS_LEN
)

type HttpEndpointStats []uint64

// Endpoint pool stats:
const (
	HTTP_ENDPOINT_POOL_STATS_HEALTHY_ROTATE_COUNT = iota
	// Must be last:
	HTTP_ENDPOINT_POOL_STATS_LEN
)

type HttpEndpointPoolStats struct {
	Stats []uint64
	// Endpoint stats are indexed by URL:
	EndpointStats map[string]HttpEndpointStats
	// Lock:
	mu *sync.Mutex
}

func (stats *HttpEndpointPoolStats) SnapDelta(prev, delta *HttpEndpointPoolStats) {
	stats.mu.Lock()
	defer stats.mu.Unlock()

	for i, crt := range stats.Stats {
		delta.Stats[i], prev.Stats[i] = crt-prev.Stats[i], crt
	}

	for url, epStats := range stats.EndpointStats {
		prevEpStats := prev.EndpointStats[url]
		if prevEpStats == nil {
			prevEpStats = make(HttpEndpointStats, HTTP_ENDPOINT_STATS_LEN)
			prev.EndpointStats[url] = prevEpStats
		}
		deltaEpStats := delta.EndpointStats[url]
		if deltaEpStats == nil {
			deltaEpStats = make(HttpEndpointStats, HTTP_ENDPOINT_STATS_LEN)
			delta.EndpointStats[url] = deltaEpStats
		}
		for i, crt := range epStats {
			deltaEpStats[i], prevEpStats[i] = crt-prevEpStats[i], crt
		}
	}
}

func NewHttpEndpointPoolStatsNoLock() *HttpEndpointPoolStats {
	return &HttpEndpointPoolStats{
		Stats:         make([]uint64, HTTP_ENDPOINT_POOL_STATS_LEN),
		EndpointStats: make(map[string]HttpEndpointStats),
	}
}

func NewHttpEndpointPoolStats() *HttpEndpointPoolStats {
	stats := NewHttpEndpointPoolStatsNoLock()
	stats.mu = &sync.Mutex{}
	return stats
}
