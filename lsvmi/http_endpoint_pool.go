// http client pool for lsvmi

package lsvmi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// LSVMI is configured with a list of URL endpoints for import. The usable
// endpoints are placed into the healthy sub-list and its head is the current
// one in use for requests. If a transport error occurs, the endpoint is moved
// to the back of the list. When the number of transport errors exceeds a
// certain threshold, the endpoint is removed from the healthy list and it will
// be checked periodically via a test HTTP request. When the latter succeeds,
// the endpoint is returned to  the tail of the healthy list. To ensure a
// balanced use of all the endpoints, the healthy list is rotated periodically
// such that each endpoint will eventually be at the head.

const (
	// Notes:
	//  1. All intervals below are time.ParseInterval() compatible.

	// Endpoint default values:
	HTTP_ENDPOINT_URL_DEFAULT                      = "http://localhost:8428/api/v1/import/prometheus"
	HTTP_ENDPOINT_MARK_UNHEALTHY_THRESHOLD_DEFAULT = 1

	// Endpoint config pool default values:
	HTTP_ENDPOINT_POOL_HEALTHY_ROTATE_INTERVAL_DEFAULT = "5m"
	HTTP_ENDPOINT_POOL_ERROR_RESET_INTERVAL_DEFAULT    = "1m"
	HTTP_ENDPOINT_POOL_HEALTH_CHECK_INTERVAL_DEFAULT   = "5s"
	HTTP_ENDPOINT_POOL_HEALTHY_MAX_WAIT_DEFAULT        = "10s"
	HTTP_ENDPOINT_POOL_SEND_BUFFER_TIMEOUT_DEFAULT     = "20s"
	HTTP_ENDPOINT_POOL_RATE_LIMIT_MBPS_DEFAULT         = ""
	// Endpoint config definitions, later they may be configurable:
	HTTP_ENDPOINT_POOL_HEALTHY_CHECK_MIN_INTERVAL    = 1 * time.Second
	HTTP_ENDPOINT_POOL_HEALTHY_POLL_INTERVAL         = 500 * time.Millisecond
	HTTP_ENDPOINT_POOL_HEALTH_CHECK_ERR_LOG_INTERVAL = 10 * time.Second

	// http.Transport config default values:
	//   Dialer config default values:
	HTTP_ENDPOINT_POOL_TCP_CONN_TIMEOUT_DEFAULT        = "2s"
	HTTP_ENDPOINT_POOL_TCP_KEEP_ALIVE_DEFAULT          = "15s"
	HTTP_ENDPOINT_POOL_MAX_IDLE_CONNS_DEFAULT          = 0 // No limit
	HTTP_ENDPOINT_POOL_MAX_IDLE_CONNS_PER_HOST_DEFAULT = 1
	HTTP_ENDPOINT_POOL_MAX_CONNS_PER_HOST_DEFAULT      = 0 // No limit
	HTTP_ENDPOINT_POOL_IDLE_CONN_TIMEOUT_DEFAULT       = "1m"
	// http.Client config default values:
	HTTP_ENDPOINT_POOL_RESPONSE_TIMEOUT_DEFAULT = "5s"
)

var epPoolLog = NewCompLogger("http_endpoint_pool")

// Define a mockable interface to substitute http.Client.Do() for testing purposes:
type HttpClientDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// Interface for a http.Request body w/ retries:
type ReadSeekRewindCloser interface {
	io.ReadSeekCloser
	Rewind() error
}

// Convert bytes.Reader into ReadSeekRewindCloser such that it can be used
// as body for http.Request w/ retries:
type BytesReadSeekCloser struct {
	rs        io.ReadSeeker
	closed    bool
	closedPos int64
}

func (brsc *BytesReadSeekCloser) Read(p []byte) (int, error) {
	if brsc.closed {
		return int(brsc.closedPos), nil
	}
	return brsc.rs.Read(p)
}

func (brsc *BytesReadSeekCloser) Seek(offset int64, whence int) (int64, error) {
	if brsc.closed {
		return 0, nil
	}
	return brsc.rs.Seek(offset, whence)
}

