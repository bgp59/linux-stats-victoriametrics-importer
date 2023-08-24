// unit64 as float64 test bed.

// Some Linux stats counters are 64 bit integers but VictoriaMetrics uses
// float64 for time series values. Values larger than 1<<53 will suffer a loss
// of precision which in turn may affect delta and rate calculations. The
// solution is to split uint64 values into lower N bits and higher 64-N ones and
// to store them in parallel timeseries.

// This sample code illustrates a way of using the high,low float64 for
// computing accurate deltas matching the results for the original uint64
// values. It should be noted that Go uint64 arithmetic handles counter
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

// The following pseudo-constants can be calculated for each lowNumBits values,
// they are listed here as convenience for building Prometheus queries:
type Uint64ToHighLowFloat64Constants struct {
	highFactor, highCounterRolloverCorrection float64
}

var Uint64ToHighLowFloat64ConstMap = map[int]Uint64ToHighLowFloat64Constants{
	0:  {1, 0},                    // delta(high[])*1+delta(low[]) >= 0 or (delta(high[])+0)*1+delta(low[])
	1:  {2, 9223372036854775808},  // delta(high[])*2+delta(low[]) >= 0 or (delta(high[])+9223372036854775808)*2+delta(low[])
	2:  {4, 4611686018427387904},  // delta(high[])*4+delta(low[]) >= 0 or (delta(high[])+4611686018427387904)*4+delta(low[])
	3:  {8, 2305843009213693952},  // delta(high[])*8+delta(low[]) >= 0 or (delta(high[])+2305843009213693952)*8+delta(low[])
	4:  {16, 1152921504606846976}, // delta(high[])*16+delta(low[]) >= 0 or (delta(high[])+1152921504606846976)*16+delta(low[])
	5:  {32, 576460752303423488},  // delta(high[])*32+delta(low[]) >= 0 or (delta(high[])+576460752303423488)*32+delta(low[])
	6:  {64, 288230376151711744},  // delta(high[])*64+delta(low[]) >= 0 or (delta(high[])+288230376151711744)*64+delta(low[])
	7:  {128, 144115188075855872}, // delta(high[])*128+delta(low[]) >= 0 or (delta(high[])+144115188075855872)*128+delta(low[])
	8:  {256, 72057594037927936},  // delta(high[])*256+delta(low[]) >= 0 or (delta(high[])+72057594037927936)*256+delta(low[])
	9:  {512, 36028797018963968},  // delta(high[])*512+delta(low[]) >= 0 or (delta(high[])+36028797018963968)*512+delta(low[])
	10: {1024, 18014398509481984}, // delta(high[])*1024+delta(low[]) >= 0 or (delta(high[])+18014398509481984)*1024+delta(low[])

	// Usable lowNumBits start:
	11: {2048, 9007199254740992},  // delta(high[])*2048+delta(low[]) >= 0 or (delta(high[])+9007199254740992)*2048+delta(low[])
	12: {4096, 4503599627370496},  // delta(high[])*4096+delta(low[]) >= 0 or (delta(high[])+4503599627370496)*4096+delta(low[])
	13: {8192, 2251799813685248},  // delta(high[])*8192+delta(low[]) >= 0 or (delta(high[])+2251799813685248)*8192+delta(low[])
	14: {16384, 1125899906842624}, // delta(high[])*16384+delta(low[]) >= 0 or (delta(high[])+1125899906842624)*16384+delta(low[])
	15: {32768, 562949953421312},  // delta(high[])*32768+delta(low[]) >= 0 or (delta(high[])+562949953421312)*32768+delta(low[])
	16: {65536, 281474976710656},  // delta(high[])*65536+delta(low[]) >= 0 or (delta(high[])+281474976710656)*65536+delta(low[])
	17: {131072, 140737488355328}, // delta(high[])*131072+delta(low[]) >= 0 or (delta(high[])+140737488355328)*131072+delta(low[])
	18: {262144, 70368744177664},  // delta(high[])*262144+delta(low[]) >= 0 or (delta(high[])+70368744177664)*262144+delta(low[])
	19: {524288, 35184372088832},  // delta(high[])*524288+delta(low[]) >= 0 or (delta(high[])+35184372088832)*524288+delta(low[])
	20: {1048576, 17592186044416}, // delta(high[])*1048576+delta(low[]) >= 0 or (delta(high[])+17592186044416)*1048576+delta(low[])
	21: {2097152, 8796093022208},  // delta(high[])*2097152+delta(low[]) >= 0 or (delta(high[])+8796093022208)*2097152+delta(low[])
	22: {4194304, 4398046511104},  // delta(high[])*4194304+delta(low[]) >= 0 or (delta(high[])+4398046511104)*4194304+delta(low[])
	23: {8388608, 2199023255552},  // delta(high[])*8388608+delta(low[]) >= 0 or (delta(high[])+2199023255552)*8388608+delta(low[])
	24: {16777216, 1099511627776}, // delta(high[])*16777216+delta(low[]) >= 0 or (delta(high[])+1099511627776)*16777216+delta(low[])
	25: {33554432, 549755813888},  // delta(high[])*33554432+delta(low[]) >= 0 or (delta(high[])+549755813888)*33554432+delta(low[])
	26: {67108864, 274877906944},  // delta(high[])*67108864+delta(low[]) >= 0 or (delta(high[])+274877906944)*67108864+delta(low[])
	27: {134217728, 137438953472}, // delta(high[])*134217728+delta(low[]) >= 0 or (delta(high[])+137438953472)*134217728+delta(low[])
	28: {268435456, 68719476736},  // delta(high[])*268435456+delta(low[]) >= 0 or (delta(high[])+68719476736)*268435456+delta(low[])
	29: {536870912, 34359738368},  // delta(high[])*536870912+delta(low[]) >= 0 or (delta(high[])+34359738368)*536870912+delta(low[])
	30: {1073741824, 17179869184}, // delta(high[])*1073741824+delta(low[]) >= 0 or (delta(high[])+17179869184)*1073741824+delta(low[])
	31: {2147483648, 8589934592},  // delta(high[])*2147483648+delta(low[]) >= 0 or (delta(high[])+8589934592)*2147483648+delta(low[])
	32: {4294967296, 4294967296},  // delta(high[])*4294967296+delta(low[]) >= 0 or (delta(high[])+4294967296)*4294967296+delta(low[])
	33: {8589934592, 2147483648},  // delta(high[])*8589934592+delta(low[]) >= 0 or (delta(high[])+2147483648)*8589934592+delta(low[])
	34: {17179869184, 1073741824}, // delta(high[])*17179869184+delta(low[]) >= 0 or (delta(high[])+1073741824)*17179869184+delta(low[])
	35: {34359738368, 536870912},  // delta(high[])*34359738368+delta(low[]) >= 0 or (delta(high[])+536870912)*34359738368+delta(low[])
	36: {68719476736, 268435456},  // delta(high[])*68719476736+delta(low[]) >= 0 or (delta(high[])+268435456)*68719476736+delta(low[])
	37: {137438953472, 134217728}, // delta(high[])*137438953472+delta(low[]) >= 0 or (delta(high[])+134217728)*137438953472+delta(low[])
	38: {274877906944, 67108864},  // delta(high[])*274877906944+delta(low[]) >= 0 or (delta(high[])+67108864)*274877906944+delta(low[])
	39: {549755813888, 33554432},  // delta(high[])*549755813888+delta(low[]) >= 0 or (delta(high[])+33554432)*549755813888+delta(low[])
	40: {1099511627776, 16777216}, // delta(high[])*1099511627776+delta(low[]) >= 0 or (delta(high[])+16777216)*1099511627776+delta(low[])
	41: {2199023255552, 8388608},  // delta(high[])*2199023255552+delta(low[]) >= 0 or (delta(high[])+8388608)*2199023255552+delta(low[])
	42: {4398046511104, 4194304},  // delta(high[])*4398046511104+delta(low[]) >= 0 or (delta(high[])+4194304)*4398046511104+delta(low[])
	43: {8796093022208, 2097152},  // delta(high[])*8796093022208+delta(low[]) >= 0 or (delta(high[])+2097152)*8796093022208+delta(low[])
	44: {17592186044416, 1048576}, // delta(high[])*17592186044416+delta(low[]) >= 0 or (delta(high[])+1048576)*17592186044416+delta(low[])
	45: {35184372088832, 524288},  // delta(high[])*35184372088832+delta(low[]) >= 0 or (delta(high[])+524288)*35184372088832+delta(low[])
	46: {70368744177664, 262144},  // delta(high[])*70368744177664+delta(low[]) >= 0 or (delta(high[])+262144)*70368744177664+delta(low[])
	47: {140737488355328, 131072}, // delta(high[])*140737488355328+delta(low[]) >= 0 or (delta(high[])+131072)*140737488355328+delta(low[])
	48: {281474976710656, 65536},  // delta(high[])*281474976710656+delta(low[]) >= 0 or (delta(high[])+65536)*281474976710656+delta(low[])
	49: {562949953421312, 32768},  // delta(high[])*562949953421312+delta(low[]) >= 0 or (delta(high[])+32768)*562949953421312+delta(low[])
	50: {1125899906842624, 16384}, // delta(high[])*1125899906842624+delta(low[]) >= 0 or (delta(high[])+16384)*1125899906842624+delta(low[])
	51: {2251799813685248, 8192},  // delta(high[])*2251799813685248+delta(low[]) >= 0 or (delta(high[])+8192)*2251799813685248+delta(low[])
	52: {4503599627370496, 4096},  // delta(high[])*4503599627370496+delta(low[]) >= 0 or (delta(high[])+4096)*4503599627370496+delta(low[])
	53: {9007199254740992, 2048},  // delta(high[])*9007199254740992+delta(low[]) >= 0 or (delta(high[])+2048)*9007199254740992+delta(low[])
	// Usable lowNumBits end.

	54: {18014398509481984, 1024}, // delta(high[])*18014398509481984+delta(low[]) >= 0 or (delta(high[])+1024)*18014398509481984+delta(low[])
	55: {36028797018963968, 512},  // delta(high[])*36028797018963968+delta(low[]) >= 0 or (delta(high[])+512)*36028797018963968+delta(low[])
	56: {72057594037927936, 256},  // delta(high[])*72057594037927936+delta(low[]) >= 0 or (delta(high[])+256)*72057594037927936+delta(low[])
	57: {144115188075855872, 128}, // delta(high[])*144115188075855872+delta(low[]) >= 0 or (delta(high[])+128)*144115188075855872+delta(low[])
	58: {288230376151711744, 64},  // delta(high[])*288230376151711744+delta(low[]) >= 0 or (delta(high[])+64)*288230376151711744+delta(low[])
	59: {576460752303423488, 32},  // delta(high[])*576460752303423488+delta(low[]) >= 0 or (delta(high[])+32)*576460752303423488+delta(low[])
	60: {1152921504606846976, 16}, // delta(high[])*1152921504606846976+delta(low[]) >= 0 or (delta(high[])+16)*1152921504606846976+delta(low[])
	61: {2305843009213693952, 8},  // delta(high[])*2305843009213693952+delta(low[]) >= 0 or (delta(high[])+8)*2305843009213693952+delta(low[])
	62: {4611686018427387904, 4},  // delta(high[])*4611686018427387904+delta(low[]) >= 0 or (delta(high[])+4)*4611686018427387904+delta(low[])
	63: {9223372036854775808, 2},  // delta(high[])*9223372036854775808+delta(low[]) >= 0 or (delta(high[])+2)*9223372036854775808+delta(low[])
	64: {0, 1},                    // delta(high[])*0+delta(low[]) >= 0 or (delta(high[])+1)*0+delta(low[])

}

