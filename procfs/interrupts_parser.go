// parser for /proc/interrupts

package procfs

import (
	"bytes"
	"fmt"
	"path"
)

// The best explanation for /proc/interrupts syntax so far: https://serverfault.com/a/1118526
//
// \/  ... linux global irq number
//             \/  ...   number of occurred irqs on CPU 0
//                         \/  ...    number of occurred irqs on CPU 1
//                               \/  ...  irq chip receiving the irq
//                                          \/ ... hw irq number and type of irq
//                                                           \/  ... assigned action of irq
//                                                                   (-> irq handler inside a driver, can also be assigned to more then just one handler / driver)
//
// cat /proc/interrupts
//            CPU0       CPU1
//   0:         22          0  IR-IO-APIC   2-edge            timer
//   1:          2          0  IR-IO-APIC   1-edge            i8042
//   8:          1          0  IR-IO-APIC   8-edge            rtc0
//   9:          0          0  IR-IO-APIC   9-fasteoi         acpi
//  12:          4          0  IR-IO-APIC   12-edge           i8042
// 120:          0          0  DMAR-MSI     0-edge            dmar0
// 122:          0          0  IR-PCI-MSI   327680-edge       xhci_hcd
// 123:      25164    5760490  IR-PCI-MSI   1048576-edge      enp2s0
// 124:         17    5424414  IR-PCI-MSI   524288-edge       amdgpu
// ...
// NMI:          0          0 Non-maskable interrupts
// LOC:          0          0 Local timer interrupts
// SPU:          0          0 Spurious interrupts
// PMI:          0          0 Performance monitoring interrupts

type InterruptsIrqInfo struct {
	// Chip receiving the IRQ:
	Controller []byte
	// H/W IRQ number and type:
	HWInterrupt []byte
	// Devices, comma separated, typically 1:
	Devices []byte
	// Whether the info has changed in the current scan or not. This information
	// may be used for generating cached metrics which may be expensive; the
	// flag can be tested to verify if the cached values are still valid or not.
	Changed bool
	// Cache the line part used to build the info above; if unchanged from the
	// previous scan, the information is still valid:
	infoLine []byte
	// The scan# where this IRQ was found, used for removing out of scope IRQs,
	// see scanNum in Interrupts.
	scanNum int
}

type InterruptsInfo struct {
	// IRQ info, indexed by IRQ:
	IrqInfo map[string]*InterruptsIrqInfo
	// Whether any info changed at this scan or not:
	Changed bool
	// Each scan has a different scan# from the previous one. IRQ's not
	// associated with the most recent scan will be removed:
	scanNum int
}

type Interrupts struct {
	// IRQ counters, indexed by IRQ:
	Counters map[string][]uint64
	// Mapping counter# -> CPU#, based on the header line; nil if no mapping is
	// needed, i.e. counter# == CPU#:
	CounterIndexToCpuNum []int
	// Info:
	Info *InterruptsInfo

	// The path file to  read:
	path string
	// Cache the line used for building the counter# -> CPU# mappings above; if
	// the line is unchanged from the previous run then the mapping is still
	// valid.
	cpuHeaderLine []byte
	// The number of counters per line; this is needed if CounterIndexToCpuNum
	// is nil, since it cannot be inferred:
	numCounters int
}

var interruptsReadFileBufPool = ReadFileBufPoolReadUnbound

func InterruptsPath(procfsRoot string) string {
	return path.Join(procfsRoot, "interrupts")
}

func NewInterruptsInfo() *InterruptsInfo {
	return &InterruptsInfo{
		IrqInfo: map[string]*InterruptsIrqInfo{},
	}
}

func NewInterrupts(procfsRoot string) *Interrupts {
	return &Interrupts{
		Counters: map[string][]uint64{},
		Info:     NewInterruptsInfo(),
		path:     InterruptsPath(procfsRoot),
	}
}

