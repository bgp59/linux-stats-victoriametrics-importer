// Collect output from logurs.Log is not in verbose mode and display it JIT at
// test Fatal time

package testutils

import (
	"bytes"
	"fmt"
	"io"
	"path"
	"runtime"
	"testing"

	"github.com/sirupsen/logrus"
)

type TestingLogCollect struct {
	buf      *bytes.Buffer
	log      *logrus.Logger
	savedOut io.Writer
	t        *testing.T
}

func NewTestingLogCollect(t *testing.T, log *logrus.Logger) *TestingLogCollect {
	tlc := &TestingLogCollect{t: t}
	if !testing.Verbose() && log != nil {
		tlc.buf = &bytes.Buffer{}
		tlc.log = log
		tlc.savedOut = log.Out
		log.SetOutput(tlc.buf)
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
	if tlc.savedOut != nil {
		tlc.log.SetOutput(tlc.savedOut)
	}
}
