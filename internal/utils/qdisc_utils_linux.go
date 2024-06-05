// Qdisc stats

//go:build linux

package utils

import (
	"github.com/ema/qdisc"
)

var QdiscAvail = true

func (uQdiscInfo *UtilsQdiscInfo) Get() error {
	qdiscInfo, err := qdisc.Get()
	if err != nil {
		return err
	}

	scanNum := uQdiscInfo.scanNum + 1
	for _, qdiscIfInfo := range qdiscInfo {
		uQdiscIfInfo := uQdiscInfo.Info[qdiscIfInfo.IfaceName]
		if uQdiscIfInfo == nil {
			uQdiscIfInfo = &UtilsQdiscIfInfo{
				Kind:    qdiscIfInfo.Kind,
				scanNum: scanNum,
			}
			uQdiscInfo.Info[qdiscIfInfo.IfaceName] = uQdiscIfInfo
		} else {
			uQdiscIfInfo.scanNum = scanNum
		}
		uQdiscIfInfo.Uint32[QDISC_PARENT] = qdiscIfInfo.Parent
		uQdiscIfInfo.Uint32[QDISC_HANDLE] = qdiscIfInfo.Handle
		uQdiscIfInfo.Uint32[QDISC_PACKETS] = qdiscIfInfo.Packets
		uQdiscIfInfo.Uint32[QDISC_DROPS] = qdiscIfInfo.Drops
		uQdiscIfInfo.Uint32[QDISC_REQUEUES] = qdiscIfInfo.Requeues
		uQdiscIfInfo.Uint32[QDISC_OVERLIMITS] = qdiscIfInfo.Overlimits
		uQdiscIfInfo.Uint32[QDISC_QLEN] = qdiscIfInfo.Qlen
		uQdiscIfInfo.Uint32[QDISC_BACKLOG] = qdiscIfInfo.Backlog

		uQdiscIfInfo.Uint64[QDISC_BYTES] = qdiscIfInfo.Bytes
		uQdiscIfInfo.Uint64[QDISC_GCFLOWS] = qdiscIfInfo.GcFlows
		uQdiscIfInfo.Uint64[QDISC_THROTTLED] = qdiscIfInfo.Throttled
		uQdiscIfInfo.Uint64[QDISC_FLOWSPLIMIT] = qdiscIfInfo.FlowsPlimit
	}

	for ifName, uQdiscIfInfo := range uQdiscInfo.Info {
		if uQdiscIfInfo.scanNum != scanNum {
			delete(uQdiscInfo.Info, ifName)
		}
	}

	uQdiscInfo.scanNum = scanNum
	return nil
}
