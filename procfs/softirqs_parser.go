// parser for /proc/softirqs

package procfs

import (
	"bytes"
	"fmt"
	"path"
)

type Softirqs struct {
	// IRQs:
	Counters map[string][]uint64

	// The CPU#NN heading; presently softirqs implementation uses all possible
	// CPU's (see:
	// https://github.com/torvalds/linux/blob/d2f51b3516dade79269ff45eae2a7668ae711b25/fs/proc/softirqs.c#L22
	// ) but to future proof for different handling of CPU Hot Plug (CPUHP),
	// maintain a mapping from col# to CPU#; set to nil if no mapping is
	// needed, i.e. counter# == CPU#:
	CpuList []int
	// Whether the mapping has changed in the current scan or not:
	CpuListChanged bool
	// The number of counters per line; this is needed if CpuList
	// is nil, since it cannot be inferred:
	NumCounters int

	// The path file to  read:
	path string
	// Cache the line used for building the counter# -> CPU# mappings above; if
	// the line is unchanged from the previous run then the mapping is still
	// valid.
	cpuHeaderLine []byte
	// Each scan has a different scan# from the previous one. IRQ's not
	// associated with the most recent scan will be removed:
	irqScanNum map[string]int
	scanNum    int
}

var softirqsReadFileBufPool = ReadFileBufPoolReadUnbound

func SoftirqsPath(procfsRoot string) string {
	return path.Join(procfsRoot, "softirqs")
}

func NewSoftirqs(procfsRoot string) *Softirqs {
	return &Softirqs{
		Counters:   make(map[string][]uint64),
		path:       SoftirqsPath(procfsRoot),
		irqScanNum: make(map[string]int),
	}
}

func (softirqs *Softirqs) Clone(full bool) *Softirqs {
	newSoftirqs := &Softirqs{
		path:           softirqs.path,
		CpuListChanged: softirqs.CpuListChanged,
		NumCounters:    softirqs.NumCounters,
		scanNum:        softirqs.scanNum,
	}
	if softirqs.CpuList != nil {
		newSoftirqs.CpuList = make([]int, len(softirqs.CpuList))
		copy(newSoftirqs.CpuList, softirqs.CpuList)
	}
	if softirqs.Counters != nil {
		newSoftirqs.Counters = make(map[string][]uint64)
		for irq, counters := range softirqs.Counters {
			newSoftirqs.Counters[irq] = make([]uint64, len(counters))
			if full {
				copy(newSoftirqs.Counters[irq], counters)
			}
		}
	}
	if softirqs.cpuHeaderLine != nil {
		newSoftirqs.cpuHeaderLine = make([]byte, len(softirqs.cpuHeaderLine))
		copy(newSoftirqs.cpuHeaderLine, softirqs.cpuHeaderLine)
	}
	if softirqs.irqScanNum != nil {
		newSoftirqs.irqScanNum = make(map[string]int)
		if full {
			for irq, scanNum := range softirqs.irqScanNum {
				newSoftirqs.irqScanNum[irq] = scanNum
			}
		}
	}
	return newSoftirqs
}

func (softirqs *Softirqs) updateCounterIndexToCpuNumMap(cpuHeaderLine []byte) error {
	needsCounterIndexToCpuNumMap := false
	fields := bytes.Fields(cpuHeaderLine)
	softirqs.NumCounters = len(fields)
	counterIndexToCpuNum := make([]int, softirqs.NumCounters)
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
		softirqs.CpuList = counterIndexToCpuNum
	} else {
		softirqs.CpuList = nil
	}
	dstHdrCap, srcHdrLen := cap(softirqs.cpuHeaderLine), len(cpuHeaderLine)
	if dstHdrCap < srcHdrLen {
		softirqs.cpuHeaderLine = make([]byte, srcHdrLen)
		copy(softirqs.cpuHeaderLine, cpuHeaderLine)
	} else {
		softirqs.cpuHeaderLine = softirqs.cpuHeaderLine[:dstHdrCap]
		copy(softirqs.cpuHeaderLine, cpuHeaderLine)
		softirqs.cpuHeaderLine = softirqs.cpuHeaderLine[:srcHdrLen]
	}
	return nil
}

