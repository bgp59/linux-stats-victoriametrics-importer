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

	// Special TID to indicate that the stats are for PID only:
	PID_ONLY_TID = -1
)

type PidTid struct {
	Pid, Tid int
}

// Define an interface to be used in tests depending on PidTidListCache (they
// will replace the real object with a simulated one):
type PidTidListCacheIF interface {
	GetPidTidList(part int, into []PidTid) ([]PidTid, error)
	Invalidate()
	GetRefreshCount() uint64
}

type PidTidListCache struct {
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

func NewPidTidListCache(procfsRoot string, nPart int, validFor time.Duration, flags uint32) PidTidListCacheIF {
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

	return &PidTidListCache{
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

func (pidTidListCache *PidTidListCache) IsEnabledFor(flags uint32) bool {
	return pidTidListCache.flags&flags > 0
}

func (pidTidListCache *PidTidListCache) Refresh(lockAcquired bool) error {
	if !lockAcquired {
		pidTidListCache.lock.Lock()
		defer pidTidListCache.lock.Unlock()
	}

	if pidTidListCache.pidLists == nil {
		pidTidListCache.pidLists = make([][]PidTid, pidTidListCache.nPart)
		for i := 0; i < pidTidListCache.nPart; i++ {
			pidTidListCache.pidLists[i] = make([]PidTid, 0)
		}
	} else {
		for i := 0; i < pidTidListCache.nPart; i++ {
			pidTidListCache.pidLists[i] = pidTidListCache.pidLists[i][0:0]
		}
	}

	names, err := getDirNames(pidTidListCache.procfsRoot)
	if err != nil {
		return err
	}

	mask, nPart, useMask := pidTidListCache.mask, pidTidListCache.nPart, pidTidListCache.mask > 0
	isPidEnabled := pidTidListCache.flags&PID_LIST_CACHE_PID_ENABLED > 0
	isTidEnabled := pidTidListCache.flags&PID_LIST_CACHE_TID_ENABLED > 0
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
			pidTidListCache.pidLists[part] = append(pidTidListCache.pidLists[part], pidTid)
			numEntries += 1
		}
		if isTidEnabled {
			// TID's belonging to PID:
			names, err := getDirNames(path.Join(pidTidListCache.procfsRoot, name, "task"))
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
					part = tid & pidTidListCache.mask
				} else {
					part = tid % pidTidListCache.nPart
				}
				pidTidListCache.pidLists[part] = append(pidTidListCache.pidLists[part], pidTid)
				numEntries += 1
			}
		}
	}
	if !pidTidListCache.initialized {
		pidTidListCache.initialized = true
	}
	pidTidListCache.retrievedTime = time.Now()
	pidTidListCache.refreshCount += 1
	return nil
}

func (pidTidListCache *PidTidListCache) GetPidTidList(part int, into []PidTid) ([]PidTid, error) {
	if part < 0 || part >= pidTidListCache.nPart {
		return nil, nil
	}
	pidTidListCache.lock.Lock()
	defer pidTidListCache.lock.Unlock()
	if !pidTidListCache.initialized || time.Since(pidTidListCache.retrievedTime) > pidTidListCache.validFor {
		err := pidTidListCache.Refresh(true)
		if err != nil {
			return nil, err
		}
	}
	pidListLen := len(pidTidListCache.pidLists[part])
	if into == nil || cap(into) < pidListLen {
		into = make([]PidTid, pidListLen)
	} else {
		into = into[:pidListLen]
	}
	copy(into, pidTidListCache.pidLists[part])
	return into, nil
}

// Mainly useful for testing:
func (pidTidListCache *PidTidListCache) Invalidate() {
	pidTidListCache.lock.Lock()
	defer pidTidListCache.lock.Unlock()
	pidTidListCache.initialized = false
}

func (pidTidListCache *PidTidListCache) GetRefreshCount() uint64 {
	pidTidListCache.lock.Lock()
	defer pidTidListCache.lock.Unlock()
	return pidTidListCache.refreshCount
}
