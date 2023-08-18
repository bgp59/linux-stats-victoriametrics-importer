// Copyright 2023 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package datamodels

// A data model is an abstract for the information returned by parsers. Rather
// than using a specialized structure, a data model packs information in slices
// of values paired with slices of corresponding names. The column and/or line
// layout of `/proc' files do not change for the uptime of the host, so the
// names need to be discovered/parsed only once and they can be reused
// subsequently.

// The common use of the parsed data is the generation of metrics in Prometheus
// exposition text format. Generally the values from the the parser are used
// as-is, therefore it is more efficient to return them in string/[]byte format,
// leaving the conversion to numbers to the caller, to be performed as needed.

// Data models are intended to be reusable, in that the parser is provided with
// a reference to the object into which the information will be stored. If the
// reference is nil, then a new object will be allocated and the discovery
// process will take place; this is normally the case for the 1st invocation. If
// the reference is non-nil, then the discovery will be skipped and only the
// data will be parsed.

// In order to increase efficiency, many metrics generators may take a delta
// approach, whereby the current scan values are compared against the ones from
// the previous scan and only the metrics whose values have changed are
// generated. To that end, metrics generators will maintain 2 data model
// container references, previous and current, flipped at every scan; the 2
// containers may share the description part (e.g. the name slices) and for that
// reason the clone method may implement a shallow copy of the latter.

/////////////////////////////////////////////////////////////////////////////////////
// FixedLayoutDataModel: word#k of line#l has the same meaning at every scan.
/////////////////////////////////////////////////////////////////////////////////////

// e.g. /proc/net/snmp:
//
// Ip: Forwarding DefaultTTL InReceives InHdrErrors InAddrErrors ForwDatagrams InUnknownProtos InDiscards InDelivers OutRequests OutDiscards OutNoRoutes ReasmTimeout ReasmReqds ReasmOKs ReasmFails FragOKs FragFails FragCreates
// Ip: 2 64 594223 0 1 0 0 0 593186 547253 20 231 0 0 0 0 0 0 0
// Icmp: InMsgs InErrors InCsumErrors InDestUnreachs InTimeExcds InParmProbs InSrcQuenchs InRedirects InEchos InEchoReps InTimestamps InTimestampReps InAddrMasks InAddrMaskReps OutMsgs OutErrors OutDestUnreachs OutTimeExcds OutParmProbs OutSrcQuenchs OutRedirects OutEchos OutEchoReps OutTimestamps OutTimestampReps OutAddrMasks OutAddrMaskReps
// Icmp: 45 1 0 45 0 0 0 0 0 0 0 0 0 0 50 0 50 0 0 0 0 0 0 0 0 0 0
// IcmpMsg: InType3 OutType3
// IcmpMsg: 45 50
// Tcp: RtoAlgorithm RtoMin RtoMax MaxConn ActiveOpens PassiveOpens AttemptFails EstabResets CurrEstab InSegs OutSegs RetransSegs InErrs OutRsts InCsumErrors
// Tcp: 1 200 120000 -1 1103 9 8 51 15 653161 594855 348 98 1038 0
// Udp: InDatagrams NoPorts InErrors OutDatagrams RcvbufErrors SndbufErrors InCsumErrors IgnoredMulti
// Udp: 10179 50 0 9846 0 0 0 58
// UdpLite: InDatagrams NoPorts InErrors OutDatagrams RcvbufErrors SndbufErrors InCsumErrors IgnoredMulti
// UdpLite: 0 0 0 0 0 0 0 0
//
// The parser should only assume that the information comes in line pairs:
//  PROTO: HEADING#1 HEADING#2 ...
//  PROTO: VAL#1 VAL#2 ...
// and not assume that the 3rd word of the 2nd line is IP DefaultTTL.

type IndexRange struct {
	Start, End int
}

type FixedLayoutDataModel struct {
	// All the values:
	Values []string
	// The unique names for the values, as a parallel array: Names[i] is the
	// name for Values[i]:
	Names []string
	// Grouping for the values, a mapping from group name -> list of index ranges
	// belonging to the group:
	Groups map[string][]IndexRange
	// Name to index map, for GetByName method. It will be populated at the 1st
	// use (JIT, that is):
	nameToIndex map[string]int
}

