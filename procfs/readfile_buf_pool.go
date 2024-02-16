// Read a file into a reusable buffer from a pool; this should be more efficient
// than allocating a buffer every time and relying on GC.

package procfs

import (
	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/utils"
)

// Predefined pools:
var (
	ReadFileBufPool16k         = utils.NewReadFileBufPool(32, 0x4000)
	ReadFileBufPool32k         = utils.NewReadFileBufPool(32, 0x8000)
	ReadFileBufPool64k         = utils.NewReadFileBufPool(32, 0x10000)
	ReadFileBufPool128k        = utils.NewReadFileBufPool(16, 0x20000)
	ReadFileBufPool256k        = utils.NewReadFileBufPool(8, 0x40000)
	ReadFileBufPool1m          = utils.NewReadFileBufPool(4, 0x100000)
	ReadFileBufPoolReadUnbound = utils.NewReadFileBufPool(4, utils.READ_FILE_BUF_POOL_MAX_READ_SIZE_UNBOUND)
)
