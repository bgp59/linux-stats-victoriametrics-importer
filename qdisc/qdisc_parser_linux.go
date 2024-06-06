// qdisc parser a-la tc -s show qdisc

// Based on https://github.com/ema/qdisc, adapted for reusable objects, w/ data
// presented as slices. The latter allows for metrics generation in a loop.

//go:build linux

package qdisc

import (
	"errors"
	"fmt"
	"math"
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

var QdiscAvail = true

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

func (qs *QdiscStats) ifIndexToNameRefreshNoLock() error {
	ifas, err := net.Interfaces()
	if err != nil {
		return err
	}
	if qs.ifIndexToNameCache == nil {
		qs.ifIndexToNameCache = make(map[uint32]string)
	}
	for _, ifa := range ifas {
		qs.ifIndexToNameCache[uint32(ifa.Index)] = ifa.Name
	}
	qs.ifIndexToNameCacheLastRefresh = time.Now()
	return nil
}

func (qs *QdiscStats) Update() error {
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

	defer c.Close()

	req := netlink.Message{
		Header: netlink.Header{
			Flags: netlink.Request | netlink.Dump,
			Type:  38, // RTM_GETQDISC
		},
		Data: make([]byte, 20),
	}

	// Perform a request, receive replies, and validate the replies
	msgs, err := c.Execute(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %v", err)
	}

	scanNum := qs.scanNum + 1
	for _, msg := range msgs {
		var qiKey QdiscInfoKey
		if len(msg.Data) < 20 {
			return fmt.Errorf("short message, len=%d < 20", len(msg.Data))
		}
		qiKey.IfIndex = nlenc.Uint32(msg.Data[4:8])
		qiKey.Handle = nlenc.Uint32(msg.Data[8:12])
		qi := qs.Info[qiKey]
		if qi == nil {
			qi = &QdiscInfo{}
			qs.Info[qiKey] = qi
		}
		qi.Uint32[QDISC_HANDLE] = qiKey.Handle
		parent := nlenc.Uint32(msg.Data[12:16])
		if parent == math.MaxUint32 {
			parent = 0
		}
		qi.Uint32[QDISC_PARENT] = parent

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

	// Resolve names and remove out-of-scope handles:
	qs.ifIndexToNameCacheLock.Lock()
	defer qs.ifIndexToNameCacheLock.Unlock()

	ifIndexToNameCacheRefreshed := false
	if qs.ifIndexToNameCache == nil ||
		time.Since(qs.ifIndexToNameCacheLastRefresh) >= IF_INDEX_TO_NAME_CACHE_REFRESH_INTERVAL {
		err = qs.ifIndexToNameRefreshNoLock()
		if err != nil {
			return err
		}
		ifIndexToNameCacheRefreshed = true
	}
	for qiKey, qi := range qs.Info {
		if qi.scanNum != scanNum {
			delete(qs.Info, qiKey)
			continue
		}
		if qi.IfName == "" {
			ifName := qs.ifIndexToNameCache[qiKey.IfIndex]
			if ifName == "" && !ifIndexToNameCacheRefreshed {
				err = qs.ifIndexToNameRefreshNoLock()
				if err != nil {
					return err
				}
				ifIndexToNameCacheRefreshed = true
			}
			qi.IfName = qs.ifIndexToNameCache[qiKey.IfIndex]
		}
	}

	qs.scanNum = scanNum
	return nil
}
