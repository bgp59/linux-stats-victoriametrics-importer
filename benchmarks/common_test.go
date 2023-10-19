// Definitions common to all benchmarks:

package benchmarks

import (
	"path"

	"github.com/eparparita/linux-stats-victoriametrics-importer/testutils"
)

const (
	PATH_TO_ROOT         = ".."
	TESTDATA_PROCFS_ROOT = "../testdata/proc"
)

var TestDataProcDir = path.Join(PATH_TO_ROOT, testutils.TESTDATA_PROC_SUBDIR)
