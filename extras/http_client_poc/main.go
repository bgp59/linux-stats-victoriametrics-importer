package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/eparparita/linux-stats-victoriametrics-importer/lsvmi"
)

const (
	DEFAULT_URL                  = "http://localhost:8080"
	DEFAULT_TARGET_SIZE          = 0x10000 // 64k
	ESTIMATED_COMPRESSION_FACTOR = 8
)

var logger = log.New(os.Stderr, "\n", log.Ldate|log.Lmicroseconds)

func generateMetrics(targetSize int, gzipCompress bool) ([]byte, error) {
	var w io.Writer

	b := &bytes.Buffer{}

	if gzipCompress {
		targetSize *= ESTIMATED_COMPRESSION_FACTOR
		w = gzip.NewWriter(b)
	} else {
		w = b
	}

	for sz, k, ts := 0, 0, int64(0); sz < targetSize; k++ {
		if k%100 == 0 {
			ts = time.Now().UnixMilli()
		}
		n, err := fmt.Fprintf(w, "metric_%06d{", k)
		if err != nil {
			return nil, err
		}
		sz += n
		for l := 0; l < k%10+1; l++ {
			if l > 0 {
				n, err = w.Write([]byte{','})
				if err != nil {
					return nil, err
				}
				sz += n
			}
			n, err = fmt.Fprintf(w, `label_%02d="val_%06d_%02d"`, l, k, l)
			if err != nil {
				return nil, err
			}
			sz += n
		}
		n, err = fmt.Fprintf(w, "} %.03f %d\n", float64(k*13+2)/11, ts)
		if err != nil {
			return nil, err
		}
		sz += n
	}
	if gzipCompress {
		w.(*gzip.Writer).Close()
	}
	return b.Bytes(), nil
}

func main() {
	var (
		url           string
		targetSize    int
		gzipCompress  bool
		chunked       bool
		numRequests   int
		rateLimitMbps string
		creditCtl     lsvmi.CreditController
	)

	flag.StringVar(&url, "url", DEFAULT_URL, "URL for import")
	flag.IntVar(&targetSize, "target-size", DEFAULT_TARGET_SIZE, "Body target size")
	flag.BoolVar(&gzipCompress, "gzip", false, "Compress body w/ gzip")
	flag.BoolVar(&chunked, "chunked", false, "Enable chunking")
	flag.IntVar(&numRequests, "num-requests", 1, "Number of requests, use 0 for unlimited")
	flag.StringVar(
		&rateLimitMbps,
		"rate-limit-mbps",
		"",
		"Rate limit as Mb[/duration], default duration 1s",
	)
	flag.Parse()

	transport := &http.Transport{
		DisableKeepAlives:   false,
		IdleConnTimeout:     5 * time.Second,
		MaxIdleConns:        5,
		MaxIdleConnsPerHost: 2,
		MaxConnsPerHost:     4,
	}

	client := &http.Client{
		Transport: transport,
	}

	if rateLimitMbps != "" {
		replenish, interval := rateLimitMbps, "1s"
		index := strings.Index(rateLimitMbps, "/")
		if index >= 0 {
			replenish = rateLimitMbps[:index]
			interval = rateLimitMbps[index+1:]
		}

		replenishFloat, err := strconv.ParseFloat(replenish, 64)
		if err != nil {
			logger.Fatal(err)
		}
		replenishValue := int(replenishFloat * 1_000_000 / 8)
		replenishInt, err := time.ParseDuration(interval)
		if err != nil {
			logger.Fatal(err)
		}
		creditCtl = lsvmi.NewCredit(replenishValue, 0, replenishInt)
		logger.Printf("credit: replenishValue=%d, replenishInt=%s\n", replenishValue, replenishInt)
	}

	metrics, err := generateMetrics(targetSize, gzipCompress)
	if err != nil {
		logger.Fatal(err)
	}

	logger.Printf("len(metrics)=%d", len(metrics))

	for k := 0; numRequests <= 0 || k < numRequests; k++ {
		var body io.Reader
		if creditCtl != nil {
			body = lsvmi.NewCreditReader(creditCtl, 256, metrics)
		} else {
			body = bytes.NewReader(metrics)
		}
		req, err := http.NewRequest("PUT", url, body)
		if err != nil {
			logger.Fatal(err)
		}
		req.Header.Add("Content-Type", "text/html")
		if chunked {
			req.ContentLength = -1 // See: https://pkg.go.dev/net/http#Request
		}
		if gzipCompress {
			req.Header.Add("Content-Encoding", "gzip")
		}
		resp, err := client.Do(req)
		if err != nil {
			logger.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			logger.Fatalf("%s\n%s", resp.Status, resp.Body)
		}
	}
}
