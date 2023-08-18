package procfs

import (
	"bytes"
	"fmt"

	"testing"
)

const (
	TEST_READ_FILE_BUF_POOL_FILE_PATH = TESTDATA_PROC_DIR + "/slabinfo"
	TEST_READ_FILE_BUF_POOL_FILE_SIZE = 17817
)

func testReadFileBufPoolGetReturn(t *testing.T, maxPoolSize int) {
	p := NewReadFileBufPool(maxPoolSize, 0)
	numGets := maxPoolSize + 1
	if maxPoolSize <= 0 {
		numGets = 13
	}
	// Retrive buffers from empty pool. Check that they are empty and that the
	// pool size stays 0.
	for k := 0; k < numGets; k++ {
		b := p.GetBuf()
		if p.poolSize != 0 {
			t.Fatalf("GetBuf(k=%d): poolSize: want: %d, got: %d", k, 0, p.poolSize)
		}
		if b.Len() != 0 {
			t.Fatalf("GetBuf(k=%d): buf.Len(): want: %d, got: %d", k, 0, b.Len())
		}
	}
	// Return seeded buffers. Check that the pool size does not exceed the max, if
	// capped.
	for k := 0; k < numGets; k++ {
		p.ReturnBuf(bytes.NewBuffer([]byte{byte(k >> 24), byte(k >> 16), byte(k >> 8), byte(k & 255)}))
		wantPoolSize := k + 1
		if maxPoolSize > 0 && wantPoolSize > maxPoolSize {
			wantPoolSize = maxPoolSize
		}
		if p.poolSize != wantPoolSize {
			t.Fatalf("ReturnBuff(k=%d): poolSize: want: %d, got: %d", k, wantPoolSize, p.poolSize)
		}
	}
	// Retrieve again and check content; note that the buffers are retrieved from the end:
	if maxPoolSize > 0 && numGets > maxPoolSize {
		numGets = maxPoolSize
	}
	for k := numGets - 1; k >= 0; k-- {
		gotBytes := p.GetBuf().Bytes()
		if p.poolSize != k {
			t.Fatalf("GetBuf(k=%d): poolSize: want: %d, got: %d", k, k, p.poolSize)
		}
		if len(gotBytes) != 0 {
			t.Fatalf("GetBuf(k=%d): buf.Len(): want: %d, got: %d", k, 0, len(gotBytes))
		}
		wantBytes := []byte{byte(k >> 24), byte(k >> 16), byte(k >> 8), byte(k & 255)}
		if cap(gotBytes) < len(wantBytes) {
			t.Fatalf("GetBuf(k=%d): cap(buf): want: >= %d, got: %d", k, len(wantBytes), cap(gotBytes))
		}
		gotBytes = gotBytes[:len(wantBytes)]
		if !bytes.Equal(wantBytes, gotBytes) {
			t.Fatalf("GetBuf(k=%d): content: want: %v, got: %v", k, wantBytes, gotBytes)
		}
	}
}

func TestReadFileBufPoolGetReturn(t *testing.T) {
	for _, maxPoolSize := range []int{
		0,
		7,
	} {
		t.Run(
			fmt.Sprintf("maxPoolSize=%d", maxPoolSize),
			func(t *testing.T) { testReadFileBufPoolGetReturn(t, maxPoolSize) },
		)
	}
}

func testReadFileBufPoolReadFile(t *testing.T, maxReadSize int64) {
	p := NewReadFileBufPool(0, maxReadSize)
	b, err := p.ReadFile(TEST_READ_FILE_BUF_POOL_FILE_PATH)
	if maxReadSize > 0 && maxReadSize <= TEST_READ_FILE_BUF_POOL_FILE_SIZE {
		if b != nil || err != ErrReadFileBufPotentialTruncation {
			t.Fatalf(
				"ReadFile(%s): want: nil, %v, got: %v, %v",
				TEST_READ_FILE_BUF_POOL_FILE_PATH, ErrReadFileBufPotentialTruncation, b, err,
			)
		}
	} else {
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", TEST_READ_FILE_BUF_POOL_FILE_PATH, err)
		}
		if b.Len() != TEST_READ_FILE_BUF_POOL_FILE_SIZE {
			t.Fatalf("ReadFile(%s): size: want: %d, got: %d",
				TEST_READ_FILE_BUF_POOL_FILE_PATH, TEST_READ_FILE_BUF_POOL_FILE_SIZE, b.Len(),
			)
		}
	}
}

func TestReadFileBufPoolReadFile(t *testing.T) {
	for _, maxReadSize := range []int64{
		0,
		TEST_READ_FILE_BUF_POOL_FILE_SIZE + 1,
		TEST_READ_FILE_BUF_POOL_FILE_SIZE,
	} {
		t.Run(
			fmt.Sprintf("maxReadSize=%d,fileSize=%d", maxReadSize, TEST_READ_FILE_BUF_POOL_FILE_SIZE),
			func(t *testing.T) { testReadFileBufPoolReadFile(t, maxReadSize) },
		)
	}
}
