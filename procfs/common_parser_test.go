package procfs

import (
	"testing"
)

type getCurrentLineTestCase struct {
	name     string
	buf      []byte
	pos      int
	wantLine string
}

var getCurrentLineTestBuf = []byte(`
line 1
line 2
line 3
`)[1:] // discard 1st `\n'

func testGetCurrentLine(tc *getCurrentLineTestCase, t *testing.T) {
	t.Logf("pos=%d", tc.pos)
	gotLine := getCurrentLine(tc.buf, tc.pos)
	if tc.wantLine != gotLine {
		t.Fatalf("getCurrentLine(%q, %q):\nwant: %q\ngot: %q", tc.buf, tc.pos, tc.wantLine, gotLine)
	}
}

func TestGetCurrentLine(t *testing.T) {
	for _, tc := range []*getCurrentLineTestCase{
		{
			"",
			getCurrentLineTestBuf,
			0,
			"line 1",
		},
		{
			"",
			getCurrentLineTestBuf,
			-1,
			"line 1",
		},
		{
			"",
			getCurrentLineTestBuf,
			7,
			"line 2",
		},
		{
			"",
			getCurrentLineTestBuf,
			-12,
			"line 2",
		},
	} {
		t.Run(
			tc.name,
			func(t *testing.T) { testGetCurrentLine(tc, t) },
		)
	}
}