func (softirqs *Softirqs) Parse() error {
	fBuf, err := softirqsReadFileBufPool.ReadFile(softirqs.path)
	defer softirqsReadFileBufPool.ReturnBuf(fBuf)
	if err != nil {
		return err
	}
	buf, l := fBuf.Bytes(), fBuf.Len()

	irqScanNum := softirqs.irqScanNum
	if irqScanNum == nil {
		irqScanNum = make(map[string]int)
		softirqs.irqScanNum = irqScanNum
	}
	scanNum := softirqs.scanNum + 1
	NumCounters := softirqs.NumCounters
	for pos, lineNum := 0, 1; pos < l; lineNum++ {
		// Line starts here:
		startLine, eol := pos, false

		if lineNum == 1 {
			// Look for changes in the CPU header line; update col# to cpu# as
			// needed:
			cpuHeaderLine := softirqs.cpuHeaderLine
			cpuHeaderLineLen := len(cpuHeaderLine)
			for ; pos < l && pos < cpuHeaderLineLen && buf[pos] == cpuHeaderLine[pos]; pos++ {
			}
			if pos != cpuHeaderLineLen || (pos < l && buf[pos] != '\n') {
				// The CPU header has changed:
				for ; pos < l && buf[pos] != '\n'; pos++ {
				}
				err = softirqs.updateCounterIndexToCpuNumMap(buf[0:pos])
				if err != nil {
					return fmt.Errorf(
						"%s#%d: %q: %v",
						softirqs.path, lineNum, getCurrentLine(buf, startLine), err,
					)
				}
				softirqs.CpuListChanged = true
				NumCounters = softirqs.NumCounters
			} else {
				softirqs.CpuListChanged = false
			}
			pos++
			continue
		}

		// IRQ: NN .. NN line:
		for ; pos < l && isWhitespace[buf[pos]]; pos++ {
		}
		irqStart, irqEnd := pos, -1
		for ; !eol && irqEnd < 0 && pos < l; pos++ {
			c := buf[pos]
			if c == ':' {
				irqEnd = pos
			} else {
				eol = (c == '\n')
			}
		}
		if irqEnd < 0 {
			return fmt.Errorf(
				"%s#%d: %q: invalid `SOFTIRQ:'",
				softirqs.path, lineNum, getCurrentLine(buf, startLine),
			)
		}
		irq := string(buf[irqStart:irqEnd])

		// Parse ` NNN NNN ... NNN' softirq counters:
		counters := softirqs.Counters[irq]
		if counters == nil || cap(counters) < NumCounters {
			counters = make([]uint64, NumCounters)
			softirqs.Counters[irq] = counters
		} else if len(counters) > NumCounters {
			counters = counters[:NumCounters]
			softirqs.Counters[irq] = counters
		}

		counterIndex := 0
		for !eol && pos < l && counterIndex < NumCounters {
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
						softirqs.path, lineNum, getCurrentLine(buf, startLine), buf[pos],
					)
				}
			}
			if foundValue {
				counters[counterIndex] = value
				counterIndex++
			}
		}
		// Enough columns?
		if counterIndex < NumCounters {
			return fmt.Errorf(
				"%s#%d: %q: missing IRQs: want: %d, got: %d",
				softirqs.path, lineNum, getCurrentLine(buf, startLine), NumCounters, counterIndex,
			)
		}

		// Locate EOL; only whitespaces are allowed at this point:
		for ; !eol && pos < l; pos++ {
			c := buf[pos]
			if eol = (c == '\n'); !eol && !isWhitespace[c] {
				return fmt.Errorf(
					"%s#%d: %q: %q unexpected content after IRQ counter(s)",
					softirqs.path, lineNum, getCurrentLine(buf, startLine), getCurrentLine(buf, pos),
				)
			}
		}

		// Mark this irq as found during the scan:
		irqScanNum[irq] = scanNum
	}

	// Cleanup IRQs no longer in use, if any:
	for irq, sNum := range irqScanNum {
		if sNum != scanNum {
			delete(softirqs.Counters, irq)
			delete(irqScanNum, irq)
		}
	}
	softirqs.scanNum = scanNum

	return nil
}
