package main

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/emypar/linux-stats-victoriametrics-importer/internal/utils"
)

func main() {

	buf := &bytes.Buffer{}
	linuxOsReleaseKeys := make([]string, len(utils.LinuxOsRelease))
	i := 0
	for key := range utils.LinuxOsRelease {
		linuxOsReleaseKeys[i] = key
		i++
	}
	sort.Strings(linuxOsReleaseKeys)
	for _, key := range linuxOsReleaseKeys {
		fmt.Fprintf(buf, "\n\t%s: %q", key, utils.LinuxOsRelease[key])
	}

	fmt.Printf(`
OSName:         %q
OSNameNorm:     %q
OSRelease:      %q
OSReleaseVer:   %v
OSMachine:      %v
OSBtime:        %s
LinuxClktck:    %d
LinuxClktckSec: %.06f
LinuxOsRelease: %s
`,
		utils.OSName,
		utils.OSNameNorm,
		utils.OSRelease,
		utils.OSReleaseVer,
		utils.OSMachine,
		utils.OSBtime,
		utils.LinuxClktck,
		utils.LinuxClktckSec,
		buf,
	)

}
