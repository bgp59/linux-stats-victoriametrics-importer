package procfs

import (
	"bytes"
	"fmt"
	"path"
	"testing"
)

type InterruptsCounterIndexToCpuNumTestCase struct {
	name                        string
	headerLine                  string
	wantCounterIndexToCpuNumMap []int
}

type InterruptsIrqInfoTestCase struct {
	name        string
	infoLine    string
	wantIrqInfo *InterruptsIrqInfo
}

type InterruptsTestCase struct {
	name            string
	procfsRoot      string
	primeInterrupts *Interrupts
	wantInterrupts  *Interrupts
	wantError       error
}

var interruptsTestdataDir = path.Join(PROCFS_TESTDATA_ROOT, "interrupts")

func cmpInterruptsCounterIndexToCpuNumMap(
	wantCounterIndexToCpuNumMap, gotCounterIndexToCpuNumMap []int,
	headerLine string,
	diffBuf *bytes.Buffer,
) *bytes.Buffer {
	if diffBuf == nil {
		diffBuf = &bytes.Buffer{}
	}

	errBuf := &bytes.Buffer{}

	if wantCounterIndexToCpuNumMap == nil && gotCounterIndexToCpuNumMap != nil {
		fmt.Fprintf(
			errBuf,
			"\nCounterIndexToCpuNum want: %v, got: %v",
			wantCounterIndexToCpuNumMap, gotCounterIndexToCpuNumMap,
		)
	} else if len(wantCounterIndexToCpuNumMap) != len(gotCounterIndexToCpuNumMap) {
		fmt.Fprintf(
			errBuf,
			"\nlen(CounterIndexToCpuNum) want: %d, got: %d",
			len(wantCounterIndexToCpuNumMap), len(gotCounterIndexToCpuNumMap),
		)
	} else {
		different := false
		for i, want := range wantCounterIndexToCpuNumMap {
			got := gotCounterIndexToCpuNumMap[i]
			if want != got {
				fmt.Fprintf(
					errBuf,
					"\nCounterIndexToCpuNum[%d]: want: %d, got: %d",
					i, want, got,
				)
				different = true
				break
			}
		}
		if different {
			fmt.Fprintf(
				errBuf,
				"\nCounterIndexToCpuNum:\n\twant: %v\n\t got: %v",
				wantCounterIndexToCpuNumMap, gotCounterIndexToCpuNumMap,
			)
		}

	}

	if errBuf.Len() > 0 {
		if headerLine != "" {
			fmt.Fprintf(diffBuf, "\nheaderLine: %q", headerLine)
		}
		diffBuf.ReadFrom(errBuf)
	}
	return diffBuf
}

func testInterruptsUpdateCounterIndexToCpuNumMap(tc *InterruptsCounterIndexToCpuNumTestCase, t *testing.T) {
	interrupts := NewInterrupts("")
	interrupts.updateCounterIndexToCpuNumMap([]byte(tc.headerLine))

	diffBuf := cmpInterruptsCounterIndexToCpuNumMap(
		tc.wantCounterIndexToCpuNumMap, interrupts.CounterIndexToCpuNum,
		tc.headerLine,
		nil,
	)

	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf)
	}
}

func cmpInterruptsIrqInfo(
	wantIrqInfo, gotIrqInfo *InterruptsIrqInfo,
	diffBuf *bytes.Buffer,
) *bytes.Buffer {
	if diffBuf == nil {
		diffBuf = &bytes.Buffer{}
	}

	errBuf := &bytes.Buffer{}

	var want, got []byte

	want, got = wantIrqInfo.Controller, gotIrqInfo.Controller
	if !bytes.Equal(want, got) {
		fmt.Fprintf(
			errBuf,
			"\nController:\n\twant: %v (%q)\n\t got: %v (%q)",
			want, want, got, got,
		)
	}

	want, got = wantIrqInfo.HWInterrupt, gotIrqInfo.HWInterrupt
	if !bytes.Equal(want, got) {
		fmt.Fprintf(
			errBuf,
			"\nHWInterrupt:\n\twant: %v (%q)\n\t got: %v (%q)",
			want, want, got, got,
		)
	}

	want, got = wantIrqInfo.Devices, gotIrqInfo.Devices
	if !bytes.Equal(want, got) {
		fmt.Fprintf(
			errBuf,
			"\nDevicest:\n\twant: %v (%q)\n\t got: %v (%q)",
			want, want, got, got,
		)
	}

	if wantIrqInfo.Changed != gotIrqInfo.Changed {
		fmt.Fprintf(
			errBuf,
			"\nChanged: want: %v, got: %v",
			wantIrqInfo.Changed, gotIrqInfo.Changed,
		)
	}

	if errBuf.Len() > 0 {
		fmt.Fprintf(diffBuf, "\ninfoLine: %q", gotIrqInfo.infoLine)
		diffBuf.ReadFrom(errBuf)
	}
	return diffBuf
}

