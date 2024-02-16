package utils

import (
	"testing"
)

func TestOsUtils(t *testing.T) {
	t.Logf(`
OSName:         %q
OSRelease:      %q
OSReleaseVer:   %v
LinuxClktck:    %d
LinuxClktckSec: %.06f
`,
		OSName, OSRelease, OSReleaseVer, LinuxClktck, LinuxClktckSec,
	)
}
