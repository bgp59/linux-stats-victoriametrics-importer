package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"

	"github.com/eparparita/linux-stats-victoriametrics-importer/lsvmi"
)

var comp1Log = lsvmi.Log.WithField(
	lsvmi.LOGGER_COMPONENT_FIELD_NAME,
	"Comp1",
)

var comp2Log = lsvmi.Log.WithField(
	lsvmi.LOGGER_COMPONENT_FIELD_NAME,
	"Comp2",
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

	for _, logger := range []*logrus.Entry{
		comp1Log, comp2Log,
	} {
		logger.Debug("debug test")
		logger.Info("info test")
		logger.Warn("warn test")
		logger.Error("error test")
	}
}