// Clone method, useful for seeding a container from a previous one: separate
// data and shared names:
func (fldm *FixedLayoutDataModel) Clone() *FixedLayoutDataModel {
	newFLDM := &FixedLayoutDataModel{
		Names:       fldm.Names,
		Groups:      fldm.Groups,
		nameToIndex: fldm.nameToIndex,
	}
	if fldm.Values != nil {
		newFLDM.Values = make([]string, len(fldm.Values), cap(fldm.Values))
	}
	return newFLDM
}

// Value by name:
func (fldm *FixedLayoutDataModel) ValueByName(name string) (string, bool) {
	nameToIndex := fldm.nameToIndex
	if nameToIndex == nil {
		nameToIndex = make(map[string]int)
		for i, name := range fldm.Names {
			nameToIndex[name] = i
		}
		fldm.nameToIndex = nameToIndex
	}
	index, ok := nameToIndex[name]
	if ok {
		return fldm.Values[index], true
	}
	return "", false
}

/////////////////////////////////////////////////////////////////////////////////////
// FixedColumnDataModel: key col#0 col#1 ... col#N-1
/////////////////////////////////////////////////////////////////////////////////////

// e.g. /proc/net/dev:
//
// Inter-|   Receive                                                |  Transmit
//  face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
//     lo:    6740      68    0    0    0     0          0         0     6740      68    0    0    0     0       0          0
//   eth0: 1936365    7267    0    0    0     0          0         0 14322183    7122    0    0    0     0       0          0
//
// The parser will assume that the 2ns line describe the columns but not the
// actual number and/or order, the latter will be discovered.

// The information associated with a key:
type FixedColumnData struct {
	// The actual values for a given key, as a parallel array: Columns[i] is the
	// name for Values[i]:
	Values []string
	// Certain keys may no longer appear in the /proc file in subsequent scans
	// (e.g. the interface was removed). To identify such keys, a scan ID is
	// used and updated if the key was found in the current scan:
	ScanID int
}

type FixedColumnDataModel struct {
	// The values for as given key:
	Data map[string]*FixedColumnData
	// The unique names for the columns, as a parallel array: Columns[i] is the
	// name for Data[key].Values[i]:
	Columns []string
	// The scan ID for the current scan, it will be copied into Data[key] for
	// all current keys. The condition Data[key].ScanID != ScanID can be used to
	// detect out-of-scope keys:
	ScanID int
	// Additional information associated w/ the key, e.g. /proc/interrupts
	// provides interrupt type and device columns:
	Info map[string]any
	// Column name  to index map, for GetByName method. It will be populated at the 1st
	// use (JIT, that is):
	columnToIndex map[string]int
	// For parser use:
	parserUse any
}

// Clone method, useful for seeding a container from a previous one: separate
// data and shared names and info:
func (fcdm *FixedColumnDataModel) Clone() *FixedColumnDataModel {
	newFCDM := &FixedColumnDataModel{
		Columns:       fcdm.Columns,
		Info:          fcdm.Info,
		columnToIndex: fcdm.columnToIndex,
		parserUse:     fcdm.parserUse,
	}
	if fcdm.Data != nil {
		newFCDM.Data = make(map[string]*FixedColumnData)
		for key, data := range fcdm.Data {
			if data.Values != nil {
				newFCDM.Data[key] = &FixedColumnData{
					Values: make([]string, len(data.Values), cap(data.Values)),
				}
			} else {
				newFCDM.Data[key] = &FixedColumnData{}
			}
		}
	}
	return newFCDM
}

// Value by column name:
func (fcdm *FixedColumnDataModel) ValueByColumn(key, col string) (string, bool) {
	columnToIndex := fcdm.columnToIndex
	if columnToIndex == nil {
		columnToIndex = make(map[string]int)
		for i, col := range fcdm.Columns {
			columnToIndex[col] = i
		}
		fcdm.columnToIndex = columnToIndex
	}
	data, ok := fcdm.Data[key]
	if !ok {
		return "", false
	}
	values := data.Values
	if values == nil {
		return "", false
	}
	index, ok := columnToIndex[col]
	if !ok || len(values) < index {
		return "", false
	}
	return values[index], true
}
