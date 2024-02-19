// LSVMI main

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/eparparita/linux-stats-victoriametrics-importer/lsvmi"
)

var mainLog = lsvmi.NewCompLogger("main")

var useStdoutMetricsQueue = flag.Bool(
	"use-stdout-metrics-queue",
	false,
	"Print metrics to stdout instead of sending to import endpoints",
)

func main() {
	var (
		err error
	)

	// Setup things in the proper order:

	// Parse args:
	flag.Parse()

	// Config:
	lsvmi.GlobalConfig, err = lsvmi.LoadLsvmiConfigFromArgs()
	if err != nil {
		mainLog.Fatal(err)
	}

	// Logger:
	err = lsvmi.SetLogger(lsvmi.GlobalConfig)
	if err != nil {
		mainLog.Fatal(err)
	}

	// Scheduler:
	lsvmi.GlobalScheduler, err = lsvmi.NewScheduler(lsvmi.GlobalConfig)
	if err != nil {
		mainLog.Fatal(err)
	}
	defer lsvmi.GlobalScheduler.Shutdown()

	// Metrics queue:
	if !*useStdoutMetricsQueue {
		// Real queue w/ compressed metrics sent to import endpoints:
		lsvmi.GlobalHttpEndpointPool, err = lsvmi.NewHttpEndpointPool(lsvmi.GlobalConfig)
		if err != nil {
			mainLog.Fatal(err)
		}
		defer lsvmi.GlobalHttpEndpointPool.Shutdown()

		lsvmi.GlobalCompressorPool, err = lsvmi.NewCompressorPool(lsvmi.GlobalConfig)
		if err != nil {
			mainLog.Fatal(err)
		}
		defer lsvmi.GlobalCompressorPool.Shutdown()

		lsvmi.GlobalMetricsQueue = lsvmi.GlobalCompressorPool
		lsvmi.GlobalCompressorPool.Start(lsvmi.GlobalHttpEndpointPool)
	} else {
		// Simulated queue w/ metrics displayed to stdout:
		lsvmi.GlobalMetricsQueue, err = lsvmi.NewStdoutMetricsQueue(lsvmi.GlobalConfig)
		if err != nil {
			mainLog.Fatal(err)
		}
		defer lsvmi.GlobalMetricsQueue.(*lsvmi.StdoutMetricsQueue).Shutdown()

		buf := lsvmi.GlobalMetricsQueue.GetBuf()
		fmt.Fprintf(buf, "# Metrics will be displayed at stdout\n")
		lsvmi.GlobalMetricsQueue.QueueBuf(buf)
	}

	// Finally block until a signal is received:
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	sig := <-sigChan
	mainLog.Warnf("Received %s signal, exiting", sig)
}
