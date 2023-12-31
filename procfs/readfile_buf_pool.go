// Read a file into a reusable buffer from a pool; this should be more efficient
// than allocating a buffer every time and relying on GC.

package procfs

import (
	"bytes"
	"errors"
	"io"
	"os"
	"sync"
)

const (
	READ_FILE_BUF_POOL_MAX_SIZE_UNBOUND      = 0
	READ_FILE_BUF_POOL_MAX_READ_SIZE_UNBOUND = 0
)

// Reading a file may be limited by a max size; if the cap is reached then it is
// possible that the file was truncated (note that stat system will report size
// 0 for /proc files, so it cannot be used to determine the actual size). Should
// such a condition occur, it should be treated as an error to signal potential
// truncation to the caller.
var ErrReadFileBufPotentialTruncation = errors.New("potential truncation")

type ReadFileBufPool struct {
	// The pool of buffers; if the pool is empty at retrieval time, a new buffer
	// is created. The buffer is returned to the pool after use.
	pool []*bytes.Buffer
	// Max pool size, if > 0, unlimited otherwise. A spike of concurrent
	// retrievals may generate more buffers than expected during normal
	// operation. Upon return, keep only up to a limit, to avoid memory waste.
	maxPoolSize int
	// Current pool size:
	poolSize int
	// Max read size, if > 0, unlimited otherwise. If the limit is reached then
	// return ErrReadFileBufPotentialTruncation.
	maxReadSize int64
	// Thread safe lock:
	lock *sync.Mutex
}

func NewReadFileBufPool(maxPoolSize int, maxReadSize int64) *ReadFileBufPool {
	return &ReadFileBufPool{
		pool:        make([]*bytes.Buffer, 0),
		maxPoolSize: maxPoolSize,
		maxReadSize: maxReadSize,
		lock:        &sync.Mutex{},
	}
}

func (p *ReadFileBufPool) GetBuf() *bytes.Buffer {
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.poolSize > 0 {
		p.poolSize--
		buf := p.pool[p.poolSize]
		buf.Reset()
		return buf
	}
	return &bytes.Buffer{}
}

func (p *ReadFileBufPool) ReturnBuf(b *bytes.Buffer) {
	p.lock.Lock()
	defer p.lock.Unlock()

	// Discard if at max capacity:
	if p.maxPoolSize > 0 && p.poolSize >= p.maxPoolSize {
		return
	}

	// Return the buffer to the pool:
	if p.poolSize >= len(p.pool) {
		p.pool = append(p.pool, b)
	} else {
		p.pool[p.poolSize] = b
	}
	p.poolSize++
}

func (p *ReadFileBufPool) ReadFile(path string) (*bytes.Buffer, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	b := p.GetBuf()
	maxReadSize := p.maxReadSize
	if maxReadSize > 0 {
		_, err = io.CopyN(b, f, maxReadSize)
		if err == io.EOF {
			// File fully read within buffer max size, i.e. no error:
			err = nil
		} else if err == nil {
			// May be truncated:
			err = ErrReadFileBufPotentialTruncation
		}
	} else {
		_, err = b.ReadFrom(f)
	}
	if err == nil {
		return b, nil
	}
	p.ReturnBuf(b)
	return nil, err
}

// Predefined pools:
var (
	ReadFileBufPool16k         = NewReadFileBufPool(32, 0x4000)
	ReadFileBufPool32k         = NewReadFileBufPool(32, 0x8000)
	ReadFileBufPool64k         = NewReadFileBufPool(32, 0x10000)
	ReadFileBufPool128k        = NewReadFileBufPool(16, 0x20000)
	ReadFileBufPool256k        = NewReadFileBufPool(8, 0x40000)
	ReadFileBufPool1m          = NewReadFileBufPool(4, 0x100000)
	ReadFileBufPoolReadUnbound = NewReadFileBufPool(4, READ_FILE_BUF_POOL_MAX_READ_SIZE_UNBOUND)
)