func (brsc *BytesReadSeekCloser) Close() error {
	brsc.closed = true
	return nil
}

// Reuse, for HTTP retries:
func (brsc *BytesReadSeekCloser) Rewind() error {
	brsc.closed = false
	_, err := brsc.Seek(0, io.SeekStart)
	return err
}

func NewBytesReadSeekCloser(b []byte) *BytesReadSeekCloser {
	return &BytesReadSeekCloser{
		rs:        bytes.NewReader(b),
		closed:    len(b) == 0,
		closedPos: int64(len(b)),
	}
}

type HttpEndpoint struct {
	// The URL that accepts PUT w/ Prometheus exposition format data:
	URL string
	// The parsed format for above, to be used for http calls:
	url *url.URL
	// The threshold for failed accesses count, used for declaring the endpoint
	// unhealthy; this may be > 1 for cases where the host name part of the URL
	// is some kind of a DNS pool which is resolved to a list of addresses, in
	// which case it should be set to the number of pool members. Just because
	// one member is unhealthy, it doesn't mean that others cannot be used. The
	// net/http Transport connection cache will remove the failed connection and
	// the name to address resolution mechanism should no longer resolve to this
	// failed IP.
	markUnhealthyThreshold int
	// State:
	healthy bool
	// The number of errors so far that is compared against the threshold above:
	numErrors int
	// The timestamp of the most recent error:
	errorTs time.Time
	// Doubly linked list:
	prev, next *HttpEndpoint
}

type HttpEndpointConfig struct {
	URL                    string
	MarkUnhealthyThreshold int `yaml:"mark_unhealthy_threshold"`
}

// The list of HTTP codes that denote success:
var HttpEndpointPoolSuccessCodes = map[int]bool{
	http.StatusOK: true,
}

// The list of HTTP codes that should be retried:
var HttpEndpointPoolRetryCodes = map[int]bool{}

func DefaultHttpEndpointConfig() *HttpEndpointConfig {
	return &HttpEndpointConfig{
		URL:                    HTTP_ENDPOINT_URL_DEFAULT,
		MarkUnhealthyThreshold: HTTP_ENDPOINT_MARK_UNHEALTHY_THRESHOLD_DEFAULT,
	}
}

func NewHttpEndpoint(cfg *HttpEndpointConfig) (*HttpEndpoint, error) {
	var err error
	if cfg == nil {
		cfg = DefaultHttpEndpointConfig()
	}
	ep := &HttpEndpoint{
		URL:                    cfg.URL,
		markUnhealthyThreshold: cfg.MarkUnhealthyThreshold,
	}
	if ep.url, err = url.Parse(ep.URL); err != nil {
		ep = nil
		err = fmt.Errorf("NewHttpEndpoint(%s): %v", ep.URL, err)
	}
	return ep, err
}

type HttpEndpointDoublyLinkedList struct {
	head, tail *HttpEndpoint
}

func (epDblLnkList *HttpEndpointDoublyLinkedList) Insert(ep, after *HttpEndpoint) {
	ep.prev = after
	if after != nil {
		ep.next = after.next
		after.next = ep
	} else {
		// Add to head:
		ep.next = epDblLnkList.head
		epDblLnkList.head = ep
	}
	if ep.next == nil {
		// Added to tail:
		epDblLnkList.tail = ep
	}
}

func (epDblLnkList *HttpEndpointDoublyLinkedList) Remove(ep *HttpEndpoint) {
	if ep.prev != nil {
		ep.prev.next = ep.next
	} else {
		epDblLnkList.head = ep.next
	}
	if ep.next != nil {
		ep.next.prev = ep.prev
	} else {
		epDblLnkList.tail = ep.prev
	}
	ep.prev = nil
	ep.next = nil
}

func (epDblLnkList *HttpEndpointDoublyLinkedList) AddToHead(ep *HttpEndpoint) {
	epDblLnkList.Insert(ep, nil)
}

func (epDblLnkList *HttpEndpointDoublyLinkedList) AddToTail(ep *HttpEndpoint) {
	epDblLnkList.Insert(ep, epDblLnkList.tail)
}

