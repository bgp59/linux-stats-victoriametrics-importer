// Compressor pool for sending metrics:
// go:build exclude
package lsvmi

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"sync"
	"time"

	"github.com/docker/go-units"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/utils"
)

// The compressor pool consists of the following:
//  - a metrics channel into which metrics generators write buffers
//  - N compressors that read from the channel and gzip compress the buffers
//    until they either reach a target size or a flush interval lapses. At that
//    point the compressed buffer is sent out to import endpoints.
//
// The batch size cannot be assessed accurately because some of it is in the
// compression buffer, which is not exposed. However it can be estimated based
// on the number of bytes processed thus far, divided by the observed
// compression factor, CF.
// CF is updated at batch end, using exponential decay alpha:
//   CF = (1 - alpha) * batchCF + alpha * CF, alpha = (0..1)
// where batchCF = (number of read bytes)/size(batch)

var compressorLog = NewCompLogger("compressor")

const (
	COMPRESSOR_POOL_COMPRESSION_LEVEL_DEFAULT    = gzip.DefaultCompression
	COMPRESSOR_POOL_NUM_COMPRESSORS_DEFAULT      = -1
	COMPRESSOR_POOL_NUM_COMPRESSORS_MAX          = 4
	COMPRESSOR_POOL_BUFFER_POOL_MAX_SIZE_DEFAULT = 64
	COMPRESSOR_POOL_METRICS_QUEUE_SIZE_DEFAULT   = 64
	COMPRESSOR_POOL_BATCH_TARGET_SIZE_DEFAULT    = "64k"
	COMPRESSOR_POOL_FLUSH_INTERVAL_DEFAULT       = "5s"
)

const (
	INITIAL_COMPRESSION_FACTOR         = 2.
	COMPRESSION_FACTOR_EXP_DECAY_ALPHA = 0.8
	// A compressed batch should be at least this size to be used for updating
	// the compression factor:
	COMPRESSED_BATCH_MIN_SIZE_FOR_CF = 128
)

const (
	COMPRESSOR_POOL_STATE_CREATED = iota
	COMPRESSOR_POOL_STATE_RUNNING
	COMPRESSOR_POOL_STATE_STOPPED
)

var compressorStateMap = map[int]string{
	COMPRESSOR_POOL_STATE_CREATED: "Created",
	COMPRESSOR_POOL_STATE_RUNNING: "Running",
	COMPRESSOR_POOL_STATE_STOPPED: "Stopped",
}

// Compressor stats:
const (
	COMPRESSOR_STATS_READ_COUNT = iota
	COMPRESSOR_STATS_READ_BYTE_COUNT
	COMPRESSOR_STATS_SEND_COUNT
	COMPRESSOR_STATS_SEND_BYTE_COUNT
	COMPRESSOR_STATS_TIMEOUT_FLUSH_COUNT
	COMPRESSOR_STATS_SEND_ERROR_COUNT
	COMPRESSOR_STATS_WRITE_ERROR_COUNT
	// Must be last:
	COMPRESSOR_STATS_UINT64_LEN
)

const (
	COMPRESSOR_STATS_COMPRESSION_FACTOR = iota
	// Must be last:
	COMPRESSOR_STATS_FLOAT64_LEN
)

// The compressor interface as seen by the metrics generators:
type Compressor interface {
	GetBuf() *bytes.Buffer
	QueueBuf(*bytes.Buffer)
}

type CompressorStats struct {
	uint64Stats  []uint64
	float64Stats []float64
}

type CompressorPoolStats struct {
	stats []*CompressorStats
	// Lock:
	mu *sync.Mutex
}

func NewCompressorStats() *CompressorStats {
	return &CompressorStats{
		uint64Stats:  make([]uint64, COMPRESSOR_STATS_UINT64_LEN),
		float64Stats: make([]float64, COMPRESSOR_STATS_FLOAT64_LEN),
	}
}

func NewCompressorPoolStatsNoLock(numCompressors int) *CompressorPoolStats {
	poolStats := &CompressorPoolStats{
		stats: make([]*CompressorStats, numCompressors),
	}
	for i := 0; i < numCompressors; i++ {
		poolStats.stats[i] = NewCompressorStats()
	}
	return poolStats
}

func NewCompressorPoolStats(numCompressors int) *CompressorPoolStats {
	poolStats := NewCompressorPoolStatsNoLock(numCompressors)
	poolStats.mu = &sync.Mutex{}
	return poolStats
}

