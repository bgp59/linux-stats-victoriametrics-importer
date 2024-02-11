package lsvmi

import (
	"net/http"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/testutils"
)

type HttpEndpointPoolTestCase struct {
	epCfgs                []*HttpEndpointConfig
	healthyRotateInterval time.Duration
}

func buildTestHttpEndpointPool(tc *HttpEndpointPoolTestCase) (*HttpEndpointPool, error) {
	epPoolCfg := DefaultHttpEndpointPoolConfig()
	epPoolCfg.Endpoints = tc.epCfgs
	epPool, err := NewHttpEndpointPool(epPoolCfg)
	if err != nil {
		epPool.healthyRotateInterval = tc.healthyRotateInterval
	}
	return epPool, err
}

func testHttpEndpointPoolCreate(tc *HttpEndpointPoolTestCase, t *testing.T) {
	tlc := testutils.NewTestingLogCollect(t, Log)
	defer tlc.RestoreLog()

	epPool, err := buildTestHttpEndpointPool(tc)
	if err != nil {
		tlc.Fatal(err)
	}
	epPool.healthyRotateInterval = -1 // Ensure it is disabled
	defer epPool.Shutdown()

	i := 0
	for ep := epPool.healthy.head; ep != nil && i < len(tc.epCfgs); ep = ep.next {
		wantUrl := tc.epCfgs[i].URL
		if wantUrl != ep.url {
			tlc.Fatalf("ep#%d url: want: %q, got: %q", i, wantUrl, ep.url)
		}
		i++
	}
	if len(tc.epCfgs) != i {
		tlc.Fatalf("len(healthy): want: %d, got: %d", len(tc.epCfgs), i)
	}
}

func testHttpEndpointPoolRotate(tc *HttpEndpointPoolTestCase, t *testing.T) {
	tlc := testutils.NewTestingLogCollect(t, Log)
	savedLogLevel := Log.GetLevel()
	setLogLevel(logrus.DebugLevel)
	defer func() {
		tlc.RestoreLog()
		setLogLevel(savedLogLevel)
	}()

	epPool, err := buildTestHttpEndpointPool(tc)
	if err != nil {
		tlc.Fatal(err)
	}
	epPool.healthyRotateInterval = 0 // Ensure rotate w/ every call
	defer epPool.Shutdown()

	for i := 0; i < len(tc.epCfgs)*4/3; i++ {
		wantUrl := tc.epCfgs[i%len(tc.epCfgs)].URL
		ep := epPool.GetCurrentHealthy(0)
		if ep == nil {
			tlc.Fatalf("GetCurrentHealthy: want: %s, got: %v", wantUrl, nil)
		} else if wantUrl != ep.url {
			tlc.Fatalf("GetCurrentHealthy: want: %s, got: %s", wantUrl, ep.url)
		}
	}
}

func testHttpEndpointPoolReportError(tc *HttpEndpointPoolTestCase, t *testing.T) {
	testTimeout := 5 * time.Second
	tlc := testutils.NewTestingLogCollect(t, Log)
	savedLogLevel := Log.GetLevel()
	setLogLevel(logrus.DebugLevel)
	defer func() {
		tlc.RestoreLog()
		setLogLevel(savedLogLevel)
	}()

	epPool, err := buildTestHttpEndpointPool(tc)
	if err != nil {
		tlc.Fatal(err)

	}
	defer epPool.Shutdown()
	// Ensure rotate w/ every call
	epPool.healthyRotateInterval = 0
	// Ensure that the health check will proceed right away, since it is paced
	// by the ClientDoer mock:
	epPool.healthCheckInterval = 0

	mock := testutils.NewHttpClientDoerMock(testTimeout)
	defer mock.Cancel()
	epPool.client = mock

	var startEp *HttpEndpoint
	healthCheckResponse := &http.Response{
		StatusCode: http.StatusOK,
		Status:     http.StatusText(http.StatusOK),
	}
	for {
		ep := epPool.GetCurrentHealthy(testTimeout)
		if ep == nil {
			tlc.Fatal(ErrHttpEndpointPoolNoHealthyEP)
		}
		if startEp == nil {
			startEp = ep
		} else if startEp == ep && ep.numErrors == 0 {
			break
		}

		epPool.ReportError(ep)
		if !ep.healthy {
			_, err = mock.GetRequest(ep.url)
			if err != nil {
				tlc.Fatal(err)
			}
			err = mock.SendResponse(ep.url, healthCheckResponse, nil)
			if err != nil {
				tlc.Fatal(err)
			}
		}
	}
}

func TestHttpEndpointPoolCreate(t *testing.T) {
	for _, tc := range []*HttpEndpointPoolTestCase{
		{
			epCfgs: []*HttpEndpointConfig{
				{"http://host1", 1},
			},
		},
		{
			epCfgs: []*HttpEndpointConfig{
				{"http://host1", 1},
				{"http://host2", 1},
			},
		},
	} {
		t.Run(
			"",
			func(t *testing.T) { testHttpEndpointPoolCreate(tc, t) },
		)
	}
}

func TestHttpEndpointPoolRotate(t *testing.T) {
	for _, tc := range []*HttpEndpointPoolTestCase{
		{
			epCfgs: []*HttpEndpointConfig{
				{"http://host1", 1},
			},
		},
		{
			epCfgs: []*HttpEndpointConfig{
				{"http://host1", 1},
				{"http://host2", 1},
				{"http://host3", 1},
				{"http://host4", 1},
			},
		},
	} {
		t.Run(
			"",
			func(t *testing.T) { testHttpEndpointPoolRotate(tc, t) },
		)
	}
}

func TestHttpEndpointPoolReportError(t *testing.T) {
	for _, tc := range []*HttpEndpointPoolTestCase{
		{
			epCfgs: []*HttpEndpointConfig{
				{"http://host1", 1},
			},
		},
		{
			epCfgs: []*HttpEndpointConfig{
				{"http://host1", 1},
				{"http://host2", 2},
				{"http://host3", 3},
				{"http://host4", 4},
			},
		},
	} {
		t.Run(
			"",
			func(t *testing.T) { testHttpEndpointPoolReportError(tc, t) },
		)
	}
}
