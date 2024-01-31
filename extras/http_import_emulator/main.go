package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	DEFAULT_PORT              = "8080"
	DEFAULT_BIND_ADDR         = "localhost"
	DEFAULT_BODY_LIMIT        = 512
	EOL_LEN                   = len("\r\n")
	DEFAULT_TRAFFIC_STATS_INT = "1s"
)

const (
	DISPLAY_REQUEST = iota
	DISPLAY_HEADERS
	DISPLAY_BODY
	// Must be last:
	NUM_DISPLAY_LEVELS
)

const (
	AUDIT_FILE_HEADER = "Timestamp,RemoteAddr,Method,URI,Proto,Size"
)

var logger = log.New(os.Stderr, "\n", log.Ldate|log.Lmicroseconds)

var (
	displayLevel      int = -1
	displayLevelNames     = [NUM_DISPLAY_LEVELS]string{
		"request",
		"headers",
		"body",
	}
	displayBodyLimit int = 0

	auditFile                *os.File
	auditFileMu              = &sync.Mutex{}
	auditFileHeaderDisplayed bool

	trafficTotalBytes int
	trafficBodyBytes  int
	trafficMu         = &sync.Mutex{}
)

func logTrafficRate(interval time.Duration) {
	prevStatsTime := time.Now()
	nextLogTime := prevStatsTime.Add(interval)
	buf := &bytes.Buffer{}

	for {
		pause := time.Until(nextLogTime)
		if pause > 0 {
			time.Sleep(pause)
		}
		statsTime := time.Now()
		statsIntMicroSec := float64(statsTime.Sub(prevStatsTime).Microseconds())
		prevStatsTime = statsTime
		trafficMu.Lock()
		buf.Reset()
		fmt.Fprintf(
			buf,
			"Traffic stats, Mbps: Body: %.03f, Total (est): %.03f\n",
			float64(trafficBodyBytes)*8/statsIntMicroSec,
			float64(trafficTotalBytes)*8/statsIntMicroSec,
		)
		trafficTotalBytes = 0
		trafficBodyBytes = 0
		trafficMu.Unlock()
		nextLogTime = nextLogTime.Add(interval)
		logger.Print(buf)
	}
}

func handleFunc(_ http.ResponseWriter, r *http.Request) {
	ts := time.Now()
	rSize, bSize := 0, 0

	var err error

	rSize += len(r.Method) + len(r.RequestURI) + len(r.Proto) + EOL_LEN

	var body []byte
	if r.Method == "PUT" || r.Method == "POST" {
		body, err = io.ReadAll(r.Body)
		if err == nil {
			bSize = len(body)
			rSize += bSize
		}
	}

	_, ok := r.Header["Transfer-Encoding"]
	if !ok && len(r.TransferEncoding) > 0 {
		r.Header["Transfer-Encoding"] = r.TransferEncoding
	}
	_, ok = r.Header["Content-Length"]
	if !ok && r.ContentLength > 0 {
		r.Header["Content-Length"] = []string{
			fmt.Sprintf("%d", r.ContentLength),
		}
	}
	isText := true
	for hdr, hdrVals := range r.Header {
		rSize += len(hdr) + EOL_LEN
		for _, val := range hdrVals {
			rSize += len(val)
		}
		switch hdr {
		case "Content-Encoding":
			if body != nil {
				for _, val := range hdrVals {
					switch val {
					case "gzip":
						b := bytes.NewBuffer(body)
						var gzipReader *gzip.Reader
						gzipReader, err = gzip.NewReader(b)
						if err == nil {
							body, err = io.ReadAll(gzipReader)
						}
					case "":
					default:
						err = fmt.Errorf("%s: unsupported encoding", val)
					}
					if err != nil {
						break
					}
				}
			}
		case "Content-Type":
			isText = false
			for _, val := range hdrVals {
				if (len(val) >= 5 && val[:5] == "text/") ||
					val == "application/x-www-form-urlencoded" {
					isText = true
					break
				}
			}
		}
	}

	buf := &bytes.Buffer{}
	if err != nil || displayLevel >= DISPLAY_REQUEST {
		fmt.Fprintf(
			buf,
			"%s %s %s %s\n",
			r.RemoteAddr,
			r.Method,
			r.RequestURI,
			r.Proto,
		)
	}
	if err != nil || displayLevel >= DISPLAY_HEADERS {
		for hdr, hdrVals := range r.Header {
			fmt.Fprintf(buf, "%s: %s\n", hdr, strings.Join(hdrVals, ", "))
		}
		buf.WriteByte('\n')
	}
	if err != nil {
		fmt.Fprintf(buf, "Error decoding request: %s\n", err)
	} else {
		if displayLevel >= DISPLAY_HEADERS {
			fmt.Fprintf(buf, "Body length: %d bytes after decoding\n", len(body))
		}
		if body != nil && displayLevel >= DISPLAY_BODY {
			buf.WriteByte('\n')
			displayBody, truncatedSize := body, 0
			if displayBodyLimit > 0 && len(body) > displayBodyLimit {
				displayBody = body[:displayBodyLimit]
				truncatedSize = len(body) - len(displayBody)
			}
			if isText {
				buf.Write(displayBody)
			} else {
				fmt.Fprintf(buf, "%v", displayBody)
			}
			if truncatedSize > 0 {
				fmt.Fprintf(buf, " ... (%d bytes truncated)", truncatedSize)
			}
			buf.WriteByte('\n')
		}

		for hdr, hdrVals := range r.Trailer {
			rSize += len(hdr) + EOL_LEN
			for _, val := range hdrVals {
				rSize += len(val)
			}
		}

		trafficMu.Lock()
		trafficTotalBytes += rSize
		trafficBodyBytes += bSize
		trafficMu.Unlock()

		if auditFile != nil {
			auditFileMu.Lock()
			if !auditFileHeaderDisplayed {
				fmt.Fprintf(auditFile, "%s\n", AUDIT_FILE_HEADER)
				auditFileHeaderDisplayed = true
			}
			fmt.Fprintf(
				auditFile,
				"%.06f,%s,%s,%s,%s,%d\n",
				float64(ts.UnixMicro())/1_000_000.,
				r.RemoteAddr, r.Method, r.RequestURI, r.Proto, rSize,
			)
			auditFileMu.Unlock()
		}
	}
	if buf.Len() > 0 {
		buf.WriteByte('\n')
		logger.Print(buf)
	}
}

