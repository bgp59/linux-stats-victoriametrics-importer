// Collectable log, (*testing.T).Log style.

// If the test is not running in verbose mode, collect the app logger's output
// and display it JIT at Fatal[f] invocation:

package testutils

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"runtime"
	"testing"
)

// The interface expected from a collectable log:
type CollectableLog interface {
	GetLevel() any
	SetLevel(level any)
	GetOutput() io.Writer
	SetOutput(out io.Writer)
}

type TestingLogCollect struct {
	buf        *bytes.Buffer
	log        CollectableLog
	savedOut   io.Writer
	savedLevel any
	t          *testing.T
}

func NewTestingLogCollect(t *testing.T, log CollectableLog, level any) *TestingLogCollect {
	tlc := &TestingLogCollect{t: t}
	if log != nil {
		if !testing.Verbose() {
			tlc.buf = &bytes.Buffer{}
			tlc.log = log
			tlc.savedOut = log.GetOutput()
			log.SetOutput(tlc.buf)
		}
		if level != nil {
			tlc.savedLevel = log.GetLevel()
			log.SetLevel(level)
		}
	}
	return tlc
}

func (tlc *TestingLogCollect) fatal(format string, args ...any) {
	if tlc.buf != nil && tlc.buf.Len() > 0 {
		tlc.t.Log("Collected log:\n\n" + tlc.buf.String())
	}
	callers := make([]uintptr, 1)
	runtime.Callers(3, callers)
	frames := runtime.CallersFrames(callers)
	frame, _ := frames.Next()
	testFileLineNum := fmt.Sprintf("from %s:%d:", path.Base(frame.File), frame.Line)
	if format != "" {
		tlc.t.Fatalf(testFileLineNum+" "+format, args...)
	} else {
		newArgs := make([]any, len(args)+1)
		newArgs[0] = testFileLineNum
		copy(newArgs[1:], args)
		tlc.t.Fatal(newArgs...)
	}
}

func (tlc *TestingLogCollect) Fatal(args ...any) {
	tlc.fatal("", args...)
}

func (tlc *TestingLogCollect) Fatalf(format string, args ...any) {
	tlc.fatal(format, args...)
}

func (tlc *TestingLogCollect) RestoreLog() {
	if tlc.log != nil {
		if tlc.savedOut != nil {
			tlc.log.SetOutput(tlc.savedOut)
		}
		if tlc.savedLevel != nil {
			tlc.log.SetLevel(tlc.savedLevel)
		}
	}
}
