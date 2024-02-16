// Misc OS related info

package utils

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/tklauser/go-sysconf"

	"golang.org/x/sys/unix"
)

var (
	OSName         string
	OSRelease      string
	OSReleaseVer   []int
	LinuxClktck    int64 = 100
	LinuxClktckSec float64
)

func init() {
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
		fmt.Fprintf(os.Stderr, "unix.Uname(): %v\n", err)
		os.Exit(1)
	}

	OSName = strings.ToLower(zeroSuffixBufToString(uname.Sysname[:]))

	OSRelease = zeroSuffixBufToString(uname.Release[:])
	semVerStr := strings.Split(OSRelease, ".")
	OSReleaseVer = make([]int, len(semVerStr))
	for i, v := range semVerStr {
		OSReleaseVer[i], err = strconv.Atoi(v)
		if err != nil {
			if i < len(semVerStr)-1 {
				fmt.Fprintf(os.Stderr, "error parsing OS OSRelease %q: %v", OSRelease, err)
				os.Exit(1)
			}
			// Maybe it is maj.min.rel<something>
			OSReleaseVer[i] = -1
			_, err = fmt.Sscanf(v, "%d", &OSReleaseVer[i])
			if err != nil {
				fmt.Fprintf(os.Stderr, "error parsing OS OSRelease %q: %v", OSRelease, err)
				os.Exit(1)

			}
			if OSReleaseVer[i] < 0 {
				fmt.Fprintf(os.Stderr, "error parsing OS OSRelease %q", OSRelease)
				os.Exit(1)
			}
		}
	}

	if OSName == "linux" {
		clktck, err := sysconf.Sysconf(sysconf.SC_CLK_TCK)
		if err == nil {
			LinuxClktck = clktck
		} else {
			fmt.Fprintf(os.Stderr, "Sysconf(SC_CLK_TCK): %v, using %d", err, LinuxClktck)
		}
	}
	LinuxClktckSec = 1. / float64(LinuxClktck)
}
