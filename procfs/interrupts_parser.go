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

type InterruptDescription struct {
	// The part of the line used for info:
	IrqInfo []byte
	// The following are populated only for numerical IRQs; the offsets are
	// applicable to IrqInfo:
	Controller, HWInterrupt, Devices SliceOffsets
	// Whether it was changed in the current scan or not:
	Changed bool
}

type Interrupts struct {
	// Maintain a mapping from col# to CPU#. If the mapping is nil, then it
	// means that CPU#NN was in col# NN.
	ColIndexToCpuNum []int
	// IRQ -> [N, N, ,,, N] map:
	Irq map[string][]uint64
	// The number of CPUs; the size of per CPU slices may be greater if a CPU
	// "vanishes" due to CPU Hot Plug.
	NumCpus int
	// Info, applicable for numerical interrupts only:
	Description map[string]*InterruptDescription
	// Track IRQs found in the current scan; each scan has a different scan#
	// from the previous one. IRQ's not associated with the most recent scan
	// will be removed:
	irqScanNum map[string]int
	scanNum    int
	// The path file to  read:
	path string
	// Cache the line used for building the mapping above; if the line is
	// unchanged from the previous run then the mapping is still valid.
	cpuHeaderLine []byte
}

var interruptsReadFileBufPool = ReadFileBufPoolReadUnbound

func NewInterrupts(procfsRoot string) *Interrupts {
	return &Interrupts{
		Irq:         map[string][]uint64{},
		Description: map[string]*InterruptDescription{},
		irqScanNum:  map[string]int{},
		path:        path.Join(procfsRoot, "interrupts"),
	}
}

func (interrupts *Interrupts) Clone(full bool) *Interrupts {
	newInterrupts := &Interrupts{
		Irq:         make(map[string][]uint64),
		NumCpus:     interrupts.NumCpus,
		Description: make(map[string]*InterruptDescription),
		irqScanNum:  map[string]int{},
		path:        interrupts.path,
	}

	if interrupts.ColIndexToCpuNum != nil {
		newInterrupts.ColIndexToCpuNum = make([]int, len(interrupts.ColIndexToCpuNum))
		copy(newInterrupts.ColIndexToCpuNum, interrupts.ColIndexToCpuNum)
	}

	for irq, perCpuIrqCounter := range interrupts.Irq {
		newPerCpuIrqCounter := make([]uint64, len(perCpuIrqCounter))
		if full {
			copy(newPerCpuIrqCounter, perCpuIrqCounter)
		}
		newInterrupts.Irq[irq] = newPerCpuIrqCounter
	}

	for irq, description := range interrupts.Description {
		newDescription := &InterruptDescription{
			Controller:  description.Controller,
			HWInterrupt: description.HWInterrupt,
			Devices:     description.Devices,
			Changed:     description.Changed,
		}
		if description.IrqInfo != nil {
			newDescription.IrqInfo = make([]byte, len(description.IrqInfo))
			copy(newDescription.IrqInfo, description.IrqInfo)
		}
		newInterrupts.Description[irq] = newDescription
	}

	if full {
		for irq, scanNum := range interrupts.irqScanNum {
			newInterrupts.irqScanNum[irq] = scanNum
		}
		newInterrupts.scanNum = interrupts.scanNum
	}

	if interrupts.cpuHeaderLine != nil {
		newInterrupts.cpuHeaderLine = make([]byte, len(interrupts.cpuHeaderLine))
		copy(newInterrupts.cpuHeaderLine, interrupts.cpuHeaderLine)
	}

	return newInterrupts
}