func testInterruptsIrqInfo(tc *InterruptsIrqInfoTestCase, t *testing.T) {
	irqInfo := &InterruptsIrqInfo{}
	irqInfo.update([]byte(tc.infoLine))

	diffBuf := cmpInterruptsIrqInfo(tc.wantIrqInfo, irqInfo, nil)

	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf)
	}
}

func cloneInterruptsAndInfo(interrupts *Interrupts) *Interrupts {
	// Normally Info is shared for testing it should be cloned as well:
	newInterrupts := interrupts.Clone(true)
	newInterrupts.Info = make(map[string]*InterruptsIrqInfo)
	for irq, irqInfo := range interrupts.Info {
		newIrqInfo := &InterruptsIrqInfo{
			scanNum: irqInfo.scanNum,
		}
		if irqInfo.Controller != nil {
			newIrqInfo.Controller = make([]byte, len(irqInfo.Controller))
			copy(newIrqInfo.Controller, irqInfo.Controller)
		}
		if irqInfo.HWInterrupt != nil {
			newIrqInfo.HWInterrupt = make([]byte, len(irqInfo.HWInterrupt))
			copy(newIrqInfo.HWInterrupt, irqInfo.HWInterrupt)
		}
		if irqInfo.Devices != nil {
			newIrqInfo.Devices = make([]byte, len(irqInfo.Devices))
			copy(newIrqInfo.Devices, irqInfo.Devices)
		}
		if irqInfo.infoLine != nil {
			newIrqInfo.infoLine = make([]byte, len(irqInfo.infoLine))
			copy(newIrqInfo.infoLine, irqInfo.infoLine)
		}
		newInterrupts.Info[irq] = newIrqInfo
	}
	return newInterrupts
}

func testInterruptsParser(tc *InterruptsTestCase, t *testing.T) {
	t.Logf(`
name=%q
procfsRoot=%q
primeInterrupts=%v
`,
		tc.name, tc.procfsRoot, (tc.primeInterrupts != nil),
	)

	var interrupts *Interrupts
	if tc.primeInterrupts != nil {
		interrupts = cloneInterruptsAndInfo(tc.primeInterrupts)
		interrupts.path = InterruptsPath(tc.procfsRoot)
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

	// Counters:
	if wantInterrupts.numCounters != interrupts.numCounters {
		fmt.Fprintf(
			diffBuf,
			"\nNumCpus: want: %d, got: %d",
			wantInterrupts.numCounters, interrupts.numCounters,
		)
	}
	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf)
	}

	for irq, wantCounters := range wantInterrupts.Counters {
		gotCounters := interrupts.Counters[irq]
		if gotCounters == nil {
			fmt.Fprintf(
				diffBuf,
				"\nCounters: missing IRQ: %s",
				irq,
			)
		} else if len(wantCounters) < len(gotCounters) {
			fmt.Fprintf(
				diffBuf,
				"\nlen(Counters[%q]): want: %d, got: %d",
				irq, len(wantCounters), len(gotCounters),
			)
		} else {
			for i, want := range wantCounters {
				got := gotCounters[i]
				if want != got {
					fmt.Fprintf(
						diffBuf,
						"\nCounters[%q][%d]: want: %d, got: %d",
						irq, i, want, got,
					)
				}
			}
		}
	}

	for irq := range interrupts.Counters {
		if wantInterrupts.Counters[irq] == nil {
			fmt.Fprintf(
				diffBuf,
				"\nCounters: unexpected IRQ: %q",
				irq,
			)
		}
	}

	cmpInterruptsCounterIndexToCpuNumMap(
		wantInterrupts.CounterIndexToCpuNum, interrupts.CounterIndexToCpuNum,
		string(interrupts.cpuHeaderLine),
		diffBuf,
	)

	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf)
	}

	// Info:
	for irq, wantIrqInfo := range wantInterrupts.Info {
		gotIrqInfo := interrupts.Info[irq]
		if gotIrqInfo == nil {
			fmt.Fprintf(
				diffBuf,
				"\nInfo: missing IRQ: %q",
				irq,
			)
		} else {
			infoDiff := cmpInterruptsIrqInfo(wantIrqInfo, gotIrqInfo, nil)
			if infoDiff.Len() > 0 {
				fmt.Fprintf(
					diffBuf,
					"\nInfo[%q]: %s",
					irq, infoDiff,
				)
			}
		}
	}

	for irq := range interrupts.Info {
		if wantInterrupts.Info[irq] == nil {
			fmt.Fprintf(
				diffBuf,
				"\nInfo: unexpected IRQ: %q",
				irq,
			)
		}
	}

	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf)
	}
}

