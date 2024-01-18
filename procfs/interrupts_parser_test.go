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

	// if !bytes.Equal(wantInterrupts.cpuHeaderLine, interrupts.cpuHeaderLine) {
	// 	fmt.Fprintf(
	// 		diffBuf,
	// 		"\ncpuHeaderLine:\n\twant: %q,\n\t got: %q",
	// 		wantInterrupts.cpuHeaderLine, interrupts.cpuHeaderLine,
	// 	)
	// }
	if wantInterrupts.IndexToCpuChanged != interrupts.IndexToCpuChanged {
		fmt.Fprintf(
			diffBuf,
			"\nIndexToCpuChanged: want: %v, got: %v",
			wantInterrupts.IndexToCpuChanged, interrupts.IndexToCpuChanged,
		)
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}

	if wantInterrupts.numCounters != interrupts.numCounters {
		fmt.Fprintf(
			diffBuf,
			"\nNumCpus: want: %d, got: %d",
			wantInterrupts.numCounters, interrupts.numCounters,
		)
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf.String())
	}

	if wantInterrupts.CounterIndexToCpuNum == nil {
		if interrupts.CounterIndexToCpuNum != nil {
			fmt.Fprintf(
				diffBuf,
				"\nCounterIndexToCpuNum: want: %v, got: %v",
				wantInterrupts.CounterIndexToCpuNum, interrupts.CounterIndexToCpuNum,
			)
		}
	} else {
		if interrupts.CounterIndexToCpuNum == nil {
			fmt.Fprintf(
				diffBuf,
				"\nCounterIndexToCpuNum: want: %v, got: %v",
				wantInterrupts.CounterIndexToCpuNum, interrupts.CounterIndexToCpuNum,
			)
		}

		if len(wantInterrupts.CounterIndexToCpuNum) != len(interrupts.CounterIndexToCpuNum) {
			fmt.Fprintf(
				diffBuf,
				"\nCounterIndexToCpuNum length: want %d, got: %d",
				len(wantInterrupts.CounterIndexToCpuNum), len(interrupts.CounterIndexToCpuNum),
			)
		}

		for i, wantCpuNum := range wantInterrupts.CounterIndexToCpuNum {
			gotCpuNum := interrupts.CounterIndexToCpuNum[i]
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

	for irq, wantInterruptsIrq := range wantInterrupts.Irq {
		gotInterruptsIrq := interrupts.Irq[irq]
		if gotInterruptsIrq == nil {
			fmt.Fprintf(
				diffBuf,
				"\nIrq: missing %q",
				irq,
			)
			continue
		}

		wantCounters, gotCounters := wantInterruptsIrq.Counters, gotInterruptsIrq.Counters
		if len(gotCounters) != wantInterrupts.numCounters {
			fmt.Fprintf(
				diffBuf,
				"\nIrq[%q].Counters length: want: %d, got: %d",
				irq, wantInterrupts.numCounters, len(gotCounters),
			)
		} else {
			for i := 0; i < wantInterrupts.numCounters; i++ {
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

		if !bytes.Equal(wantInterruptsIrq.Controller, gotInterruptsIrq.Controller) {
			fmt.Fprintf(
				diffBuf,
				"\nIrq[%q].Controller:\n\twant: %v (%q)\n\tgot: %v (%q)",
				irq,
				wantInterruptsIrq.Controller, wantInterruptsIrq.Controller,
				gotInterruptsIrq.Controller, gotInterruptsIrq.Controller,
			)
		}
		if !bytes.Equal(wantInterruptsIrq.HWInterrupt, gotInterruptsIrq.HWInterrupt) {
			fmt.Fprintf(
				diffBuf,
				"\nIrq[%q].HWInterrupt:\n\twant: %v (%q)\n\tgot: %v (%q)",
				irq,
				wantInterruptsIrq.HWInterrupt, wantInterruptsIrq.HWInterrupt,
				gotInterruptsIrq.HWInterrupt, gotInterruptsIrq.HWInterrupt,
			)
		}
		if !bytes.Equal(wantInterruptsIrq.Devices, gotInterruptsIrq.Devices) {
			fmt.Fprintf(
				diffBuf,
				"\nIrq[%q].Devices:\n\twant: %v (%q)\n\tgot: %v (%q)",
				irq,
				wantInterruptsIrq.Devices, wantInterruptsIrq.Devices,
				gotInterruptsIrq.Devices, gotInterruptsIrq.Devices,
			)
		}
		if wantInterruptsIrq.InfoChanged != gotInterruptsIrq.InfoChanged {
			fmt.Fprintf(
				diffBuf,
				"\nIrq[%q].InfoChanged: want: %v, got: %v",
				irq, wantInterruptsIrq.InfoChanged, gotInterruptsIrq.InfoChanged,
			)
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
				CounterIndexToCpuNum: nil,
				Irq: map[string]*InterruptsIrq{
					"0": {
						Counters:    []uint64{0, 1},
						Controller:  []byte("controller-0"),
						HWInterrupt: []byte("hw-irq-0"),
						Devices:     []byte("device0"),
						InfoChanged: true,
					},
					"1": {
						Counters:    []uint64{1000, 1001},
						Controller:  []byte("controller-1"),
						HWInterrupt: []byte("hw-irq-1"),
						Devices:     []byte("device1-1,device1-2"),
						InfoChanged: true,
					},
					"4": {
						Counters:    []uint64{4000, 4001},
						Controller:  []byte("controller-4"),
						HWInterrupt: []byte("hw-irq-4"),
						Devices:     []byte("device4-1,device4-2"),
						InfoChanged: true,
					},
					"non-numeric": {
						Counters: []uint64{1000000, 1000001},
					},
					"no-info": {
						Counters: []uint64{2000000, 2000001},
					},
				},
				IndexToCpuChanged: true,
				numCounters:       2,
			},
		},
		{
			name:       "remove_irq",
			procfsRoot: path.Join(interruptsTestdataDir, "field_mapping"),
			primeInterrupts: &Interrupts{
				CounterIndexToCpuNum: nil,
				Irq: map[string]*InterruptsIrq{
					"0": {
						Counters:    []uint64{0, 1},
						Controller:  []byte("controller-0"),
						HWInterrupt: []byte("hw-irq-0"),
						Devices:     []byte("device0"),
						InfoChanged: true,
						info:        []byte("controller-0   hw-irq-0    device0"),
						scanNum:     1,
					},
					"1": {
						Counters:    []uint64{1000, 1001},
						Controller:  []byte("controller-1"),
						HWInterrupt: []byte("hw-irq-1"),
						Devices:     []byte("device1-1,device1-2"),
						InfoChanged: true,
						info:        []byte("controller-1   hw-irq-1    device1-1,device1-2  "),
						scanNum:     1,
					},
					"4": {
						Counters:    []uint64{4000, 4001},
						Controller:  []byte("controller-4"),
						HWInterrupt: []byte("hw-irq-4"),
						Devices:     []byte("device4-1,device4-2"),
						InfoChanged: true,
						info:        []byte("controller-4   hw-irq-4    device4-1,device4-2"),
						scanNum:     1,
					},
					"non-numeric": {
						Counters: []uint64{1000000, 1000001},
						scanNum:  1,
					},
					"no-info": {
						Counters: []uint64{2000000, 2000001},
						scanNum:  1,
					},
					// removed IRQs:
					"11": {
						Counters:    []uint64{11000, 11001},
						Controller:  []byte("controller-11"),
						HWInterrupt: []byte("hw-irq-11"),
						Devices:     []byte("device11-1,device11-2"),
						InfoChanged: false,
						scanNum:     1,
					},
					"delete": {
						Counters: []uint64{3000000, 3000001},
						scanNum:  1,
					},
				},
				IndexToCpuChanged: true,
				numCounters:       2,
				cpuHeaderLine:     []byte("                  CPU0           CPU1"),
				scanNum:           1,
			},
			wantInterrupts: &Interrupts{
				CounterIndexToCpuNum: nil,
				Irq: map[string]*InterruptsIrq{
					"0": {
						Counters:    []uint64{0, 1},
						Controller:  []byte("controller-0"),
						HWInterrupt: []byte("hw-irq-0"),
						Devices:     []byte("device0"),
						InfoChanged: false,
					},
					"1": {
						Counters:    []uint64{1000, 1001},
						Controller:  []byte("controller-1"),
						HWInterrupt: []byte("hw-irq-1"),
						Devices:     []byte("device1-1,device1-2"),
						InfoChanged: false,
					},
					"4": {
						Counters:    []uint64{4000, 4001},
						Controller:  []byte("controller-4"),
						HWInterrupt: []byte("hw-irq-4"),
						Devices:     []byte("device4-1,device4-2"),
						InfoChanged: false,
					},
					"non-numeric": {
						Counters: []uint64{1000000, 1000001},
					},
					"no-info": {
						Counters: []uint64{2000000, 2000001},
					},
				},
				IndexToCpuChanged: false,
				numCounters:       2,
			},
		},
		{
			procfsRoot: path.Join(interruptsTestdataDir, "remove_cpu"),
			primeInterrupts: &Interrupts{
				CounterIndexToCpuNum: nil,
				Irq: map[string]*InterruptsIrq{
					"0": {
						Counters:    []uint64{20, 21},
						Controller:  []byte("controller-0"),
						HWInterrupt: []byte("hw-irq-0"),
						Devices:     []byte("device0"),
						InfoChanged: true,
						info:        []byte("controller-0   hw-irq-0    device0"),
						scanNum:     10,
					},
					"1": {
						Counters:    []uint64{21000, 21001},
						Controller:  []byte("controller-1"),
						HWInterrupt: []byte("hw-irq-1"),
						Devices:     []byte("device1-1,device1-2"),
						InfoChanged: true,
						info:        []byte("controller-1   hw-irq-1    device1-1,device1-2  "),
						scanNum:     10,
					},
					"4": {
						Counters:    []uint64{24000, 24001},
						Controller:  []byte("controller-4"),
						HWInterrupt: []byte("hw-irq-4"),
						Devices:     []byte("device4-1,device4-2"),
						InfoChanged: true,
						info:        []byte("controller-4   hw-irq-4    device4-1,device4-2"),
						scanNum:     10,
					},
					"non-numeric": {
						Counters: []uint64{21000000, 21000001},
						scanNum:  10,
					},
					"no-info": {
						Counters: []uint64{22000000, 22000001},
						scanNum:  10,
					},
				},
				IndexToCpuChanged: true,
				numCounters:       2,
				cpuHeaderLine:     []byte("                  CPU0           CPU1"),
				scanNum:           10,
			},
			wantInterrupts: &Interrupts{
				CounterIndexToCpuNum: []int{1},
				Irq: map[string]*InterruptsIrq{
					"0": {
						Counters:    []uint64{1},
						Controller:  []byte("controller-0"),
						HWInterrupt: []byte("hw-irq-0"),
						Devices:     []byte("device0"),
						InfoChanged: false,
					},
					"1": {
						Counters:    []uint64{1001},
						Controller:  []byte("controller-1"),
						HWInterrupt: []byte("hw-irq-1"),
						Devices:     []byte("device1-1,device1-2"),
						InfoChanged: true,
					},
					"4": {
						Counters:    []uint64{4001},
						Controller:  []byte("controller-4"),
						HWInterrupt: []byte("hw-irq-4"),
						Devices:     []byte("device4-1,device4-2"),
						InfoChanged: false,
					},
					"non-numeric": {
						Counters: []uint64{1000001},
					},
					"no-info": {
						Counters: []uint64{2000001},
					},
				},
				IndexToCpuChanged: true,
				numCounters:       1,
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
