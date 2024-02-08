package main

import (
	"bytes"
	"flag"
	"fmt"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/lsvmi"
)

var log = lsvmi.Log

func main() {
	minInterval := 1 * time.Second

	flag.Parse()
	err := lsvmi.LoadLsvmiConfigFromArgs()
	if err != nil {
		log.Fatal(err)
	}

	err = lsvmi.SetLogger(lsvmi.LsvmiCfg)
	if err != nil {
		log.Fatal(err)
	}

	epPool, err := lsvmi.NewHttpEndpointPool(lsvmi.LsvmiCfg)
	if err != nil {
		log.Fatal(err)
	}

	buf := &bytes.Buffer{}
	for k := 1; ; k++ {
		nextAfter := time.Now().Add(minInterval)
		buf.Reset()
		fmt.Fprintf(buf, "Hello World# %d", k)
		err = epPool.SendBuffer(buf.Bytes(), -1, false)
		if err != nil {
			log.Warn(err)
		}
		pause := time.Until(nextAfter)
		if pause > 0 {
			time.Sleep(pause)
		}
	}
}
