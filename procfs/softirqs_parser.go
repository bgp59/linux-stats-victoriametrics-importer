// parser for /proc/softirqs

package procfs

import (
	"bytes"
	"fmt"
	"path"
)

type SoftirqsIrq struct {
	// IRQ counters:
	Counters []uint64
	// The scan# where this IRQ was found, used for removing out of scope IRQs,
	// see scanNum in Softirqs.
	scanNum int
}

type Softirqs struct {
	// IRQs:
	Irq map[string]*SoftirqsIrq

	// The CPU#NN heading; presently softirqs implementation uses all possible
	// CPU's (see:
	// https://github.com/torvalds/linux/blob/d2f51b3516dade79269ff45eae2a7668ae711b25/fs/proc/softirqs.c#L22
	// ) but to future proof for different handling of CPU Hot Plug (CPUHP),
	// maintain a mapping from col# to CPU#; set to nil if no mapping is
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

var softirqsReadFileBufPool = ReadFileBufPoolReadUnbound

func SoftirqsPath(procfsRoot string) string {
	return path.Join(procfsRoot, "softirqs")
}

func NewSoftirqs(procfsRoot string) *Softirqs {
	return &Softirqs{
		Irq:  make(map[string]*SoftirqsIrq),
		path: SoftirqsPath(procfsRoot),
	}
}

func (softirqs *Softirqs) Clone(full bool) *Softirqs {
	newSoftirqs := &Softirqs{
		path:              softirqs.path,
		IndexToCpuChanged: softirqs.IndexToCpuChanged,
		numCounters:       softirqs.numCounters,
		scanNum:           softirqs.scanNum,
	}
	if softirqs.CounterIndexToCpuNum != nil {
		newSoftirqs.CounterIndexToCpuNum = make([]int, len(softirqs.CounterIndexToCpuNum))
		copy(newSoftirqs.CounterIndexToCpuNum, softirqs.CounterIndexToCpuNum)
	}
	if softirqs.Irq != nil {
		newSoftirqs.Irq = make(map[string]*SoftirqsIrq)
		for irq, softirqsIrq := range softirqs.Irq {
			newSoftirqsIrq := &SoftirqsIrq{
				scanNum: softirqsIrq.scanNum,
			}
			if softirqsIrq.Counters != nil {
				newSoftirqsIrq.Counters = make([]uint64, len(softirqsIrq.Counters))
				if full {
					copy(newSoftirqsIrq.Counters, softirqsIrq.Counters)
				}
			}
			newSoftirqs.Irq[irq] = newSoftirqsIrq
		}
	}
	if softirqs.cpuHeaderLine != nil {
		newSoftirqs.cpuHeaderLine = make([]byte, len(softirqs.cpuHeaderLine))
		copy(newSoftirqs.cpuHeaderLine, softirqs.cpuHeaderLine)
	}
	return newSoftirqs
}

func (softirqs *Softirqs) updateCounterIndexToCpuNumMap(cpuHeaderLine []byte) error {
	needsCounterIndexToCpuNumMap := false
	fields := bytes.Fields(cpuHeaderLine)
	softirqs.numCounters = len(fields)
	counterIndexToCpuNum := make([]int, softirqs.numCounters)
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
		softirqs.CounterIndexToCpuNum = counterIndexToCpuNum
	} else {
		softirqs.CounterIndexToCpuNum = nil
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

	scanNum := softirqs.scanNum + 1
	numCounters := softirqs.numCounters
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
				softirqs.IndexToCpuChanged = true
				numCounters = softirqs.numCounters
			} else {
				softirqs.IndexToCpuChanged = false
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
		var counters []uint64
		softirqsIrq := softirqs.Irq[irq]
		if softirqsIrq == nil {
			softirqsIrq = &SoftirqsIrq{
				Counters: make([]uint64, numCounters),
			}
			softirqs.Irq[irq] = softirqsIrq
			counters = softirqsIrq.Counters
		} else {
			counters = softirqsIrq.Counters
			if cap(counters) < numCounters {
				counters = make([]uint64, numCounters)
				softirqsIrq.Counters = counters
			} else if len(counters) != numCounters {
				counters = counters[:numCounters]
				softirqsIrq.Counters = counters
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
		if counterIndex < numCounters {
			return fmt.Errorf(
				"%s#%d: %q: missing IRQs: want: %d, got: %d",
				softirqs.path, lineNum, getCurrentLine(buf, startLine), numCounters, counterIndex,
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
		softirqsIrq.scanNum = scanNum
	}

	// Cleanup IRQs no longer in use, if any:
	for irq, softirqsIrq := range softirqs.Irq {
		if softirqsIrq.scanNum != scanNum {
			delete(softirqs.Irq, irq)
		}
	}
	softirqs.scanNum = scanNum

	return nil
}
