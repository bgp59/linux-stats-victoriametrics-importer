// LSVMI main

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/lsvmi"
)

const (
	SHUTDOWN_TIMEOUT = 5 * time.Second
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
	lsvmi.GlobalLsvmiConfig, err = lsvmi.LoadLsvmiConfigFromArgs()
	if err != nil {
		mainLog.Fatal(err)
	}

	// Logger:
	err = lsvmi.SetLogger(lsvmi.GlobalLsvmiConfig)
	if err != nil {
		mainLog.Fatal(err)
	}

	// Scheduler:
	lsvmi.GlobalScheduler, err = lsvmi.NewScheduler(lsvmi.GlobalLsvmiConfig)
	if err != nil {
		mainLog.Fatal(err)
	}
	defer lsvmi.GlobalScheduler.Shutdown()

	// Metrics queue:
	if !*useStdoutMetricsQueue {
		// Real queue w/ compressed metrics sent to import endpoints:
		lsvmi.GlobalHttpEndpointPool, err = lsvmi.NewHttpEndpointPool(lsvmi.GlobalLsvmiConfig)
		if err != nil {
			mainLog.Fatal(err)
		}
		defer lsvmi.GlobalHttpEndpointPool.Shutdown()

		lsvmi.GlobalCompressorPool, err = lsvmi.NewCompressorPool(lsvmi.GlobalLsvmiConfig)
		if err != nil {
			mainLog.Fatal(err)
		}
		defer lsvmi.GlobalCompressorPool.Shutdown()

		lsvmi.GlobalMetricsQueue = lsvmi.GlobalCompressorPool
		lsvmi.GlobalCompressorPool.Start(lsvmi.GlobalHttpEndpointPool)
	} else {
		// Simulated queue w/ metrics displayed to stdout:
		lsvmi.GlobalMetricsQueue, err = lsvmi.NewStdoutMetricsQueue(lsvmi.GlobalLsvmiConfig)
		if err != nil {
			mainLog.Fatal(err)
		}
		defer lsvmi.GlobalMetricsQueue.(*lsvmi.StdoutMetricsQueue).Shutdown()

		buf := lsvmi.GlobalMetricsQueue.GetBuf()
		fmt.Fprintf(buf, "# Metrics will be displayed at stdout\n")
		lsvmi.GlobalMetricsQueue.QueueBuf(buf)
	}

	// Initialize metrics generators:
	err = lsvmi.InitCommonMetrics(lsvmi.GlobalLsvmiConfig)
	if err != nil {
		mainLog.Fatal(err)
	}

	// Block until a signal is received:
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	sig := <-sigChan
	mainLog.Warnf("Received %s signal, exiting", sig)

	// Set a timeout watchdog, just in case:
	go func() {
		timer := time.NewTimer(SHUTDOWN_TIMEOUT)
		<-timer.C
		mainLog.Fatalf("shutdown timed out after %s", SHUTDOWN_TIMEOUT)
	}()
}