// N.B. The cloned object below will *share* the Info. Metrics generators geared
// for deltas will use 2 objects, current and previous, for detecting
// changes; after each scan the 2 objects are flipped. While this is useful for
// counters, it is counterproductive for Info, which assumes previous scan info,
// rather than 2 scans back (an object becomes current every other scan).
func (interrupts *Interrupts) Clone(full bool) *Interrupts {
	newInterrupts := &Interrupts{
		Info:        interrupts.Info, // i.e. shared
		path:        interrupts.path,
		numCounters: interrupts.numCounters,
	}
	if interrupts.Counters != nil {
		newInterrupts.Counters = make(map[string][]uint64)
		for irq, irqCounters := range interrupts.Counters {
			newInterrupts.Counters[irq] = make([]uint64, len(irqCounters))
			if full {
				copy(newInterrupts.Counters[irq], irqCounters)
			}
		}
	}
	if interrupts.CounterIndexToCpuNum != nil {
		newInterrupts.CounterIndexToCpuNum = make([]int, len(interrupts.CounterIndexToCpuNum))
		copy(newInterrupts.CounterIndexToCpuNum, interrupts.CounterIndexToCpuNum)
	}
	if interrupts.cpuHeaderLine != nil {
		newInterrupts.cpuHeaderLine = make([]byte, len(interrupts.cpuHeaderLine))
		copy(newInterrupts.cpuHeaderLine, interrupts.cpuHeaderLine)
	}

	return newInterrupts
}

func (interrupts *Interrupts) updateCounterIndexToCpuNumMap(cpuHeaderLine []byte) error {
	needsCounterIndexToCpuNumMap := false
	fields := bytes.Fields(cpuHeaderLine)
	interrupts.numCounters = len(fields)
	counterIndexToCpuNum := make([]int, interrupts.numCounters)
	for index, cpu := range fields {
		cpuNum, l := 0, len(cpu)
		if l <= 3 {
			return fmt.Errorf("%q: invalid cpu spec, not CPUNNN", string(cpu))
		}
		for pos := 3; pos < l; pos++ {
			digit := cpu[pos] - '0'
			if digit < 10 {
				cpuNum = (cpuNum << 3) + (cpuNum << 1) + int(digit)
			} else {
				return fmt.Errorf("%q: invalid cpu spec, not CPUNNN", string(cpu))
			}
		}
		counterIndexToCpuNum[index] = cpuNum
		if index != cpuNum {
			needsCounterIndexToCpuNumMap = true
		}
	}
	if needsCounterIndexToCpuNumMap {
		interrupts.CounterIndexToCpuNum = counterIndexToCpuNum
	} else {
		interrupts.CounterIndexToCpuNum = nil
	}
	dstHdrCap, srcHdrLen := cap(interrupts.cpuHeaderLine), len(cpuHeaderLine)
	if dstHdrCap < srcHdrLen {
		interrupts.cpuHeaderLine = make([]byte, srcHdrLen)
		copy(interrupts.cpuHeaderLine, cpuHeaderLine)
	} else {
		interrupts.cpuHeaderLine = interrupts.cpuHeaderLine[:dstHdrCap]
		copy(interrupts.cpuHeaderLine, cpuHeaderLine)
		interrupts.cpuHeaderLine = interrupts.cpuHeaderLine[:srcHdrLen]
	}
	return nil
}

func (irqInfo *InterruptsIrqInfo) update(infoLine []byte) {
	dstCap, l := cap(irqInfo.infoLine), len(infoLine)
	if dstCap < l {
		irqInfo.infoLine = make([]byte, l)
		copy(irqInfo.infoLine, infoLine)
	} else {
		irqInfo.infoLine = irqInfo.infoLine[:l]
		copy(irqInfo.infoLine, infoLine)
	}

	pos := 0

	// Controller:
	for ; pos < l && isWhitespace[infoLine[pos]]; pos++ {
	}
	start := pos
	for ; pos < l && !isWhitespace[infoLine[pos]]; pos++ {
	}
	irqInfo.Controller = irqInfo.infoLine[start:pos]

	// H/W IRQ number and type:
	for ; pos < l && isWhitespace[infoLine[pos]]; pos++ {
	}
	start = pos
	for ; pos < l && !isWhitespace[infoLine[pos]]; pos++ {
	}
	irqInfo.HWInterrupt = irqInfo.infoLine[start:pos]

	// Devices:
	for ; pos < l && isWhitespace[infoLine[pos]]; pos++ {
	}
	start = pos
	// Strip trailing spaces, if any:
	pos = l - 1
	for ; start <= pos && isWhitespace[infoLine[pos]]; pos-- {
	}
	irqInfo.Devices = irqInfo.infoLine[start : pos+1]
}

