// Misc OS related info

package utils

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"
)

const (
	DEFAULT_LINUX_CLKTCK = 100
)

var (
	OSName         string
	OSNameNorm     string
	OSRelease      string
	OSReleaseVer   []int
	OSBtime        = getOsBtime()
	LinuxClktck    = int64(DEFAULT_LINUX_CLKTCK)
	LinuxClktckSec = 1. / float64(LinuxClktck)
	LinuxOsRelease = make(map[string]string)
)

func setUnameOSInfo() error {
	zeroSuffixBufToString := func(buf []byte) string {
		i := bytes.IndexByte(buf, 0)
		if i < 0 {
			i = len(buf)
		}
		return string(buf[:i])
	}

	uname := unix.Utsname{}
	err := unix.Uname(&uname)
	if err != nil {
		return fmt.Errorf("unix.Uname(): %v", err)
	}

	OSName = zeroSuffixBufToString(uname.Sysname[:])
	OSNameNorm = strings.ToLower(OSName)

	OSRelease = zeroSuffixBufToString(uname.Release[:])
	semVerStr := strings.Split(OSRelease, ".")
	OSReleaseVer = make([]int, len(semVerStr))
	for i, v := range semVerStr {
		OSReleaseVer[i], err = strconv.Atoi(v)
		if err != nil {
			if i < len(semVerStr)-1 {
				return fmt.Errorf("error parsing OS OSRelease %q: %v", OSRelease, err)
			}
			// Maybe it is maj.min.rel<something>
			OSReleaseVer[i] = -1
			_, err = fmt.Sscanf(v, "%d", &OSReleaseVer[i])
			if err != nil {
				return fmt.Errorf("error parsing OS OSRelease %q: %v", OSRelease, err)
			}
			if OSReleaseVer[i] < 0 {
				return fmt.Errorf("error parsing OS OSRelease %q", OSRelease)
			}
		}
	}
	return nil
}

func init() {
	if err := setUnameOSInfo(); err != nil {
		fmt.Fprintf(os.Stderr, "setUnameOSInfo(): %v\n", err)
	}

	if clktck, err := getClktckSec(); err == nil {
		LinuxClktck = clktck
		LinuxClktckSec = 1. / float64(LinuxClktck)
	} else {
		fmt.Fprintf(os.Stderr, "getClktckSec(): %v, using %d\n", err, LinuxClktck)
	}

	if linuxOsRelease, err := getLinuxOsRelease(); err == nil {
		LinuxOsRelease = linuxOsRelease
	} else {
		fmt.Fprintf(os.Stderr, "linuxOsRelease(): %v\n", err)
	}
}
