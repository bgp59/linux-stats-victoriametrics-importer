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

var softirqsTestdataDir = path.Join(TESTDATA_PROCFS_ROOT, "softirqs")

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

	if wantSoftirqs.CpuNumLine != softirqs.CpuNumLine {
		fmt.Fprintf(
			diffBuf,
			"\nCpuNumLine:\n\twant: %q,\n\t got: %q",
			wantSoftirqs.CpuNumLine, softirqs.CpuNumLine,
		)
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}

	if wantSoftirqs.NumCpus != softirqs.NumCpus {
		fmt.Fprintf(
			diffBuf,
			"\\nNumCpus: want: %d, got: %d",
			wantSoftirqs.NumCpus, softirqs.NumCpus,
		)
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}

	if wantSoftirqs.ColIndexToCpuNum == nil {
		if softirqs.ColIndexToCpuNum != nil {
			fmt.Fprintf(
				diffBuf,
				"\nColIndexToCpuNum: want: %v, got: %v",
				wantSoftirqs.ColIndexToCpuNum, softirqs.ColIndexToCpuNum,
			)
		}
	} else {
		if softirqs.ColIndexToCpuNum == nil {
			fmt.Fprintf(
				diffBuf,
				"\nColIndexToCpuNum: want: %v, got: %v",
				wantSoftirqs.ColIndexToCpuNum, softirqs.ColIndexToCpuNum,
			)
		}

		if len(wantSoftirqs.ColIndexToCpuNum) != len(softirqs.ColIndexToCpuNum) {
			fmt.Fprintf(
				diffBuf,
				"\nColIndexToCpuNum length: want %d, got: %d",
				len(wantSoftirqs.ColIndexToCpuNum), len(softirqs.ColIndexToCpuNum),
			)
		}

		for i, wantCpuNum := range wantSoftirqs.ColIndexToCpuNum {
			gotCpuNum := softirqs.ColIndexToCpuNum[i]
			if wantCpuNum != gotCpuNum {
				fmt.Fprintf(
					diffBuf,
					"\nColIndexToCpuNum[%d]: want: %d, got: %d",
					i, wantCpuNum, gotCpuNum,
				)
			}
		}
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}

	for irq, wantPerCpuCounter := range wantSoftirqs.Irq {
		gotPerCpuCount := softirqs.Irq[irq]
		if gotPerCpuCount == nil {
			fmt.Fprintf(
				diffBuf,
				"\nIrq: missing %q",
				irq,
			)
			continue
		}
		if len(gotPerCpuCount) < wantSoftirqs.NumCpus {
			fmt.Fprintf(
				diffBuf,
				"\nIrq[%q] length: want: >= %d, got: %d",
				irq, wantSoftirqs.NumCpus, len(gotPerCpuCount),
			)
			continue
		}
		for i := 0; i < wantSoftirqs.NumCpus; i++ {
			wantCounter := wantPerCpuCounter[i]
			gotCounter := gotPerCpuCount[i]
			if wantCounter != gotCounter {
				fmt.Fprintf(
					diffBuf,
					"\nIrq[%q][%d]: want: %d, got: %d",
					irq, i, wantCounter, gotCounter,
				)
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
	for _, tc := range []*SoftirqsTestCase{
		{
			procfsRoot: path.Join(softirqsTestdataDir, "field_mapping"),
			wantSoftirqs: &Softirqs{
				ColIndexToCpuNum: nil,
				CpuNumLine:       "                    CPU0       CPU1       CPU2       CPU3",
				Irq: map[string][]uint64{
					"HI":     []uint64{0, 1, 2, 3},
					"TIMER":  []uint64{4, 5, 6, 7},
					"NET_TX": []uint64{8, 9, 10, 11},
					"NET_RX": []uint64{12, 13, 14, 15},
				},
				NumCpus: 4,
			},
		},
		{
			name:       "remove_irq",
			procfsRoot: path.Join(softirqsTestdataDir, "field_mapping"),
			primeSoftirqs: &Softirqs{
				ColIndexToCpuNum: nil,
				CpuNumLine:       "                    CPU0       CPU1       CPU2       CPU3",
				Irq: map[string][]uint64{
					"HRTIMER": []uint64{10000, 10001, 10002, 10003},
					"RCU":     []uint64{10004, 10005, 10006, 10007},
					"NET_TX":  []uint64{1008, 1009, 10010, 10011},
					"NET_RX":  []uint64{10012, 10013, 10014, 10015},
				},
				irqScanNum: map[string]int{
					"HRTIMER": 10,
					"RCU":     10,
					"NET_TX":  10,
					"NET_RX":  10,
				},
				scanNum: 10,
				NumCpus: 4,
			},
			wantSoftirqs: &Softirqs{
				ColIndexToCpuNum: nil,
				CpuNumLine:       "                    CPU0       CPU1       CPU2       CPU3",
				Irq: map[string][]uint64{
					"HI":     []uint64{0, 1, 2, 3},
					"TIMER":  []uint64{4, 5, 6, 7},
					"NET_TX": []uint64{8, 9, 10, 11},
					"NET_RX": []uint64{12, 13, 14, 15},
				},
				NumCpus: 4,
			},
		},
		{
			procfsRoot: path.Join(softirqsTestdataDir, "remove_cpu"),
			primeSoftirqs: &Softirqs{
				ColIndexToCpuNum: nil,
				CpuNumLine:       "                    CPU0       CPU1       CPU2       CPU3",
				Irq: map[string][]uint64{
					"HI":     []uint64{10000, 10001, 10002, 10003},
					"TIMER":  []uint64{10004, 10005, 10006, 10007},
					"NET_TX": []uint64{1008, 1009, 10010, 10011},
					"NET_RX": []uint64{10012, 10013, 10014, 10015},
				},
				NumCpus: 4,
			},
			wantSoftirqs: &Softirqs{
				ColIndexToCpuNum: []int{0, 1, 3},
				CpuNumLine:       "                    CPU0       CPU1       CPU3",
				Irq: map[string][]uint64{
					"HI":     []uint64{0, 1, 3},
					"TIMER":  []uint64{4, 5, 7},
					"NET_TX": []uint64{8, 9, 11},
					"NET_RX": []uint64{12, 13, 15},
				},
				NumCpus: 3,
			},
		},
	} {
		var name string
		if tc.name != "" {
			name = fmt.Sprintf("name=%s,procfsRoot=%s", tc.name, tc.procfsRoot)
		} else {
			name = fmt.Sprintf("procfsRoot=%s", tc.procfsRoot)
		}
		t.Run(
			name,
			func(t *testing.T) { testSoftirqsParser(tc, t) },
		)
	}
}
