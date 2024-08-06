// Return the list of PID, TID to scan

// The task of scanning processes (PIDs) and threads (TIDs) may be divided upon
// multiple goroutines. Rather than having each goroutine scan /proc, maintain a
// cache with the most recent scan with an expiration date. If multiple
// goroutines ask for the relevant lists within the cache window then the
// results may be re-used.

package procfs

import (
	"os"
	"path"
	"sync"
	"time"
)

const (
	PID_LIST_CACHE_PID_ENABLED = uint32(1 << 0)
	PID_LIST_CACHE_TID_ENABLED = uint32(1 << 1)
	PID_LIST_CACHE_ALL_ENABLED = PID_LIST_CACHE_PID_ENABLED | PID_LIST_CACHE_TID_ENABLED
)

type PidTid struct {
	Pid, Tid int
}

type PidListCache struct {
	// The number of partitions, N, that divide the workload (the number of
	// worker goroutines, that is). Each goroutine identifies with a number i =
	// 0..(N-1) and handles only PID/TID such that [PT]ID % N == i.
	nPart int

	// If nPart is a power of 2, then use a mask instead of % (modulo). The mask
	// will be set to a negative number if disabled:
	mask int

	// Whether it was initialized once and the timestamp of the latest retrieval:
	initialized   bool
	retrievedTime time.Time

	// How long a scan is valid:
	validFor time.Duration

	// What is being cached:
	flags uint32

	// The actual lists, indexed by the partition#:
	pidLists [][]PidTid

	// The root of the file system; typically /proc:
	procfsRoot string

	// Lock protection:
	lock *sync.Mutex

	// Refresh count (mainly for testing):
	refreshCount uint64
}

func NewPidListCache(procfsRoot string, nPart int, validFor time.Duration, flags uint32) *PidListCache {
	if nPart < 1 {
		nPart = 1
	}
	mask := int(-1)
	if nPart > 1 {
		for m := 1; m <= (1 << 30); m = m << 1 {
			if m == nPart {
				mask = m - 1
				break
			}
		}
	}

	return &PidListCache{
		nPart:      nPart,
		mask:       mask,
		validFor:   validFor,
		flags:      flags,
		procfsRoot: procfsRoot,
		lock:       &sync.Mutex{},
	}
}

func getDirNames(dir string) ([]string, error) {
	d, err := os.Open(dir)
	if err != nil {
		return nil, err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return nil, err
	}
	return names, err
}

func (pidListCache *PidListCache) IsEnabledFor(flags uint32) bool {
	return pidListCache.flags&flags > 0
}

func (pidListCache *PidListCache) Refresh(lockAcquired bool) error {
	if !lockAcquired {
		pidListCache.lock.Lock()
		defer pidListCache.lock.Unlock()
	}

	if pidListCache.pidLists == nil {
		pidListCache.pidLists = make([][]PidTid, pidListCache.nPart)
		for i := 0; i < pidListCache.nPart; i++ {
			pidListCache.pidLists[i] = make([]PidTid, 0)
		}
	} else {
		for i := 0; i < pidListCache.nPart; i++ {
			pidListCache.pidLists[i] = pidListCache.pidLists[i][0:0]
		}
	}

	names, err := getDirNames(pidListCache.procfsRoot)
	if err != nil {
		return err
	}

	mask, nPart, useMask := pidListCache.mask, pidListCache.nPart, pidListCache.mask > 0
	isPidEnabled := pidListCache.flags&PID_LIST_CACHE_PID_ENABLED > 0
	isTidEnabled := pidListCache.flags&PID_LIST_CACHE_TID_ENABLED > 0
	numEntries := 0
	for _, name := range names {
		var (
			part     int
			pid, tid int
			pidTid   PidTid
		)

		// Convert to number by hand, saving a nanosec or two:
		pid = 0
		for _, c := range []byte(name) {
			if d := int(c - '0'); 0 <= d && d <= 9 {
				pid = (pid << 3) + (pid << 1) + d
			} else {
				pid = -1
				break
			}
		}
		if pid <= 0 {
			continue
		}

		pidTid.Pid = pid
		pidTid.Tid = PID_ONLY_TID

		if isPidEnabled {
			if useMask {
				part = pid & mask
			} else {
				part = pid % nPart
			}
			pidListCache.pidLists[part] = append(pidListCache.pidLists[part], pidTid)
			numEntries += 1
		}
		if isTidEnabled {
			// TID's belonging to PID:
			names, err := getDirNames(path.Join(pidListCache.procfsRoot, name, "task"))
			if err != nil {
				// Silently ignore, maybe the PID just went away:
				continue
			}
			for _, name := range names {
				tid = 0
				for _, c := range []byte(name) {
					if d := int(c - '0'); 0 <= d && d <= 9 {
						tid = (tid << 3) + (tid << 1) + d
					} else {
						tid = -1
						break
					}
				}
				if tid <= 0 {
					continue
				}
				pidTid.Tid = tid
				if useMask {
					part = tid & pidListCache.mask
				} else {
					part = tid % pidListCache.nPart
				}
				pidListCache.pidLists[part] = append(pidListCache.pidLists[part], pidTid)
				numEntries += 1
			}
		}
	}
	if !pidListCache.initialized {
		pidListCache.initialized = true
	}
	pidListCache.retrievedTime = time.Now()
	pidListCache.refreshCount += 1
	return nil
}

func (pidListCache *PidListCache) GetPidTidList(part int, into []PidTid) ([]PidTid, error) {
	if part < 0 || part >= pidListCache.nPart {
		return nil, nil
	}
	pidListCache.lock.Lock()
	defer pidListCache.lock.Unlock()
	if !pidListCache.initialized || time.Since(pidListCache.retrievedTime) > pidListCache.validFor {
		err := pidListCache.Refresh(true)
		if err != nil {
			return nil, err
		}
	}
	pidListLen := len(pidListCache.pidLists[part])
	if into == nil || cap(into) < pidListLen {
		into = make([]PidTid, pidListLen)
	} else {
		into = into[:pidListLen]
	}
	copy(into, pidListCache.pidLists[part])
	return into, nil
}

// Mainly useful for testing:
func (pidListCache *PidListCache) Invalidate() {
	pidListCache.lock.Lock()
	defer pidListCache.lock.Unlock()
	pidListCache.initialized = false
}

func (pidListCache *PidListCache) GetRefreshCount() uint64 {
	pidListCache.lock.Lock()
	defer pidListCache.lock.Unlock()
	return pidListCache.refreshCount
}
