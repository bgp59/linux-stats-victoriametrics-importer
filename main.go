// LSVMI main

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bgp59/linux-stats-victoriametrics-importer/buildinfo"
	"github.com/bgp59/linux-stats-victoriametrics-importer/lsvmi"
)

const (
	SHUTDOWN_TIMEOUT = 3 * time.Second
)

var mainLog = lsvmi.NewCompLogger("main")

var useStdoutMetricsQueueArg = flag.Bool(
	"use-stdout-metrics-queue",
	false,
	lsvmi.FormatFlagUsage(
		`Print metrics to stdout instead of sending to import endpoints`,
	),
)

var printVerArg = flag.Bool(
	"version",
	false,
	lsvmi.FormatFlagUsage(
		`Print version and other build info and exit`,
	),
)

func main() {
	var (
		err error
	)

	// Setup things in the proper order:

	// Parse args:
	flag.Parse()

	if *printVerArg {
		fmt.Fprintf(
			os.Stderr,
			"Version: %s\nGit Info: %s\n", buildinfo.Version, buildinfo.GitInfo,
		)
		return
	}

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

	// Info version:
	mainLog.Infof("Version: %s, Git Info: %s", buildinfo.Version, buildinfo.GitInfo)

	// Metrics queue:
	if !*useStdoutMetricsQueueArg {
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
		defer lsvmi.GlobalHttpEndpointPool.Shutdown() // may timeout if all endpoints are down
		defer lsvmi.GlobalCompressorPool.Shutdown()
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

	// Log instance and hostname, useful for dashboard variable selection:
	mainLog.Infof("Instance: %s, Hostname: %s", lsvmi.GlobalInstance, lsvmi.GlobalHostname)

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
