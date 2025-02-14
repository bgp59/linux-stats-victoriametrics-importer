package main

import (
	"flag"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/bgp59/linux-stats-victoriametrics-importer/lsvmi"
)

const (
	DEFAULT_SEND_RATE  = "1"
	DEFAULT_SEND_LIMIT = 0
	MIN_SEND_SIZE      = 64
	MAX_SEND_SIZE      = 0x10000 //64k
)

var log = lsvmi.Log

func main() {
	var (
		sendRate   string
		sendLimit  uint64
		sendSize   int
		sendCredit *lsvmi.Credit
	)

	flag.StringVar(
		&sendRate,
		"poc-send-rate",
		DEFAULT_SEND_RATE,
		"Send rate, op[:interval] (implied interval is 1s)",
	)

	flag.Uint64Var(
		&sendLimit,
		"poc-send-limit",
		DEFAULT_SEND_LIMIT,
		"Stop after N sends, 0 for unlimited",
	)

	flag.IntVar(
		&sendSize,
		"poc-send-size",
		MIN_SEND_SIZE,
		fmt.Sprintf("Send size, %d..%d", MIN_SEND_SIZE, MAX_SEND_SIZE),
	)

	flag.Parse()

	lsvmiConfig, err := lsvmi.LoadLsvmiConfigFromArgs()
	if err != nil {
		log.Fatal(err)
	}

	err = lsvmi.SetLogger(lsvmiConfig)
	if err != nil {
		log.Fatal(err)
	}

	epPool, err := lsvmi.NewHttpEndpointPool(lsvmiConfig)
	if err != nil {
		log.Fatal(err)
	}

	if sendSize < MIN_SEND_SIZE {
		sendSize = MIN_SEND_SIZE
	}
	if sendSize > MAX_SEND_SIZE {
		sendSize = MAX_SEND_SIZE
	}

	buf := make([]byte, sendSize)
	text := buf[16:]
	for i, c := 0, byte(' '); i < len(text); i++ {
		if i%65 == 0 {
			text[i] = '\n'
		} else {
			text[i] = c
			c += 1
			if c > '~' {
				c = ' '
			}
		}
	}

	if sendRate != "" {
		var replenishValue int
		replenishValueStr, replenishInt := sendRate, 1*time.Second
		i := strings.Index(sendRate, ":")
		if i > 0 {
			replenishValueStr = sendRate[:i]
			replenishInt, err = time.ParseDuration(sendRate[i+1:])
			if err != nil {
				log.Fatal(err)
			}
		}
		replenishValue, err = strconv.Atoi(replenishValueStr)
		if err != nil {
			log.Fatal(err)
		}
		if replenishValue > 0 {
			sendCredit = lsvmi.NewCredit(replenishValue, 0, replenishInt)
		}
	}

	for sendCount, done := uint64(0), false; !done; {
		n := 1
		if sendCredit != nil {
			n = sendCredit.GetCredit(math.MaxInt, 1)
		}
		for ; !done && n > 0; n-- {
			copy(buf, fmt.Sprintf("%016d", sendCount))
			err = epPool.SendBuffer(buf, -1, false)
			if err != nil {
				log.Warn(err)
			}
			sendCount++
			done = sendLimit > 0 && sendCount >= sendLimit
		}
	}
}
