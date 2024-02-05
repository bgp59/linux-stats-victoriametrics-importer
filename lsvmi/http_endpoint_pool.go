// http client pool for lsvmi

package lsvmi

import (
	"fmt"
	"net/http"
	"sync"
	"time"
)

// LSVMI is configured with a list of URL endpoints for import. The list is
// divided into 2 sub-lists: healthy and unhealthy. The endpoint at the head of
// the healthy list is the current one in use for requests. If a transport
// errors occur, the endpoint is moved to the back of the list. When the number
// of transport errors exceeds a certain threshold, the endpoint is moved to the
// unhealthy list, where it will be checked periodically via a test HTTP
// request. When the latter succeeds, the endpoint is returned to  the tail of
// the healthy list. To ensure a balanced use of all the endpoints, the healthy
// list is rotated periodically such that each endpoint will eventually be at
// the head.

const (
	// Notes:
	//  1. All intervals below are time.ParseInterval() compatible.

	// Endpoint default values:
	HTTP_ENDPOINT_URL_DEFAULT                      = "http://localhost:8428/api/v1/import/prometheus"
	HTTP_ENDPOINT_MARK_UNHEALTHY_THRESHOLD_DEFAULT = 1

	// Endpoint config pool default values:
	HTTP_ENDPOINT_POOL_HEALTHY_ROTATE_INTERVAL_DEFAULT = "5m"
	HTTP_ENDPOINT_POOL_ERROR_RESET_INTERVAL_DEFAULT    = "1m"
	HTTP_ENDPOINT_POOL_HEALTHY_CHECK_INTERVAL_DEFAULT  = "5s"
	HTTP_ENDPOINT_POOL_HEALTHY_CHECK_MIN_INTERVAL      = 1 * time.Second
	// http.Transport config default values:
	HTTP_ENDPOINT_POOL_MAX_IDLE_CONNS_DEFAULT          = 0 // No limit
	HTTP_ENDPOINT_POOL_MAX_IDLE_CONNS_PER_HOST_DEFAULT = 1
	HTTP_ENDPOINT_POOL_MAX_CONNS_PER_HOST_DEFAULT      = 0 // No limit
	HTTP_ENDPOINT_POOL_IDLE_CONN_TIMEOUT_DEFAULT       = "1m"
	HTTP_ENDPOINT_POOL_RESPONSE_HEADER_TIMEOUT_DEFAULT = "15s"
)

var epPoolLog = NewCompLogger("http_endpoint_pool")

