// parser for /proc/interrupts

package procfs

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
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
	// The following are populated only for numerical IRQs:
	Controller  string
	HWInterrupt string
	Devices     []string
	// Whether it was changed in the current scan or not:
	Changed bool
	// Parsing the info columns is expensive and that information doesn't
	// change very often, if at all. Cache the part of the line used for it and
	// reuse the description if the former is unchanged.
	irqInfo []byte
}

type Interrupts struct {
	// Maintain a mapping from col# to CPU#. If the mapping is nil, then it
	// means that CPU#NN was in column index NN.
	ColIndexToCpuNum []int
	// Cache the line used for building the mapping above; if the line is
	// unchanged from the previous run then the mapping is still valid.
	CpuNumLine string
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
}

var interruptsLineSeparators = [256]bool{' ': true, '\t': true}

func NewInterrupts(procfsRoot string) *Interrupts {
	return &Interrupts{
		Irq:        make(map[string][]uint64),
		irqScanNum: map[string]int{},
		path:       path.Join(procfsRoot, "interrupts"),
	}
}

func (interrupts *Interrupts) Clone(full bool) *Interrupts {
	newInterrupts := &Interrupts{
		CpuNumLine:  interrupts.CpuNumLine,
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
		var newDevices []string
		if description.Devices != nil {
			devices := description.Devices
			newDevices = make([]string, len(devices))
			copy(newDevices, devices)
		}
		newDescription := &InterruptDescription{
			Controller:  description.Controller,
			HWInterrupt: description.HWInterrupt,
			Devices:     newDevices,
			Changed:     description.Changed,
		}
		if description.irqInfo != nil {
			newDescription.irqInfo = make([]byte, len(description.irqInfo))
			copy(newDescription.irqInfo, description.irqInfo)
		}
		newInterrupts.Description[irq] = newDescription
	}

	if full {
		for irq, scanNum := range interrupts.irqScanNum {
			newInterrupts.irqScanNum[irq] = scanNum
		}
		newInterrupts.scanNum = interrupts.scanNum
	}

	return newInterrupts
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
		// Header, look for changes in CPU#NN cols:
		if lineNum == 1 {
			line := scanner.Text()
			if line != interrupts.CpuNumLine {
				needsCpuNumMap := false
				fields := strings.Fields(line)
				interrupts.NumCpus = len(fields)
				numCpus = interrupts.NumCpus
				colIndexToCpuNum := make([]int, numCpus)
				for index, cpu := range fields {
					if len(cpu) <= 3 {
						return fmt.Errorf(
							"%s: line# %d: %s: invalid cpu spec",
							interrupts.path, lineNum, line,
						)
					}
					cpuNum, err := strconv.Atoi(cpu[3:])
					if err != nil {
						return fmt.Errorf(
							"%s: line# %d: %s: %v",
							interrupts.path, lineNum, line, err,
						)
					}
					colIndexToCpuNum[index] = cpuNum
					if index != cpuNum {
						needsCpuNumMap = true
					}
				}
				interrupts.CpuNumLine = line
				if needsCpuNumMap {
					interrupts.ColIndexToCpuNum = colIndexToCpuNum
				} else {
					interrupts.ColIndexToCpuNum = nil
				}
			}
			continue
		}

		// IRQ line, handle it as bytes to parse it in a single pass:
		line := scanner.Bytes()
		l := len(line)
		pos := 0

		// Parse `   IRQ:':
		for ; pos < l && interruptsLineSeparators[line[pos]]; pos++ {
		}
		// Silently ignore empty line; there shouldn't be any but then they do no harm either:
		if pos == l {
			continue
		}
		irq, start, irqIsNum := "", pos, true
		for ; pos < l; pos++ {
			c := line[pos]
			if c == ':' {
				end := pos
				// Strip ending spaces (between IRQ and `:', that is), if any:
				for ; start < end && interruptsLineSeparators[line[end-1]]; end-- {
				}
				if start < end {
					irq = string(line[start:end])
				}
				pos++
				break
			}
			if c < '0' || c > '9' {
				irqIsNum = false
			}
		}
		if irq == "" {
			return fmt.Errorf(
				"%s: line# %d: %s: missing IRQ",
				interrupts.path, lineNum, line,
			)
		}

		// Parse ` NNNN NNN ... NNN' interrupt counters:
		perCpuIrqCounter := interrupts.Irq[irq]
		if len(perCpuIrqCounter) < numCpus {
			perCpuIrqCounter = make([]uint64, numCpus)
			interrupts.Irq[irq] = perCpuIrqCounter
		}

		counterIndex := 0
		for pos < l && counterIndex < numCpus {
			// Locate next NNNN:
			for ; pos < l && interruptsLineSeparators[line[pos]]; pos++ {
			}

			for counter := uint64(0); pos < l; pos++ {
				c := line[pos]
				if interruptsLineSeparators[c] {
					perCpuIrqCounter[counterIndex] = counter
					counterIndex++
					pos++
					break
				}
				if '0' <= c && c <= '9' {
					counter = counter*10 + uint64(c-'0')
				} else {
					return fmt.Errorf(
						"%s: line# %d: %s: invalid number for counter index %d",
						interrupts.path, lineNum, line, counterIndex,
					)
				}
			}
		}
		if counterIndex < numCpus {
			return fmt.Errorf(
				"%s: line# %d: %s: invalid number of counters %d (< %d)",
				interrupts.path, lineNum, line, counterIndex, numCpus,
			)
		}

		// Handle description, applicable for numerical IRQs only:
		if irqIsNum {
			for ; pos < l && interruptsLineSeparators[line[pos]]; pos++ {
			}
			irqInfo := line[pos:]
			description := interrupts.Description[irq]
			if description == nil {
				description = &InterruptDescription{}
				interrupts.Description[irq] = description
			}

			description.Changed = !bytes.Equal(description.irqInfo, irqInfo)
			if description.Changed {
				description.irqInfo = make([]byte, len(irqInfo))
				copy(description.irqInfo, irqInfo)
				l := len(irqInfo)

				start, pos := 0, 0
				for ; pos < l && !interruptsLineSeparators[irqInfo[pos]]; pos++ {
				}
				description.Controller = string(irqInfo[start:pos])

				for ; pos < l && interruptsLineSeparators[irqInfo[pos]]; pos++ {
				}
				start = pos
				for ; pos < l && !interruptsLineSeparators[irqInfo[pos]]; pos++ {
				}
				description.HWInterrupt = string(irqInfo[start:pos])

				devices := make([]string, 0)
				for pos < l {
					for ; pos < l && interruptsLineSeparators[irqInfo[pos]]; pos++ {
					}
					start = pos
					for ; pos < l && irqInfo[pos] != ','; pos++ {
					}
					// Strip ending spaces (directly preceding `,', that is), if any:
					end := pos
					for ; start < end && interruptsLineSeparators[irqInfo[end-1]]; end-- {
					}
					if start < end {
						devices = append(devices, string(irqInfo[start:end]))
					}
				}
				description.Devices = devices
			}
		}
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