type HttpEndpointPool struct {
	// The healthy list:
	healthy *HttpEndpointDoublyLinkedList
	// How often to rotate the healthy list. Set to 0 to rotate after every use
	// or to -1 to disable the rotation:
	healthyRotateInterval time.Duration
	// The time stamp when the last change to the head of the healthy list
	// occurred (most likely due to rotation):
	healthyHeadChangeTs time.Time
	// The rotation, which occurs *before* the endpoint is selected, should be
	// disabled for the 1st use (the pool has just been built after all):
	firstUse bool
	// A failed endpoint is moved to the back of the usable list, as long as the
	// cumulative error count is less than the threshold. If enough time passes
	// before it makes it back to the head of the list, then the error count
	// used to declare it unhealthy is no longer relevant and it should be
	// reset. The following defines the interval after which older errors may be
	// ignored; use 0 to disable:
	errorResetInterval time.Duration
	// How often to check if an unhealthy endpoint has become healthy:
	healthCheckInterval time.Duration
	// How long to wait for a healthy endpoint, in case healthy list is empty;
	// normally this should be > HealthCheckInterval.
	healthyMaxWait time.Duration
	// How often to poll for a healthy endpoint; this is not configurable for now:
	healthyPollInterval time.Duration
	// How often to log health check errors, if repeated:
	healthCheckErrLogInterval time.Duration
	// How long to wait for a SendBuffer call to succeed; normally this should
	// be longer than healthyMaxWait or other HTTP timeouts:
	sendBufferTimeout time.Duration
	// Rate limiting credit mechanism, if not nil:
	credit CreditController
	// The http client, as a mockable interface:
	client HttpClientDoer
	// Access lock:
	mu *sync.Mutex
	// Context and wait group for health checking goroutines:
	ctx         context.Context
	ctxCancelFn context.CancelFunc
	wg          *sync.WaitGroup
}

type HttpEndpointPoolConfig struct {
	Endpoints             []*HttpEndpointConfig `yaml:"endpoints"`
	HealthyRotateInterval string                `yaml:"healthy_rotate_interval"`
	ErrorResetInterval    string                `yaml:"error_reset_interval"`
	HealthCheckInterval   string                `yaml:"health_check_interval"`
	HealthyMaxWait        string                `yaml:"healthy_max_wait"`
	SendBufferTimeout     string                `yaml:"send_buffer_timeout"`
	RateLimitMbps         string                `yaml:"rate_limit_mbps"`
	TcpConnTimeout        string                `yaml:"tcp_conn_timeout"`
	TcpKeepAlive          string                `yaml:"tcp_keep_alive"`
	MaxIdleConns          int                   `yaml:"max_idle_conns"`
	MaxIdleConnsPerHost   int                   `yaml:"max_idle_conns_per_host"`
	MaxConnsPerHost       int                   `yaml:"max_conns_per_host"`
	IdleConnTimeout       string                `yaml:"idle_conn_timeout"`
	ResponseTimeout       string                `yaml:"response_timeout"`
}

func DefaultHttpEndpointPoolConfig() *HttpEndpointPoolConfig {
	return &HttpEndpointPoolConfig{
		HealthyRotateInterval: HTTP_ENDPOINT_POOL_HEALTHY_ROTATE_INTERVAL_DEFAULT,
		ErrorResetInterval:    HTTP_ENDPOINT_POOL_ERROR_RESET_INTERVAL_DEFAULT,
		HealthCheckInterval:   HTTP_ENDPOINT_POOL_HEALTH_CHECK_INTERVAL_DEFAULT,
		HealthyMaxWait:        HTTP_ENDPOINT_POOL_HEALTHY_MAX_WAIT_DEFAULT,
		SendBufferTimeout:     HTTP_ENDPOINT_POOL_SEND_BUFFER_TIMEOUT_DEFAULT,
		RateLimitMbps:         HTTP_ENDPOINT_POOL_RATE_LIMIT_MBPS_DEFAULT,
		TcpConnTimeout:        HTTP_ENDPOINT_POOL_TCP_CONN_TIMEOUT_DEFAULT,
		TcpKeepAlive:          HTTP_ENDPOINT_POOL_TCP_KEEP_ALIVE_DEFAULT,
		MaxIdleConns:          HTTP_ENDPOINT_POOL_MAX_IDLE_CONNS_DEFAULT,
		MaxIdleConnsPerHost:   HTTP_ENDPOINT_POOL_MAX_IDLE_CONNS_PER_HOST_DEFAULT,
		MaxConnsPerHost:       HTTP_ENDPOINT_POOL_MAX_CONNS_PER_HOST_DEFAULT,
		IdleConnTimeout:       HTTP_ENDPOINT_POOL_IDLE_CONN_TIMEOUT_DEFAULT,
		ResponseTimeout:       HTTP_ENDPOINT_POOL_RESPONSE_TIMEOUT_DEFAULT,
	}
}

