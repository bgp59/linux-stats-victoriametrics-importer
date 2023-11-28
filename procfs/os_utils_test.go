package procfs

import (
	"testing"
)

func TestOsUtils(t *testing.T) {
	t.Logf(
		"\nOSName: %q\nOSRelease: %q\nOSReleaseVer: %v\nLinuxClktck: %d\nLinuxClktckSec: %.06f\n",
		OSName, OSRelease, OSReleaseVer, LinuxClktck, LinuxClktckSec,
	)
}
