// ClientDoer interface for testing.

package testutils

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// The request <-> response mapping is keyed by URL and it consists of a pair of
// channels of length 1.

type HttpClientDoerMockResp struct {
	resp *http.Response
	err  error
}

type HttpClientDoerMockChannels struct {
	req     chan *http.Request
	respErr chan *HttpClientDoerMockResp
}

type HttpClientDoerMock struct {
	channels map[string]*HttpClientDoerMockChannels
	ctx      context.Context
	cancelFn context.CancelFunc
	mu       *sync.Mutex
	wg       *sync.WaitGroup
}

func NewHttpClientDoerMock(timeout time.Duration) *HttpClientDoerMock {
	mock := &HttpClientDoerMock{
		channels: make(map[string]*HttpClientDoerMockChannels, 0),
		mu:       &sync.Mutex{},
		wg:       &sync.WaitGroup{},
	}
	if timeout > 0 {
		mock.ctx, mock.cancelFn = context.WithTimeout(context.Background(), timeout)
	} else {
		mock.ctx, mock.cancelFn = context.WithCancel(context.Background())
	}
	return mock
}

func (mock *HttpClientDoerMock) Stop() {
	mock.cancelFn()
	mock.wg.Wait()
}

func (mock *HttpClientDoerMock) getChannels(url string) *HttpClientDoerMockChannels {
	mock.mu.Lock()
	defer mock.mu.Unlock()
	channels := mock.channels[url]
	if channels == nil {
		channels = &HttpClientDoerMockChannels{
			req:     make(chan *http.Request, 1),
			respErr: make(chan *HttpClientDoerMockResp, 1),
		}
		mock.channels[url] = channels
	}
	return channels
}

func (mock *HttpClientDoerMock) Do(req *http.Request) (*http.Response, error) {
	mock.wg.Add(1)
	defer mock.wg.Done()
	url := req.URL.String()
	channels := mock.getChannels(url)
	cancelErr := fmt.Errorf("%s %q: mock cancelled", req.Method, url)
	select {
	case <-mock.ctx.Done():
		return nil, cancelErr
	case channels.req <- req:
	}
	select {
	case <-mock.ctx.Done():
		return nil, cancelErr
	case respErr := <-channels.respErr:
		return respErr.resp, respErr.err
	}
}

func (mock *HttpClientDoerMock) GetRequest(url string) (*http.Request, error) {
	mock.wg.Add(1)
	defer mock.wg.Done()
	channels := mock.getChannels(url)
	select {
	case <-mock.ctx.Done():
		return nil, fmt.Errorf("get req for %q: mock cancelled", url)
	case req := <-channels.req:
		return req, nil
	}
}

func (mock *HttpClientDoerMock) SendResponse(url string, resp *http.Response, err error) error {
	mock.wg.Add(1)
	defer mock.wg.Done()
	channels := mock.getChannels(url)
	select {
	case <-mock.ctx.Done():
		return fmt.Errorf("send resp to %q: mock cancelled", url)
	case channels.respErr <- &HttpClientDoerMockResp{resp, err}:
		return nil
	}
}

func (mock *HttpClientDoerMock) CloseIdleConnections() {}