func main() {
	var (
		port, bindAddr   string
		displayLevelName string
		auditFileName    string
		trafficStatsInt  string
	)

	flag.StringVar(
		&port,
		"port",
		DEFAULT_PORT,
		"Listen port",
	)
	flag.StringVar(
		&bindAddr,
		"bind-addr",
		DEFAULT_BIND_ADDR,
		"Listen bind address",
	)
	flag.StringVar(
		&displayLevelName,
		"display-level",
		"",
		fmt.Sprintf("Display level, one of: %q", displayLevelNames),
	)
	flag.StringVar(
		&auditFileName,
		"audit-file",
		"",
		"Audit file, use `-' for stdout",
	)
	flag.IntVar(
		&displayBodyLimit,
		"display-body-limit",
		DEFAULT_BODY_LIMIT,
		"Display only the first N bytes of the body, use 0 for no limit",
	)
	flag.StringVar(
		&trafficStatsInt,
		"traffic-stats-int",
		DEFAULT_TRAFFIC_STATS_INT,
		"Traffic stats interval, use 0 to disable",
	)
	flag.Parse()

	if displayLevelName != "" {
		found := false
		for level, name := range displayLevelNames {
			if name == displayLevelName {
				displayLevel = level
				found = true
				break
			}
		}
		if !found {
			logger.Fatalf(
				"%q: invalid level, it should be one of %q\n",
				displayLevelName, displayLevelNames,
			)
		}
	}

	if auditFileName == "-" {
		auditFile = os.Stdout
	} else if auditFileName != "" {
		var err error
		auditFile, err = os.OpenFile(
			auditFileName, os.O_CREATE|os.O_WRONLY, fs.ModePerm,
		)
		if err != nil {
			logger.Fatal(err)
		}
		defer auditFile.Close()
	}

	interval, err := time.ParseDuration(trafficStatsInt)
	if err != nil {
		logger.Fatal(err)
	}

	if interval > 0 {
		go logTrafficRate(interval)
	}

	addr := bindAddr
	if port != "" {
		addr += ":" + port
	}
	http.HandleFunc("/", handleFunc)
	logger.Printf("Listening on %s, display level: %d", addr, displayLevel)
	err = http.ListenAndServe(addr, nil)
	if err != nil {
		logger.Fatal(err)
	}
}
