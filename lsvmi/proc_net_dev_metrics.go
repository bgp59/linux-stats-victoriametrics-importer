// /proc/net/dev metrics

package lsvmi

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"
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

	PROC_NET_DEV_PRESENCE_METRIC = "proc_net_dev_present"

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

// Per dev info, grouped together to ensure a single lookup:
type ProcNetDevInfo struct {
	// Delta metrics cache, indexed by procfs.NET_DEV_...:
	deltaMetrics [][]byte

	// Presence metric:
	presentMetric []byte

	// Cycle#:
	cycleNum int

	// Delta/rate metrics are generated with skip-zero-after-zero rule, i.e. if
	// the current and previous deltas are both zero, then the current metric is
	// skipped, save for full cycles. Keep track of zero deltas, indexed by
	// procfs.NET_DEV_...:
	zeroDelta []bool
}

type ProcNetDevMetrics struct {
	// id/task_id:
	id string

	// Scan interval:
	interval time.Duration

	// Full metric factor:
	fullMetricsFactor int

	// Dual storage for parsed stats used as previous, current:
	procNetDev [2]*procfs.NetDev
	// Timestamp when the stats were collected:
	procNetDevTs [2]time.Time
	// Index for current stats, toggled after each use:
	currIndex int

	// Device info, indexed by device:
	devInfoMap map[string]*ProcNetDevInfo

	// Interval metric:
	intervalMetric []byte

	// A buffer for the timestamp suffix:
	tsSuffixBuf *bytes.Buffer

	// The total number of metrics, evaluated every time there is a change:
	totalMetricsCount int

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
		fullMetricsFactor: procNetDevMetricsCfg.FullMetricsFactor,
		devInfoMap:        make(map[string]*ProcNetDevInfo),
		tsSuffixBuf:       &bytes.Buffer{},
	}

	procNetDevMetricsLog.Infof("id=%s", procNetDevMetrics.id)
	procNetDevMetricsLog.Infof("interval=%s", procNetDevMetrics.interval)
	procNetDevMetricsLog.Infof("full_metrics_factor=%d", procNetDevMetrics.fullMetricsFactor)
	return procNetDevMetrics, nil
}

func (pndm *ProcNetDevMetrics) updateDevInfo(dev string) {
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
	pndm.devInfoMap[dev] = &ProcNetDevInfo{
		deltaMetrics: deltaMetrics,
		presentMetric: []byte(fmt.Sprintf(
			`%s{%s="%s",%s="%s",%s="%s"} `, // N.B. the space before the value is included!
			PROC_NET_DEV_PRESENCE_METRIC,
			INSTANCE_LABEL_NAME, instance,
			HOSTNAME_LABEL_NAME, hostname,
			PROC_NET_DEV_LABEL_NAME, dev,
		)),
		cycleNum:  initialCycleNum.Get(pndm.fullMetricsFactor),
		zeroDelta: make([]bool, procfs.NET_DEV_NUM_STATS),
	}
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
	currProcNetDev, prevProcNetDev := pndm.procNetDev[pndm.currIndex], pndm.procNetDev[1-pndm.currIndex]
	currTs, prevTs := pndm.procNetDevTs[pndm.currIndex], pndm.procNetDevTs[1-pndm.currIndex]
	pndm.currIndex = 1 - pndm.currIndex

	// All metrics are delta, a previous state is required:
	if prevProcNetDev == nil {
		return 0, 0
	}

	actualMetricsCount := 0
	pndm.tsSuffixBuf.Reset()
	fmt.Fprintf(
		pndm.tsSuffixBuf, " %d\n", currTs.UnixMilli(),
	)
	promTs := pndm.tsSuffixBuf.Bytes()
	deltaSec := currTs.Sub(prevTs).Seconds()
	evalTotalMetricsCount := false
	for dev, currDevStats := range currProcNetDev.DevStats {
		prevDevStats := prevProcNetDev.DevStats[dev]
		if prevDevStats == nil {
			continue
		}

		devInfo := pndm.devInfoMap[dev]
		fullMetrics := devInfo == nil || devInfo.cycleNum == 0
		if devInfo == nil {
			pndm.updateDevInfo(dev)
			devInfo = pndm.devInfoMap[dev]
			evalTotalMetricsCount = true
		}
		deltaMetrics := devInfo.deltaMetrics
		zeroDelta := devInfo.zeroDelta

		for index, metric := range deltaMetrics {
			val := currDevStats[index] - prevDevStats[index]
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

		if fullMetrics {
			buf.Write(devInfo.presentMetric)
			buf.WriteByte('1')
			buf.Write(promTs)
			actualMetricsCount++
		}

		if devInfo.cycleNum++; devInfo.cycleNum >= pndm.fullMetricsFactor {
			devInfo.cycleNum = 0
		}
	}

	// Network devices may be created/enabled dynamically, remove out of
	// scope ones:
	if len(pndm.devInfoMap) > len(currProcNetDev.DevStats) {
		evalTotalMetricsCount = true
		for dev, devInfo := range pndm.devInfoMap {
			if _, ok := currProcNetDev.DevStats[dev]; !ok {
				buf.Write(devInfo.presentMetric)
				buf.WriteByte('0')
				buf.Write(promTs)
				actualMetricsCount++
				delete(pndm.devInfoMap, dev)
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

	if evalTotalMetricsCount {
		// The total number of metrics:
		//		delta metrics#: (number of dev) * (number of counters + 1 (presence))
		//		interval metric#: 1
		pndm.totalMetricsCount = len(currProcNetDev.DevStats)*(procfs.NET_DEV_NUM_STATS+1) + 1
	}

	return actualMetricsCount, pndm.totalMetricsCount
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

	currProcNetDev := pndm.procNetDev[pndm.currIndex]
	if currProcNetDev == nil {
		prevProcNetDev := pndm.procNetDev[1-pndm.currIndex]
		if prevProcNetDev != nil {
			currProcNetDev = prevProcNetDev.Clone(false)
		} else {
			procfsRoot := GlobalProcfsRoot
			if pndm.procfsRoot != "" {
				procfsRoot = pndm.procfsRoot
			}
			currProcNetDev = procfs.NewNetDev(procfsRoot)
		}
		pndm.procNetDev[pndm.currIndex] = currProcNetDev
	}
	err := currProcNetDev.Parse()
	if err != nil {
		procNetDevMetricsLog.Warnf("%v: proc net dev metrics will be disabled", err)
		return false
	}
	pndm.procNetDevTs[pndm.currIndex] = timeNowFn()

	buf := metricsQueue.GetBuf()
	actualMetricsCount, totalMetricsCount := pndm.generateMetrics(buf)
	if totalMetricsCount > 0 {
		byteCount := buf.Len()
		metricsQueue.QueueBuf(buf)
		GlobalMetricsGeneratorStatsContainer.Update(
			pndm.id, uint64(actualMetricsCount), uint64(totalMetricsCount), uint64(byteCount),
		)
	} else {
		metricsQueue.ReturnBuf(buf)
	}

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