func (interrupts *Interrupts) Parse() error {
	fBuf, err := interruptsReadFileBufPool.ReadFile(interrupts.path)
	defer interruptsReadFileBufPool.ReturnBuf(fBuf)
	if err != nil {
		return err
	}

	buf, l := fBuf.Bytes(), fBuf.Len()

	numCounters := interrupts.numCounters
	info := interrupts.Info
	info.Changed = false
	scanNum := info.scanNum + 1
	for pos, lineNum := 0, 1; pos < l; lineNum++ {
		// Line starts here:
		startLine, eol := pos, false

		if lineNum == 1 {
			// Look for changes in the CPU header line; update col# to cpu# as
			// needed:
			cpuHeaderLine := interrupts.cpuHeaderLine
			cpuHeaderLineLen := len(cpuHeaderLine)
			for ; pos < l && pos < cpuHeaderLineLen && buf[pos] == cpuHeaderLine[pos]; pos++ {
			}
			if pos != cpuHeaderLineLen || (pos < l && buf[pos] != '\n') {
				// The CPU header has changed:
				for ; pos < l && buf[pos] != '\n'; pos++ {
				}
				err = interrupts.updateCounterIndexToCpuNumMap(buf[0:pos])
				if err != nil {
					return fmt.Errorf(
						"%s#%d: %q: %v",
						interrupts.path, lineNum, getCurrentLine(buf, startLine), err,
					)
				}
				numCounters = interrupts.numCounters
			}
			pos++
			continue
		}

		// IRQ line:

		// Parse IRQ:
		for ; pos < l && isWhitespace[buf[pos]]; pos++ {
		}
		irqStart, irqEnd, irqIsNum := pos, -1, true
		for ; !eol && irqEnd < 0 && pos < l; pos++ {
			c := buf[pos]
			if c == ':' {
				irqEnd = pos
			} else if irqIsNum && c-'0' >= 10 {
				irqIsNum = false
			} else {
				eol = (c == '\n')
			}
		}
		if irqEnd < 0 {
			return fmt.Errorf(
				"%s#%d: %q: missing `IRQ:'",
				interrupts.path, lineNum, getCurrentLine(buf, startLine),
			)
		}
		irq := string(buf[irqStart:irqEnd])

		// Parse ` NNN NNN ... NNN' interrupt counters:
		counters := interrupts.Counters[irq]
		if counters == nil {
			counters = make([]uint64, numCounters)
			interrupts.Counters[irq] = counters
		} else {
			if cap(counters) < numCounters {
				counters = make([]uint64, numCounters)
				interrupts.Counters[irq] = counters
			} else if len(counters) != numCounters {
				counters = counters[:numCounters]
				interrupts.Counters[irq] = counters
			}
		}

		counterIndex := 0
		for !eol && pos < l && counterIndex < numCounters {
			for ; pos < l && isWhitespace[buf[pos]]; pos++ {
			}
			value, foundValue := uint64(0), false
			for done := false; !done && pos < l; pos++ {
				c := buf[pos]
				if digit := c - '0'; digit < 10 {
					value = (value << 3) + (value << 1) + uint64(digit)
					foundValue = true
				} else if eol = (c == '\n'); eol || isWhitespace[c] {
					done = true
				} else {
					return fmt.Errorf(
						"%s#%d: %q: `%c' not a valid digit",
						interrupts.path, lineNum, getCurrentLine(buf, startLine), buf[pos],
					)
				}
			}
			if foundValue {
				counters[counterIndex] = value
				counterIndex++
			}
		}
		// Enough columns?
		if counterIndex < numCounters {
			return fmt.Errorf(
				"%s#%d: %q: missing IRQs: want: %d, got: %d",
				interrupts.path, lineNum, getCurrentLine(buf, startLine), numCounters, counterIndex,
			)
		}

		// Info:
		irqInfo := info.IrqInfo[irq]
		if irqInfo == nil {
			irqInfo = &InterruptsIrqInfo{}
			info.IrqInfo[irq] = irqInfo
		}

		// Handle description, that's applicable for numerical IRQ's only:
		if irqIsNum {
			var infoLine []byte
			for ; pos < l && isWhitespace[buf[pos]]; pos++ {
			}
			startInfo := pos
			for ; !eol && pos < l; pos++ {
				eol = (buf[pos] == '\n')
			}
			if eol {
				infoLine = buf[startInfo : pos-1]
			} else {
				infoLine = buf[startInfo:pos]
			}

			if irqInfo.Changed = !bytes.Equal(irqInfo.infoLine, infoLine); irqInfo.Changed {
				irqInfo.update(infoLine)
				info.Changed = true
			}
		}

		// Locate EOL:
		for ; !eol && pos < l; pos++ {
			eol = (buf[pos] == '\n')
		}

		// Mark IRQ as found at this scan:
		irqInfo.scanNum = scanNum
	}

	// Cleanup IRQs no longer in use, if any:
	for irq, irqInfo := range info.IrqInfo {
		if irqInfo.scanNum != scanNum {
			delete(interrupts.Counters, irq)
			delete(info.IrqInfo, irq)
		}
	}
	info.scanNum = scanNum

	return nil
}
