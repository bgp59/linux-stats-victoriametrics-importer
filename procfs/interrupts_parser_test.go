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

	if wantInterrupts.CpuHeaderLine != interrupts.CpuHeaderLine {
		fmt.Fprintf(
			diffBuf,
			"\nCpuNumLine:\n\twant: %q,\n\t got: %q",
			wantInterrupts.CpuHeaderLine, interrupts.CpuHeaderLine,
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

	for irq, wantDescription := range tc.wantInterrupts.Description {
		gotDescription := interrupts.Description[irq]
		if gotDescription == nil {
			fmt.Fprintf(
				diffBuf,
				"\nDescription: missing  %q",
				irq,
			)
			continue
		}
		if wantDescription.Controller != gotDescription.Controller {
			fmt.Fprintf(
				diffBuf,
				"\nDescription[%q].Controller: want: %q, got: %q",
				irq, wantDescription.Controller, gotDescription.Controller,
			)
		}
		if wantDescription.HWInterrupt != gotDescription.HWInterrupt {
			fmt.Fprintf(
				diffBuf,
				"\nDescription[%q].HWInterrupt: want: %q, got: %q",
				irq, wantDescription.HWInterrupt, gotDescription.HWInterrupt,
			)
		}
		devicesEq := len(wantDescription.Devices) == len(gotDescription.Devices)
		if devicesEq {
			for i, wantDevice := range wantDescription.Devices {
				if wantDevice != gotDescription.Devices[i] {
					devicesEq = false
					break
				}
			}
		}
		if !devicesEq {
			fmt.Fprintf(
				diffBuf,
				"\nDescription[%q].Devices:\n\twant: %q\n\t got: %q",
				irq, wantDescription.Devices, gotDescription.Devices,
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

	if tc.primeInterrupts == nil || len(tc.primeInterrupts.Description) == 0 {
		for irq, description := range interrupts.Description {
			if tc.wantInterrupts.Description[irq] == nil {
				fmt.Fprintf(
					diffBuf,
					"\nDescription: unexpected irq %q (Controller: %q, HWInterrupt: %q, Devices: %q)",
					irq, description.Controller, description.HWInterrupt, description.Devices,
				)
			}
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
				CpuHeaderLine:    "                  CPU0           CPU1",
				Irq: map[string][]uint64{
					"0":           []uint64{0, 1},
					"1":           []uint64{1000, 1001},
					"4":           []uint64{4000, 4001},
					"non-numeric": []uint64{1000000, 1000001},
					"no-info":     []uint64{2000000, 2000001},
				},
				NumCpus: 2,
				Description: map[string]*InterruptDescription{
					"0": &InterruptDescription{
						Controller:  "controller-0",
						HWInterrupt: "hw-irq-0",
						Devices:     []string{"device0"},
						Changed:     true,
					},
					"1": &InterruptDescription{
						Controller:  "controller-1",
						HWInterrupt: "hw-irq-1",
						Devices:     []string{"device1-1", "device1-2"},
						Changed:     true,
					},
					"4": &InterruptDescription{
						Controller:  "controller-4",
						HWInterrupt: "hw-irq-4",
						Devices:     []string{"device4-1", "device4-2"},
						Changed:     true,
					},
				},
			},
		},
		{
			name:       "remove_irq",
			procfsRoot: path.Join(interruptsTestdataDir, "field_mapping"),
			primeInterrupts: &Interrupts{
				ColIndexToCpuNum: nil,
				CpuHeaderLine:    "                  CPU0           CPU1",
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
				CpuHeaderLine:    "                  CPU0           CPU1",
				Irq: map[string][]uint64{
					"0":           []uint64{0, 1},
					"1":           []uint64{1000, 1001},
					"4":           []uint64{4000, 4001},
					"non-numeric": []uint64{1000000, 1000001},
					"no-info":     []uint64{2000000, 2000001},
				},
				NumCpus: 2,
				Description: map[string]*InterruptDescription{
					"0": &InterruptDescription{
						Controller:  "controller-0",
						HWInterrupt: "hw-irq-0",
						Devices:     []string{"device0"},
						Changed:     true,
					},
					"1": &InterruptDescription{
						Controller:  "controller-1",
						HWInterrupt: "hw-irq-1",
						Devices:     []string{"device1-1", "device1-2"},
						Changed:     true,
					},
					"4": &InterruptDescription{
						Controller:  "controller-4",
						HWInterrupt: "hw-irq-4",
						Devices:     []string{"device4-1", "device4-2"},
						Changed:     true,
					},
				},
			},
		},
		{
			procfsRoot: path.Join(interruptsTestdataDir, "remove_cpu"),
			primeInterrupts: &Interrupts{
				ColIndexToCpuNum: nil,
				CpuHeaderLine:    "                  CPU0           CPU1",
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
				Description: map[string]*InterruptDescription{
					"0": &InterruptDescription{
						Controller:  "controller-0",
						HWInterrupt: "hw-irq-0",
						Devices:     []string{"device0"},
						Changed:     true,
						irqInfo:     []byte("controller-0   hw-irq-0    device0"),
					},
					"1": &InterruptDescription{
						Controller:  "controller-1",
						HWInterrupt: "hw-irq-1",
						Devices:     []string{"device1-1", "device1-2"},
						Changed:     true,
						irqInfo:     []byte("controller-1   hw-irq-1    device1-1, device1-2"),
					},
					"4": &InterruptDescription{
						Controller:  "controller-4",
						HWInterrupt: "hw-irq-4",
						Devices:     []string{"device4-1", "device4-2"},
						Changed:     true,
						irqInfo:     []byte("controller-4   hw-irq-4    device4-1  , device4-2"),
					},
				},
			},
			wantInterrupts: &Interrupts{
				ColIndexToCpuNum: []int{1},
				CpuHeaderLine:    "                          CPU1",
				Irq: map[string][]uint64{
					"0":           []uint64{1},
					"1":           []uint64{1001},
					"4":           []uint64{4001},
					"non-numeric": []uint64{1000001},
					"no-info":     []uint64{2000001},
				},
				NumCpus: 1,
				Description: map[string]*InterruptDescription{
					"0": &InterruptDescription{
						Controller:  "controller-0",
						HWInterrupt: "hw-irq-0",
						Devices:     []string{"device0"},
						Changed:     false,
					},
					"1": &InterruptDescription{
						Controller:  "controller-1",
						HWInterrupt: "hw-irq-1",
						Devices:     []string{"device1-1", "device1-2"},
						Changed:     false,
					},
					"4": &InterruptDescription{
						Controller:  "controller-4",
						HWInterrupt: "hw-irq-4",
						Devices:     []string{"device4-1", "device4-2"},
						Changed:     false,
					},
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
