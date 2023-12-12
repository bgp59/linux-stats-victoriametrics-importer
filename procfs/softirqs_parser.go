// parser for /proc/softirqs

package procfs

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path"
)

type Softirqs struct {
	// The CPU#NN heading; presently softirqs implementation uses all possible
	// CPU's (see:
	// https://github.com/torvalds/linux/blob/d2f51b3516dade79269ff45eae2a7668ae711b25/fs/proc/softirqs.c#L22
	// ) but to future proof for different handling of CPU Hot Plug (CPUHP),
	// maintain a mapping from col# to CPU#. If the mapping is nil, then it
	// means that CPU#NN was in column index NN.
	ColIndexToCpuNum []int
	// The number of CPU's, needed in case ColIndexToCpuNum is nil:
	NumCpus int
	// IRQ -> [N, N, ,,, N] map:
	Irq map[string][]uint64
	// Cache the line used for building the ColIndexToCpuNum mapping; if the
	// line is unchanged from the previous run then the mapping is still valid.
	cpuHeaderLine []byte
	// IRQs may disappear from one scan to the next one keep track of IRQs found
	// in the current scan; each scan has a different scan# from the previous
	// one. IRQ's not associated with the most recent scan will be removed from
	// Irq map:
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
		Irq:        make(map[string][]uint64),
		NumCpus:    softirqs.NumCpus,
		irqScanNum: map[string]int{},
		path:       softirqs.path,
	}
	if softirqs.ColIndexToCpuNum != nil {
		newSoftirqs.ColIndexToCpuNum = make([]int, len(softirqs.ColIndexToCpuNum))
		copy(newSoftirqs.ColIndexToCpuNum, softirqs.ColIndexToCpuNum)
	}
	if softirqs.cpuHeaderLine != nil {
		newSoftirqs.cpuHeaderLine = make([]byte, len(softirqs.cpuHeaderLine))
		copy(newSoftirqs.cpuHeaderLine, softirqs.cpuHeaderLine)
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

func (softirqs *Softirqs) updateColIndexToCpuNumMap(cpuHeaderLine []byte) error {
	needsColIndexToCpuNumMap := false
	fields := bytes.Fields(cpuHeaderLine)
	softirqs.NumCpus = len(fields)
	colIndexToCpuNum := make([]int, softirqs.NumCpus)
	for index, cpuSpec := range fields {
		if len(cpuSpec) <= 3 {
			return fmt.Errorf("invalid cpu spec")
		}
		cpuNum := 0
		for pos := 3; pos < len(cpuSpec); pos++ {
			if digit := cpuSpec[pos] - '0'; digit < 10 {
				cpuNum = (cpuNum << 3) + (cpuNum << 1) + int(digit)
			} else {
				return fmt.Errorf(
					"%q: invalid CPUNN, `%c' not a valid digit",
					string(cpuSpec), cpuSpec[pos],
				)
			}
		}
		colIndexToCpuNum[index] = cpuNum
		if index != cpuNum {
			needsColIndexToCpuNumMap = true
		}
	}
	softirqs.cpuHeaderLine = make([]byte, len(cpuHeaderLine))
	copy(softirqs.cpuHeaderLine, cpuHeaderLine)
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
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanNum := softirqs.scanNum + 1
	numCpus := 0
	for lineNum := 1; scanner.Scan(); lineNum++ {
		line := scanner.Bytes()
		if lineNum == 1 {
			if !bytes.Equal(line, softirqs.cpuHeaderLine) {
				err = softirqs.updateColIndexToCpuNumMap(line)
				if err != nil {
					return fmt.Errorf("%s#%d: %q: %v", softirqs.path, lineNum, string(line), err)
				}
			}
			numCpus = softirqs.NumCpus
			continue
		}

		// IRQ: NN .. NN:
		pos, l := 0, len(line)
		for ; pos < l && isWhitespace[line[pos]]; pos++ {
		}
		irqStart := pos
		for ; pos < l && line[pos] != ':'; pos++ {
		}
		if irqStart >= pos {
			return fmt.Errorf("%s#%d: %q: invalid `SOFTIRQ:'", softirqs.path, lineNum, string(line))
		}
		irq := string(line[irqStart:pos])
		pos++

		perCpuIrqCounter := softirqs.Irq[irq]
		if len(perCpuIrqCounter) < numCpus {
			perCpuIrqCounter = make([]uint64, numCpus)
			softirqs.Irq[irq] = perCpuIrqCounter
		}

		irqIndex := 0
		for pos < l && irqIndex < numCpus {
			for ; pos < l && isWhitespace[line[pos]]; pos++ {
			}
			value, done := uint64(0), false
			for ; !done && pos < l; pos++ {
				c := line[pos]
				if digit := c - '0'; digit < 10 {
					value = (value << 3) + (value << 1) + uint64(digit)
				} else if isWhitespace[c] {
					done = true
				} else {
					return fmt.Errorf(
						"%s#%d: %q: `%c' not a valid digit",
						softirqs.path, lineNum, string(line), line[pos],
					)
				}
			}
			perCpuIrqCounter[irqIndex] = value
			irqIndex++
		}

		if irqIndex < numCpus {
			return fmt.Errorf(
				"%s#%d: %q: missing IRQs: want: %d, got: %d",
				softirqs.path, lineNum, string(line), numCpus, irqIndex,
			)
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
