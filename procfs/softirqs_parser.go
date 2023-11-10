// parser for /proc/softirqs

package procfs

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
)

type Softirqs struct {
	// The CPU#NN heading; presently softirqs implementation uses all possible
	// CPU's (see:
	// https://github.com/torvalds/linux/blob/d2f51b3516dade79269ff45eae2a7668ae711b25/fs/proc/softirqs.c#L22
	// ) but to future proof for different handling of CPU Hot Plug (CPUHP),
	// maintain a mapping from col# to CPU#. If the mapping is nil, then it
	// means that CPU#NN was in column index NN.
	ColIndexToCpuNum []int
	// Cache the line used for building the mapping above; if the line is
	// unchanged from the previous run then the mapping is still valid.
	CpuHeaderLine string
	// IRQ -> [N, N, ,,, N] map:
	Irq map[string][]uint64
	// The number of CPUs; the size of per CPU slices may be greater if a CPU
	// "vanishes" due to CPUHP.
	NumCpus int
	// Track IRQs found in the current scan; each scan has a different scan#
	// from the previous one. IRQ's not associated with the most recent scan
	// will be removed:
	irqScanNum map[string]int
	scanNum    int
	// The path file to  read:
	path string
}

func NewSoftirqs(procfsRoot string) *Softirqs {
	return &Softirqs{
		Irq:        make(map[string][]uint64),
		irqScanNum: map[string]int{},
		path:       path.Join(procfsRoot, "softirqs"),
	}
}

func (softirqs *Softirqs) Clone(full bool) *Softirqs {
	newSoftirqs := &Softirqs{
		CpuHeaderLine: softirqs.CpuHeaderLine,
		Irq:           make(map[string][]uint64),
		NumCpus:       softirqs.NumCpus,
		irqScanNum:    map[string]int{},
		path:          softirqs.path,
	}
	if softirqs.ColIndexToCpuNum != nil {
		newSoftirqs.ColIndexToCpuNum = make([]int, len(softirqs.ColIndexToCpuNum))
		copy(newSoftirqs.ColIndexToCpuNum, softirqs.ColIndexToCpuNum)
	}
	for irq, perCpuIrqCounter := range softirqs.Irq {
		newSoftirqs.Irq[irq] = make([]uint64, len(perCpuIrqCounter))
		if full {
			copy(newSoftirqs.Irq[irq], perCpuIrqCounter)
		}
	}
	if full {
		for irq, scanNum := range softirqs.irqScanNum {
			newSoftirqs.irqScanNum[irq] = scanNum
		}
		newSoftirqs.scanNum = softirqs.scanNum
	}
	return newSoftirqs
}

func (softirqs *Softirqs) updateColIndexToCpuNumMap(cpuHeaderLine string) error {
	needsColIndexToCpuNumMap := false
	fields := strings.Fields(cpuHeaderLine)
	softirqs.NumCpus = len(fields)
	colIndexToCpuNum := make([]int, softirqs.NumCpus)
	for index, cpu := range fields {
		if len(cpu) <= 3 {
			return fmt.Errorf("invalid cpu spec")
		}
		cpuNum, err := strconv.Atoi(cpu[3:])
		if err != nil {
			return err
		}
		colIndexToCpuNum[index] = cpuNum
		if index != cpuNum {
			needsColIndexToCpuNumMap = true
		}
	}
	softirqs.CpuHeaderLine = cpuHeaderLine
	if needsColIndexToCpuNumMap {
		softirqs.ColIndexToCpuNum = colIndexToCpuNum
	} else {
		softirqs.ColIndexToCpuNum = nil
	}
	return nil
}

func (softirqs *Softirqs) Parse() error {
	file, err := os.Open(softirqs.path)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(file)
	numCpus := softirqs.NumCpus
	scanNum := softirqs.scanNum + 1
	for lineNum := 1; scanner.Scan(); lineNum++ {
		line := scanner.Text()
		// CPU# header:
		if lineNum == 1 {
			if line != softirqs.CpuHeaderLine {
				err = softirqs.updateColIndexToCpuNumMap(line)
				if err != nil {
					return fmt.Errorf("%s#%d: %q: %v", softirqs.path, lineNum, line, err)
				}
				numCpus = softirqs.NumCpus
			}
			continue
		}

		// IRQ: NN .. NN:
		fields := strings.Fields(line)
		expectedNumFields := numCpus + 1
		if len(fields) != expectedNumFields {
			return fmt.Errorf(
				"%s#%d: %q: field# %d (!= %d)",
				softirqs.path, lineNum, line, len(fields), expectedNumFields,
			)
		}
		irq := fields[0]
		irqLen := len(irq) - 1
		if irqLen < 1 || irq[irqLen] != ':' {
			return fmt.Errorf("%s#%d: %q: invalid `SOFTIRQ:'", softirqs.path, lineNum, line)
		}
		irq = irq[:irqLen]
		perCpuIrqCounter := softirqs.Irq[irq]
		if len(perCpuIrqCounter) < numCpus {
			perCpuIrqCounter = make([]uint64, numCpus)
			softirqs.Irq[irq] = perCpuIrqCounter
		}
		for i := 0; i < numCpus; i++ {
			perCpuIrqCounter[i], err = strconv.ParseUint(fields[i+1], 10, 64)
			if err != nil {
				return fmt.Errorf("%s#%d: %q: %v", softirqs.path, lineNum, line, err)
			}
		}
		softirqs.irqScanNum[irq] = scanNum
	}
	err = scanner.Err()
	if err != nil {
		return fmt.Errorf("%s: %v", softirqs.path, err)
	}

	// Cleanup IRQs no longer in use, if any:
	for irq, irqScanNum := range softirqs.irqScanNum {
		if irqScanNum != scanNum {
			delete(softirqs.Irq, irq)
		}
	}
	softirqs.scanNum = scanNum

	return nil
}
