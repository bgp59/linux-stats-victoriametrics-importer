// Globals for LSVMI

package lsvmi

var (
	GlobalConfig           *LsvmiConfig
	GlobalHttpEndpointPool *HttpEndpointPool
	GlobalCompressorPool   *CompressorPool
	GlobalMetricsQueue     MetricsQueue
	GlobalScheduler        *Scheduler
)
