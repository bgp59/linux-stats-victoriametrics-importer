package lsvmi

import (
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

func testHttpEndpointPoolHealthyListCreate(tc *HttpEndpointPoolTestCase, t *testing.T) {
	tlc := testutils.NewTestingLogCollect(t, Log)
	defer tlc.RestoreLog()

	epPool, err := buildTestHttpEndpointPool(tc)
	if err != nil {
		tlc.Fatal(err)
	}
	epPool.healthyRotateInterval = -1 // Ensure it is disabled
	defer epPool.Stop()

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

func testHttpEndpointPoolHealthyListRotate(tc *HttpEndpointPoolTestCase, t *testing.T) {
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
	defer epPool.Stop()

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

func TestHttpEndpointPoolHealthyListCreate(t *testing.T) {
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
			func(t *testing.T) { testHttpEndpointPoolHealthyListCreate(tc, t) },
		)
	}
}

func TestHttpEndpointPoolHealthyListRotate(t *testing.T) {
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
			func(t *testing.T) { testHttpEndpointPoolHealthyListRotate(tc, t) },
		)
	}
}
