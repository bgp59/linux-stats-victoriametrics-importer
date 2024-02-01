// Credit based rate limit controller.
//
// The credit is a numerical quantity replenished periodically, at intervals T,
// with a constant number N. The replenished value may by capped to a max M>=N,
// or it may be unbound. The value R=N/T represents the target rate limit and
// M-N represents the burst limit.
//
// A user in need of n resources should request a credit ==/<= n before
// proceeding (the user may specify an interval nMin..n, nMin <= n). If credit
// is available the user receives a value c within the requested interval and it
// then should use no more than c.
//
// Use case: limit network utilization by choosing N/T = target bandwidth.

package lsvmi

import (
	"context"
	"io"
	"sync"
	"time"
)

const (
	CREDIT_NO_LIMIT    = 0
	CREDIT_EXACT_MATCH = 0
)

// Define an interface for testing:
type CreditController interface {
	GetCredit(desired, minAcceptable int) int
}

// The actual implementation:
type Credit struct {
	ctx            context.Context
	cancelFunc     context.CancelFunc
	wg             *sync.WaitGroup
	cond           *sync.Cond
	current        int
	max            int
	replenishValue int
	replenishInt   time.Duration
}

// Credit based reader:
type CreditReader struct {
	// Credit control:
	cc CreditController
	// Minimum acceptable credit:
	minC int
	// Bytes to return with the controlled rate:
	b []byte
	// Read pointer in b:
	r int
	// Total size of b:
	n int
}

func NewCredit(replenishValue, max int, replenishInt time.Duration) *Credit {
	if max != CREDIT_NO_LIMIT && max < replenishValue {
		max = replenishValue
	}

	ctx, cancelFunc := context.WithCancel(context.Background())

	c := &Credit{
		ctx:            ctx,
		cancelFunc:     cancelFunc,
		wg:             &sync.WaitGroup{},
		cond:           sync.NewCond(&sync.Mutex{}),
		max:            max,
		replenishValue: replenishValue,
		replenishInt:   replenishInt,
	}
	c.startReplenish()
	return c
}

func (c *Credit) startReplenish() {
	replenishInt := c.replenishInt
	nextReplenishTime := time.Now().Add(replenishInt)
	c.current = c.replenishValue
	ctx, wg := c.ctx, c.wg
	wg.Add(1)
	go func() {
		for {
			select {
			case <-ctx.Done():
				wg.Done()
				return
			default:
				pause := time.Until(nextReplenishTime)
				if pause > 0 {
					time.Sleep(pause)
				}
				nextReplenishTime = nextReplenishTime.Add(replenishInt)
				c.cond.L.Lock()
				c.current += c.replenishValue
				if c.max != CREDIT_NO_LIMIT && c.current > c.max {
					c.current = c.max
				}
				c.cond.Broadcast()
				c.cond.L.Unlock()
			}
		}
	}()
}

func (c *Credit) StopReplenish() {
	c.cancelFunc()
}

func (c *Credit) StopReplenishWait() {
	c.cancelFunc()
	c.wg.Wait()
}

func (c *Credit) GetCredit(desired, minAcceptable int) (got int) {
	if minAcceptable == CREDIT_EXACT_MATCH ||
		minAcceptable > desired {
		minAcceptable = desired
	}

	c.cond.L.Lock()
	defer c.cond.L.Unlock()

	for c.current < minAcceptable {
		c.cond.Wait()
	}

	if c.current >= desired {
		got = desired
	} else {
		got = c.current
	}
	c.current -= got
	return
}

func NewCreditReader(cc CreditController, minAcceptable int, b []byte) *CreditReader {
	if minAcceptable < 0 {
		minAcceptable = 0
	}
	return &CreditReader{
		cc:   cc,
		minC: int(minAcceptable),
		b:    b,
		r:    0,
		n:    len(b),
	}
}

// Implement the Read interface:
func (cr *CreditReader) Read(p []byte) (int, error) {
	available := cr.n - cr.r
	if available <= 0 {
		return 0, io.EOF
	}
	toRead := len(p)
	if toRead == 0 {
		return 0, nil
	}
	if available < toRead {
		toRead = available
	}
	toRead = int(cr.cc.GetCredit(toRead, cr.minC))
	if toRead == 0 {
		return 0, nil
	}
	s := cr.r
	cr.r += toRead
	copy(p, cr.b[s:cr.r])
	return toRead, nil
}
