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
	var softirqs *Softirqs
	if tc.primeSoftirqs != nil {
		softirqs = tc.primeSoftirqs.Clone(true)
		if softirqs.path == "" {
			softirqs.path = path.Join(tc.procfsRoot, "softirqs")
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
	if wantSoftirqs.IndexToCpuChanged != softirqs.IndexToCpuChanged {
		fmt.Fprintf(
			diffBuf,
			"\nIndexToCpuChanged: want: %v, got: %v",
			wantSoftirqs.IndexToCpuChanged, softirqs.IndexToCpuChanged,
		)
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}

	if wantSoftirqs.CounterIndexToCpuNum == nil {
		if softirqs.CounterIndexToCpuNum != nil {
			fmt.Fprintf(
				diffBuf,
				"\nCounterIndexToCpuNum: want: %v, got: %v",
				wantSoftirqs.CounterIndexToCpuNum, softirqs.CounterIndexToCpuNum,
			)
		}
	} else {
		if softirqs.CounterIndexToCpuNum == nil {
			fmt.Fprintf(
				diffBuf,
				"\nCounterIndexToCpuNum: want: %v, got: %v",
				wantSoftirqs.CounterIndexToCpuNum, softirqs.CounterIndexToCpuNum,
			)
		}

		if len(wantSoftirqs.CounterIndexToCpuNum) != len(softirqs.CounterIndexToCpuNum) {
			fmt.Fprintf(
				diffBuf,
				"\nCounterIndexToCpuNum length: want %d, got: %d",
				len(wantSoftirqs.CounterIndexToCpuNum), len(softirqs.CounterIndexToCpuNum),
			)
		}

		for i, wantCpuNum := range wantSoftirqs.CounterIndexToCpuNum {
			gotCpuNum := softirqs.CounterIndexToCpuNum[i]
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

	for irq, wantSoftirqsIrq := range wantSoftirqs.Irq {
		gotSoftirqsIrq := softirqs.Irq[irq]
		if gotSoftirqsIrq == nil {
			fmt.Fprintf(
				diffBuf,
				"\nIrq: missing %q",
				irq,
			)
			continue
		}

		wantCounters, gotCounters := wantSoftirqsIrq.Counters, gotSoftirqsIrq.Counters
		if len(gotCounters) != wantSoftirqs.numCounters {
			fmt.Fprintf(
				diffBuf,
				"\nIrq[%q].Counters length: want: %d, got: %d",
				irq, wantSoftirqs.numCounters, len(gotCounters),
			)
		} else {
			for i := 0; i < wantSoftirqs.numCounters; i++ {
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

	for irq := range softirqs.Irq {
		if wantSoftirqs.Irq[irq] == nil {
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
	for i, tc := range []*SoftirqsTestCase{
		{
			procfsRoot: path.Join(softirqsTestdataDir, "field_mapping"),
			wantSoftirqs: &Softirqs{
				CounterIndexToCpuNum: nil,
				Irq: map[string]*SoftirqsIrq{
					"HI":     {Counters: []uint64{0, 1, 2, 3}},
					"TIMER":  {Counters: []uint64{4, 5, 6, 7}},
					"NET_TX": {Counters: []uint64{8, 9, 10, 11}},
					"NET_RX": {Counters: []uint64{12, 13, 14, 15}},
				},
				IndexToCpuChanged: true,
				numCounters:       4,
			},
		},
		{
			name:       "remove_irq",
			procfsRoot: path.Join(softirqsTestdataDir, "field_mapping"),
			primeSoftirqs: &Softirqs{
				CounterIndexToCpuNum: nil,
				Irq: map[string]*SoftirqsIrq{
					"HRTIMER": {Counters: []uint64{10000, 10001, 10002, 10003}, scanNum: 1},
					"RCU":     {Counters: []uint64{10004, 10005, 10006, 10007}, scanNum: 1},
					"NET_TX":  {Counters: []uint64{10008, 10009, 100010, 100011}, scanNum: 1},
					"NET_RX":  {Counters: []uint64{100012, 100013, 100014, 100015}, scanNum: 1},
				},
				IndexToCpuChanged: true,
				numCounters:       4,
				cpuHeaderLine:     []byte("                    CPU0       CPU1       CPU2       CPU3"),
				scanNum:           1,
			},
			wantSoftirqs: &Softirqs{
				CounterIndexToCpuNum: nil,
				Irq: map[string]*SoftirqsIrq{
					"HI":     {Counters: []uint64{0, 1, 2, 3}},
					"TIMER":  {Counters: []uint64{4, 5, 6, 7}},
					"NET_TX": {Counters: []uint64{8, 9, 10, 11}},
					"NET_RX": {Counters: []uint64{12, 13, 14, 15}},
				},
				IndexToCpuChanged: false,
				numCounters:       4,
			},
		},
		{
			procfsRoot: path.Join(softirqsTestdataDir, "remove_cpu"),
			primeSoftirqs: &Softirqs{
				CounterIndexToCpuNum: nil,
				Irq: map[string]*SoftirqsIrq{
					"HI":     {Counters: []uint64{10000, 10001, 10002, 10003}, scanNum: 1},
					"TIMER":  {Counters: []uint64{10004, 10005, 10006, 10007}, scanNum: 1},
					"NET_TX": {Counters: []uint64{10008, 10009, 100010, 100011}, scanNum: 1},
					"NET_RX": {Counters: []uint64{100012, 100013, 100014, 100015}, scanNum: 1},
				},
				IndexToCpuChanged: true,
				numCounters:       4,
				cpuHeaderLine:     []byte("                    CPU0       CPU1       CPU2       CPU3"),
				scanNum:           1,
			},
			wantSoftirqs: &Softirqs{
				CounterIndexToCpuNum: []int{0, 1, 3},
				Irq: map[string]*SoftirqsIrq{
					"HI":     {Counters: []uint64{0, 1, 3}},
					"TIMER":  {Counters: []uint64{4, 5, 7}},
					"NET_TX": {Counters: []uint64{8, 9, 11}},
					"NET_RX": {Counters: []uint64{12, 13, 15}},
				},
				IndexToCpuChanged: true,
				numCounters:       3,
			},
		},
	} {
		var name string
		if tc.name != "" {
			name = fmt.Sprintf("tc=%d,name=%s,procfsRoot=%s", i, tc.name, tc.procfsRoot)
		} else {
			name = fmt.Sprintf("tc=%d,procfsRoot=%s", i, tc.procfsRoot)
		}
		t.Run(
			name,
			func(t *testing.T) { testSoftirqsParser(tc, t) },
		)
	}
}
