package lsvmi

import (
	"bytes"
	"fmt"
	"io"
	"testing"
)

type CreditMock struct {
	// What to return in the next GetCredit:
	retVal int
}

func (cm *CreditMock) GetCredit(desired, minAcceptable int) int {
	return cm.retVal
}

type CreditReaderTestStep struct {
	getCreditRetVal int
	wantReadN       int
	wantReadErr     error
}

type CreditReaderTestCase struct {
	name        string
	readBufSize int
	crBufSize   int
	steps       []*CreditReaderTestStep
}

func testCreditReader(tc *CreditReaderTestCase, t *testing.T) {
	buf := &bytes.Buffer{}
	fmt.Fprintf(
		buf, `
name=%q
readBufSize=%d
crBufSize=%d
steps:
`,
		tc.name, tc.readBufSize, tc.crBufSize,
	)

	for _, step := range tc.steps {
		fmt.Fprintf(
			buf,
			"\t%d, %d, %v\n",
			step.getCreditRetVal, step.wantReadN, step.wantReadErr,
		)
	}
	t.Log(buf)

	cc := &CreditMock{}
	cr := NewCreditReader(cc, 0, make([]byte, tc.crBufSize))
	p, s := make([]byte, tc.readBufSize), 0
	for i, step := range tc.steps {
		cc.retVal = step.getCreditRetVal
		gotN, gotErr := cr.Read(p[s:])
		if step.wantReadN != gotN || step.wantReadErr != gotErr {
			t.Fatalf(
				"step[%d]: (n, err): want: (%d, %v), got: (%d, %v)",
				i,
				step.wantReadN, step.wantReadErr,
				gotN, gotErr,
			)
		}
		s += gotN
	}
}

func TestCreditReader(t *testing.T) {
	for _, tc := range []*CreditReaderTestCase{
		{
			name:        "read_match",
			readBufSize: 10,
			crBufSize:   10,
			steps: []*CreditReaderTestStep{
				{3, 3, nil},
				{4, 4, nil},
				{3, 3, nil},
				// At EOF, the credit will not be invoked, hence the unrealistic
				// value:
				{10000, 0, io.EOF},
				{10000, 0, io.EOF},
			},
		},
		{
			name:        "zero_len_read_buf",
			readBufSize: 0,
			crBufSize:   10,
			steps: []*CreditReaderTestStep{
				// The credit will not be invoked, hence the unrealistic value:
				{10000, 0, nil},
				{10000, 0, nil},
			},
		},
		{
			name:        "under_read",
			readBufSize: 10,
			crBufSize:   20,
			steps: []*CreditReaderTestStep{
				{3, 3, nil},
				{4, 4, nil},
				{3, 3, nil},
				// The credit will not be invoked, hence the unrealistic
				// value:
				{10000, 0, nil},
				{10000, 0, nil},
			},
		},
		{
			name:        "over_read",
			readBufSize: 20,
			crBufSize:   10,
			steps: []*CreditReaderTestStep{
				{3, 3, nil},
				{4, 4, nil},
				{3, 3, nil},
				// At EOF, the credit will not be invoked, hence the unrealistic
				// value:
				{10000, 0, io.EOF},
				{10000, 0, io.EOF},
			},
		},
	} {
		t.Run(
			tc.name,
			func(t *testing.T) { testCreditReader(tc, t) },
		)
	}
}