func NewHttpEndpointPool(cfg any) (*HttpEndpointPool, error) {
	var (
		err     error
		poolCfg *HttpEndpointPoolConfig
	)

	switch cfg := cfg.(type) {
	case *LsvmiConfig:
		poolCfg = cfg.HttpEndpointPoolConfig
	case *HttpEndpointPoolConfig:
		poolCfg = cfg
	case nil:
	default:
		return nil, fmt.Errorf("NewHttpEndpointPool: %T invalid config type", cfg)
	}
	if poolCfg == nil {
		poolCfg = DefaultHttpEndpointPoolConfig()
	}

	dialer := &net.Dialer{}
	if dialer.Timeout, err = time.ParseDuration(poolCfg.TcpConnTimeout); err != nil {
		return nil, fmt.Errorf("NewHttpEndpointPool: tcp_conn_timeout: %v", err)
	}
	if dialer.KeepAlive, err = time.ParseDuration(poolCfg.TcpKeepAlive); err != nil {
		return nil, fmt.Errorf("NewHttpEndpointPool: tcp_keep_alive: %v", err)
	}

	transport := &http.Transport{
		DialContext:         dialer.DialContext,
		MaxIdleConns:        poolCfg.MaxIdleConns,
		MaxIdleConnsPerHost: poolCfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:     poolCfg.MaxConnsPerHost,
	}

	if transport.IdleConnTimeout, err = time.ParseDuration(poolCfg.IdleConnTimeout); err != nil {
		return nil, fmt.Errorf("NewHttpEndpointPool: idle_conn_timeout: %v", err)
	}

	client := &http.Client{
		Transport: transport,
	}

	if client.Timeout, err = time.ParseDuration(poolCfg.ResponseTimeout); err != nil {
		return nil, fmt.Errorf("NewHttpEndpointPool: response_timeout: %v", err)
	}

	epPool := &HttpEndpointPool{
		healthy:                   &HttpEndpointDoublyLinkedList{},
		healthyPollInterval:       HTTP_ENDPOINT_POOL_HEALTHY_POLL_INTERVAL,
		healthCheckErrLogInterval: HTTP_ENDPOINT_POOL_HEALTH_CHECK_ERR_LOG_INTERVAL,
		firstUse:                  true,
		client:                    client,
		mu:                        &sync.Mutex{},
		wg:                        &sync.WaitGroup{},
	}
	epPool.ctx, epPool.ctxCancelFn = context.WithCancel(context.Background())
	if epPool.healthyRotateInterval, err = time.ParseDuration(poolCfg.HealthyRotateInterval); err != nil {
		return nil, fmt.Errorf("NewHttpEndpointPool: healthy_rotate_interval: %v", err)
	}
	if epPool.errorResetInterval, err = time.ParseDuration(poolCfg.ErrorResetInterval); err != nil {
		return nil, fmt.Errorf("NewHttpEndpointPool: error_reset_interval: %v", err)
	}
	if epPool.healthyRotateInterval, err = time.ParseDuration(poolCfg.HealthyRotateInterval); err != nil {
		return nil, fmt.Errorf("NewHttpEndpointPool: healthy_rotate_interval: %v", err)
	}
	if epPool.healthCheckInterval, err = time.ParseDuration(poolCfg.HealthCheckInterval); err != nil {
		return nil, fmt.Errorf("NewHttpEndpointPool: healthy_check_interval: %v", err)
	}
	if epPool.sendBufferTimeout, err = time.ParseDuration(poolCfg.SendBufferTimeout); err != nil {
		return nil, fmt.Errorf("NewHttpEndpointPool: send_buffer_timeout: %v", err)
	}
	if epPool.healthyMaxWait, err = time.ParseDuration(poolCfg.HealthyMaxWait); err != nil {
		return nil, fmt.Errorf("NewHttpEndpointPool: healthy_max_wait: %v", err)
	}
	if epPool.healthCheckInterval < HTTP_ENDPOINT_POOL_HEALTHY_CHECK_MIN_INTERVAL {
		epPoolLog.Warnf(
			"healthy_check_interval %s too small, it will be adjusted to %s",
			epPool.healthCheckInterval, HTTP_ENDPOINT_POOL_HEALTHY_CHECK_MIN_INTERVAL,
		)
		epPool.healthCheckInterval = HTTP_ENDPOINT_POOL_HEALTHY_CHECK_MIN_INTERVAL
	}
	if epPool.healthyMaxWait, err = time.ParseDuration(poolCfg.HealthyMaxWait); err != nil {
		return nil, fmt.Errorf("NewHttpEndpointPool: healthy_max_wait: %v", err)
	}
	if epPool.sendBufferTimeout, err = time.ParseDuration(poolCfg.SendBufferTimeout); err != nil {
		return nil, fmt.Errorf("NewHttpEndpointPool: send_buffer_timeout: %v", err)
	}
	if poolCfg.RateLimitMbps != "" {
		if epPool.credit, err = NewCreditFromSpec(poolCfg.RateLimitMbps); err != nil {
			return nil, fmt.Errorf("NewHttpEndpointPool: rate_limit_mbps: %v", err)
		}
	}

	epPoolLog.Infof("healthy_rotate_interval=%s", epPool.healthyRotateInterval)
	epPoolLog.Infof("error_reset_interval=%s", epPool.errorResetInterval)
	epPoolLog.Infof("health_check_interval=%s", epPool.healthCheckInterval)
	epPoolLog.Infof("healthy_max_wait=%s", epPool.healthyMaxWait)
	epPoolLog.Infof("healthy_poll_interval=%s", epPool.healthyPollInterval)
	epPoolLog.Infof("max_idle_conns=%d", transport.MaxIdleConns)
	epPoolLog.Infof("send_buffer_timeout=%s", epPool.sendBufferTimeout)
	epPoolLog.Infof("rate_limit_mbps=%v", epPool.credit)
	epPoolLog.Infof("tcp_conn_timeout=%s", dialer.Timeout)
	epPoolLog.Infof("tcp_keep_alive=%s", dialer.KeepAlive)
	epPoolLog.Infof("max_idle_conns_per_host=%d", transport.MaxIdleConnsPerHost)
	epPoolLog.Infof("max_conns_per_host=%d", transport.MaxConnsPerHost)
	epPoolLog.Infof("idle_conn_timeout=%s", transport.IdleConnTimeout)
	epPoolLog.Infof("response_timeout=%s", client.Timeout)

	endpoints := poolCfg.Endpoints
	defaultEpCfg := DefaultHttpEndpointConfig()
	if len(endpoints) == 0 {
		endpoints = []*HttpEndpointConfig{defaultEpCfg}
	}
	for _, epCfg := range endpoints {
		cfg := *epCfg
		if cfg.URL == "" {
			cfg.URL = defaultEpCfg.URL
		}
		if cfg.MarkUnhealthyThreshold <= 0 {
			cfg.MarkUnhealthyThreshold = defaultEpCfg.MarkUnhealthyThreshold
		}
		if ep, err := NewHttpEndpoint(&cfg); err != nil {
			return nil, err
		} else {
			epPool.MoveToHealthy(ep)
		}
	}
	epPoolLog.Infof("%s is at the head of the healthy list", epPool.healthy.head.URL)

	return epPool, nil
}