type HttpEndpoint struct {
	// The URL that accepts PUT w/ Prometheus exposition format data:
	URL string
	// State:
	healthy bool
	// The threshold for failed accesses count, used for declaring the endpoint
	// unhealthy; this may be > 1 for cases where the host name part of the URL
	// is some kind of a DNS pool which is resolved to a list of addresses, in
	// which case it should be set to the number of pool members. Just because
	// one member is unhealthy, it doesn't mean that others cannot be used. The
	// net/http Transport connection cache will remove the failed connection and
	// the name to address resolution mechanism should no longer resolve to this
	// failed IP.
	markUnhealthyThreshold int
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

func NewHttpEndpoint(cfg *HttpEndpointConfig) *HttpEndpoint {
	if cfg == nil {
		cfg = DefaultHttpEndpointConfig()
	}
	return &HttpEndpoint{
		URL:                    cfg.URL,
		markUnhealthyThreshold: cfg.MarkUnhealthyThreshold,
	}
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
	// How often to rotate the healthy list; use 0 to disable the rotation:
	healthyRotateInterval time.Duration
	// The time stamp when the last change to the head of the healthy list
	// occurred (most likely due to rotation):
	healthyHeadChangeTs time.Time
	// A failed endpoint is moved to the back of the usable list, as long as the
	// cumulative error count is less than the threshold. If enough time passes
	// before it makes it back to the head of the list, then the error count
	// used to declare it unhealthy is no longer relevant and it should be
	// reset. The following defines the interval after which older errors may be
	// ignored; use 0 to disable:
	errorResetInterval time.Duration
	// How often to check if an unhealthy endpoint has become healthy:
	healthyCheckInterval time.Duration
	// The http transport underlying the http clients:
	transport *http.Transport
	// Condition lock for access:
	cond *sync.Cond
}

type HttpEndpointPoolConfig struct {
	Endpoints             []*HttpEndpointConfig `yaml:"endpoints"`
	HealthyRotateInterval string                `yaml:"healthy_rotate_interval"`
	ErrorResetInterval    string                `yaml:"error_reset_interval"`
	HealthyCheckInterval  string                `yaml:"healthy_check_interval"`
	// Params for http.Transport:
	MaxIdleConns          int    `yaml:"max_idle_conns"`
	MaxIdleConnsPerHost   int    `yaml:"max_idle_conns_per_host"`
	MaxConnsPerHost       int    `yaml:"max_conns_per_host"`
	IdleConnTimeout       string `yaml:"idle_conn_timeout"`
	ResponseHeaderTimeout string `yaml:"response_header_timeout"`
}

func DefaultHttpEndpointPoolConfig() *HttpEndpointPoolConfig {
	return &HttpEndpointPoolConfig{
		HealthyRotateInterval: HTTP_ENDPOINT_POOL_HEALTHY_ROTATE_INTERVAL_DEFAULT,
		ErrorResetInterval:    HTTP_ENDPOINT_POOL_ERROR_RESET_INTERVAL_DEFAULT,
		HealthyCheckInterval:  HTTP_ENDPOINT_POOL_HEALTHY_CHECK_INTERVAL_DEFAULT,
		MaxIdleConns:          HTTP_ENDPOINT_POOL_MAX_IDLE_CONNS_DEFAULT,
		MaxIdleConnsPerHost:   HTTP_ENDPOINT_POOL_MAX_IDLE_CONNS_PER_HOST_DEFAULT,
		MaxConnsPerHost:       HTTP_ENDPOINT_POOL_MAX_CONNS_PER_HOST_DEFAULT,
		IdleConnTimeout:       HTTP_ENDPOINT_POOL_IDLE_CONN_TIMEOUT_DEFAULT,
		ResponseHeaderTimeout: HTTP_ENDPOINT_POOL_RESPONSE_HEADER_TIMEOUT_DEFAULT,
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

	transport := &http.Transport{
		MaxIdleConns:        poolCfg.MaxIdleConns,
		MaxIdleConnsPerHost: poolCfg.MaxIdleConnsPerHost,
		MaxConnsPerHost:     poolCfg.MaxConnsPerHost,
	}

	if transport.IdleConnTimeout, err = time.ParseDuration(poolCfg.IdleConnTimeout); err != nil {
		return nil, fmt.Errorf("NewHttpEndpointPool: idle_conn_timeout: %v", err)
	}
	if transport.ResponseHeaderTimeout, err = time.ParseDuration(poolCfg.ResponseHeaderTimeout); err != nil {
		return nil, fmt.Errorf("NewHttpEndpointPool: response_header_timeout: %v", err)
	}

	epPool := &HttpEndpointPool{
		healthy:   &HttpEndpointDoublyLinkedList{},
		transport: transport,
		cond:      sync.NewCond(&sync.Mutex{}),
	}
	if epPool.healthyRotateInterval, err = time.ParseDuration(poolCfg.HealthyRotateInterval); err != nil {
		return nil, fmt.Errorf("NewHttpEndpointPool: healthy_rotate_interval: %v", err)
	}
	if epPool.errorResetInterval, err = time.ParseDuration(poolCfg.ErrorResetInterval); err != nil {
		return nil, fmt.Errorf("NewHttpEndpointPool: error_reset_interval: %v", err)
	}
	if epPool.healthyRotateInterval, err = time.ParseDuration(poolCfg.HealthyRotateInterval); err != nil {
		return nil, fmt.Errorf("NewHttpEndpointPool: healthy_rotate_interval: %v", err)
	}
	if epPool.healthyCheckInterval, err = time.ParseDuration(poolCfg.HealthyCheckInterval); err != nil {
		return nil, fmt.Errorf("NewHttpEndpointPool: healthy_check_interval: %v", err)
	}
	if epPool.healthyCheckInterval < HTTP_ENDPOINT_POOL_HEALTHY_CHECK_MIN_INTERVAL {
		epPoolLog.Warnf(
			"healthy_check_interval %s too small, it will be adjusted to %s",
			epPool.healthyCheckInterval, HTTP_ENDPOINT_POOL_HEALTHY_CHECK_MIN_INTERVAL,
		)
		epPool.healthyCheckInterval = HTTP_ENDPOINT_POOL_HEALTHY_CHECK_MIN_INTERVAL
	}

	epPoolLog.Infof("healthy_rotate_interval=%s", epPool.healthyRotateInterval)
	epPoolLog.Infof("error_reset_interval=%s", epPool.errorResetInterval)
	epPoolLog.Infof("healthy_check_interval=%s", epPool.healthyCheckInterval)
	epPoolLog.Infof("max_idle_conns=%d", epPool.transport.MaxIdleConns)
	epPoolLog.Infof("max_idle_conns_per_host=%d", epPool.transport.MaxIdleConnsPerHost)
	epPoolLog.Infof("max_conns_per_host=%d", epPool.transport.MaxConnsPerHost)
	epPoolLog.Infof("idle_conn_timeout=%s", epPool.transport.IdleConnTimeout)
	epPoolLog.Infof("response_header_timeout=%s", epPool.transport.ResponseHeaderTimeout)

	if len(poolCfg.Endpoints) == 0 {
		epPool.MoveToHealthy(NewHttpEndpoint(nil))
	} else {
		for _, epCfg := range poolCfg.Endpoints {
			epPool.MoveToHealthy(NewHttpEndpoint(epCfg))
		}
	}
	epPoolLog.Infof("%s is at the head of the healthy list", epPool.healthy.head.URL)
	epPool.healthyHeadChangeTs = time.Now()
	return epPool, nil
}

func (epPool *HttpEndpointPool) MoveToUnhealthy(ep *HttpEndpoint) {
	epPool.cond.L.Lock()
	defer epPool.cond.L.Unlock()
	if !ep.healthy {
		// Already in the unhealthy state:
		return
	}
	ep.healthy = false
	epPool.healthy.Remove(ep)
}

func (epPool *HttpEndpointPool) MoveToHealthy(ep *HttpEndpoint) {
	epPool.cond.L.Lock()
	defer epPool.cond.L.Unlock()
	if ep.healthy {
		// Already in the healthy state:
		return
	}
	ep.healthy = true
	ep.numErrors = 0
	epPool.healthy.AddToTail(ep)
	epPool.cond.Broadcast()
	epPoolLog.Infof(
		"url=%s, mark_unhealthy_threshold=%d added to the healthy list",
		ep.URL, ep.markUnhealthyThreshold,
	)
}

func (epPool *HttpEndpointPool) GetCurrentHealthy() *HttpEndpoint {
	var ep *HttpEndpoint
	epPool.cond.L.Lock()
	defer epPool.cond.L.Unlock()
	for epPool.healthy.head == nil {
		epPool.cond.Wait()
	}
	ep = epPool.healthy.head
	// Rotate as needed:
	if epPool.healthyRotateInterval > 0 &&
		epPool.healthy.head != epPool.healthy.tail &&
		time.Since(epPool.healthyHeadChangeTs) >= epPool.healthyRotateInterval {
		epPool.healthy.Remove(ep)
		epPool.healthy.AddToTail(ep)
		epPoolLog.Infof("%s rotated to healthy list tail", ep.URL)
		ep = epPool.healthy.head
		epPool.healthyHeadChangeTs = time.Now()
		epPoolLog.Infof("%s rotated to healthy list head", ep.URL)

	}
	// Apply error reset as needed:
	if epPool.errorResetInterval > 0 &&
		ep.numErrors > 0 &&
		time.Since(ep.errorTs) >= epPool.errorResetInterval {
		epPoolLog.Infof("clear error count for %s (was: %d)", ep.URL, ep.numErrors)
		ep.numErrors = 0
	}
	return ep
}
