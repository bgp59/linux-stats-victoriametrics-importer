// Qdisc stats

//go:build !linux

package utils

import (
	"fmt"
	"runtime"
)

var QdiscAvail = false

func (uQdiscInfo *UtilsQdiscInfo) Get() error {
	return fmt.Errorf("qdisc not supported for GOOS=%s", runtime.GOOS)
}
