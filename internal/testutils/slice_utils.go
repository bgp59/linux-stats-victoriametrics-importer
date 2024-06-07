package testutils

import (
	"bytes"
	"fmt"
)

func CompareSlices[T comparable](want, got []T, name string, errBuf *bytes.Buffer) bool {
	if len(want) != len(got) {
		fmt.Fprintf(
			errBuf,
			"\nlen(%s): want: %d, got: %d",
			name, len(want), len(got),
		)
		return false
	}

	ok := true
	for i, wantVal := range want {
		gotVal := got[i]
		if wantVal != gotVal {
			fmt.Fprintf(
				errBuf,
				"\n%s[%d]: want: %v, got: %v",
				name, i, wantVal, gotVal,
			)
			ok = false
		}
	}

	return ok
}
