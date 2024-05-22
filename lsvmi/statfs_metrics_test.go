package lsvmi

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/eparparita/linux-stats-victoriametrics-importer/internal/testutils"
)

type StatfsKeepFsTypeTestCase struct {
	name        string
	includeList []string
	excludeList []string
	wantKeep    []string
	wantNotKeep []string
}

func testStatfsKeepFsType(tc *StatfsKeepFsTypeTestCase, t *testing.T) {
	tlc := testutils.NewTestLogCollect(t, Log, nil)
	defer tlc.RestoreLog()

	cfg := DefaultStatfsMetricsConfig()
	if tc.includeList != nil {
		cfg.IncludeFilesystemTypes = make([]string, len(tc.includeList))
		copy(cfg.IncludeFilesystemTypes, tc.includeList)
	}
	if tc.excludeList != nil {
		cfg.ExcludeFilesystemTypes = make([]string, len(tc.excludeList))
		copy(cfg.ExcludeFilesystemTypes, tc.excludeList)
	}

	sfsm, err := NewStatfsMetrics(cfg)
	if err != nil {
		t.Fatal(err)
	}

	errBuf := &bytes.Buffer{}
	errFsType := make([]string, 0)

	errFsType = errFsType[:0]
	for _, fsType := range tc.wantKeep {
		if !sfsm.keepFsType(fsType) {
			errFsType = append(errFsType, fsType)
		}
	}
	if len(errFsType) > 0 {
		fmt.Fprintf(errBuf, "\nmissing keep: %q", errFsType)
	}

	errFsType = errFsType[:0]
	for _, fsType := range tc.wantNotKeep {
		if sfsm.keepFsType(fsType) {
			errFsType = append(errFsType, fsType)
		}
	}
	if len(errFsType) > 0 {
		fmt.Fprintf(errBuf, "\nunexpected keep: %q", errFsType)
	}

	if errBuf.Len() > 0 {
		t.Fatal(errBuf)
	}
}

func TestStatfsKeepFsType(t *testing.T) {
	for _, tc := range []*StatfsKeepFsTypeTestCase{
		{
			name: "include_all,exclude_none",
			wantKeep: []string{
				"incFsType",
				"otherFsType",
			},
		},
		{
			name: "include_some,exclude_none",
			includeList: []string{
				"incFsType",
			},
			wantKeep: []string{
				"incFsType",
			},
			wantNotKeep: []string{
				"excFsType",
				"otherFsType",
			},
		},
		{
			name: "include_all,exclude_some",
			excludeList: []string{
				"excFsType",
			},
			wantKeep: []string{
				"incFsType",
				"otherFsType",
			},
			wantNotKeep: []string{
				"excFsType",
			},
		},
		{
			name: "include_some,exclude_some",
			includeList: []string{
				"incFsType",
			},
			excludeList: []string{
				"excFsType",
			},
			wantKeep: []string{
				"incFsType",
			},
			wantNotKeep: []string{
				"excFsType",
				"otherFsType",
			},
		},
	} {
		t.Run(
			tc.name,
			func(t *testing.T) { testStatfsKeepFsType(tc, t) },
		)
	}
}
