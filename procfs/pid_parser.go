// Common utils for /proc/PID parsers:

package procfs

// Notes about PID stats parsers:
//
// 1. The metrics generator for PID stats will maintain a cache of previous scan
// objects on a per thread basis + an extra scratch object for the current scan.
// The flow is as follows:
//   pull previous_pid_stat object from cache for the target PID, TID
//   invoke parse method on scratch_pid_stat using the file path from previous_pid_stat
//   compare scratch_pid_stat v. previous_pid_stat for deltas
//   scratch_pid_stat becomes the new state in the cache and the freed previous_pid_stat
//   becomes the new scratch object
// Therefore, unlike other parsers, the parse method should provide the ability
// to change the path without allocating new objects.
//
// 2. To assist with unit testing PID stats parsers will be defined as
// interfaces which can be replaced by test objects returning test case data.

import (
	"path"
	"strconv"
)

const (
	// Special PID to indicate self:
	SELF_PID = -1
)

type PidTidPath struct {
	procfsRoot string
}

func NewPidTidPath(procfsRoot string) *PidTidPath {
	return &PidTidPath{
		procfsRoot: procfsRoot,
	}
}

func BuildPidTidPath(procfsRoot string, pid, tid int) string {
	pidStr := strconv.Itoa(pid)
	if pid <= 0 {
		pidStr = "self"
	}
	if tid <= 0 {
		return path.Join(procfsRoot, pidStr)
	} else {
		return path.Join(procfsRoot, pidStr, "task", strconv.Itoa(tid))
	}
}

func (pidTidPath *PidTidPath) Path(pid, tid int, fName string) string {
	return BuildPidTidPath(pidTidPath.procfsRoot, pid, tid)
}