func Uint64ToHighLowFloat64(i uint64, lowNumBits int) (float64, float64) {
	return float64(i >> lowNumBits), float64(i & (1<<lowNumBits - 1))
}

// Emulate PromQL delta:
func HigLowFloat64Delta(high1, low1, high2, low2 float64, lowNumBits int) float64 {
	highLowFloat64Const := Uint64ToHighLowFloat64ConstMap[lowNumBits]
	deltaHigh, deltaLow := high1-high2, low1-low2

	if deltaHigh < 0 || (deltaHigh == 0 && low1 < low2) {
		return (deltaHigh+highLowFloat64Const.highCounterRolloverCorrection)*highLowFloat64Const.highFactor + deltaLow
	} else {
		return deltaHigh*highLowFloat64Const.highFactor + deltaLow
	}
}

func Uint64ToFloat64VerifyDelta(i1, i2 uint64, lowNumBits int) (err error) {
	wantDelta := i1 - i2
	high1, low1 := Uint64ToHighLowFloat64(i1, lowNumBits)
	high2, low2 := Uint64ToHighLowFloat64(i2, lowNumBits)
	gotDelta := HigLowFloat64Delta(high1, low1, high2, low2, lowNumBits)
	if wantDelta != uint64(gotDelta) {
		err = fmt.Errorf("lowNumBits=%d: %d-%d: want: %d, got: %f", lowNumBits, i1, i2, wantDelta, gotDelta)
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
		for j := 0; j <= i; j++ {
			diff := uint64(1<<i - 1<<j)
			for m := 0; m <= 64; m++ {
				for n := 0; n <= m; n++ {
					i1 := uint64(1<<n - 1<<m)
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
		}
	}

	fmt.Printf("case# %d, error# %d\n", nTotal, nErr)
}
