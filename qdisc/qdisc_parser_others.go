//go:build !linux

package qdisc

import (
	"fmt"
	"runtime"
)

var QdiscAvail = false

func (qs *QdiscStats) Update() error {
	return fmt.Errorf("qdisc not supported for GOOS=%s", runtime.GOOS)
}
