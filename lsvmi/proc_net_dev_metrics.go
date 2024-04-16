// /proc/net/dev metrics

package lsvmi

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"
)

const (
	PROC_NET_DEV_METRICS_CONFIG_INTERVAL_DEFAULT            = "1s"
	PROC_NET_DEV_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT = 15

	// This generator id:
	PROC_NET_DEV_METRICS_ID = "proc_net_dev_metrics"
)

const (
	// METRIC{instance="INSTANCE",hostname="HOSTNAME",PROC_NET_DEV_LABEL_NAME="DEV"}:
	PROC_NET_DEV_RX_RATE_METRIC             = "proc_net_dev_rx_kbps"
	PROC_NET_DEV_RX_PACKETS_DELTA_METRIC    = "proc_net_dev_rx_pkts_delta"
	PROC_NET_DEV_RX_ERRS_DELTA_METRIC       = "proc_net_dev_rx_errs_delta"
	PROC_NET_DEV_RX_DROP_DELTA_METRIC       = "proc_net_dev_rx_drop_delta"
	PROC_NET_DEV_RX_FIFO_DELTA_METRIC       = "proc_net_dev_rx_fifo_delta"
	PROC_NET_DEV_RX_FRAME_DELTA_METRIC      = "proc_net_dev_rx_frame_delta"
	PROC_NET_DEV_RX_COMPRESSED_DELTA_METRIC = "proc_net_dev_rx_compressed_delta"
	PROC_NET_DEV_RX_MULTICAST_DELTA_METRIC  = "proc_net_dev_rx_mcast_delta"
	PROC_NET_DEV_TX_RATE_METRIC             = "proc_net_dev_tx_kbps"
	PROC_NET_DEV_TX_PACKETS_DELTA_METRIC    = "proc_net_dev_tx_pkts_delta"
	PROC_NET_DEV_TX_ERRS_DELTA_METRIC       = "proc_net_dev_tx_errs_delta"
	PROC_NET_DEV_TX_DROP_DELTA_METRIC       = "proc_net_dev_tx_drop_delta"
	PROC_NET_DEV_TX_FIFO_DELTA_METRIC       = "proc_net_dev_tx_fifo_delta"
	PROC_NET_DEV_TX_COLLS_DELTA_METRIC      = "proc_net_dev_tx_colls_delta"
	PROC_NET_DEV_TX_CARRIER_DELTA_METRIC    = "proc_net_dev_tx_carrier_delta"
	PROC_NET_DEV_TX_COMPRESSED_DELTA_METRIC = "proc_net_dev_tx_compressed_delta"

	PROC_NET_DEV_LABEL_NAME = "dev"

	// Interval since last generation, i.e. the interval underlying the deltas.
	// Normally this should be close to scan interval, but this is the actual
	// value, rather than the desired one:
	PROC_NET_DEV_INTERVAL_METRIC_NAME = "proc_net_dev_metrics_delta_sec"
)

