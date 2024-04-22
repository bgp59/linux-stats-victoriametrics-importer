package procfs

import (
	"bytes"
	"fmt"
	"path"
	"testing"
)

type SoftirqsTestCase struct {
	name          string
	procfsRoot    string
	primeSoftirqs *Softirqs
	wantSoftirqs  *Softirqs
	wantError     error
}

var softirqsTestdataDir = path.Join(PROCFS_TESTDATA_ROOT, "softirqs")

func testSoftirqsParser(tc *SoftirqsTestCase, t *testing.T) {
	t.Logf(`
name=%q
procfsRoot=%q
primeSoftirqs=%v
`,
		tc.name, tc.procfsRoot, (tc.primeSoftirqs != nil),
	)

	var softirqs *Softirqs
	if tc.primeSoftirqs != nil {
		softirqs = tc.primeSoftirqs.Clone(true)
		if tc.procfsRoot != "" {
			softirqs.path = SoftirqsPath(tc.procfsRoot)
		}
	} else {
		softirqs = NewSoftirqs(tc.procfsRoot)
	}

	err := softirqs.Parse()

	if tc.wantError != nil {
		if err == nil || tc.wantError.Error() != err.Error() {
			t.Fatalf("want: %v error, got: %v", tc.wantError, err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}

	wantSoftirqs := tc.wantSoftirqs
	diffBuf := &bytes.Buffer{}

	// if !bytes.Equal(wantSoftirqs.cpuHeaderLine, softirqs.cpuHeaderLine) {
	// 	fmt.Fprintf(
	// 		diffBuf,
	// 		"\ncpuHeaderLine:\n\twant: %q,\n\t got: %q",
	// 		string(wantSoftirqs.cpuHeaderLine), string(softirqs.cpuHeaderLine),
	// 	)
	// }
	if wantSoftirqs.CpuListChanged != softirqs.CpuListChanged {
		fmt.Fprintf(
			diffBuf,
			"\nIndexToCpuChanged: want: %v, got: %v",
			wantSoftirqs.CpuListChanged, softirqs.CpuListChanged,
		)
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}

	if wantSoftirqs.CpuList == nil {
		if softirqs.CpuList != nil {
			fmt.Fprintf(
				diffBuf,
				"\nCounterIndexToCpuNum: want: %v, got: %v",
				wantSoftirqs.CpuList, softirqs.CpuList,
			)
		}
	} else {
		if softirqs.CpuList == nil {
			fmt.Fprintf(
				diffBuf,
				"\nCounterIndexToCpuNum: want: %v, got: %v",
				wantSoftirqs.CpuList, softirqs.CpuList,
			)
		}

		if len(wantSoftirqs.CpuList) != len(softirqs.CpuList) {
			fmt.Fprintf(
				diffBuf,
				"\nCounterIndexToCpuNum length: want %d, got: %d",
				len(wantSoftirqs.CpuList), len(softirqs.CpuList),
			)
		}

		for i, wantCpuNum := range wantSoftirqs.CpuList {
			gotCpuNum := softirqs.CpuList[i]
			if wantCpuNum != gotCpuNum {
				fmt.Fprintf(
					diffBuf,
					"\nCounterIndexToCpuNum[%d]: want: %d, got: %d",
					i, wantCpuNum, gotCpuNum,
				)
			}
		}
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}

	for irq, wantCounters := range wantSoftirqs.Counters {
		gotCounters := softirqs.Counters[irq]
		if gotCounters == nil {
			fmt.Fprintf(
				diffBuf, "\nCounters: missing %q", irq,
			)
		}
		if len(gotCounters) != wantSoftirqs.NumCounters {
			fmt.Fprintf(
				diffBuf,
				"\nIrq[%q].Counters length: want: %d, got: %d",
				irq, wantSoftirqs.NumCounters, len(gotCounters),
			)
		} else {
			for i := 0; i < wantSoftirqs.NumCounters; i++ {
				wantCounter := wantCounters[i]
				gotCounter := gotCounters[i]
				if wantCounter != gotCounter {
					fmt.Fprintf(
						diffBuf,
						"\nIrq[%q].Counters[%d]: want: %d, got: %d",
						irq, i, wantCounter, gotCounter,
					)
				}
			}
		}
	}

	for irq := range softirqs.Counters {
		if wantSoftirqs.Counters[irq] == nil {
			fmt.Fprintf(
				diffBuf,
				"\nIrq: unexpected %q",
				irq,
			)
		}
	}

	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}
}

func TestSoftirqsParser(t *testing.T) {
	for _, tc := range []*SoftirqsTestCase{
		{
			name:       "field_mapping",
			procfsRoot: path.Join(softirqsTestdataDir, "field_mapping"),
			wantSoftirqs: &Softirqs{
				CpuList: nil,
				Counters: map[string][]uint64{
					"HI":     {0, 1, 2, 3},
					"TIMER":  {4, 5, 6, 7},
					"NET_TX": {8, 9, 10, 11},
					"NET_RX": {12, 13, 14, 15},
				},
				CpuListChanged: true,
				NumCounters:    4,
			},
		},
		{
			name:       "remove_irq",
			procfsRoot: path.Join(softirqsTestdataDir, "field_mapping"),
			primeSoftirqs: &Softirqs{
				CpuList: nil,
				Counters: map[string][]uint64{
					"HRTIMER": {10000, 10001, 10002, 10003},
					"RCU":     {10004, 10005, 10006, 10007},
					"NET_TX":  {10008, 10009, 100010, 100011},
					"NET_RX":  {100012, 100013, 100014, 100015},
				},
				CpuListChanged: true,
				NumCounters:    4,
				cpuHeaderLine:  []byte("                    CPU0       CPU1       CPU2       CPU3"),
				irqScanNum: map[string]int{
					"HRTIMER": 1,
					"RCU":     1,
					"NET_TX":  1,
					"NET_RX":  1,
				},
				scanNum: 1,
			},
			wantSoftirqs: &Softirqs{
				CpuList: nil,
				Counters: map[string][]uint64{
					"HI":     {0, 1, 2, 3},
					"TIMER":  {4, 5, 6, 7},
					"NET_TX": {8, 9, 10, 11},
					"NET_RX": {12, 13, 14, 15},
				},
				CpuListChanged: false,
				NumCounters:    4,
			},
		},
		{
			name:       "remove_cpu",
			procfsRoot: path.Join(softirqsTestdataDir, "remove_cpu"),
			primeSoftirqs: &Softirqs{
				CpuList: nil,
				Counters: map[string][]uint64{
					"HI":     {10000, 10001, 10002, 10003},
					"TIMER":  {10004, 10005, 10006, 10007},
					"NET_TX": {10008, 10009, 100010, 100011},
					"NET_RX": {100012, 100013, 100014, 100015},
				},
				CpuListChanged: true,
				NumCounters:    4,
				cpuHeaderLine:  []byte("                    CPU0       CPU1       CPU2       CPU3"),
				scanNum:        1,
			},
			wantSoftirqs: &Softirqs{
				CpuList: []int{0, 1, 3},
				Counters: map[string][]uint64{
					"HI":     {0, 1, 3},
					"TIMER":  {4, 5, 7},
					"NET_TX": {8, 9, 11},
					"NET_RX": {12, 13, 15},
				},
				CpuListChanged: true,
				NumCounters:    3,
			},
		},
	} {
		t.Run(
			tc.name,
			func(t *testing.T) { testSoftirqsParser(tc, t) },
		)
	}
}
