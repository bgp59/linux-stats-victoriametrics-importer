// qdisc parser a-la tc -s show qdisc

// Based on https://github.com/ema/qdisc, adapted for reusable objects, w/ data
// presented as slices. The latter allows for metrics generation in a loop.

//go:build linux

package qdisc

import (
	"errors"
	"fmt"
	"net"
	"syscall"
	"time"

	"github.com/mdlayher/netlink"
	"github.com/mdlayher/netlink/nlenc"
)

const (
	TCA_UNSPEC = iota
	TCA_KIND
	TCA_OPTIONS
	TCA_STATS
	TCA_XSTATS
	TCA_RATE
	TCA_FCNT
	TCA_STATS2
	TCA_STAB
	// __TCA_MAX
)

const (
	TCA_STATS_UNSPEC = iota
	TCA_STATS_BASIC
	TCA_STATS_RATE_EST
	TCA_STATS_QUEUE
	TCA_STATS_APP
	TCA_STATS_RATE_EST64
	// __TCA_STATS_MAX
)

var QdiscAvailable = true

func (qi *QdiscInfo) parseTCAStats(attr netlink.Attribute) error {
	qi.Uint64[QDISC_BYTES] = nlenc.Uint64(attr.Data[0:8])
	qi.Uint32[QDISC_PACKETS] = nlenc.Uint32(attr.Data[8:12])
	qi.Uint32[QDISC_DROPS] = nlenc.Uint32(attr.Data[12:16])
	qi.Uint32[QDISC_OVERLIMITS] = nlenc.Uint32(attr.Data[16:20])
	qi.Uint32[QDISC_QLEN] = nlenc.Uint32(attr.Data[28:32])
	qi.Uint32[QDISC_BACKLOG] = nlenc.Uint32(attr.Data[32:36])
	return nil
}

func (qi *QdiscInfo) parseTCAStats2(attr netlink.Attribute) error {
	nested, err := netlink.UnmarshalAttributes(attr.Data)
	if err != nil {
		return err
	}

	for _, a := range nested {
		switch a.Type {
		case TCA_STATS_BASIC:
			qi.Uint64[QDISC_BYTES] = nlenc.Uint64(a.Data[0:8])
			qi.Uint32[QDISC_PACKETS] = nlenc.Uint32(a.Data[8:12])
		case TCA_STATS_QUEUE:
			qi.Uint32[QDISC_QLEN] = nlenc.Uint32(a.Data[0:4])
			qi.Uint32[QDISC_BACKLOG] = nlenc.Uint32(a.Data[4:8])
			qi.Uint32[QDISC_DROPS] = nlenc.Uint32(a.Data[8:12])
			qi.Uint32[QDISC_REQUEUES] = nlenc.Uint32(a.Data[12:16])
			qi.Uint32[QDISC_OVERLIMITS] = nlenc.Uint32(a.Data[16:20])
		default:
		}
	}
	return nil
}

func (qs *QdiscStats) ifIndexToNameCacheRefresh() error {
	ifas, err := net.Interfaces()
	if err != nil {
		return err
	}
	qsShared := qs.shared

	if qsShared.ifIndexToNameCache == nil {
		qsShared.ifIndexToNameCache = make(map[uint32]string)
	}
	for _, ifa := range ifas {
		qsShared.ifIndexToNameCache[uint32(ifa.Index)] = ifa.Name
	}
	qsShared.ifIndexToNameCacheLastRefresh = time.Now()
	return nil
}

func (qs *QdiscStats) Parse() error {
	qsShared := qs.shared
	if qsShared.netConn == nil {
		const familyRoute = 0

		c, err := netlink.Dial(familyRoute, nil)
		if err != nil {
			return fmt.Errorf("failed to dial netlink: %v", err)
		}

		if err := c.SetOption(netlink.GetStrictCheck, true); err != nil {
			// silently accept ENOPROTOOPT errors when kernel is not > 4.20
			if !errors.Is(err, syscall.ENOPROTOOPT) {
				return fmt.Errorf("unexpected error trying to set option NETLINK_GET_STRICT_CHK: %v", err)
			}
		}
		qsShared.netConn = c
	}

	if qsShared.netReqMessage == nil {
		qsShared.netReqMessage = &netlink.Message{
			Header: netlink.Header{
				Flags: netlink.Request | netlink.Dump,
				Type:  38, // RTM_GETQDISC
			},
			Data: make([]byte, 20),
		}
	}

	// Perform a request, receive replies, and validate the replies:
	c := qsShared.netConn.(*netlink.Conn)
	msgs, err := c.Execute(*(qsShared.netReqMessage.(*netlink.Message)))
	if err != nil {
		c.Close()
		qsShared.netConn = nil
		return fmt.Errorf("failed to execute netlink request: %v", err)
	}

	scanNum := *qsShared.scanNum + 1

	ifIndexToNameCacheRefreshed := false
	if qsShared.ifIndexToNameCache == nil ||
		time.Since(qsShared.ifIndexToNameCacheLastRefresh) >= QDISC_IF_INDEX_TO_NAME_CACHE_REFRESH_INTERVAL {
		err = qs.ifIndexToNameCacheRefresh()
		if err != nil {
			return err
		}
		ifIndexToNameCacheRefreshed = true
	}

	for _, msg := range msgs {
		var qiKey QdiscInfoKey
		if len(msg.Data) < 20 {
			return fmt.Errorf("short message, len=%d < 20", len(msg.Data))
		}
		qiKey.IfIndex = nlenc.Uint32(msg.Data[4:8])
		qiKey.Handle = nlenc.Uint32(msg.Data[8:12])
		qi := qs.Info[qiKey]
		if qi == nil {
			ifName := qsShared.ifIndexToNameCache[qiKey.IfIndex]
			if ifName == "" && !ifIndexToNameCacheRefreshed {
				err = qs.ifIndexToNameCacheRefresh()
				if err != nil {
					return err
				}
			}
			qi = &QdiscInfo{
				IfName: ifName,
			}
			qs.Info[qiKey] = qi
		} else if ifIndexToNameCacheRefreshed {
			// Verify every so often (at refresh time, that is), that the name
			// of the interface hasn't changed:
			if ifName := qsShared.ifIndexToNameCache[qiKey.IfIndex]; qi.IfName != ifName {
				qi.IfName = ifName
			}
		}
		qi.Uint32[QDISC_HANDLE] = qiKey.Handle
		qi.Uint32[QDISC_PARENT] = nlenc.Uint32(msg.Data[12:16])

		// The first 20 bytes are taken by tcmsg:
		attrs, err := netlink.UnmarshalAttributes(msg.Data[20:])
		if err != nil {
			return fmt.Errorf("failed to unmarshal attributes: %v", err)
		}
		for _, attr := range attrs {
			switch attr.Type {
			case TCA_KIND:
				qi.Kind = nlenc.String(attr.Data)
			case TCA_STATS2:
				err = qi.parseTCAStats2(attr)
				if err != nil {
					return err
				}
			case TCA_STATS:
				// Legacy
				err = qi.parseTCAStats(attr)
				if err != nil {
					return err
				}
			default:
				// TODO: TCA_OPTIONS and TCA_XSTATS
			}
		}

		qi.scanNum = scanNum
	}

	// Remove out-of-scope qdiscs as needed:
	if len(msgs) != len(qs.Info) {
		for qiKey, qi := range qs.Info {
			if qi.scanNum != scanNum {
				delete(qs.Info, qiKey)
				continue
			}
		}
	}

	*qsShared.scanNum = scanNum
	return nil
}
