package lsvmi

import (
	"net/http"
	"sync"
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/testutils"
)

// ClientDoer interface for testing.

// The req -> resp mapping is keyed by URL only; Do will block until a response
// is written and Respond will block unless the previous response was read. Do
// and Respond may be invoked from different goroutines and the blocking helps
// w/ the synchronization.

type HttpEndpointPoolTestClientDoerResp struct {
	resp *http.Response
	err  error
}

type HttpEndpointPoolTestClientDoer struct {
	reqRespMap map[string]chan *HttpEndpointPoolTestClientDoerResp
	mu         *sync.Mutex
}

func (d *HttpEndpointPoolTestClientDoer) Do(req *http.Request) (*http.Response, error) {
	url := req.URL.String()
	d.mu.Lock()
	c := d.reqRespMap[url]
	if c == nil {
		c = make(chan *HttpEndpointPoolTestClientDoerResp, 1)
		d.reqRespMap[url] = c
	}
	d.mu.Unlock()
	dResp := <-c
	return dResp.resp, dResp.err
}

func (d *HttpEndpointPoolTestClientDoer) Respond(url string, resp *http.Response, err error) {
	d.mu.Lock()
	c := d.reqRespMap[url]
	if c == nil {
		c = make(chan *HttpEndpointPoolTestClientDoerResp, 1)
		d.reqRespMap[url] = c
	}
	d.mu.Unlock()
	c <- &HttpEndpointPoolTestClientDoerResp{resp, err}
}

func NewHttpEndpointPoolTestClientDoer() *HttpEndpointPoolTestClientDoer {
	return &HttpEndpointPoolTestClientDoer{
		reqRespMap: make(map[string]chan *HttpEndpointPoolTestClientDoerResp, 0),
		mu:         &sync.Mutex{},
	}
}

func testHttpEndpointPoolHealthyListCreate(urlList []string, t *testing.T) {
	tlc := testutils.NewTestingLogCollect(t, Log)
	defer tlc.RestoreLog()

	epPool, err := NewHttpEndpointPool(nil)
	epPool.healthyRotateInterval = -1 // disable rotation
	if err != nil {
		tlc.Fatal(err)
	}
	ep := epPool.GetCurrentHealthy(0)
	if ep != nil {
		tlc.Fatalf("unexpected endpoint: %s", ep.url)
	}
	for _, url := range urlList {
		ep, err := NewHttpEndpoint(&HttpEndpointConfig{URL: url})
		if err != nil {
			tlc.Fatalf("NewHttpEndpoint(%s): %v", url, err)
		}
		epPool.MoveToHealthy(ep)
	}
	i := 0
	for ep := epPool.healthy.head; ep != nil && i < len(urlList); ep = ep.next {
		if urlList[i] != ep.url {
			tlc.Fatalf("ep[%d] url: want: %q, got: %q", i, urlList[i], ep.url)
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

	epPoolCfg := DefaultHttpEndpointPoolConfig()
	epPoolCfg.HealthyRotateInterval = "0" // rotate w/ every call
	for _, url := range urlList {
		epPoolCfg.Endpoints = append(epPoolCfg.Endpoints, &HttpEndpointConfig{URL: url})
	}

	epPool, err := NewHttpEndpointPool(epPoolCfg)
	if err != nil {
		tlc.Fatal(err)
	}

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
