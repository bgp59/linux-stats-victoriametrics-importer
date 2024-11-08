# LSVMI Qdics Metrics (id: `proc_qdisc_metrics`)

<!-- TOC tocDepth:2..3 chapterDepth:2..6 -->

- [General Information](#general-information)
- [Metrics](#metrics)
  - [qdisc_rate_kbps](#qdisc_rate_kbps)
  - [qdisc_packets_delta](#qdisc_packets_delta)
  - [qdisc_drops_delta](#qdisc_drops_delta)
  - [qdisc_requeues_delta](#qdisc_requeues_delta)
  - [qdisc_overlimits_delta](#qdisc_overlimits_delta)
  - [qdisc_qlen](#qdisc_qlen)
  - [qdisc_backlog](#qdisc_backlog)
  - [qdisc_gcflows_delta](#qdisc_gcflows_delta)
  - [qdisc_throttled_delta](#qdisc_throttled_delta)
  - [qdisc_flowsplimit_delta](#qdisc_flowsplimit_delta)
  - [qdisc_present](#qdisc_present)
  - [qdisc_metrics_delta_sec](#qdisc_metrics_delta_sec)

<!-- /TOC -->

## General Information

Based on `tc -s show qdisc` like info, see [tc](https://man7.org/linux/man-pages/man8/tc.8.html)

Additional information at [Linux Advanced Routing & Traffic Control](https://lartc.org/): [Chapter 9. Queueing Disciplines for Bandwidth Management](https://lartc.org/howto/lartc.qdisc.html)

## Metrics

Unless otherwise specified all metrics in this section have the following label set:

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| kind | _kind_ |
| handle | _maj_ |
| parent | _maj:min_ |
| if | _interface_ |

### qdisc_rate_kbps

Average throughput, in kbit/sec, over the interval since the last scan.

### qdisc_packets_delta

Number of packets since the last scan.

### qdisc_drops_delta

Number of dropped packets since the last scan.

### qdisc_requeues_delta

Number of re-queued packets since the last scan.

### qdisc_overlimits_delta

Number over the limit packets since the last scan.

### qdisc_qlen

Queue length.

### qdisc_backlog

Queue backlog.

### qdisc_gcflows_delta

Number of GC'ed flows since the last scan.

### qdisc_throttled_delta

Number of throttled flows since the last scan.

### qdisc_flowsplimit_delta

Number of split flows since the last scan.

### qdisc_present

Metric indicating the presence (value `1`) or disappearance (value `0`) of a qdisc. This metric could be used as a qualifier for the values above, since qdisc's can be added/deleted at any time. When a qdisc is deleted its last metrics may still linger in queries due to lookback interval. For a more precise cutoff point, the metrics can by qualified as follows:

  ```text
  
  qdisc_rate_kbps \
  and on (instance, hostname, kind, handle, parent, if) \
  (last_over_time(qdisc_present) > 0)

  ```

### qdisc_metrics_delta_sec

Time in seconds since the last scan. The real life counterpart (i.e. measured value) to the desired (configured) `interval`.

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
