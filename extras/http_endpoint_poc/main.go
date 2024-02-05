package main

import (
	"github.com/eparparita/linux-stats-victoriametrics-importer/lsvmi"
)

func main() {
	lsvmi.NewHttpEndpointPool(nil)
}