// Map stats index (see procfs/net_dev_parser.go) into metrics names:
var procNetDevIndexDeltaMetricNameMap = map[int]string{
	procfs.NET_DEV_RX_BYTES:      PROC_NET_DEV_RX_RATE_METRIC,
	procfs.NET_DEV_RX_PACKETS:    PROC_NET_DEV_RX_PACKETS_DELTA_METRIC,
	procfs.NET_DEV_RX_ERRS:       PROC_NET_DEV_RX_ERRS_DELTA_METRIC,
	procfs.NET_DEV_RX_DROP:       PROC_NET_DEV_RX_DROP_DELTA_METRIC,
	procfs.NET_DEV_RX_FIFO:       PROC_NET_DEV_RX_FIFO_DELTA_METRIC,
	procfs.NET_DEV_RX_FRAME:      PROC_NET_DEV_RX_FRAME_DELTA_METRIC,
	procfs.NET_DEV_RX_COMPRESSED: PROC_NET_DEV_RX_COMPRESSED_DELTA_METRIC,
	procfs.NET_DEV_RX_MULTICAST:  PROC_NET_DEV_RX_MULTICAST_DELTA_METRIC,
	procfs.NET_DEV_TX_BYTES:      PROC_NET_DEV_TX_RATE_METRIC,
	procfs.NET_DEV_TX_PACKETS:    PROC_NET_DEV_TX_PACKETS_DELTA_METRIC,
	procfs.NET_DEV_TX_ERRS:       PROC_NET_DEV_TX_ERRS_DELTA_METRIC,
	procfs.NET_DEV_TX_DROP:       PROC_NET_DEV_TX_DROP_DELTA_METRIC,
	procfs.NET_DEV_TX_FIFO:       PROC_NET_DEV_TX_FIFO_DELTA_METRIC,
	procfs.NET_DEV_TX_COLLS:      PROC_NET_DEV_TX_COLLS_DELTA_METRIC,
	procfs.NET_DEV_TX_CARRIER:    PROC_NET_DEV_TX_CARRIER_DELTA_METRIC,
	procfs.NET_DEV_TX_COMPRESSED: PROC_NET_DEV_TX_COMPRESSED_DELTA_METRIC,
}

// Certain values are used to generate rates:
type ProcNetDevRate struct {
	factor float64 // dVal/dTime * factor
	prec   int     // FormatFloat prec arg
}

var procNetDevIndexRate = [procfs.NET_DEV_NUM_STATS]*ProcNetDevRate{
	procfs.NET_DEV_RX_BYTES: {8. / 1000., 1},
	procfs.NET_DEV_TX_BYTES: {8. / 1000., 1},
}

var procNetDevMetricsLog = NewCompLogger(PROC_NET_DEV_METRICS_ID)

type ProcNetDevMetricsConfig struct {
	// How often to generate the metrics in time.ParseDuration() format:
	Interval string `yaml:"interval"`
	// Normally metrics are generated only if there is a change in value from
	// the previous scan. However every N cycles the full set is generated. Use
	// 0 to generate full metrics every cycle.
	FullMetricsFactor int `yaml:"full_metrics_factor"`
}

func DefaultProcNetDevMetricsConfig() *ProcNetDevMetricsConfig {
	return &ProcNetDevMetricsConfig{
		Interval:          PROC_NET_DEV_METRICS_CONFIG_INTERVAL_DEFAULT,
		FullMetricsFactor: PROC_NET_DEV_METRICS_CONFIG_FULL_METRICS_FACTOR_DEFAULT,
	}
}

type ProcNetDevMetrics struct {
	// id/task_id:
	id string
	// Scan interval:
	interval time.Duration
	// Dual storage for parsed stats used as previous, current:
	procNetDev [2]*procfs.NetDev
	// Timestamp when the stats were collected:
	procNetDevTs [2]time.Time
	// Index for current stats, toggled after each use:
	crtIndex int
	// Current cycle#:
	cycleNum int
	// Full metric factor:
	fullMetricsFactor int

	// Delta/rate metrics are generated with skip-zero-after-zero rule, i.e.
	// if the current and previous deltas are both zero, then the current metric
	// is skipped, save for full cycles. Keep track of zero deltas,
	// indexed by device and procfs.NET_DEV_...:
	zeroDeltaMap map[string][]bool

	// Metrics cache, indexed by device and procfs.NET_DEV_...:
	deltaMetricsCache map[string][][]byte

	// Interval metric:
	intervalMetric []byte

	// A buffer for the timestamp suffix:
	tsSuffixBuf *bytes.Buffer

	// The following are needed for testing only. Left to their default values,
	// the usual objects will be used.
	instance, hostname string
	timeNowFn          func() time.Time
	metricsQueue       MetricsQueue
	procfsRoot         string
}

