package procfs

import (
	"bytes"
	"fmt"
	"path"
	"testing"
)

type InterruptsTestCase struct {
	name            string
	procfsRoot      string
	primeInterrupts *Interrupts
	wantInterrupts  *Interrupts
	wantError       error
}

var interruptsTestdataDir = path.Join(TESTDATA_PROCFS_ROOT, "interrupts")

func testInterruptsParser(tc *InterruptsTestCase, t *testing.T) {
	var interrupts *Interrupts
	if tc.primeInterrupts != nil {
		interrupts = tc.primeInterrupts.Clone(true)
		if interrupts.path == "" {
			interrupts.path = path.Join(tc.procfsRoot, "interrupts")
		}
	} else {
		interrupts = NewInterrupts(tc.procfsRoot)
	}

	err := interrupts.Parse()

	if tc.wantError != nil {
		if err == nil || tc.wantError.Error() != err.Error() {
			t.Fatalf("want: %v error, got: %v", tc.wantError, err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}

	wantInterrupts := tc.wantInterrupts
	diffBuf := &bytes.Buffer{}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}

	if wantInterrupts.CpuNumLine != interrupts.CpuNumLine {
		fmt.Fprintf(
			diffBuf,
			"\nCpuNumLine:\n\twant: %q,\n\t got: %q",
			wantInterrupts.CpuNumLine, interrupts.CpuNumLine,
		)
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}

	if wantInterrupts.NumCpus != interrupts.NumCpus {
		fmt.Fprintf(
			diffBuf,
			"\nexpectedNumFields: want: %q, got: %q",
			wantInterrupts.NumCpus, interrupts.NumCpus,
		)
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}

	if wantInterrupts.ColIndexToCpuNum == nil {
		if interrupts.ColIndexToCpuNum != nil {
			fmt.Fprintf(
				diffBuf,
				"\nColIndexToCpuNum: want: %v, got: %v",
				wantInterrupts.ColIndexToCpuNum, interrupts.ColIndexToCpuNum,
			)
		}
	} else {
		if interrupts.ColIndexToCpuNum == nil {
			fmt.Fprintf(
				diffBuf,
				"\nColIndexToCpuNum: want: %v, got: %v",
				wantInterrupts.ColIndexToCpuNum, interrupts.ColIndexToCpuNum,
			)
		}

		if len(wantInterrupts.ColIndexToCpuNum) != len(interrupts.ColIndexToCpuNum) {
			fmt.Fprintf(
				diffBuf,
				"\nColIndexToCpuNum length: want %d, got: %d",
				len(wantInterrupts.ColIndexToCpuNum), len(interrupts.ColIndexToCpuNum),
			)
		}

		for i, wantCpuNum := range wantInterrupts.ColIndexToCpuNum {
			gotCpuNum := interrupts.ColIndexToCpuNum[i]
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

	for irq, wantPerCpuCounter := range wantInterrupts.Irq {
		gotPerCpuCount := interrupts.Irq[irq]
		if gotPerCpuCount == nil {
			fmt.Fprintf(
				diffBuf,
				"\nIrq: missing %q",
				irq,
			)
			continue
		}
		if len(gotPerCpuCount) < wantInterrupts.NumCpus {
			fmt.Fprintf(
				diffBuf,
				"\nIrq[%q] length: want: >= %d, got: %d",
				irq, wantInterrupts.NumCpus, len(gotPerCpuCount),
			)
			continue
		}
		for i := 0; i < wantInterrupts.NumCpus; i++ {
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

	for irq := range interrupts.Irq {
		if wantInterrupts.Irq[irq] == nil {
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

func TestInterruptsParser(t *testing.T) {
	for _, tc := range []*InterruptsTestCase{
		{
			procfsRoot: path.Join(interruptsTestdataDir, "field_mapping"),
			wantInterrupts: &Interrupts{
				ColIndexToCpuNum: nil,
				CpuNumLine:       "                    CPU0       CPU1",
				Irq:              map[string][]uint64{},
				NumCpus:          2,
			},
		},
	} {
		var name string
		if tc.name != "" {
			name = fmt.Sprintf("name=%s/procfsRoot=%s", tc.name, tc.procfsRoot)
		} else {
			name = fmt.Sprintf("procfsRoot=%s", tc.procfsRoot)
		}
		t.Run(
			name,
			func(t *testing.T) { testInterruptsParser(tc, t) },
		)
	}

}
