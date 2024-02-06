package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/eparparita/linux-stats-victoriametrics-importer/lsvmi"
)

func main() {
	flag.Parse()
	err := lsvmi.LoadLsvmiConfigFromArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = lsvmi.SetLogger(lsvmi.LsvmiCfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	epPool, err := lsvmi.NewHttpEndpointPool(lsvmi.LsvmiCfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	for {
		ep := epPool.GetCurrentHealthy()
		if ep == nil {
			break
		}
		epPool.DeclareUnhealthy(ep)
	}

	for {
		ep := epPool.GetCurrentHealthy()
		if ep != nil {
			break
		}
	}

}