func NewProcNetDevMetrics(cfg any) (*ProcNetDevMetrics, error) {
	var (
		err                  error
		procNetDevMetricsCfg *ProcNetDevMetricsConfig
	)

	switch cfg := cfg.(type) {
	case *LsvmiConfig:
		procNetDevMetricsCfg = cfg.ProcNetDevMetricsConfig
	case *ProcNetDevMetricsConfig:
		procNetDevMetricsCfg = cfg
	case nil:
		procNetDevMetricsCfg = DefaultProcNetDevMetricsConfig()
	default:
		return nil, fmt.Errorf("NewProcNetDevMetrics: %T invalid config type", cfg)
	}

	interval, err := time.ParseDuration(procNetDevMetricsCfg.Interval)
	if err != nil {
		return nil, err
	}
	procNetDevMetrics := &ProcNetDevMetrics{
		id:                PROC_NET_DEV_METRICS_ID,
		interval:          interval,
		zeroDeltaMap:      make(map[string][]bool),
		deltaMetricsCache: make(map[string][][]byte),
		fullMetricsFactor: procNetDevMetricsCfg.FullMetricsFactor,
		tsSuffixBuf:       &bytes.Buffer{},
	}

	procNetDevMetricsLog.Infof("id=%s", procNetDevMetrics.id)
	procNetDevMetricsLog.Infof("interval=%s", procNetDevMetrics.interval)
	procNetDevMetricsLog.Infof("full_metrics_factor=%d", procNetDevMetrics.fullMetricsFactor)
	return procNetDevMetrics, nil
}

func (pndm *ProcNetDevMetrics) updateDeltaMetricsCache(dev string) {
	instance, hostname := GlobalInstance, GlobalHostname
	if pndm.instance != "" {
		instance = pndm.instance
	}
	if pndm.hostname != "" {
		hostname = pndm.hostname
	}

	deltaMetrics := make([][]byte, procfs.NET_DEV_NUM_STATS)
	for index, name := range procNetDevIndexDeltaMetricNameMap {
		deltaMetrics[index] = []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. the space before the value is included!
			name,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			PROC_NET_DEV_LABEL_NAME, dev,
		))
	}
	pndm.deltaMetricsCache[dev] = deltaMetrics

}

func (pndm *ProcNetDevMetrics) updateMetricsCache() {
	instance, hostname := GlobalInstance, GlobalHostname
	if pndm.instance != "" {
		instance = pndm.instance
	}
	if pndm.hostname != "" {
		hostname = pndm.hostname
	}
	pndm.intervalMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. include space before val
		PROC_NET_DEV_INTERVAL_METRIC_NAME,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))

}

