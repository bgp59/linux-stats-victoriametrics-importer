// parser for /proc/interrupts

package procfs

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
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
	Controller, HWInterrupt, Devices ByteSliceOffsets
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

func updateInterruptDescription(description *InterruptDescription, irqInfo []byte) {
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
	file, err := os.Open(interrupts.path)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(file)
	numCpus := interrupts.NumCpus
	scanNum := interrupts.scanNum + 1
	for lineNum := 1; scanner.Scan(); lineNum++ {
		line := scanner.Bytes()

		if lineNum == 1 {
			// CPU header, look for changes in CPU#NN cols:
			if !bytes.Equal(line, interrupts.cpuHeaderLine) {
				err = interrupts.updateColIndexToCpuNumMap(line)
				if err != nil {
					return fmt.Errorf("%s#%d: %q: %v", interrupts.path, lineNum, line, err)
				}
				numCpus = interrupts.NumCpus
			}
			continue
		}

		// IRQ line:
		pos, l := 0, len(line)

		// Parse `   IRQ:':
		for ; pos < l && isWhitespace[line[pos]]; pos++ {
		}
		irqStart, irqEnd, irqIsNum := pos, pos, true
		for ; pos < l; pos++ {
			c := line[pos]
			if c == ':' {
				irqEnd = pos
				pos++
				break
			}
			if c-'0' >= 10 {
				irqIsNum = false
			}
		}
		if irqStart >= irqEnd {
			return fmt.Errorf("%s#%d: %q: missing `IRQ:'", interrupts.path, lineNum, line)
		}
		irq := string(line[irqStart:irqEnd])

		// Parse ` NNNN NNN ... NNN' interrupt counters:
		perCpuIrqCounter := interrupts.Irq[irq]
		if cap(perCpuIrqCounter) < numCpus {
			perCpuIrqCounter = make([]uint64, numCpus)
			interrupts.Irq[irq] = perCpuIrqCounter
		} else if len(perCpuIrqCounter) < numCpus {
			perCpuIrqCounter = perCpuIrqCounter[:numCpus]
			interrupts.Irq[irq] = perCpuIrqCounter
		}

		counterIndex := 0
		for pos < l && counterIndex < numCpus {
			// Locate next NNNN:
			for ; pos < l && isWhitespace[line[pos]]; pos++ {
			}
			startPos, counter := pos, uint64(0)
			for done := false; !done && pos < l; pos++ {
				c := line[pos]
				if digit := c - '0'; digit <= '9' {
					counter = (counter << 3) + (counter << 1) + uint64(digit)
				} else if isWhitespace[c] {
					done = true
				} else {
					return fmt.Errorf(
						"%s#%d: %q: `%c' invalid value for digit",
						interrupts.path, lineNum, line, c,
					)
				}
			}
			if startPos < pos {
				perCpuIrqCounter[counterIndex] = counter
				counterIndex++
			}
		}
		if counterIndex < numCpus {
			return fmt.Errorf(
				"%s#%d: %q: invalid number of counters %d (< %d)",
				interrupts.path, lineNum, line, counterIndex, numCpus,
			)
		}

		// Handle description, applicable for numerical IRQs only:
		if irqIsNum {
			for ; pos < l && isWhitespace[line[pos]]; pos++ {
			}
			irqInfo := line[pos:]
			description := interrupts.Description[irq]
			if description == nil {
				description = &InterruptDescription{}
				interrupts.Description[irq] = description
			}
			description.Changed = !bytes.Equal(description.IrqInfo, irqInfo)
			if description.Changed {
				updateInterruptDescription(description, irqInfo)
			}
		}

		// Mark IRQ as found at this scan:
		interrupts.irqScanNum[irq] = scanNum
	}

	err = scanner.Err()
	if err != nil {
		return fmt.Errorf("%s: %v", interrupts.path, err)
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
