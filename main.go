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

	// Metrics queue:
	if !*useStdoutMetricsQueue {
		// Real queue w/ compressed metrics sent to import endpoints:
		lsvmi.GlobalHttpEndpointPool, err = lsvmi.NewHttpEndpointPool(lsvmi.GlobalLsvmiConfig)
		if err != nil {
			mainLog.Fatal(err)
		}

		lsvmi.GlobalCompressorPool, err = lsvmi.NewCompressorPool(lsvmi.GlobalLsvmiConfig)
		if err != nil {
			mainLog.Fatal(err)
		}
		lsvmi.GlobalMetricsQueue = lsvmi.GlobalCompressorPool

		lsvmi.GlobalCompressorPool.Start(lsvmi.GlobalHttpEndpointPool)
		// N.B. stop the HTTP pool *before* the compressor pool, otherwise the
		// latter may be stuck in send:
		defer lsvmi.GlobalCompressorPool.Shutdown()
		defer lsvmi.GlobalHttpEndpointPool.Shutdown()
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

	// Scheduler:
	lsvmi.GlobalScheduler, err = lsvmi.NewScheduler(lsvmi.GlobalLsvmiConfig)
	if err != nil {
		mainLog.Fatal(err)
	}
	lsvmi.GlobalScheduler.Start()
	defer lsvmi.GlobalScheduler.Shutdown()

	// Initialize metrics generators:
	err = lsvmi.InitCommonMetrics(lsvmi.GlobalLsvmiConfig)
	if err != nil {
		mainLog.Fatal(err)
	}

	taskList := make([]*lsvmi.Task, 0)
	for _, tb := range lsvmi.TaskBuilders.List() {
		tasks, err := tb(lsvmi.GlobalLsvmiConfig)
		if err != nil {
			mainLog.Fatal(err)
		}
		if len(tasks) > 0 {
			taskList = append(taskList, tasks...)
		}
	}

	for _, task := range taskList {
		lsvmi.GlobalScheduler.AddNewTask(task)
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