func TestInterruptsUpdateCounterIndexToCpuNumMap(t *testing.T) {
	for _, tc := range []*InterruptsCounterIndexToCpuNumTestCase{
		{
			"",
			" CPU0 CPU1",
			nil,
		},
		{
			"",
			" CPU0 CPU12",
			[]int{0, 12},
		},
	} {
		t.Run(
			tc.name,
			func(t *testing.T) { testInterruptsUpdateCounterIndexToCpuNumMap(tc, t) },
		)
	}
}

func TestInterruptsIrqInfo(t *testing.T) {
	for _, tc := range []*InterruptsIrqInfoTestCase{
		{
			infoLine: "controller hw-interrupt device",
			wantIrqInfo: &InterruptsIrqInfo{
				Controller:  []byte("controller"),
				HWInterrupt: []byte("hw-interrupt"),
				Devices:     []byte("device"),
				Changed:     true,
			},
		},
		{
			infoLine: "controller hw-interrupt dev1,dev2",
			wantIrqInfo: &InterruptsIrqInfo{
				Controller:  []byte("controller"),
				HWInterrupt: []byte("hw-interrupt"),
				Devices:     []byte("dev1,dev2"),
				Changed:     true,
			},
		},
		{
			infoLine: "\tcontroller    hw-interrupt\t\tdevice          \t",
			wantIrqInfo: &InterruptsIrqInfo{
				Controller:  []byte("controller"),
				HWInterrupt: []byte("hw-interrupt"),
				Devices:     []byte("device"),
				Changed:     true,
			},
		},
		{
			infoLine: "controller hw-interrupt ",
			wantIrqInfo: &InterruptsIrqInfo{
				Controller:  []byte("controller"),
				HWInterrupt: []byte("hw-interrupt"),
				Devices:     make([]byte, 0),
				Changed:     true,
			},
		},
	} {
		t.Run(
			tc.name,
			func(t *testing.T) { testInterruptsIrqInfo(tc, t) },
		)
	}
}