func (poolStats *CompressorPoolStats) Snap(snapStats *CompressorPoolStats) *CompressorPoolStats {
	if snapStats == nil {
		snapStats = NewCompressorPoolStatsNoLock(len(poolStats.stats))
	}

	poolStats.mu.Lock()
	defer poolStats.mu.Unlock()

	for compressorIndx, stats := range poolStats.stats {
		copy(snapStats.stats[compressorIndx].uint64Stats, stats.uint64Stats)
		copy(snapStats.stats[compressorIndx].float64Stats, stats.float64Stats)
	}

	return snapStats
}

type CompressorPool struct {
	// The number of compressors:
	numCompressors int
	// The buffer pool for queued metrics:
	bufPool *utils.ReadFileBufPool
	// The metrics channel (queue):
	metricsQueue chan *bytes.Buffer
	// The compression level:
	compressionLevel int
	// Compressed batch target size; when the compressed data becomes greater
	// than the latter, the batch is sent out:
	batchTargetSize int
	// How long to wait before sending out a partially filled batch, to avoid
	// staleness. A timer is set with the value below when the batch starts and
	// if it fires before the target size is reached then the batch is sent out.
	flushInterval time.Duration
	// State:
	state   int
	stateMu *sync.Mutex
	// Stats:
	poolStats *CompressorPoolStats
	// Wait group to sync on exit:
	wg *sync.WaitGroup
}

type CompressorPoolConfig struct {
	// The number of compressors. If set to -1 it will match the number of
	// available cores but not more than COMPRESSOR_POOL_NUM_COMPRESSORS_MAX:
	NumCompressors int `yaml:"num_compressors"`
	// Buffer pool size; buffers are pulled by metrics generators as needed and
	// they are returned after they are compressed. The pool max size controls
	// only how many idle buffers are being kept around, since they are created
	// as many as requested but they are discarded if they exceed the value
	// below. A value is too small leads to object churning and a value too
	// large may waste memory.
	BufferPoolMaxSize int `yaml:"buffer_pool_max_size"`
	// Metrics queue size, it should be deep enough to accommodate metrics up to
	// send_buffer_timeout:
	MetricsQueueSize int `yaml:"metrics_queue_size"`
	// Compression level: 0..9:
	CompressionLevel int `yaml:"compression_level"`
	// Batch target size; metrics will be read from the queue until the
	// compressed size is ~ to the value below. The value can have the usual `k`
	// or `m` suffixes for KiB or MiB accordingly.
	BatchTargetSize string `yaml:"batch_target_size"`
	// Flush interval. If batch_target_size is not reached before this interval
	// expires, the metrics compressed thus far are being sent anyway. Use 0 to
	// disable time flush. The value should be compatible with
	// time.ParseDuration().
	FlushInterval string `yaml:"flush_interval"`
}

func DefaultCompressorPoolConfig() *CompressorPoolConfig {
	return &CompressorPoolConfig{
		NumCompressors:    COMPRESSOR_POOL_NUM_COMPRESSORS_DEFAULT,
		BufferPoolMaxSize: COMPRESSOR_POOL_BUFFER_POOL_MAX_SIZE_DEFAULT,
		MetricsQueueSize:  COMPRESSOR_POOL_METRICS_QUEUE_SIZE_DEFAULT,
		CompressionLevel:  COMPRESSOR_POOL_COMPRESSION_LEVEL_DEFAULT,
		BatchTargetSize:   COMPRESSOR_POOL_BATCH_TARGET_SIZE_DEFAULT,
		FlushInterval:     COMPRESSOR_POOL_FLUSH_INTERVAL_DEFAULT,
	}
}

