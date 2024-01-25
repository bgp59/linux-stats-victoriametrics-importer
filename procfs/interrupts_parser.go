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

type InterruptsIrq struct {
	// IRQ counters:
	Counters []uint64
	// The following are populated only for numerical IRQs:
	Controller, HWInterrupt, Devices []byte
	// Whether the info has changed in the current scan or not:
	InfoChanged bool
	// Cache line part used to build the info above; if unchanged from the
	// previous scan, the information is still valid:
	info []byte
	// The scan# where this IRQ was found, used for removing out of scope IRQs,
	// see scanNum in Interrupts.
	scanNum int
}

type Interrupts struct {
	// IRQs:
	Irq map[string]*InterruptsIrq
	// Mapping counter# -> CPU#, based on the header line; nil if no mapping is
	// needed, i.e. counter# == CPU#:
	CounterIndexToCpuNum []int
	// Whether the mapping has changed in the current scan or not:
	IndexToCpuChanged bool

	// The path file to  read:
	path string
	// Cache the line used for building the counter# -> CPU# mappings above; if
	// the line is unchanged from the previous run then the mapping is still
	// valid.
	cpuHeaderLine []byte
	// The number of counters per line; this is needed if CounterIndexToCpuNum
	// is nil, since it cannot be inferred:
	numCounters int
	// Each scan has a different scan# from the previous one. IRQ's not
	// associated with the most recent scan will be removed:
	scanNum int
}

var interruptsReadFileBufPool = ReadFileBufPoolReadUnbound

func InterruptsPath(procfsRoot string) string {
	return path.Join(procfsRoot, "interrupts")
}

func NewInterrupts(procfsRoot string) *Interrupts {
	return &Interrupts{
		Irq:  map[string]*InterruptsIrq{},
		path: InterruptsPath(procfsRoot),
	}
}

func (interrupts *Interrupts) Clone(full bool) *Interrupts {
	newInterrupts := &Interrupts{
		path:              interrupts.path,
		IndexToCpuChanged: interrupts.IndexToCpuChanged,
		numCounters:       interrupts.numCounters,
		scanNum:           interrupts.scanNum,
	}
	if interrupts.CounterIndexToCpuNum != nil {
		newInterrupts.CounterIndexToCpuNum = make([]int, len(interrupts.CounterIndexToCpuNum))
		copy(newInterrupts.CounterIndexToCpuNum, interrupts.CounterIndexToCpuNum)
	}
	if interrupts.Irq != nil {
		newInterrupts.Irq = make(map[string]*InterruptsIrq)
		for irq, interruptsIrq := range interrupts.Irq {
			newInterruptsIrq := &InterruptsIrq{
				InfoChanged: interruptsIrq.InfoChanged,
				scanNum:     interruptsIrq.scanNum,
			}
			if interruptsIrq.Counters != nil {
				newInterruptsIrq.Counters = make([]uint64, len(interruptsIrq.Counters))
				if full {
					copy(newInterruptsIrq.Counters, interruptsIrq.Counters)
				}
			}
			if interruptsIrq.info != nil {
				newInterruptsIrq.updateInfo(interruptsIrq.info)
			}
			newInterrupts.Irq[irq] = newInterruptsIrq
		}
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

func (interruptsIrq *InterruptsIrq) updateInfo(info []byte) {
	dstIrqInfoCap, l := cap(interruptsIrq.info), len(info)
	if dstIrqInfoCap < l {
		interruptsIrq.info = make([]byte, l)
		copy(interruptsIrq.info, info)
	} else {
		interruptsIrq.info = interruptsIrq.info[:dstIrqInfoCap]
		copy(interruptsIrq.info, info)
		interruptsIrq.info = interruptsIrq.info[:l]
	}

	pos := 0
	start := pos
	for ; pos < l && !isWhitespace[info[pos]]; pos++ {
	}
	interruptsIrq.Controller = interruptsIrq.info[start:pos]

	for ; pos < l && isWhitespace[info[pos]]; pos++ {
	}
	start = pos
	for ; pos < l && !isWhitespace[info[pos]]; pos++ {
	}
	interruptsIrq.HWInterrupt = interruptsIrq.info[start:pos]

	for ; pos < l && isWhitespace[info[pos]]; pos++ {
	}
	start = pos
	// Trim ending white spaces, if any:
	end := l
	for ; end > pos && isWhitespace[info[end-1]]; end-- {
	}
	interruptsIrq.Devices = interruptsIrq.info[start:end]
}

func (interrupts *Interrupts) Parse() error {
	fBuf, err := interruptsReadFileBufPool.ReadFile(interrupts.path)
	defer interruptsReadFileBufPool.ReturnBuf(fBuf)
	if err != nil {
		return err
	}

	buf, l := fBuf.Bytes(), fBuf.Len()

	scanNum := interrupts.scanNum + 1
	numCounters := interrupts.numCounters
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
				interrupts.IndexToCpuChanged = true
				numCounters = interrupts.numCounters
			} else {
				interrupts.IndexToCpuChanged = false
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
		var counters []uint64
		interruptsIrq := interrupts.Irq[irq]
		if interruptsIrq == nil {
			interruptsIrq = &InterruptsIrq{
				Counters: make([]uint64, numCounters),
			}
			interrupts.Irq[irq] = interruptsIrq
			counters = interruptsIrq.Counters
		} else {
			counters = interruptsIrq.Counters
			if cap(counters) < numCounters {
				counters = make([]uint64, numCounters)
				interruptsIrq.Counters = counters
			} else if len(counters) != numCounters {
				counters = counters[:numCounters]
				interruptsIrq.Counters = counters
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

		// Handle description, applicable for numerical IRQs only:
		if irqIsNum {
			for ; pos < l && isWhitespace[buf[pos]]; pos++ {
			}
			infoStart := pos

			info := interruptsIrq.info
			i, infoLen := 0, len(info)
			for pos < l && i < infoLen && buf[pos] == info[i] {
				pos++
				i++
			}
			if i != infoLen || (pos < l && buf[pos] != '\n') {
				// The description has changed:
				interruptsIrq.InfoChanged = true
				for ; pos < l && buf[pos] != '\n'; pos++ {
				}
				interruptsIrq.updateInfo(buf[infoStart:pos])
			} else {
				interruptsIrq.InfoChanged = false
			}
		}

		// Locate EOL:
		for ; !eol && pos < l; pos++ {
			eol = (buf[pos] == '\n')
		}

		// Mark IRQ as found at this scan:
		interruptsIrq.scanNum = scanNum
	}

	// Cleanup IRQs no longer in use, if any:
	for irq, interruptsIrq := range interrupts.Irq {
		if interruptsIrq.scanNum != scanNum {
			delete(interrupts.Irq, irq)
		}
	}
	interrupts.scanNum = scanNum

	return nil
}
