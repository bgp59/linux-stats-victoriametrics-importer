// Benchmark strconv.Atoi v. manual conversion.

package benchmarks

import (
	"fmt"
	"strconv"
	"testing"
)

type BenchAtoiTestCase struct {
	numStr    string
	wantVal   int
	expectErr bool
}

var benchAtoiTestCases = []*BenchAtoiTestCase{
	{"1", 1, false},
	{"12", 12, false},
	{"12345678", 12345678, false},
	{"12345", 12345, false},
	{"1", 1, false},
	{"12", 12, false},
	{"12345678", 12345678, false},
	{"12345", 12345, false},
	{"1", 1, false},
	{"12", 12, false},
	{"12345678", 12345678, false},
	{"12345", 12345, false},
	{"1", 1, false},
	{"12", 12, false},
	{"12345678", 12345678, false},
	{"12345", 12345, false},
	{"1", 1, false},
	{"12", 12, false},
	{"12345678", 12345678, false},
	{"12345", 12345, false},
	{"1", 1, false},
	{"12", 12, false},
	{"12345678", 12345678, false},
	{"12345", 12345, false},
	{"1", 1, false},
	{"12", 12, false},
	{"12345678", 12345678, false},
	{"12345", 12345, false},
}

func BenchmarkStrconvAtoi(b *testing.B) {
	for n := 0; n < b.N; n++ {
		for _, tc := range benchAtoiTestCases {
			gotVal, err := strconv.Atoi(tc.numStr)
			if tc.expectErr {
				if err == nil {
					b.Fatalf("%#v: want err, got: %v", tc, err)
				}
			} else if err != nil {
				b.Fatalf("%#v: %v", tc, err)
			}
			if tc.wantVal != gotVal {
				b.Fatalf("%#v: got: %d", tc, gotVal)
			}
		}
	}
}

func BenchmarkManualAtoi(b *testing.B) {
	cvtErr := fmt.Errorf("Manual Atoi Error")
	for n := 0; n < b.N; n++ {
		for _, tc := range benchAtoiTestCases {
			gotVal, err := 0, error(nil)
			for _, c := range []byte(tc.numStr) {
				if d := int(c - '0'); 0 <= d && d <= 9 {
					gotVal = (gotVal << 3) + (gotVal << 1) + d
				} else {
					err = cvtErr
					break
				}
			}
			if tc.expectErr {
				if err == nil {
					b.Fatalf("%#v: want err, got: %v", tc, err)
				}
			} else if err != nil {
				b.Fatalf("%#v: %v", tc, err)
			}
			if tc.wantVal != gotVal {
				b.Fatalf("%#v: got: %d", tc, gotVal)
			}
		}
	}
}
