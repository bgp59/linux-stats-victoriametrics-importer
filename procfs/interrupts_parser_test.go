package procfs

import (
	"bytes"
	"fmt"
	"path"
	"testing"
)

type TestInterruptDescription struct {
	Controller, HWInterrupt, Devices string
	Changed                          bool
}

type InterruptsTestCase struct {
	name            string
	procfsRoot      string
	primeInterrupts *Interrupts
	wantInterrupts  *Interrupts
	wantDescription map[string]*TestInterruptDescription
	wantError       error
}

var interruptsTestdataDir = path.Join(PROCFS_TESTDATA_ROOT, "interrupts")

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
	if wantInterrupts == nil {
		return
	}

	diffBuf := &bytes.Buffer{}

	if !bytes.Equal(wantInterrupts.cpuHeaderLine, interrupts.cpuHeaderLine) {
		fmt.Fprintf(
			diffBuf,
			"\ncpuHeaderLine:\n\twant: %q,\n\t got: %q",
			wantInterrupts.cpuHeaderLine, interrupts.cpuHeaderLine,
		)
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}

	if wantInterrupts.NumCpus != interrupts.NumCpus {
		fmt.Fprintf(
			diffBuf,
			"\nNumCpus: want: %d, got: %d",
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

	for irq, wantDescription := range tc.wantDescription {
		gotDescription := interrupts.Description[irq]
		if gotDescription == nil {
			fmt.Fprintf(
				diffBuf,
				"\nDescription: missing  %q",
				irq,
			)
			continue
		}
		irqInfo := gotDescription.IrqInfo

		gotController := string(irqInfo[gotDescription.Controller.Start:gotDescription.Controller.End])
		if wantDescription.Controller != gotController {
			fmt.Fprintf(
				diffBuf,
				"\nDescription[%q].Controller: want: %q, got: %q",
				irq, wantDescription.Controller, gotController,
			)
		}

		gotHWInterrupt := string(irqInfo[gotDescription.HWInterrupt.Start:gotDescription.HWInterrupt.End])
		if wantDescription.HWInterrupt != gotHWInterrupt {
			fmt.Fprintf(
				diffBuf,
				"\nDescription[%q].HWInterrupt: want: %q, got: %q",
				irq, wantDescription.HWInterrupt, gotHWInterrupt,
			)
		}

		gotDevices := string(irqInfo[gotDescription.Devices.Start:gotDescription.Devices.End])
		if wantDescription.Devices != gotDevices {
			fmt.Fprintf(
				diffBuf,
				"\nDescription[%q].Devices: want: %q, got: %q",
				irq, wantDescription.Devices, gotDevices,
			)
		}

		if wantDescription.Changed != gotDescription.Changed {
			fmt.Fprintf(
				diffBuf,
				"\nDescription[%q].Changed: want: %v, got: %v",
				irq, wantDescription.Changed, gotDescription.Changed,
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
				cpuHeaderLine:    []byte("                  CPU0           CPU1"),
				Irq: map[string][]uint64{
					"0":           []uint64{0, 1},
					"1":           []uint64{1000, 1001},
					"4":           []uint64{4000, 4001},
					"non-numeric": []uint64{1000000, 1000001},
					"no-info":     []uint64{2000000, 2000001},
				},
				NumCpus: 2,
			},
			wantDescription: map[string]*TestInterruptDescription{
				"0": &TestInterruptDescription{
					Controller:  "controller-0",
					HWInterrupt: "hw-irq-0",
					Devices:     "device0",
					Changed:     true,
				},
				"1": &TestInterruptDescription{
					Controller:  "controller-1",
					HWInterrupt: "hw-irq-1",
					Devices:     "device1-1,device1-2",
					Changed:     true,
				},
				"4": &TestInterruptDescription{
					Controller:  "controller-4",
					HWInterrupt: "hw-irq-4",
					Devices:     "device4-1,device4-2",
					Changed:     true,
				},
			},
		},
		{
			name:       "remove_irq",
			procfsRoot: path.Join(interruptsTestdataDir, "field_mapping"),
			primeInterrupts: &Interrupts{
				ColIndexToCpuNum: nil,
				cpuHeaderLine:    []byte("                  CPU0           CPU1"),
				Irq: map[string][]uint64{
					"0":           []uint64{20, 21},
					"1":           []uint64{21000, 21001},
					"4":           []uint64{24000, 24001},
					"non-numeric": []uint64{21000000, 21000001},
					"no-info":     []uint64{22000000, 22000001},
					"11":          []uint64{2110, 2111},
					"delete":      []uint64{31000000, 31000001},
				},
				NumCpus: 2,
				irqScanNum: map[string]int{
					"0":           10,
					"1":           10,
					"4":           10,
					"non-numeric": 10,
					"no-info":     10,
					"11":          10,
					"delete":      10,
				},
				scanNum: 10,
			},
			wantInterrupts: &Interrupts{
				ColIndexToCpuNum: nil,
				cpuHeaderLine:    []byte("                  CPU0           CPU1"),
				Irq: map[string][]uint64{
					"0":           []uint64{0, 1},
					"1":           []uint64{1000, 1001},
					"4":           []uint64{4000, 4001},
					"non-numeric": []uint64{1000000, 1000001},
					"no-info":     []uint64{2000000, 2000001},
				},
				NumCpus: 2,
			},
			wantDescription: map[string]*TestInterruptDescription{
				"0": &TestInterruptDescription{
					Controller:  "controller-0",
					HWInterrupt: "hw-irq-0",
					Devices:     "device0",
					Changed:     true,
				},
				"1": &TestInterruptDescription{
					Controller:  "controller-1",
					HWInterrupt: "hw-irq-1",
					Devices:     "device1-1,device1-2",
					Changed:     true,
				},
				"4": &TestInterruptDescription{
					Controller:  "controller-4",
					HWInterrupt: "hw-irq-4",
					Devices:     "device4-1,device4-2",
					Changed:     true,
				},
			},
		},
		{
			procfsRoot: path.Join(interruptsTestdataDir, "remove_cpu"),
			primeInterrupts: &Interrupts{
				ColIndexToCpuNum: nil,
				cpuHeaderLine:    []byte("                  CPU0           CPU1"),
				Irq: map[string][]uint64{
					"0":           []uint64{20, 21},
					"1":           []uint64{21000, 21001},
					"4":           []uint64{24000, 24001},
					"non-numeric": []uint64{21000000, 21000001},
					"no-info":     []uint64{22000000, 22000001},
					"11":          []uint64{2110, 2111},
					"delete":      []uint64{31000000, 31000001},
				},
				NumCpus: 2,
				irqScanNum: map[string]int{
					"0":           10,
					"1":           10,
					"4":           10,
					"non-numeric": 10,
					"11":          10,
					"delete":      10,
				},
				scanNum: 10,
			},
			wantInterrupts: &Interrupts{
				ColIndexToCpuNum: []int{1},
				cpuHeaderLine:    []byte("                          CPU1"),
				Irq: map[string][]uint64{
					"0":           []uint64{1},
					"1":           []uint64{1001},
					"4":           []uint64{4001},
					"non-numeric": []uint64{1000001},
					"no-info":     []uint64{2000001},
				},
				NumCpus: 1,
			},
			wantDescription: map[string]*TestInterruptDescription{
				"0": &TestInterruptDescription{
					Controller:  "controller-0",
					HWInterrupt: "hw-irq-0",
					Devices:     "device0",
					Changed:     true,
				},
				"1": &TestInterruptDescription{
					Controller:  "controller-1",
					HWInterrupt: "hw-irq-1",
					Devices:     "device1-1,device1-2",
					Changed:     true,
				},
				"4": &TestInterruptDescription{
					Controller:  "controller-4",
					HWInterrupt: "hw-irq-4",
					Devices:     "device4-1,device4-2",
					Changed:     true,
				},
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
			func(t *testing.T) { testInterruptsParser(tc, t) },
		)
	}
}
