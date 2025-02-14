// Display metrics at stdout instead of sending them to import endpoints.

package lsvmi

import (
	"bytes"
	"fmt"
	"os"
	"sync"

	"github.com/docker/go-units"

	"github.com/bgp59/linux-stats-victoriametrics-importer/internal/utils"
)

type StdoutMetricsQueue struct {
	// The buffer pool for queued metrics:
	bufPool *utils.ReadFileBufPool
	// The metrics channel (queue):
	metricsQueue chan *bytes.Buffer
	// Fill with metrics up to the target size:
	batchTargetSize int
	// Wait goroutine on shutdown:
	wg *sync.WaitGroup
}

func NewStdoutMetricsQueue(cfg any) (*StdoutMetricsQueue, error) {
	var (
		err     error
		poolCfg *CompressorPoolConfig
	)

	switch cfg := cfg.(type) {
	case *LsvmiConfig:
		poolCfg = cfg.CompressorPoolConfig
	case *CompressorPoolConfig:
		poolCfg = cfg
	case nil:
		poolCfg = DefaultCompressorPoolConfig()
	default:
		return nil, fmt.Errorf("NewStdoutMetricsQueue: %T invalid config type", cfg)
	}

	batchTargetSize, err := units.RAMInBytes(poolCfg.BatchTargetSize)
	if err != nil {
		return nil, fmt.Errorf(
			"NewStdoutMetricsQueue: invalid batch_target_size %q: %v",
			poolCfg.BatchTargetSize, err,
		)
	}

	metricsQueue := &StdoutMetricsQueue{
		bufPool:         utils.NewBufPool(poolCfg.BufferPoolMaxSize),
		metricsQueue:    make(chan *bytes.Buffer, poolCfg.MetricsQueueSize),
		batchTargetSize: int(batchTargetSize),
		wg:              &sync.WaitGroup{},
	}

	metricsQueue.wg.Add(1)
	go metricsQueue.loop()

	return metricsQueue, nil
}

func (mq *StdoutMetricsQueue) GetBuf() *bytes.Buffer {
	return mq.bufPool.GetBuf()
}

func (mq *StdoutMetricsQueue) ReturnBuf(buf *bytes.Buffer) {
	mq.bufPool.ReturnBuf(buf)
}

func (mq *StdoutMetricsQueue) QueueBuf(buf *bytes.Buffer) {
	mq.metricsQueue <- buf
}

func (mq *StdoutMetricsQueue) GetTargetSize() int {
	return mq.batchTargetSize
}

func (mq *StdoutMetricsQueue) loop() {
	defer mq.wg.Done()

	for {
		buf, isOpen := <-mq.metricsQueue
		if !isOpen {
			return
		}
		os.Stdout.Write(buf.Bytes())
		os.Stdout.WriteString("\n")
		mq.bufPool.ReturnBuf(buf)
	}
}

func (mq *StdoutMetricsQueue) Shutdown() {
	close(mq.metricsQueue)
	mq.wg.Wait()
}
