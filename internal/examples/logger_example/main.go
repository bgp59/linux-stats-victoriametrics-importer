package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/eparparita/linux-stats-victoriametrics-importer/lsvmi"
)

var log1 = lsvmi.NewCompLogger("Comp1")
var log2 = lsvmi.NewCompLogger("Comp2")

func main() {
	flag.Parse()
	cfg, err := lsvmi.LoadLsvmiConfigFromArgs()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	err = lsvmi.SetLogger(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	log1.Debug("debug test")
	log1.Info("info test")
	log1.Warn("warn test")
	log1.Error("error test")

	log2.Debug("debug test")
	log2.Info("info test")
	log2.Warn("warn test")
	log2.Error("error test")
}