func (epPool *HttpEndpointPool) HealthCheck(ep *HttpEndpoint) {
	sameErr := func(err1, err2 error) bool {
		return err1 == nil && err2 == nil ||
			err1 != nil && err2 != nil && err1.Error() == err2.Error()
	}

	sameStatus := func(preStatusCode int, resp *http.Response) bool {
		return resp == nil && preStatusCode == -1 ||
			resp != nil && preStatusCode == resp.StatusCode
	}

	req := &http.Request{
		Method: http.MethodPut,
		URL:    ep.url,
		Header: http.Header{},
	}
	req.Header.Add("Content-Type", "text/html")
	checkTime := time.Now().Add(epPool.healthCheckInterval)
	timer := time.NewTimer(time.Until(checkTime))
	var (
		prevErr        error
		prevStatusCode int       = -1
		errorLogTs     time.Time = time.Now()
	)
	for done := false; !done; {
		select {
		case <-epPool.ctx.Done():
			timer.Stop()
			if !timer.Stop() {
				<-timer.C
			}
			done = true
		case <-timer.C:
			res, err := epPool.client.Do(req)
			if res != nil && res.Body != nil {
				res.Body.Close()
			}
			if err == nil && res != nil && HttpEndpointPoolSuccessCodes[res.StatusCode] {
				epPoolLog.Infof("%s %q: %s", req.Method, req.URL, res.Status)
				done = true
			} else {
				checkTime = checkTime.Add(epPool.healthCheckInterval)
				timer.Reset(time.Until(checkTime))
				if !sameErr(err, prevErr) ||
					!sameStatus(prevStatusCode, res) ||
					time.Since(errorLogTs) >= epPool.healthCheckErrLogInterval {
					errorLogTs = time.Now()
					if err != nil {
						epPoolLog.Warn(err)
					} else {
						epPoolLog.Warnf("%s %q: %s", req.Method, req.URL, res.Status)
					}
				}
				prevErr = err
				if res != nil {
					prevStatusCode = res.StatusCode
				} else {
					prevStatusCode = -1
				}
			}
		}
	}
	epPool.MoveToHealthy(ep)
	epPool.wg.Done()
}

