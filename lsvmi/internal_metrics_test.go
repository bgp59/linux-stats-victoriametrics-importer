// Common definitions for internal metrics tests

package lsvmi

import "time"

type InternalMetricsTestCase struct {
	Name             string
	Description      string
	Instance         string
	Hostname         string
	PromTs           int64
	WantMetricsCount int
	WantMetrics      []string
	ReportExtra      bool
}

func newTestInternalMetrics(tc *InternalMetricsTestCase) (*InternalMetrics, error) {
	internalMetrics, err := NewInternalMetrics(nil)
	if err != nil {
		return nil, err
	}
	internalMetrics.instance = tc.Instance
	internalMetrics.hostname = tc.Hostname
	timeNowRetVal := time.UnixMilli(tc.PromTs)
	internalMetrics.timeNowFn = func() time.Time { return timeNowRetVal }

	return internalMetrics, nil
}