func NewCompressorPool(cfg any) (*CompressorPool, error) {
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
		return nil, fmt.Errorf("NewCompressorPool: %T invalid config type", cfg)
	}

	// Create a dummy compressor to verify the compression level:
	_, err = gzip.NewWriterLevel(nil, poolCfg.CompressionLevel)
	if err != nil {
		return nil, fmt.Errorf("NewCompressorPool: %v", err)
	}

	batchTargetSize, err := units.RAMInBytes(poolCfg.BatchTargetSize)
	if err != nil {
		return nil, fmt.Errorf(
			"NewCompressorPool: invalid batch_target_size %q: %v",
			poolCfg.BatchTargetSize, err,
		)
	}

	flushInterval, err := time.ParseDuration(poolCfg.FlushInterval)
	if err != nil {
		return nil, fmt.Errorf(
			"NewCompressorPool: invalid flush_interval %q: %v",
			poolCfg.FlushInterval, err,
		)
	}

	numCompressors := poolCfg.NumCompressors
	if numCompressors <= 0 {
		numCompressors = utils.AvailableCpusCount
	}
	if numCompressors > COMPRESSOR_POOL_NUM_COMPRESSORS_MAX {
		numCompressors = COMPRESSOR_POOL_NUM_COMPRESSORS_MAX
	}

	pool := &CompressorPool{
		numCompressors:   numCompressors,
		bufPool:          utils.NewBufPool(poolCfg.BufferPoolMaxSize),
		metricsQueue:     make(chan *bytes.Buffer, poolCfg.MetricsQueueSize),
		compressionLevel: poolCfg.CompressionLevel,
		batchTargetSize:  int(batchTargetSize),
		flushInterval:    flushInterval,
		state:            COMPRESSOR_POOL_STATE_CREATED,
		stateMu:          &sync.Mutex{},
		poolStats:        NewCompressorPoolStats(numCompressors),
		wg:               &sync.WaitGroup{},
	}

	compressorLog.Infof("num_compressors=%d", pool.numCompressors)
	compressorLog.Infof("buffer_pool_max_size=%d", poolCfg.BufferPoolMaxSize)
	compressorLog.Infof("metrics_queue_size=%d", poolCfg.MetricsQueueSize)
	compressorLog.Infof("compression_level=%d", pool.compressionLevel)
	compressorLog.Infof("batch_target_size=%d", pool.batchTargetSize)
	compressorLog.Infof("flush_interval=%s", pool.flushInterval)

	return pool, nil
}

func (pool *CompressorPool) Start(sender Sender) {
	pool.stateMu.Lock()
	defer pool.stateMu.Unlock()

	if pool.state != COMPRESSOR_POOL_STATE_CREATED {
		compressorLog.Warnf(
			"compressor pool can only be started from state %d '%s', not from %d '%s'",
			COMPRESSOR_POOL_STATE_CREATED, compressorStateMap[COMPRESSOR_POOL_STATE_CREATED],
			pool.state, compressorStateMap[pool.state],
		)
		return
	}

	for compressorIndx := 0; compressorIndx < pool.numCompressors; compressorIndx++ {
		pool.wg.Add(1)
		go pool.loop(compressorIndx, sender)
	}
	pool.state = COMPRESSOR_POOL_STATE_RUNNING
}

func (pool *CompressorPool) Shutdown() {
	pool.stateMu.Lock()
	defer pool.stateMu.Unlock()

	if pool.state == COMPRESSOR_POOL_STATE_STOPPED {
		compressorLog.Warnf(
			"compressor pool already in state %d '%s'",
			COMPRESSOR_POOL_STATE_STOPPED, compressorStateMap[COMPRESSOR_POOL_STATE_STOPPED],
		)
		return
	}

	compressorLog.Info("close metrics queue")
	close(pool.metricsQueue)
	pool.wg.Wait()
	compressorLog.Info("all compressors stopped")
	pool.state = COMPRESSOR_POOL_STATE_STOPPED
}

func (pool *CompressorPool) GetBuf() *bytes.Buffer {
	return pool.bufPool.GetBuf()
}

func (pool *CompressorPool) QueueBuf(b *bytes.Buffer) {
	pool.metricsQueue <- b
}