func (pndm *ProcNetDevMetrics) generateMetrics(buf *bytes.Buffer) (int, int) {
	actualMetricsCount := 0
	crtProcNetDev, prevProcNetDev := pndm.procNetDev[pndm.crtIndex], pndm.procNetDev[1-pndm.crtIndex]
	if prevProcNetDev != nil {
		crtTs, prevTs := pndm.procNetDevTs[pndm.crtIndex], pndm.procNetDevTs[1-pndm.crtIndex]
		pndm.tsSuffixBuf.Reset()
		fmt.Fprintf(
			pndm.tsSuffixBuf, " %d\n", crtTs.UnixMilli(),
		)
		promTs := pndm.tsSuffixBuf.Bytes()

		fullMetrics := pndm.cycleNum == 0
		deltaSec := crtTs.Sub(prevTs).Seconds()
		for dev, crtDevStats := range crtProcNetDev.DevStats {
			prevDevStats := prevProcNetDev.DevStats[dev]
			if prevDevStats == nil {
				continue
			}
			deltaMetrics := pndm.deltaMetricsCache[dev]
			if deltaMetrics == nil {
				pndm.updateDeltaMetricsCache(dev)
				deltaMetrics = pndm.deltaMetricsCache[dev]
			}
			zeroDelta := pndm.zeroDeltaMap[dev]
			if zeroDelta == nil {
				zeroDelta = make([]bool, procfs.NET_DEV_NUM_STATS)
				pndm.zeroDeltaMap[dev] = zeroDelta
			}

			for index, metric := range deltaMetrics {
				val := crtDevStats[index] - prevDevStats[index]
				if val != 0 || fullMetrics || !zeroDelta[index] {
					buf.Write(metric)
					rate := procNetDevIndexRate[index]
					if rate != nil {
						buf.WriteString(strconv.FormatFloat(
							float64(val)/deltaSec*rate.factor, 'f', rate.prec, 64,
						))
					} else {
						buf.WriteString(strconv.FormatUint(val, 10))
					}
					buf.Write(promTs)
					actualMetricsCount++
				}
				zeroDelta[index] = val == 0
			}
		}

		// Network devices may be created/enabled dynamically. Check and delete zero
		// flags as needed.
		if len(pndm.zeroDeltaMap) > len(crtProcNetDev.DevStats) {
			for dev := range pndm.zeroDeltaMap {
				if _, ok := crtProcNetDev.DevStats[dev]; !ok {
					delete(pndm.zeroDeltaMap, dev)
				}
			}
		}

		if pndm.intervalMetric == nil {
			pndm.updateMetricsCache()
		}
		buf.Write(pndm.intervalMetric)
		buf.WriteString(strconv.FormatFloat(deltaSec, 'f', 6, 64))
		buf.Write(promTs)
		actualMetricsCount++
	}

	// The total number of metrics:
	//		delta metrics#: number of dev * number of counters
	//		interval metric#: 1
	totalMetricsCount := len(crtProcNetDev.DevStats)*procfs.NET_DEV_NUM_STATS + 1

	// Toggle the buffers, update the collection time and the cycle#:
	pndm.crtIndex = 1 - pndm.crtIndex
	if pndm.cycleNum++; pndm.cycleNum >= pndm.fullMetricsFactor {
		pndm.cycleNum = 0
	}

	return actualMetricsCount, totalMetricsCount
}

// Satisfy the TaskActivity interface:
func (pndm *ProcNetDevMetrics) Execute() bool {
	timeNowFn := time.Now
	if pndm.timeNowFn != nil {
		timeNowFn = pndm.timeNowFn
	}

	metricsQueue := GlobalMetricsQueue
	if pndm.metricsQueue != nil {
		metricsQueue = pndm.metricsQueue
	}

	crtProcNetDev := pndm.procNetDev[pndm.crtIndex]
	if crtProcNetDev == nil {
		prevProcNetDev := pndm.procNetDev[1-pndm.crtIndex]
		if prevProcNetDev != nil {
			crtProcNetDev = prevProcNetDev.Clone(false)
		} else {
			procfsRoot := GlobalProcfsRoot
			if pndm.procfsRoot != "" {
				procfsRoot = pndm.procfsRoot
			}
			crtProcNetDev = procfs.NewNetDev(procfsRoot)
		}
		pndm.procNetDev[pndm.crtIndex] = crtProcNetDev
	}
	err := crtProcNetDev.Parse()
	if err != nil {
		procNetDevMetricsLog.Warnf("%v: proc net dev metrics will be disabled", err)
		return false
	}
	pndm.procNetDevTs[pndm.crtIndex] = timeNowFn()

	buf := metricsQueue.GetBuf()
	actualMetricsCount, totalMetricsCount := pndm.generateMetrics(buf)
	byteCount := buf.Len()
	metricsQueue.QueueBuf(buf)

	GlobalMetricsGeneratorStatsContainer.Update(
		pndm.id, uint64(actualMetricsCount), uint64(totalMetricsCount), uint64(byteCount),
	)

	return true
}

// Define and register the task builder:
func ProcNetDevMetricsTaskBuilder(cfg *LsvmiConfig) ([]*Task, error) {
	pndm, err := NewProcNetDevMetrics(cfg)
	if err != nil {
		return nil, err
	}
	if pndm.interval <= 0 {
		procNetDevMetricsLog.Infof(
			"interval=%s, metrics disabled", pndm.interval,
		)
		return nil, nil
	}
	tasks := []*Task{
		NewTask(pndm.id, pndm.interval, pndm),
	}
	return tasks, nil
}

func init() {
	TaskBuilders.Register(ProcNetDevMetricsTaskBuilder)
}