func (epPool *HttpEndpointPool) ReportError(ep *HttpEndpoint) {
	epPool.mu.Lock()
	defer epPool.mu.Unlock()
	ep.numErrors += 1
	epPoolLog.Warnf(
		"%s: error#: %d, threshold: %d",
		ep.URL, ep.numErrors, ep.markUnhealthyThreshold,
	)
	if ep.numErrors >= ep.markUnhealthyThreshold {
		if !ep.healthy {
			// Already in the unhealthy state:
			return
		}
	}
	ep.healthy = false
	epPool.healthy.Remove(ep)
	epPool.wg.Add(1)
	epPoolLog.Warnf("%s moved to health check", ep.URL)
	go epPool.HealthCheck(ep)
}

func (epPool *HttpEndpointPool) MoveToHealthy(ep *HttpEndpoint) {
	epPool.mu.Lock()
	defer epPool.mu.Unlock()
	if ep.healthy {
		// Already in the healthy state:
		return
	}
	ep.healthy = true
	ep.numErrors = 0
	epPool.healthy.AddToTail(ep)
	epPoolLog.Infof("%s added to the healthy list", ep.URL)
}

// Get the current healthy endpoint or nil if none available after max wait; if
// maxWait < 0 then the pool healthyMaxWait is used:
func (epPool *HttpEndpointPool) GetCurrentHealthy(maxWait time.Duration) *HttpEndpoint {
	var ep *HttpEndpoint
	epPool.mu.Lock()
	defer epPool.mu.Unlock()
	// There is no sync.Condition Wait with timeout, so poll until deadline,
	// waiting for a healthy endpoint. It shouldn't impact the overall
	// efficiency since this is not the normal operating condition.
	if maxWait < 0 {
		maxWait = epPool.healthyMaxWait
	}
	deadline := time.Now().Add(maxWait)
	for epPool.healthy.head == nil {
		timeLeft := time.Until(deadline)
		if timeLeft <= 0 {
			return nil
		}
		epPool.mu.Unlock()
		pause := epPool.healthyPollInterval
		if pause > timeLeft {
			pause = timeLeft
		}
		time.Sleep(pause)
		epPool.mu.Lock()
	}
	ep = epPool.healthy.head
	// Rotate as needed:
	if epPool.firstUse {
		epPool.healthyHeadChangeTs = time.Now()
		epPool.firstUse = false
	} else if epPool.healthy.head != epPool.healthy.tail &&
		(epPool.healthyRotateInterval == 0 ||
			epPool.healthyRotateInterval > 0 &&
				time.Since(epPool.healthyHeadChangeTs) >= epPool.healthyRotateInterval) {
		epPool.healthy.Remove(ep)
		epPool.healthy.AddToTail(ep)
		epPoolLog.Infof("%s rotated to healthy list tail", ep.URL)
		ep = epPool.healthy.head
		epPool.healthyHeadChangeTs = time.Now()
		epPoolLog.Infof("%s rotated to healthy list head", ep.URL)
	}
	// Apply error reset as needed:
	if ep.numErrors > 0 &&
		epPool.errorResetInterval > 0 &&
		time.Since(ep.errorTs) >= epPool.errorResetInterval {
		epPoolLog.Infof("%s: error#: %d->0)", ep.URL, ep.numErrors)
		ep.numErrors = 0
	}
	return ep
}

