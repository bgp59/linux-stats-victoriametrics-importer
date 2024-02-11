// ClientDoer interface for testing.

package testutils

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"
)

var ErrHttpClientDoerMockCancelled = errors.New("HttpClientDoerMock cancelled")
var ErrHttpClientDoerMockPlayback = errors.New("HttpClientDoerMock playback error")

// The request <-> response mapping is keyed by URL and it consists of a pair of
// channels of length 1.
type HttpClientDoerMockRespErr struct {
	Response *http.Response
	Error    error
}

type HttpClientDoerMockChannels struct {
	req     chan *http.Request
	respErr chan *HttpClientDoerMockRespErr
}

type HttpClientDoerMock struct {
	channels map[string]*HttpClientDoerMockChannels
	ctx      context.Context
	cancelFn context.CancelFunc
	mu       *sync.Mutex
	wg       *sync.WaitGroup
}

type HttpClientDoerPlaybookRespErr struct {
	Url string
	HttpClientDoerMockRespErr
}

type HttpClientDoerPlaybookReq struct {
	Url     string
	Request *http.Request
}

type HttpClientDoerPlaybook struct {
	// The requests made by the doers, in the order they were received. The
	// order is always consistent for a given URL, but it may vary across URL's
	// if the play is executed in parallel.
	Reqs []*HttpClientDoerPlaybookReq
	// The responses:
	RespErrs []*HttpClientDoerPlaybookRespErr
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

func (mock *HttpClientDoerMock) Cancel() {
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
			respErr: make(chan *HttpClientDoerMockRespErr, 1),
		}
		mock.channels[url] = channels
	}
	return channels
}

func httpClientDoerMockAddReqToRes(req *http.Request, resp *http.Response) *http.Response {
	newResp := *resp
	resReq := *req
	resReq.Body = nil
	newResp.Request = &resReq
	return &newResp
}

func (mock *HttpClientDoerMock) Do(req *http.Request) (*http.Response, error) {
	mock.wg.Add(1)
	defer mock.wg.Done()
	url := req.URL.String()
	channels := mock.getChannels(url)
	cancelErr := fmt.Errorf("%s %q: %w", req.Method, url, ErrHttpClientDoerMockCancelled)
	select {
	case <-mock.ctx.Done():
		return nil, cancelErr
	case channels.req <- req:
	}

	select {
	case <-mock.ctx.Done():
		return nil, cancelErr
	case respErr := <-channels.respErr:
		return httpClientDoerMockAddReqToRes(req, respErr.Response), respErr.Error
	}
}

func (mock *HttpClientDoerMock) GetRequest(url string) (*http.Request, error) {
	mock.wg.Add(1)
	defer mock.wg.Done()
	channels := mock.getChannels(url)
	select {
	case <-mock.ctx.Done():
		return nil, fmt.Errorf("get req for %q: %w", url, ErrHttpClientDoerMockCancelled)
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
		return fmt.Errorf("send resp to %q: %w", url, ErrHttpClientDoerMockCancelled)
	case channels.respErr <- &HttpClientDoerMockRespErr{resp, err}:
		return nil
	}
}

func (mock *HttpClientDoerMock) CloseIdleConnections() {}

func (mock *HttpClientDoerMock) playOneUrl(
	playbook *HttpClientDoerPlaybook,
	url string,
	respErrs []*HttpClientDoerPlaybookRespErr,
	retErrs map[string]error,
) {
	var (
		req *http.Request
		err error
	)
	defer func() {
		mock.mu.Lock()
		retErrs[url] = err
		mock.mu.Unlock()
	}()

	for _, respErr := range respErrs {
		req, err = mock.GetRequest(url)
		if err != nil {
			return
		}
		err = mock.SendResponse(url, respErr.Response, respErr.Error)
		if err != nil {
			return
		}
		mock.mu.Lock()
		playbook.Reqs = append(playbook.Reqs, &HttpClientDoerPlaybookReq{Url: url, Request: req})
		mock.mu.Unlock()
	}
}

func (mock *HttpClientDoerMock) PlayParallel(playbook *HttpClientDoerPlaybook) error {
	byUrlRespErrs := make(map[string][]*HttpClientDoerPlaybookRespErr)
	retErrs := make(map[string]error)

	for _, urlRespErr := range playbook.RespErrs {
		if byUrlRespErrs[urlRespErr.Url] == nil {
			byUrlRespErrs[urlRespErr.Url] = make([]*HttpClientDoerPlaybookRespErr, 0)
		}
		byUrlRespErrs[urlRespErr.Url] = append(byUrlRespErrs[urlRespErr.Url], urlRespErr)
	}

	wg := &sync.WaitGroup{}
	for url, respErrs := range byUrlRespErrs {
		wg.Add(1)
		go func() {
			mock.playOneUrl(playbook, url, respErrs, retErrs)
			wg.Done()
		}()
	}
	wg.Wait()

	var err error
	for _, retErr := range retErrs {
		if retErr == nil {
			continue
		}
		if err == nil {
			err = retErr
		} else {
			err = fmt.Errorf("%w, %w", err, retErr)
		}
	}

	if err != nil {
		err = fmt.Errorf("%w: %w", ErrHttpClientDoerMockPlayback, err)
	}
	return err
}

func (mock *HttpClientDoerMock) Play(playbook *HttpClientDoerPlaybook) error {
	var (
		err error
		req *http.Request
	)
	for _, urlRespErr := range playbook.RespErrs {
		url := urlRespErr.Url
		req, err = mock.GetRequest(url)
		if err != nil {
			break
		}
		err = mock.SendResponse(url, urlRespErr.Response, urlRespErr.Error)
		if err != nil {
			break
		}
		playbook.Reqs = append(playbook.Reqs, &HttpClientDoerPlaybookReq{Url: url, Request: req})
	}

	if err != nil {
		err = fmt.Errorf("%w: %w", ErrHttpClientDoerMockPlayback, err)
	}
	return err
}
