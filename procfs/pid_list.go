// Return the list of PID, TID to scan

// The task of scanning processes (PIDs) and threads (TIDs) may be divided upon
// multiple goroutines. Rather than having each goroutine scan /proc,
// maintaining a cache with the most recent scan with an expiration date. If
// multiple goroutines ask for the relevant lists within the cache window then
// the results may be re-used.

package procfs

import (
	"os"
	"path"
	"strconv"
	"sync"
	"time"
)

const (
	PID_LIST_CACHE_PID_ENABLED = uint32(1 << 0)
	PID_LIST_CACHE_TID_ENABLED = uint32(1 << 1)
	PID_LIST_CACHE_ALL_ENABLED = PID_LIST_CACHE_PID_ENABLED | PID_LIST_CACHE_TID_ENABLED
)

type PidTidPair struct {
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

	// The timestamp of the latest retrieval:
	retrievedTime time.Time

	// How long a scan is valid:
	validFor time.Duration

	// What is being cached:
	flags uint32

	// The actual lists, indexed by the number of partitions:
	pidLists [][]PidTidPair

	// The root of the file system; typically /proc:
	procfsRoot string

	// Lock protection:
	lock sync.Mutex
}

func NewPidListCache(nPart int, validFor time.Duration, procfsRoot string, flags uint32) *PidListCache {
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
		nPart:         nPart,
		mask:          mask,
		retrievedTime: time.Now().Add(-2 * validFor), // this should force a refresh at the next call
		validFor:      validFor,
		flags:         flags,
		procfsRoot:    procfsRoot,
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

func (c *PidListCache) RetrievedTime() time.Time {
	return c.retrievedTime
}

func (c *PidListCache) IsEnabledFor(flags uint32) bool {
	return c.flags&flags > 0
}

func (c *PidListCache) Refresh(lockAcquired bool) error {
	if !lockAcquired {
		c.lock.Lock()
		defer c.lock.Unlock()
	}

	if c.pidLists == nil {
		c.pidLists = make([][]PidTidPair, c.nPart)
		for i := 0; i < c.nPart; i++ {
			c.pidLists[i] = make([]PidTidPair, 0)
		}
	} else {
		for i := 0; i < c.nPart; i++ {
			c.pidLists[i] = c.pidLists[i][0:0]
		}
	}

	names, err := getDirNames(c.procfsRoot)
	if err != nil {
		return err
	}

	mask, nPart, useMask := c.mask, c.nPart, c.mask > 0
	isPidEnabled := c.flags&PID_LIST_CACHE_PID_ENABLED > 0
	isTidEnabled := c.flags&PID_LIST_CACHE_TID_ENABLED > 0
	numEntries := 0
	for _, name := range names {
		var part int

		pid64, err := strconv.ParseInt(name, 10, 64)
		if err != nil {
			continue
		}
		pid := int(pid64)
		if pid == 0 {
			// Not a real life scenario, however test data may have it:
			continue
		}

		if isPidEnabled {
			if useMask {
				part = pid & mask
			} else {
				part = pid % nPart
			}
			c.pidLists[part] = append(c.pidLists[part], PidTidPair{pid, PID_STAT_PID_ONLY_TID})
			numEntries += 1
		}
		if isTidEnabled {
			// TID's belonging to PID:
			names, err := getDirNames(path.Join(c.procfsRoot, name, "task"))
			if err != nil {
				// Silently ignore, maybe the PID just went away:
				continue
			}
			for _, name := range names {
				tid64, err := strconv.ParseInt(name, 10, 64)
				if err != nil {
					continue
				}
				tid := int(tid64)
				if useMask {
					part = tid & c.mask
				} else {
					part = tid % c.nPart
				}
				c.pidLists[part] = append(c.pidLists[part], PidTidPair{pid, tid})
				numEntries += 1
			}
		}
	}
	c.retrievedTime = time.Now()
	return nil
}

func (c *PidListCache) GetPidTidList(part int, into []PidTidPair) ([]PidTidPair, error) {
	if part < 0 || part >= c.nPart {
		return nil, nil
	}
	c.lock.Lock()
	defer c.lock.Unlock()
	if time.Since(c.retrievedTime) > c.validFor {
		err := c.Refresh(true)
		if err != nil {
			return nil, err
		}
	}
	pidListLen := len(c.pidLists[part])
	if into == nil || cap(into) < pidListLen {
		into = make([]PidTidPair, pidListLen)
	} else {
		into = into[:pidListLen]
	}
	copy(into, c.pidLists[part])
	return into, nil
}