func TestInterruptsParser(t *testing.T) {
	for _, tc := range []*InterruptsTestCase{
		{
			name:       "field_mapping",
			procfsRoot: path.Join(interruptsTestdataDir, "field_mapping"),
			wantInterrupts: &Interrupts{
				Counters: map[string][]uint64{
					"0":           {0, 1},
					"1":           {1000, 1001},
					"4":           {4000, 4001},
					"non-numeric": {1000000, 1000001},
					"no-info":     {2000000, 2000001},
				},
				CounterIndexToCpuNum: nil,
				Info: map[string]*InterruptsIrqInfo{
					"0": {
						Controller:  []byte("controller-0"),
						HWInterrupt: []byte("hw-irq-0"),
						Devices:     []byte("device0"),
						Changed:     true,
					},
					"1": {
						Controller:  []byte("controller-1"),
						HWInterrupt: []byte("hw-irq-1"),
						Devices:     []byte("device1-1,device1-2"),
						Changed:     true,
					},
					"4": {
						Controller:  []byte("controller-4"),
						HWInterrupt: []byte("hw-irq-4"),
						Devices:     []byte("device4-1,device4-2"),
						Changed:     true,
					},
					"non-numeric": {},
					"no-info":     {},
				},
				numCounters: 2,
			},
		},
		{
			name:       "remove_irq",
			procfsRoot: path.Join(interruptsTestdataDir, "field_mapping"),
			primeInterrupts: &Interrupts{
				Counters: map[string][]uint64{
					"0":           {0, 1},
					"1":           {1000, 1001},
					"4":           {4000, 4001},
					"non-numeric": {1000000, 1000001},
					"no-info":     {2000000, 2000001},
					// To be removed:
					"11":     {11000, 11001},
					"delete": {3000000, 3000001},
				},
				CounterIndexToCpuNum: nil,
				Info: map[string]*InterruptsIrqInfo{
					"0": {
						Controller:  []byte("controller-0"),
						HWInterrupt: []byte("hw-irq-0"),
						Devices:     []byte("device0"),
						Changed:     true,
						scanNum:     1,
					},
					"1": {
						Controller:  []byte("controller-1"),
						HWInterrupt: []byte("hw-irq-1"),
						Devices:     []byte("device1-1,device1-2"),
						Changed:     true,
						scanNum:     1,
					},
					"4": {
						Controller:  []byte("controller-4"),
						HWInterrupt: []byte("hw-irq-4"),
						Devices:     []byte("device4-1,device4-2"),
						Changed:     true,
						scanNum:     1,
					},
					"non-numeric": {},
					"no-info":     {},
					// To be removed:
					"11": {
						Controller:  []byte("controller-11"),
						HWInterrupt: []byte("hw-irq-11"),
						Devices:     []byte("device11-1,device11-2"),
						Changed:     false,
						scanNum:     1,
					},
					"delete": {
						scanNum: 1,
					},
				},
				numCounters: 2,
				scanNum:     1,
			},
			wantInterrupts: &Interrupts{
				Counters: map[string][]uint64{
					"0":           {0, 1},
					"1":           {1000, 1001},
					"4":           {4000, 4001},
					"non-numeric": {1000000, 1000001},
					"no-info":     {2000000, 2000001},
				},
				CounterIndexToCpuNum: nil,
				Info: map[string]*InterruptsIrqInfo{
					"0": {
						Controller:  []byte("controller-0"),
						HWInterrupt: []byte("hw-irq-0"),
						Devices:     []byte("device0"),
						Changed:     true,
					},
					"1": {
						Controller:  []byte("controller-1"),
						HWInterrupt: []byte("hw-irq-1"),
						Devices:     []byte("device1-1,device1-2"),
						Changed:     true,
					},
					"4": {
						Controller:  []byte("controller-4"),
						HWInterrupt: []byte("hw-irq-4"),
						Devices:     []byte("device4-1,device4-2"),
						Changed:     true,
					},
					"non-numeric": {},
					"no-info":     {},
				},
				numCounters: 2,
			},
		},
		{
			name:       "remove_cpu",
			procfsRoot: path.Join(interruptsTestdataDir, "remove_cpu"),
			primeInterrupts: &Interrupts{
				Counters: map[string][]uint64{
					"0":           {20, 21},
					"1":           {21000, 21001},
					"4":           {24000, 24001},
					"non-numeric": {21000000, 21000001},
					"no-info":     {22000000, 22000001},
				},
				CounterIndexToCpuNum: nil,
				Info: map[string]*InterruptsIrqInfo{
					"0": {
						Controller:  []byte("controller-0"),
						HWInterrupt: []byte("hw-irq-0"),
						Devices:     []byte("device0"),
						Changed:     true,
						infoLine:    []byte("controller-0   hw-irq-0    device0"),
						scanNum:     1,
					},
					"1": {
						Controller:  []byte("controller-1"),
						HWInterrupt: []byte("hw-irq-1"),
						Devices:     []byte("device1-1,device1-2"),
						Changed:     true,
						infoLine:    []byte("controller-1   hw-irq-1    device1-1,device1-2"),
						scanNum:     1,
					},
					"4": {
						Controller:  []byte("controller-4"),
						HWInterrupt: []byte("hw-irq-4"),
						Devices:     []byte("device4-1,device4-2"),
						Changed:     true,
						infoLine:    []byte("controller-4   hw-irq-4    device4-1,device4-2"),
						scanNum:     1,
					},
					"non-numeric": {},
					"no-info":     {},
				},
				numCounters: 2,
				scanNum:     1,
			},
			wantInterrupts: &Interrupts{
				Counters: map[string][]uint64{
					"0":           {1},
					"1":           {1001},
					"4":           {4001},
					"non-numeric": {1000001},
					"no-info":     {2000001},
				},
				CounterIndexToCpuNum: []int{1},
				Info: map[string]*InterruptsIrqInfo{
					"0": {
						Controller:  []byte("controller-0"),
						HWInterrupt: []byte("hw-irq-0"),
						Devices:     []byte("device0"),
						Changed:     false,
					},
					"1": {
						Controller:  []byte("controller-1"),
						HWInterrupt: []byte("hw-irq-1"),
						Devices:     []byte("device1-1,device1-2"),
						Changed:     false,
					},
					"4": {
						Controller:  []byte("controller-4"),
						HWInterrupt: []byte("hw-irq-4"),
						Devices:     []byte("device4-1,device4-2"),
						Changed:     false,
					},
					"non-numeric": {},
					"no-info":     {},
				},
				numCounters: 1,
			},
		},
	} {
		t.Run(
			tc.name,
			func(t *testing.T) { testInterruptsParser(tc, t) },
		)
	}
}
