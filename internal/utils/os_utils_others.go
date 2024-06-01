// Misc Other OS related info

//go:build !linux

package utils

import "time"

var dummyBtime = time.Now()

func getClktckSec() (int64, error) {
	return DEFAULT_LINUX_CLKTCK, nil
}

func getOsBtime() time.Time {
	return dummyBtime
}

func getLinuxOsRelease() (map[string]string, error) {
	return map[string]string{}, nil
}
