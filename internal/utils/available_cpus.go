// Cache available cpu#.
//
// For Linux the latter is based on cpu affinity mask, whereas for non Linux it
// is based on runtime.NumCPU.

package utils

var AvailableCpusCount = CountAvailableCPUs()
