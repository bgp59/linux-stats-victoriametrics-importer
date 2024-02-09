package lsvmi

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/testutils"
)

func getTestHttpEndpointPool(urlList []string) (*HttpEndpointPool, error) {
	epPoolCfg := DefaultHttpEndpointPoolConfig()
	for _, url := range urlList {
		epPoolCfg.Endpoints = append(epPoolCfg.Endpoints, &HttpEndpointConfig{URL: url})
	}
	return NewHttpEndpointPool(epPoolCfg)
}

func testHttpEndpointPoolHealthyListCreate(urlList []string, t *testing.T) {
	tlc := testutils.NewTestingLogCollect(t, Log)
	defer tlc.RestoreLog()

	epPool, err := getTestHttpEndpointPool(urlList)
	if err != nil {
		tlc.Fatal(err)
	}
	epPool.healthyRotateInterval = -1 // disable rotation
	i := 0
	for ep := epPool.healthy.head; ep != nil && i < len(urlList); ep = ep.next {
		if urlList[i] != ep.url {
			tlc.Fatalf("ep#%d url: want: %q, got: %q", i, urlList[i], ep.url)
		}
		i++
	}
	if len(urlList) != i {
		tlc.Fatalf("len(healthy): want: %d, got: %d", len(urlList), i)
	}
}

func testHttpEndpointPoolHealthyListRotate(urlList []string, t *testing.T) {
	tlc := testutils.NewTestingLogCollect(t, Log)
	defer tlc.RestoreLog()

	epPool, err := getTestHttpEndpointPool(urlList)
	if err != nil {
		tlc.Fatal(err)
	}
	epPool.healthyRotateInterval = 0 // rotate w/ every call
	for i := 0; i < len(urlList)*4/3; i++ {
		wantUrl := urlList[i%len(urlList)]
		ep := epPool.GetCurrentHealthy(0)
		if ep == nil {
			tlc.Fatalf("GetCurrentHealthy: want: %s, got: %v", wantUrl, nil)
		} else if wantUrl != ep.url {
			tlc.Fatalf("GetCurrentHealthy: want: %s, got: %s", wantUrl, ep.url)
		}
	}
}

func TestHttpEndpointPoolHealthyListCreate(t *testing.T) {
	for _, urlList := range [][]string{
		{"http://host1"},
		{"http://host1", "http://host2"},
	} {
		t.Run(
			"",
			func(t *testing.T) { testHttpEndpointPoolHealthyListCreate(urlList, t) },
		)
	}
}

func TestHttpEndpointPoolHealthyListRotate(t *testing.T) {
	for _, urlList := range [][]string{
		{"http://host1"},
		{"http://host1", "http://host2"},
	} {
		t.Run(
			"",
			func(t *testing.T) { testHttpEndpointPoolHealthyListRotate(urlList, t) },
		)
	}
}

type HttpEndpointPoolSendBufferTestCase struct {
	urlList []string
	b       []byte
}

func testHttpEndpointPoolSendBuffer(tc *HttpEndpointPoolSendBufferTestCase, t *testing.T) {
	tlc := testutils.NewTestingLogCollect(t, Log)
	doer := testutils.NewHttpClientDoerMock(5 * time.Second)
	defer doer.Stop()
	defer tlc.RestoreLog()

	epPool, err := getTestHttpEndpointPool(tc.urlList)
	if err != nil {
		tlc.Fatal(err)
	}
	epPool.healthyRotateInterval = 0
	epPool.client = doer

	url := tc.urlList[0]
	c := make(chan error, 1)
	go func() {
		var err error
		defer func() { c <- err }()

		// doer is blocked on sending the request until we read it:
		req, err := doer.GetRequest(url)
		if err != nil {
			return
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			return
		}
		if !bytes.Equal(tc.b, body) {
			err = fmt.Errorf("body different in req")
			return
		}
		// next doer is blocked on sending the request until we provide a response:
		statusCode := http.StatusOK
		resp := &http.Response{
			Status:     http.StatusText(statusCode),
			StatusCode: statusCode,
			Request:    req,
		}
		err = doer.SendResponse(url, resp, fmt.Errorf("Expect cancellation!"))
		if err != nil {
			return
		}
	}()

	err = epPool.SendBuffer(tc.b, 0, false)
	if err != nil {
		tlc.Fatal(err)
	}
}

func TestHttpEndpointPoolSendBuffer(t *testing.T) {
	b := []byte("TestHttpEndpointPoolSendBuffer")
	for _, tc := range []*HttpEndpointPoolSendBufferTestCase{
		{
			urlList: []string{"http://host1", "http://host2"},
			b:       b,
		},
	} {
		t.Run(
			"",
			func(t *testing.T) { testHttpEndpointPoolSendBuffer(tc, t) },
		)
	}

}
