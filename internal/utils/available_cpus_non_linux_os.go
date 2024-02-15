// Count available CPUs based on affinity

//go:build !linux

package utils

import (
	"runtime"
)

func CountAvailableCPUs() int {
	return runtime.NumCPU()
}