// SendBuffer: the main reason for the pool is to send buffers w/ load balancing
// and retries. If timeout is < 0 then the pool's sendBufferTimeout is used:
func (epPool *HttpEndpointPool) SendBuffer(b []byte, timeout time.Duration, gzipped bool) error {
	var body ReadSeekRewindCloser

	header := http.Header{
		"Content-Type": {"text/html"},
	}
	if gzipped {
		header.Add("Content-Encoding", "gzip")
	}

	if epPool.credit != nil {
		body = NewCreditReader(epPool.credit, 128, b)
	} else {
		body = NewBytesReadSeekCloser(b)
	}

	if timeout < 0 {
		timeout = epPool.sendBufferTimeout
	}
	deadline := time.Now().Add(timeout)
	for attempt := 1; ; attempt++ {
		maxWait := time.Until(deadline)
		if maxWait < 0 {
			maxWait = 0
		}
		ep := epPool.GetCurrentHealthy(maxWait)
		if ep == nil {
			return fmt.Errorf(
				"SendBuffer attempt# %d: no healthy HTTP endpoint available", attempt,
			)
		}
		if attempt > 1 {
			body.Rewind()
		}
		req := &http.Request{
			Method: http.MethodPut,
			Header: header.Clone(),
			URL:    ep.url,
			Body:   body,
		}
		res, err := epPool.client.Do(req)
		if err == nil && res != nil && HttpEndpointPoolSuccessCodes[res.StatusCode] {
			// The request succeeded:
			return nil
		}
		if err == nil && res != nil && !HttpEndpointPoolRetryCodes[res.StatusCode] {
			// The request failed and it shouldn't be retried (for instance the
			// data is malformed) since it will yield the same result on a
			// different endpoint:
			return fmt.Errorf(
				"SendBuffer attempt# %d: %s %s: %s", attempt, req.Method, ep.URL, res.Status,
			)
		}
		// Report the failure:
		if err != nil {
			Log.Warnf("SendBuffer attempt# %d: %v", attempt, err)
		} else if res != nil {
			Log.Warnf("SendBuffer attempt# %d: %s %s: %s", attempt, req.Method, ep.URL, res.Status)
		} else {
			Log.Warnf("SendBuffer attempt# %d: %s %s: no response", attempt, req.Method, ep.URL)
		}
		// There is something wrong w/ the endpoint:
		epPool.ReportError(ep)
	}
}

// Needed for testing or clean exit in general:
func (epPool *HttpEndpointPool) Stop() {
	epPoolLog.Info("stop health check goroutines")
	epPool.ctxCancelFn()
	epPool.wg.Wait()
	epPoolLog.Info("all health check goroutines completed")
	if credit, ok := epPool.credit.(*Credit); ok {
		credit.StopReplenishWait()
	}
}
