// lsvmi process proper metrics

package lsvmi

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/utils"
	"github.com/eparparita/linux-stats-victoriametrics-importer/procfs"
)

// Generate basic process metrics such as memory and CPU utilization for for
// this process:

const (
	LSVMI_OS_VSIZE_METRIC       = "lsvmi_os_vsize"
	LSVMI_OS_RSS_METRIC         = "lsvmi_os_rss"
	LSVMI_OS_PCPU_METRIC        = "lsvmi_os_pcpu"
	LSVMI_OS_NUM_THREADS_METRIC = "lsvmi_os_num_threads"
)

type OsInternalMetrics struct {
	// Internal metrics, for common values:
	internalMetrics *InternalMetrics
	// Dual storage for snapping the stats, used as current, previous, toggled
	// after every metrics generation:
	pidStat [2]*procfs.PidStat
	// When the stats were collected:
	statsTs [2]time.Time
	// The current index:
	crtIndex int
	// Own PID:
	pid int
	// Page size:
	pagesize uint64
	// Metrics cache:
	vszMetric, rssMetric, pcpuMetric, numThreadsMetric []byte
}

func NewOsInternalMetrics(internalMetrics *InternalMetrics) *OsInternalMetrics {
	if utils.OSName != "linux" {
		internalMetricsLog.Warnf(
			"OS internal metrics not supported for %q", utils.OSName,
		)
		return nil
	}

	return &OsInternalMetrics{
		internalMetrics: internalMetrics,
		pid:             os.Getpid(),
		pagesize:        uint64(os.Getpagesize()),
	}
}

func (osim *OsInternalMetrics) SnapStats() {
	pidStat := osim.pidStat[osim.crtIndex]
	if pidStat == nil {
		procfsRoot := GlobalProcfsRoot
		if osim.internalMetrics.procfsRoot != "" {
			procfsRoot = osim.internalMetrics.procfsRoot
		}
		pidStat = procfs.NewPidStat(procfsRoot, osim.pid, procfs.PID_STAT_PID_ONLY_TID)
		osim.pidStat[osim.crtIndex] = pidStat
	}
	err := pidStat.Parse(nil)
	statsTs := time.Now()
	if err != nil {
		internalMetricsLog.Warnf("pidStat.Parse(pid=%d): %v", osim.pid, err)
		osim.pidStat[osim.crtIndex] = nil
	}
	osim.statsTs[osim.crtIndex] = statsTs
}

func (osim *OsInternalMetrics) updateMetricsCache() {
	instance, hostname := GlobalInstance, GlobalHostname
	if osim.internalMetrics.instance != "" {
		instance = osim.internalMetrics.instance
	}
	if osim.internalMetrics.hostname != "" {
		hostname = osim.internalMetrics.hostname
	}
	osim.vszMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
		LSVMI_OS_VSIZE_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))
	osim.rssMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
		LSVMI_OS_RSS_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))
	osim.pcpuMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
		LSVMI_OS_PCPU_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))
	osim.numThreadsMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
		LSVMI_OS_NUM_THREADS_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))

}

func (osim *OsInternalMetrics) generateMetrics(
	buf *bytes.Buffer, tsSuffix []byte,
) int {

	crtPidStat, prevPidStat := osim.pidStat[osim.crtIndex], osim.pidStat[1-osim.crtIndex]
	if crtPidStat == nil {
		// Cannot generate metrics since stats couldn't be collected:
		return 0
	}

	if tsSuffix == nil {
		// This should happen only during unit testing:
		tsSuffix = osim.internalMetrics.getTsSuffix()
	}

	if osim.vszMetric == nil {
		// This will update all metrics:
		osim.updateMetricsCache()
	}

	metricsCount := 0

	buf.Write(osim.vszMetric)
	buf.Write(crtPidStat.ByteSliceFields[procfs.PID_STAT_VSIZE])
	buf.Write(tsSuffix)
	metricsCount++

	buf.Write(osim.rssMetric)
	rss := uint64(0)
	for _, c := range crtPidStat.ByteSliceFields[procfs.PID_STAT_RSS] {
		rss = (rss << 3) + (rss << 1) + uint64(c-'0')
	}
	rss *= osim.pagesize
	buf.WriteString(strconv.FormatUint(rss, 10))
	buf.Write(tsSuffix)
	metricsCount++

	buf.Write(osim.numThreadsMetric)
	buf.Write(crtPidStat.ByteSliceFields[procfs.PID_STAT_NUM_THREADS])
	buf.Write(tsSuffix)
	metricsCount++

	if prevPidStat != nil {
		dTime := osim.statsTs[osim.crtIndex].Sub(osim.statsTs[1-osim.crtIndex]).Seconds()
		dTimeCpu := float64(
			crtPidStat.NumericFields[procfs.PID_STAT_UTIME]+
				crtPidStat.NumericFields[procfs.PID_STAT_STIME]-
				prevPidStat.NumericFields[procfs.PID_STAT_UTIME]-
				prevPidStat.NumericFields[procfs.PID_STAT_STIME]) *
			utils.LinuxClktckSec
		pcpu := dTimeCpu / dTime * 100
		buf.Write(osim.pcpuMetric)
		buf.WriteString(strconv.FormatFloat(pcpu, 'f', 1, 64))
		buf.Write(tsSuffix)
		metricsCount++
	}

	// Flip the stats storage:
	osim.crtIndex = 1 - osim.crtIndex

	return metricsCount
}
