package utils

import (
	"bytes"
	"fmt"
	"sort"
	"testing"
)

func TestOsUtils(t *testing.T) {
	buf := &bytes.Buffer{}
	linuxOsReleaseKeys := make([]string, len(LinuxOsRelease))
	i := 0
	for key := range LinuxOsRelease {
		linuxOsReleaseKeys[i] = key
		i++
	}
	sort.Strings(linuxOsReleaseKeys)
	for _, key := range linuxOsReleaseKeys {
		fmt.Fprintf(buf, "\n\t%s: %q", key, LinuxOsRelease[key])
	}

	t.Logf(`
OSName:         %q
OSNameNorm:     %q
OSRelease:      %q
OSReleaseVer:   %v
OSBtime:        %s
LinuxClktck:    %d
LinuxClktckSec: %.06f
LinuxOsRelease: %s
`,
		OSName, OSNameNorm, OSRelease, OSReleaseVer, OSBtime, LinuxClktck, LinuxClktckSec,
		buf,
	)
}
