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
	// ) but to future proof for different handling of CPU Hot Plug, maintain a
	// mapping from col# to CPU#. If the mapping is nil, then it means that
	// CPU#NN was in column index NN.
	ColIndexToCpuNum []int
	// Cache the line used for building the mapping above; if the line is
	// unchanged from the previous run then the mapping is still valid.
	CpuNumLine string
	// IRQ -> [N, N, ,,, N] map:
	Irq map[string][]uint64
	// The number of fields expected for each IRQ line (the number of cols in
	// the 1st line + 1); this should be store explicitly since ColIndexToCpuNum
	// may be nil.
	expectedNumFields int
	// Track IRQs found in the current scan; each scan has a scan# different
	// from the previous one. IRQ's not associated with the most recent scan
	// will be removed:
	irqScanNum map[string]int
	scanNum    int
	// The path file to  read:
	path string
}

func NewSoftirq(procfsRoot string) *Softirqs {
	return &Softirqs{
		Irq:        make(map[string][]uint64),
		irqScanNum: map[string]int{},
		path:       path.Join(procfsRoot, "softirqs"),
	}
}

func (softirqs *Softirqs) Clone(full bool) *Softirqs {
	newSoftirqs := &Softirqs{
		CpuNumLine:        softirqs.CpuNumLine,
		Irq:               make(map[string][]uint64),
		expectedNumFields: softirqs.expectedNumFields,
		irqScanNum:        map[string]int{},
		path:              softirqs.path,
	}
	if softirqs.ColIndexToCpuNum != nil {
		newSoftirqs.ColIndexToCpuNum = make([]int, len(softirqs.ColIndexToCpuNum))
		copy(newSoftirqs.ColIndexToCpuNum, softirqs.ColIndexToCpuNum)
	}
	for irq, perCpuIrqCount := range softirqs.Irq {
		newSoftirqs.Irq[irq] = make([]uint64, len(perCpuIrqCount))
		if full {
			copy(newSoftirqs.Irq[irq], perCpuIrqCount)
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

func (softirqs *Softirqs) Parse() error {
	file, err := os.Open(softirqs.path)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(file)
	expectedNumFields, perCpuIrqCountLen := 0, 0
	scanNum := softirqs.scanNum + 1
	for lineNum := 1; scanner.Scan(); lineNum++ {
		line := scanner.Text()
		// CPU# header:
		if lineNum == 1 {
			if line != softirqs.CpuNumLine {
				needsCpuNumMap := false
				fields := strings.Fields(line)
				softirqs.expectedNumFields = len(fields) + 1
				colIndexToCpuNum := make([]int, len(fields))
				for index, cpu := range fields {
					cpuNum, err := strconv.Atoi(cpu[3:])
					if err != nil {
						return fmt.Errorf(
							"%s: line# %d: %s: %v",
							softirqs.path, lineNum, line, err,
						)
					}
					colIndexToCpuNum[index] = cpuNum
					if index != cpuNum {
						needsCpuNumMap = true
					}
				}
				softirqs.CpuNumLine = line
				if needsCpuNumMap {
					softirqs.ColIndexToCpuNum = colIndexToCpuNum
				} else {
					softirqs.ColIndexToCpuNum = nil
				}
			}
			expectedNumFields = softirqs.expectedNumFields
			perCpuIrqCountLen = expectedNumFields - 1
			continue
		}

		// IRQ: NN .. NN:
		fields := strings.Fields(line)
		if len(fields) != expectedNumFields {
			return fmt.Errorf(
				"%s: line# %d: %s: field# %d (!= %d)",
				softirqs.path, lineNum, line, len(fields), expectedNumFields,
			)
		}
		irqLen := len(fields[0]) - 1
		if irqLen < 1 || fields[0][irqLen] != ':' {
			return fmt.Errorf(
				"%s: line# %d: %s: invalid SOFTIRQ",
				softirqs.path, lineNum, line,
			)
		}
		irq := fields[0][:irqLen]
		perCpuIrqCount := softirqs.Irq[irq]
		if len(perCpuIrqCount) < perCpuIrqCountLen {
			perCpuIrqCount = make([]uint64, perCpuIrqCountLen)
			softirqs.Irq[irq] = perCpuIrqCount
		}
		for i := 0; i < perCpuIrqCountLen; i++ {
			perCpuIrqCount[i], err = strconv.ParseUint(fields[i+1], 10, 64)
			if err != nil {
				return fmt.Errorf(
					"%s: line# %d: %s: %v",
					softirqs.path, lineNum, line, err,
				)
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
