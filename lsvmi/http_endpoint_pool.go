// http client pool for lsvmi

package lsvmi

import (
	"context"
	"fmt"
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
	// Endpoint config definitions, later they may be configurable:
	HTTP_ENDPOINT_POOL_HEALTHY_CHECK_MIN_INTERVAL    = 1 * time.Second
	HTTP_ENDPOINT_POOL_HEALTHY_POLL_INTERVAL         = 500 * time.Microsecond
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
	// How long to wait for a healthy endpoint, in case healthy is empty; normally
	// this should be > HealthCheckInterval.
	healthyMaxWait time.Duration
	// How often to poll for a healthy endpoint; this is not configurable for now:
	healthyPollInterval time.Duration
	// How often to log health check errors, if repeated:
	healthCheckErrLogInterval time.Duration
	// The http client, as a mockable interface:
	client HttpClientDoer
	// Access lock:
	mu *sync.Mutex
	// Context and wait group for health checking goroutines, in case the pool
	// is shutdown:
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
	// Params for http.Transport:
	//  Params for Dialer:
	TcpConnTimeout      string `yaml:"tcp_conn_timeout"`
	TcpKeepAlive        string `yaml:"tcp_keep_alive"`
	MaxIdleConns        int    `yaml:"max_idle_conns"`
	MaxIdleConnsPerHost int    `yaml:"max_idle_conns_per_host"`
	MaxConnsPerHost     int    `yaml:"max_conns_per_host"`
	IdleConnTimeout     string `yaml:"idle_conn_timeout"`
	// Params for http.Client:
	ResponseTimeout string `yaml:"response_timeout"`
}

func DefaultHttpEndpointPoolConfig() *HttpEndpointPoolConfig {
	return &HttpEndpointPoolConfig{
		HealthyRotateInterval: HTTP_ENDPOINT_POOL_HEALTHY_ROTATE_INTERVAL_DEFAULT,
		ErrorResetInterval:    HTTP_ENDPOINT_POOL_ERROR_RESET_INTERVAL_DEFAULT,
		HealthCheckInterval:   HTTP_ENDPOINT_POOL_HEALTH_CHECK_INTERVAL_DEFAULT,
		HealthyMaxWait:        HTTP_ENDPOINT_POOL_HEALTHY_MAX_WAIT_DEFAULT,
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

	epPoolLog.Infof("healthy_rotate_interval=%s", epPool.healthyRotateInterval)
	epPoolLog.Infof("error_reset_interval=%s", epPool.errorResetInterval)
	epPoolLog.Infof("healthy_check_interval=%s", epPool.healthCheckInterval)
	epPoolLog.Infof("healthy_max_wait=%s", epPool.healthyMaxWait)
	epPoolLog.Infof("max_idle_conns=%d", transport.MaxIdleConns)
	epPoolLog.Infof("tcp_conn_timeout=%s", dialer.Timeout)
	epPoolLog.Infof("tcp_keep_alive=%s", dialer.KeepAlive)
	epPoolLog.Infof("max_idle_conns_per_host=%d", transport.MaxIdleConnsPerHost)
	epPoolLog.Infof("max_conns_per_host=%d", transport.MaxConnsPerHost)
	epPoolLog.Infof("idle_conn_timeout=%s", transport.IdleConnTimeout)
	epPoolLog.Infof("response_timeout=%s", client.Timeout)

	endpoints := poolCfg.Endpoints
	if len(endpoints) == 0 {
		endpoints = []*HttpEndpointConfig{nil}
	}
	for _, epCfg := range endpoints {
		if ep, err := NewHttpEndpoint(epCfg); err != nil {
			return nil, err
		} else {
			epPool.MoveToHealthy(ep)
		}
	}
	epPoolLog.Infof("%s is at the head of the healthy list", epPool.healthy.head.URL)

	return epPool, nil
}

// Needed for testing:
func (epPool *HttpEndpointPool) StopHealthCheck() {
	epPoolLog.Info("Stop health check goroutines")
	epPool.ctxCancelFn()
	epPool.wg.Wait()
	epPoolLog.Info("All health check goroutines completed")
}

func (epPool *HttpEndpointPool) HealthCheck(ep *HttpEndpoint) {
	isSameErr := func(err1, err2 error) bool {
		return err1 == nil && err2 == nil ||
			err1 != nil && err2 != nil && err1.Error() == err2.Error()
	}

	isSameStatus := func(preStatusCode int, resp *http.Response) bool {
		return resp == nil && preStatusCode == -1 ||
			resp != nil && preStatusCode == resp.StatusCode
	}

	req := &http.Request{
		Method: http.MethodPut,
		URL:    ep.url,
		Header: http.Header{},
	}
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
			if err == nil && res != nil && res.StatusCode == http.StatusOK {
				done = true
			} else {
				checkTime = checkTime.Add(epPool.healthCheckInterval)
				timer.Reset(time.Until(checkTime))
				if !isSameErr(err, prevErr) ||
					!isSameStatus(prevStatusCode, res) ||
					time.Since(errorLogTs) >= epPool.healthCheckErrLogInterval {
					errorLogTs = time.Now()
					if err != nil {
						epPoolLog.Warn(err)
					} else {
						epPoolLog.Warn("%s %q: %s", req.Method, req.URL, res.Status)
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

func (epPool *HttpEndpointPool) DeclareUnhealthy(ep *HttpEndpoint) {
	epPool.mu.Lock()
	defer epPool.mu.Unlock()
	if !ep.healthy {
		// Already in the unhealthy state:
		return
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

// Get the current healthy endpoint or nil if none available after max wait:
func (epPool *HttpEndpointPool) GetCurrentHealthy() *HttpEndpoint {
	var ep *HttpEndpoint
	epPool.mu.Lock()
	defer epPool.mu.Unlock()
	// There is no sync.Condition Wait with timeout, so poll until deadline,
	// waiting for a healthy endpoint. It shouldn't impact the overall
	// efficiency since this is not the normal operating condition.
	waitStart := time.Now()
	for epPool.healthy.head == nil {
		if time.Since(waitStart) > epPool.healthyMaxWait {
			return nil
		}
		epPool.mu.Unlock()
		time.Sleep(epPool.healthyPollInterval)
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
		epPoolLog.Infof("clear error count for %s (was: %d)", ep.URL, ep.numErrors)
		ep.numErrors = 0
	}
	return ep
}
