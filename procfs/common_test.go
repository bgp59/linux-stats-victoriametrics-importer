// Definitions common to all tests:

package procfs

import (
	"path"

	"github.com/eparparita/linux-stats-victoriametrics-importer/testutils"
)

const (
	PATH_TO_ROOT         = ".."
	TESTDATA_PROCFS_ROOT = "../testdata/procfs_test"
)

var TestDataProcDir = path.Join(PATH_TO_ROOT, testutils.TESTDATA_PROC_SUBDIR)
