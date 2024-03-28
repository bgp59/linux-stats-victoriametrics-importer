package procfs

import (
	"bytes"
	"fmt"
	"path"
	"testing"
)

type InterruptsCpuListTestCase struct {
	name        string
	headerLine  string
	wantCpuList []int
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

func cmpCpuList(
	wantCpuList, gotCpuList []int,
	headerLine string,
	diffBuf *bytes.Buffer,
) *bytes.Buffer {
	if diffBuf == nil {
		diffBuf = &bytes.Buffer{}
	}

	errBuf := &bytes.Buffer{}

	if wantCpuList == nil && gotCpuList != nil {
		fmt.Fprintf(
			errBuf,
			"\nCpuList want: %v, got: %v",
			wantCpuList, gotCpuList,
		)
	} else if len(wantCpuList) != len(gotCpuList) {
		fmt.Fprintf(
			errBuf,
			"\nlen(CpuList) want: %d, got: %d",
			len(wantCpuList), len(gotCpuList),
		)
	} else {
		different := false
		for i, want := range wantCpuList {
			got := gotCpuList[i]
			if want != got {
				fmt.Fprintf(
					errBuf,
					"\nCpuList[%d]: want: %d, got: %d",
					i, want, got,
				)
				different = true
				break
			}
		}
		if different {
			fmt.Fprintf(
				errBuf,
				"\nCpuList:\n\twant: %v\n\t got: %v",
				wantCpuList, gotCpuList,
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

func testInterruptsUpdateCpuList(tc *InterruptsCpuListTestCase, t *testing.T) {
	interrupts := NewInterrupts("")
	interrupts.updateCpuList([]byte(tc.headerLine))

	diffBuf := cmpCpuList(
		tc.wantCpuList, interrupts.CpuList,
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

	if wantIrqInfo.IrqChanged != gotIrqInfo.IrqChanged {
		fmt.Fprintf(
			errBuf,
			"\nChanged: want: %v, got: %v",
			wantIrqInfo.IrqChanged, gotIrqInfo.IrqChanged,
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

// InterruptsInfo is normally shared, except for testing:
func cloneInterruptsInfo(info *InterruptsInfo) *InterruptsInfo {
	newInfo := &InterruptsInfo{
		IrqInfo:        map[string]*InterruptsIrqInfo{},
		IrqChanged:     info.IrqChanged,
		CpuListChanged: info.CpuListChanged,
		scanNum:        info.scanNum,
	}
	if info.cpuHeaderLine != nil {
		newInfo.cpuHeaderLine = make([]byte, len(info.cpuHeaderLine))
		copy(newInfo.cpuHeaderLine, info.cpuHeaderLine)
	}
	for irq, irqInfo := range info.IrqInfo {
		newIrqInfo := &InterruptsIrqInfo{
			IrqChanged: irqInfo.IrqChanged,
			scanNum:    irqInfo.scanNum,
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
		newInfo.IrqInfo[irq] = newIrqInfo
	}
	return newInfo
}

func cloneInterruptsAndInfo(interrupts *Interrupts) *Interrupts {
	// Normally Info is shared for testing it should be cloned as well:
	newInterrupts := interrupts.Clone(true)
	newInterrupts.Info = cloneInterruptsInfo(interrupts.Info)
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
	if wantInterrupts.NumCounters != interrupts.NumCounters {
		fmt.Fprintf(
			diffBuf,
			"\nNumCpus: want: %d, got: %d",
			wantInterrupts.NumCounters, interrupts.NumCounters,
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

	cmpCpuList(
		wantInterrupts.CpuList, interrupts.CpuList,
		string(interrupts.Info.cpuHeaderLine),
		diffBuf,
	)

	if wantInterrupts.Info.CpuListChanged != interrupts.Info.CpuListChanged {
		fmt.Fprintf(
			diffBuf,
			"\nInfo.CpuListChanged: want: %v, got: %v",
			wantInterrupts.Info.CpuListChanged, interrupts.Info.CpuListChanged,
		)
	}

	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf)
	}

	// Info:
	for irq, wantIrqInfo := range wantInterrupts.Info.IrqInfo {
		gotIrqInfo := interrupts.Info.IrqInfo[irq]
		if gotIrqInfo == nil {
			fmt.Fprintf(
				diffBuf,
				"\nInfo.IrqInfo: missing IRQ: %q",
				irq,
			)
		} else {
			infoDiff := cmpInterruptsIrqInfo(wantIrqInfo, gotIrqInfo, nil)
			if infoDiff.Len() > 0 {
				fmt.Fprintf(
					diffBuf,
					"\nInfo.IrqInfo[%q]: %s",
					irq, infoDiff,
				)
			}
		}
	}

	for irq := range interrupts.Info.IrqInfo {
		if wantInterrupts.Info.IrqInfo[irq] == nil {
			fmt.Fprintf(
				diffBuf,
				"\nInfo.IrqInfo: unexpected IRQ: %q",
				irq,
			)
		}
	}

	if wantInterrupts.Info.IrqChanged != interrupts.Info.IrqChanged {
		fmt.Fprintf(
			diffBuf,
			"\nInfo.IrqChanged: want: %v, got: %v",
			wantInterrupts.Info.IrqChanged, interrupts.Info.IrqChanged,
		)
	}

	if diffBuf.Len() > 0 {
		t.Fatal(diffBuf)
	}
}

func TestInterruptsUpdateCpuList(t *testing.T) {
	for _, tc := range []*InterruptsCpuListTestCase{
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
			func(t *testing.T) { testInterruptsUpdateCpuList(tc, t) },
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
			},
		},
		{
			infoLine: "controller hw-interrupt dev1,dev2",
			wantIrqInfo: &InterruptsIrqInfo{
				Controller:  []byte("controller"),
				HWInterrupt: []byte("hw-interrupt"),
				Devices:     []byte("dev1,dev2"),
			},
		},
		{
			infoLine: "\tcontroller    hw-interrupt\t\tdevice          \t",
			wantIrqInfo: &InterruptsIrqInfo{
				Controller:  []byte("controller"),
				HWInterrupt: []byte("hw-interrupt"),
				Devices:     []byte("device"),
			},
		},
		{
			infoLine: "controller hw-interrupt ",
			wantIrqInfo: &InterruptsIrqInfo{
				Controller:  []byte("controller"),
				HWInterrupt: []byte("hw-interrupt"),
				Devices:     make([]byte, 0),
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
				CpuList: nil,
				Info: &InterruptsInfo{
					IrqInfo: map[string]*InterruptsIrqInfo{
						"0": {
							Controller:  []byte("controller-0"),
							HWInterrupt: []byte("hw-irq-0"),
							Devices:     []byte("device0"),
							IrqChanged:  true,
						},
						"1": {
							Controller:  []byte("controller-1"),
							HWInterrupt: []byte("hw-irq-1"),
							Devices:     []byte("device1-1,device1-2"),
							IrqChanged:  true,
						},
						"4": {
							Controller:  []byte("controller-4"),
							HWInterrupt: []byte("hw-irq-4"),
							Devices:     []byte("device4-1,device4-2"),
							IrqChanged:  true,
						},
						"non-numeric": {},
						"no-info":     {},
					},
					IrqChanged:     true,
					CpuListChanged: true,
				},
				NumCounters: 2,
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
				CpuList: nil,
				Info: &InterruptsInfo{
					IrqInfo: map[string]*InterruptsIrqInfo{
						"0": {
							Controller:  []byte("controller-0"),
							HWInterrupt: []byte("hw-irq-0"),
							Devices:     []byte("device0"),
							IrqChanged:  true,
							scanNum:     1,
						},
						"1": {
							Controller:  []byte("controller-1"),
							HWInterrupt: []byte("hw-irq-1"),
							Devices:     []byte("device1-1,device1-2"),
							IrqChanged:  true,
							scanNum:     1,
						},
						"4": {
							Controller:  []byte("controller-4"),
							HWInterrupt: []byte("hw-irq-4"),
							Devices:     []byte("device4-1,device4-2"),
							IrqChanged:  true,
							scanNum:     1,
						},
						"non-numeric": {},
						"no-info":     {},
						// To be removed:
						"11": {
							Controller:  []byte("controller-11"),
							HWInterrupt: []byte("hw-irq-11"),
							Devices:     []byte("device11-1,device11-2"),
							IrqChanged:  false,
							scanNum:     1,
						},
						"delete": {
							scanNum: 1,
						},
					},
					IrqChanged:     false,
					CpuListChanged: true,
					cpuHeaderLine:  []byte("                  CPU0           CPU1"),
					scanNum:        1,
				},
				NumCounters: 2,
			},
			wantInterrupts: &Interrupts{
				Counters: map[string][]uint64{
					"0":           {0, 1},
					"1":           {1000, 1001},
					"4":           {4000, 4001},
					"non-numeric": {1000000, 1000001},
					"no-info":     {2000000, 2000001},
				},
				CpuList: nil,
				Info: &InterruptsInfo{
					IrqInfo: map[string]*InterruptsIrqInfo{
						"0": {
							Controller:  []byte("controller-0"),
							HWInterrupt: []byte("hw-irq-0"),
							Devices:     []byte("device0"),
							IrqChanged:  true,
						},
						"1": {
							Controller:  []byte("controller-1"),
							HWInterrupt: []byte("hw-irq-1"),
							Devices:     []byte("device1-1,device1-2"),
							IrqChanged:  true,
						},
						"4": {
							Controller:  []byte("controller-4"),
							HWInterrupt: []byte("hw-irq-4"),
							Devices:     []byte("device4-1,device4-2"),
							IrqChanged:  true,
						},
						"non-numeric": {},
						"no-info":     {},
					},
					IrqChanged:     true,
					CpuListChanged: false,
				},
				NumCounters: 2,
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
				CpuList: nil,
				Info: &InterruptsInfo{
					IrqInfo: map[string]*InterruptsIrqInfo{
						"0": {
							Controller:  []byte("controller-0"),
							HWInterrupt: []byte("hw-irq-0"),
							Devices:     []byte("device0"),
							IrqChanged:  true,
							infoLine:    []byte("controller-0   hw-irq-0    device0"),
							scanNum:     1,
						},
						"1": {
							Controller:  []byte("controller-1"),
							HWInterrupt: []byte("hw-irq-1"),
							Devices:     []byte("device1-1,device1-2"),
							IrqChanged:  true,
							infoLine:    []byte("controller-1   hw-irq-1    device1-1,device1-2"),
							scanNum:     1,
						},
						"4": {
							Controller:  []byte("controller-4"),
							HWInterrupt: []byte("hw-irq-4"),
							Devices:     []byte("device4-1,device4-2"),
							IrqChanged:  true,
							infoLine:    []byte("controller-4   hw-irq-4    device4-1,device4-2"),
							scanNum:     1,
						},
						"non-numeric": {},
						"no-info":     {},
					},
					IrqChanged:     true,
					CpuListChanged: false,
					cpuHeaderLine:  []byte("                  CPU0           CPU1"),
					scanNum:        1,
				},
				NumCounters: 2,
			},
			wantInterrupts: &Interrupts{
				Counters: map[string][]uint64{
					"0":           {1},
					"1":           {1001},
					"4":           {4001},
					"non-numeric": {1000001},
					"no-info":     {2000001},
				},
				CpuList: []int{1},
				Info: &InterruptsInfo{
					IrqInfo: map[string]*InterruptsIrqInfo{
						"0": {
							Controller:  []byte("controller-0"),
							HWInterrupt: []byte("hw-irq-0"),
							Devices:     []byte("device0"),
							IrqChanged:  false,
						},
						"1": {
							Controller:  []byte("controller-1"),
							HWInterrupt: []byte("hw-irq-1"),
							Devices:     []byte("device1-1,device1-2"),
							IrqChanged:  false,
						},
						"4": {
							Controller:  []byte("controller-4"),
							HWInterrupt: []byte("hw-irq-4"),
							Devices:     []byte("device4-1,device4-2"),
							IrqChanged:  false,
						},
						"non-numeric": {},
						"no-info":     {},
					},
					IrqChanged:     false,
					CpuListChanged: true,
				},
				NumCounters: 1,
			},
		},
	} {
		t.Run(
			tc.name,
			func(t *testing.T) { testInterruptsParser(tc, t) },
		)
	}
}
