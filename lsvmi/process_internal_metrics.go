// lsvmi process proper metrics

package lsvmi

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/emypar/linux-stats-victoriametrics-importer/internal/utils"
	"github.com/emypar/linux-stats-victoriametrics-importer/procfs"
)

// Generate basic process metrics such as memory and CPU utilization for for
// this process:

const (
	LSVMI_PROC_VSIZE_METRIC       = "lsvmi_proc_vsize"
	LSVMI_PROC_RSS_METRIC         = "lsvmi_proc_rss"
	LSVMI_PROC_PCPU_METRIC        = "lsvmi_proc_pcpu"
	LSVMI_PROC_NUM_THREADS_METRIC = "lsvmi_proc_num_threads"
)

type ProcessInternalMetrics struct {
	// Internal metrics, for common values:
	internalMetrics *InternalMetrics
	// Dual storage for snapping the stats, used as current, previous, toggled
	// after every metrics generation:
	pidStat    [2]procfs.PidStatParser
	pidTidPath string
	// When the stats were collected:
	statsTs [2]time.Time
	// The current index:
	currIndex int
	// Own PID:
	pid int
	// Page size:
	pagesize uint64
	// Metrics cache:
	vszMetric, rssMetric, pcpuMetric, numThreadsMetric []byte
}

func NewProcessInternalMetrics(internalMetrics *InternalMetrics) *ProcessInternalMetrics {
	if utils.OSNameNorm != "linux" {
		internalMetricsLog.Warnf(
			"OS internal metrics not supported for %q", utils.OSNameNorm,
		)
		return nil
	}

	return &ProcessInternalMetrics{
		internalMetrics: internalMetrics,
		pid:             os.Getpid(),
		pagesize:        uint64(os.Getpagesize()),
	}
}

func (pim *ProcessInternalMetrics) SnapStats() {
	if pim.pidTidPath == "" {
		procfsRoot := GlobalProcfsRoot
		if pim.internalMetrics.procfsRoot != "" {
			procfsRoot = pim.internalMetrics.procfsRoot
		}
		pim.pidTidPath = procfs.BuildPidTidPath(procfsRoot, pim.pid, procfs.PID_ONLY_TID)
	}
	pidStat := pim.pidStat[pim.currIndex]
	if pidStat == nil {
		pidStat = procfs.NewPidStat()
		pim.pidStat[pim.currIndex] = pidStat
	}
	err := pidStat.Parse(pim.pidTidPath)
	statsTs := time.Now()
	if err != nil {
		internalMetricsLog.Warnf("pidStat.Parse(pid=%d): %v", pim.pid, err)
		pim.pidStat[pim.currIndex] = nil
	}
	pim.statsTs[pim.currIndex] = statsTs
}

func (pim *ProcessInternalMetrics) updateMetricsCache() {
	instance, hostname := GlobalInstance, GlobalHostname
	if pim.internalMetrics.instance != "" {
		instance = pim.internalMetrics.instance
	}
	if pim.internalMetrics.hostname != "" {
		hostname = pim.internalMetrics.hostname
	}
	pim.vszMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
		LSVMI_PROC_VSIZE_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))
	pim.rssMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
		LSVMI_PROC_RSS_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))
	pim.pcpuMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
		LSVMI_PROC_PCPU_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))
	pim.numThreadsMetric = []byte(fmt.Sprintf(
		`%s{%s="%s",%s="%s"} `, // N.B. include the whitespace separating the metric from value
		LSVMI_PROC_NUM_THREADS_METRIC,
		INSTANCE_LABEL_NAME, instance,
		HOSTNAME_LABEL_NAME, hostname,
	))

}

func (pim *ProcessInternalMetrics) generateMetrics(
	buf *bytes.Buffer, tsSuffix []byte,
) int {

	currPidStat, prevPidStat := pim.pidStat[pim.currIndex], pim.pidStat[1-pim.currIndex]
	if currPidStat == nil {
		// Cannot generate metrics since stats couldn't be collected:
		return 0
	}

	if tsSuffix == nil {
		// This should happen only during unit testing:
		tsSuffix = pim.internalMetrics.getTsSuffix()
	}

	if pim.vszMetric == nil {
		// This will update all metrics:
		pim.updateMetricsCache()
	}

	metricsCount := 0

	currPidStatBSF, currPidStatNF := currPidStat.GetData()

	buf.Write(pim.vszMetric)
	buf.Write(currPidStatBSF[procfs.PID_STAT_VSIZE])
	buf.Write(tsSuffix)
	metricsCount++

	buf.Write(pim.rssMetric)
	rss := currPidStatNF[procfs.PID_STAT_RSS] * pim.pagesize
	buf.WriteString(strconv.FormatUint(rss, 10))
	buf.Write(tsSuffix)
	metricsCount++

	buf.Write(pim.numThreadsMetric)
	buf.Write(currPidStatBSF[procfs.PID_STAT_NUM_THREADS])
	buf.Write(tsSuffix)
	metricsCount++

	if prevPidStat != nil {
		_, prevPidStatNF := prevPidStat.GetData()
		dTime := pim.statsTs[pim.currIndex].Sub(pim.statsTs[1-pim.currIndex]).Seconds()
		dTimeCpu := float64(
			currPidStatNF[procfs.PID_STAT_UTIME]+
				currPidStatNF[procfs.PID_STAT_STIME]-
				prevPidStatNF[procfs.PID_STAT_UTIME]-
				prevPidStatNF[procfs.PID_STAT_STIME]) *
			utils.LinuxClktckSec
		pcpu := dTimeCpu / dTime * 100
		buf.Write(pim.pcpuMetric)
		buf.WriteString(strconv.FormatFloat(pcpu, 'f', 1, 64))
		buf.Write(tsSuffix)
		metricsCount++
	}

	// Flip the stats storage:
	pim.currIndex = 1 - pim.currIndex

	return metricsCount
}
