// Definitions common to all tests:

package procfs

import (
	"path"

	"github.com/eparparita/linux-stats-victoriametrics-importer/testutils"
)

const (
	PATH_TO_ROOT = ".."
)

var TestDataProcDir = path.Join(PATH_TO_ROOT, testutils.TESTDATA_PROC_SUBDIR)
