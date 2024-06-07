//go:build !linux

package qdisc

import (
	"fmt"
	"runtime"
)

var QdiscAvailable = false

func (qs *QdiscStats) Parse() error {
	return fmt.Errorf("qdisc not supported for GOOS=%s", runtime.GOOS)
}