func (pool *CompressorPool) loop(compressorIndx int, sender Sender) {
	var (
		buf      *bytes.Buffer
		err      error
		stats    *CompressorStats
		statsMu  *sync.Mutex
		gzWriter *gzip.Writer
		sendFn   func([]byte, time.Duration, bool) error
	)

	defer func() {
		compressorLog.Infof("compressor %d stopped", compressorIndx)
		pool.wg.Done()
	}()

	if sender != nil {
		sendFn = sender.SendBuffer
	}
	bufPool := pool.bufPool
	metricsQueue := pool.metricsQueue
	compressionLevel := pool.compressionLevel
	batchTargetSize := pool.batchTargetSize
	flushInterval := pool.flushInterval
	if pool.poolStats != nil {
		stats = pool.poolStats.stats[compressorIndx]
		statsMu = pool.poolStats.mu
	}
	alpha := COMPRESSION_FACTOR_EXP_DECAY_ALPHA

	// Initialize a stopped timer:
	flushTimer := time.NewTimer(time.Hour)
	if !flushTimer.Stop() {
		<-flushTimer.C
	}

	gzipped, estimatedCF := true, INITIAL_COMPRESSION_FACTOR
	if compressionLevel == gzip.NoCompression {
		estimatedCF = 1.
	}

	gzBuf := &bytes.Buffer{}

	batchReadCount, batchReadByteCount, batchTimeoutCount, doSend, timerSet := 0, 0, 0, false, false
	batchReadByteLimit := int(float64(batchTargetSize) * estimatedCF)
	compressorLog.Infof("start compressor %d", compressorIndx)
	for isOpen := true; isOpen; {
		select {
		case buf, isOpen = <-metricsQueue:
			if buf != nil && buf.Len() > 0 {
				if batchReadCount == 0 {
					// First read of the batch:
					gzBuf.Reset()
					// Create a gzWriter if none exists or repurpose the existent one:
					if gzWriter == nil {
						gzWriter, err = gzip.NewWriterLevel(gzBuf, compressionLevel)
						if err != nil {
							compressorLog.Warnf("compressor %d: %v", compressorIndx, err)
							return
						}
					} else {
						gzWriter.Reset(gzBuf)
					}
					// Reset the flush timer:
					if flushInterval > 0 {
						flushTimer.Reset(flushInterval)
						timerSet = true
					}
				}
				batchReadCount += 1
				batchReadByteCount += buf.Len()
				_, err := gzWriter.Write(buf.Bytes())
				if bufPool != nil {
					bufPool.ReturnBuf(buf)
				}
				if err != nil {
					// This should never happen, since the write is to a buffer, but
					// for completeness it should be handled:
					compressorLog.Warnf("compressor %d: %v", compressorIndx, err)
					if timerSet && !flushTimer.Stop() {
						<-flushTimer.C
					}
					batchReadCount, batchReadByteCount, batchTimeoutCount, doSend, timerSet = 0, 0, 0, false, false
					// Force the recreation of the compressor:
					gzWriter = nil
					if stats != nil {
						statsMu.Lock()
						stats.uint64Stats[COMPRESSOR_STATS_WRITE_ERROR_COUNT] += 1
						statsMu.Unlock()
					}
				}
			}
			doSend = !isOpen && batchReadByteCount > 0 ||
				batchReadByteCount >= batchReadByteLimit
		case <-flushTimer.C:
			doSend, batchTimeoutCount, timerSet = true, 1, false
		}

		if doSend {
			if timerSet && !flushTimer.Stop() {
				<-flushTimer.C
			}
			gzWriter.Close()
			batchSentCount, batchSentByteCount, batchSentErrCount := 1, gzBuf.Len(), 0
			if batchSentByteCount >= COMPRESSED_BATCH_MIN_SIZE_FOR_CF {
				batchCF := float64(batchReadByteCount) / float64(batchSentByteCount)
				estimatedCF = (1-alpha)*batchCF + alpha*estimatedCF
				batchReadByteLimit = int(float64(batchTargetSize) * estimatedCF)
			}

			if sendFn != nil {
				err = sendFn(gzBuf.Bytes(), -1, gzipped)
				if err != nil {
					compressorLog.Warnf("compressor %d: %v", compressorIndx, err)
					batchSentByteCount, batchSentErrCount = 0, 1
				}
			} else {
				batchSentCount, batchSentByteCount = 0, 0
			}

			if stats != nil {
				statsMu.Lock()
				stats.uint64Stats[COMPRESSOR_STATS_READ_COUNT] += uint64(batchReadCount)
				stats.uint64Stats[COMPRESSOR_STATS_READ_BYTE_COUNT] += uint64(batchReadByteCount)
				stats.uint64Stats[COMPRESSOR_STATS_SEND_COUNT] += uint64(batchSentCount)
				stats.uint64Stats[COMPRESSOR_STATS_SEND_BYTE_COUNT] += uint64(batchSentByteCount)
				stats.uint64Stats[COMPRESSOR_STATS_TIMEOUT_FLUSH_COUNT] += uint64(batchTimeoutCount)
				stats.uint64Stats[COMPRESSOR_STATS_SEND_ERROR_COUNT] += uint64(batchSentErrCount)
				stats.float64Stats[COMPRESSOR_STATS_COMPRESSION_FACTOR] = estimatedCF
				statsMu.Unlock()
			}

			batchReadCount, batchReadByteCount, batchTimeoutCount, doSend, timerSet = 0, 0, 0, false, false
		}
	}
}
