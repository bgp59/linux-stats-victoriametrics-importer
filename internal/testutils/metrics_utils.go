// Utils for metrics testing:

package testutils

import (
	"bytes"
	"fmt"
	"strings"
)

// A test metrics queue which collects and indexes metrics:
type TestMetricsQueue struct {
	metrics         map[string]int
	batchTargetSize int
}

func NewTestMetricsQueue(batchTargetSize int) *TestMetricsQueue {
	return &TestMetricsQueue{
		metrics:         make(map[string]int, 0),
		batchTargetSize: batchTargetSize,
	}
}

// The MetricsQueue interface:
func (mq *TestMetricsQueue) GetBuf() *bytes.Buffer {
	return &bytes.Buffer{}
}

func (mq *TestMetricsQueue) ReturnBuf(buf *bytes.Buffer) {
}

func (mq *TestMetricsQueue) QueueBuf(buf *bytes.Buffer) {
	if buf == nil || buf.Len() == 0 {
		return
	}
	for _, metric := range strings.Split(buf.String(), "\n") {
		metric = strings.TrimSpace(metric)
		if metric != "" {
			mq.metrics[metric] += 1
		}
	}
}

func (mq *TestMetricsQueue) GetTargetSize() int {
	return mq.batchTargetSize
}

func (mq *TestMetricsQueue) GenerateReport(wantMetrics []string, reportExtra bool, errBuf *bytes.Buffer) *bytes.Buffer {
	if errBuf == nil {
		errBuf = &bytes.Buffer{}
	}

	foundMetrics := make(map[string]bool)
	for _, wantMetric := range wantMetrics {
		wantMetric = strings.TrimSpace(wantMetric)
		if mq.metrics[wantMetric] == 0 {
			fmt.Fprintf(errBuf, "\nmissing metric: %s", wantMetric)
		} else {
			foundMetrics[wantMetric] = true
		}
	}

	if reportExtra {
		for gotMetric, count := range mq.metrics {
			if !foundMetrics[gotMetric] {
				fmt.Fprintf(errBuf, "\nunexpected metric: %s", gotMetric)
			}
			if count > 1 {
				fmt.Fprintf(errBuf, "\nmetric: %s: count: %d > 1", gotMetric, count)
			}
		}
	}
	return errBuf
}