func (interrupts *Interrupts) updateColIndexToCpuNumMap(cpuHeaderLine []byte) error {
	needsColIndexToCpuNumMap := false
	fields := bytes.Fields(cpuHeaderLine)
	interrupts.NumCpus = len(fields)
	colIndexToCpuNum := make([]int, interrupts.NumCpus)
	for index, cpu := range fields {
		cpuNum, l := 0, len(cpu)
		if l <= 3 {
			return fmt.Errorf("invalid cpu spec")
		}
		for pos := 3; pos < l; pos++ {
			cpuNum = (cpuNum << 3) + (cpuNum << 1) + int(cpu[pos]-'0')
		}
		colIndexToCpuNum[index] = cpuNum
		if index != cpuNum {
			needsColIndexToCpuNumMap = true
		}
	}
	if needsColIndexToCpuNumMap {
		interrupts.ColIndexToCpuNum = colIndexToCpuNum
	} else {
		interrupts.ColIndexToCpuNum = nil
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
	if needsColIndexToCpuNumMap {
		interrupts.ColIndexToCpuNum = colIndexToCpuNum
	} else {
		interrupts.ColIndexToCpuNum = nil
	}
	return nil
}

func (description *InterruptDescription) update(irqInfo []byte) {
	dstIrqInfoCap, l := cap(description.IrqInfo), len(irqInfo)
	if dstIrqInfoCap < l {
		description.IrqInfo = make([]byte, l)
		copy(description.IrqInfo, irqInfo)
	} else {
		description.IrqInfo = description.IrqInfo[:dstIrqInfoCap]
		copy(description.IrqInfo, irqInfo)
		description.IrqInfo = description.IrqInfo[:l]
	}

	pos := 0
	description.Controller.Start = pos
	for ; pos < l && !isWhitespace[irqInfo[pos]]; pos++ {
	}
	description.Controller.End = pos

	for ; pos < l && isWhitespace[irqInfo[pos]]; pos++ {
	}
	description.HWInterrupt.Start = pos
	for ; pos < l && !isWhitespace[irqInfo[pos]]; pos++ {
	}
	description.HWInterrupt.End = pos

	for ; pos < l && isWhitespace[irqInfo[pos]]; pos++ {
	}
	description.Devices.Start = pos
	// Trim ending white spaces, if any:
	endDevPos := l - 1
	for ; endDevPos >= pos && isWhitespace[irqInfo[endDevPos]]; endDevPos-- {
	}
	description.Devices.End = endDevPos + 1
}

func (interrupts *Interrupts) Parse() error {
	fBuf, err := interruptsReadFileBufPool.ReadFile(interrupts.path)
	defer interruptsReadFileBufPool.ReturnBuf(fBuf)
	if err != nil {
		return err
	}

	buf, l := fBuf.Bytes(), fBuf.Len()

	numCpus := interrupts.NumCpus
	scanNum := interrupts.scanNum + 1
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
				err = interrupts.updateColIndexToCpuNumMap(buf[0:pos])
				if err != nil {
					return fmt.Errorf(
						"%s#%d: %q: %v",
						interrupts.path, lineNum, getCurrentLine(buf, startLine), err,
					)
				}
			}
			numCpus = interrupts.NumCpus
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
		perCpuIrqCounter := interrupts.Irq[irq]
		if cap(perCpuIrqCounter) < numCpus {
			perCpuIrqCounter = make([]uint64, numCpus)
			interrupts.Irq[irq] = perCpuIrqCounter
		} else if len(perCpuIrqCounter) < numCpus {
			perCpuIrqCounter = perCpuIrqCounter[:numCpus]
			interrupts.Irq[irq] = perCpuIrqCounter
		}

		counterIndex := 0
		for !eol && pos < l && counterIndex < numCpus {
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
				perCpuIrqCounter[counterIndex] = value
				counterIndex++
			}
		}
		// Enough columns?
		if counterIndex < numCpus {
			return fmt.Errorf(
				"%s#%d: %q: missing IRQs: want: %d, got: %d",
				interrupts.path, lineNum, getCurrentLine(buf, startLine), numCpus, counterIndex,
			)
		}

		// Handle description, applicable for numerical IRQs only:
		if irqIsNum {
			for ; pos < l && isWhitespace[buf[pos]]; pos++ {
			}
			irqInfoStart := pos

			description := interrupts.Description[irq]
			if description == nil {
				description = &InterruptDescription{}
				interrupts.Description[irq] = description
			}
			irqInfo := description.IrqInfo
			irqInfoLen := len(irqInfo)
			for i := 0; pos < l && i < irqInfoLen && buf[pos] == irqInfo[i]; pos++ {
				i++
			}
			if pos != irqInfoStart+irqInfoLen || (pos < l && buf[pos] != '\n') {
				// The description has changed:
				description.Changed = true
				for ; pos < l && buf[pos] != '\n'; pos++ {
				}
				description.update(buf[irqInfoStart:pos])
			} else {
				description.Changed = false
			}
		}

		// Locate EOL:
		for ; !eol && pos < l; pos++ {
			eol = (buf[pos] == '\n')
		}

		// Mark IRQ as found at this scan:
		interrupts.irqScanNum[irq] = scanNum
	}

	// Cleanup IRQs no longer in use, if any:
	for irq, irqScanNum := range interrupts.irqScanNum {
		if irqScanNum != scanNum {
			delete(interrupts.Irq, irq)
		}
	}
	interrupts.scanNum = scanNum

	return nil
}
