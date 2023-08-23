// unit64 as float64 test bed.

// Some Linux stats counters are 64 bit integers but VictoriaMetrics uses
// float64 for time series values. Values larger than 1<<53 will suffer a loss
// of precision which in turn may affect delta and rate calculations. The
// solution is to split uint64 values into lower N bits and higher 64-N ones and
// to store them in parallel timeseries.

// This sample code illustrates a way of using the high,low float64 for
// computing accurate deltas matching the results for the original uint64
// values. It should be noted that var Golang uint64 arithmetic handles counter
// rollover, e.g.
//      var i1, i2 uint64 = 0, math.MaxUint64
//      fmt.Println(i1 - i2)
// will yield 1, so the logic using only float64 should apply a correction.
//
// The code below checks that the calculation for delta = i1-i2, i1,i2 =
// 0..(1<<64 -1), delta = 0..(1<<53-1), using the floats matches the direct uint64
// result for both cases where i1 >= i2 and i1 < i2 (rollover).

package main

import (
	"flag"
	"fmt"
)

func Uint64ToHighLowFloat64(i uint64, lowNumBits int) (float64, float64) {
	return float64(i >> lowNumBits), float64(i & (1<<lowNumBits - 1))
}

// Emulate PromQL delta:
func HigLowFloat64Delta(high1, low1, high2, low2 float64, lowNumBits int) float64 {
	highFactor := float64(uint64(1) << lowNumBits)
	deltaHigh, deltaLow := high1-high2, low1-low2

	if deltaHigh < 0 || (deltaHigh == 0 && low1 < low2) {
		highCounterRolloverCorrection := float64(uint64(1) << (64 - lowNumBits))
		deltaHigh += highCounterRolloverCorrection
	}

	return deltaHigh*highFactor + deltaLow
}

func Uint64ToFloat64VerifyDelta(i1, i2 uint64, lowNumBits int) (err error) {
	wantDelta := i1 - i2
	high1, low1 := Uint64ToHighLowFloat64(i1, lowNumBits)
	high2, low2 := Uint64ToHighLowFloat64(i2, lowNumBits)
	gotDelta := uint64(HigLowFloat64Delta(high1, low1, high2, low2, lowNumBits))
	if wantDelta != gotDelta {
		err = fmt.Errorf("lowNumBits=%d: %d-%d: want: %d, got: %d", lowNumBits, i1, i2, wantDelta, gotDelta)
	}
	return
}

func main() {
	diffMaxNumBits := flag.Int(
		"diffMaxNumBits",
		53,
		"Delta checked for 0..1 << diffMaxNumBits-1",
	)
	minLowNumBits := flag.Int(
		"minLowNumBits",
		11,
		"Delta checked for low part w/ minLowNumBits..maxLowNumBits",
	)
	maxLowNumBits := flag.Int(
		"maxLowNumBits",
		53,
		"Delta checked for low part w/ minLowNumBits..maxLowNumBits",
	)

	flag.Parse()

	var nTotal, nErr uint64

	for i := 0; i <= *diffMaxNumBits; i++ {
		diff := uint64(1<<i - 1)
		for n := 0; n <= 64; n++ {
			i1 := uint64(1<<n - 1)
			i2 := i1 - diff
			for lowNumBits := *minLowNumBits; lowNumBits <= *maxLowNumBits; lowNumBits++ {
				nTotal++
				err := Uint64ToFloat64VerifyDelta(i1, i2, lowNumBits)
				if err != nil {
					nErr++
					fmt.Println(err)
				}
			}
		}
	}

	fmt.Printf("case# %d, error# %d\n", nTotal, nErr)
}
