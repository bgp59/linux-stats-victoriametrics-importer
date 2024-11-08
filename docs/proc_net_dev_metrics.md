# LSVMI Network Interface Metrics (id: `proc_net_dev_metrics`)
<!-- TOC tocDepth:2..3 chapterDepth:2..6 -->

- [General Information](#general-information)
- [Metrics](#metrics)
  - [proc_net_dev_rx_kbps](#proc_net_dev_rx_kbps)
  - [proc_net_dev_rx_pkts_delta](#proc_net_dev_rx_pkts_delta)
  - [proc_net_dev_rx_errs_delta](#proc_net_dev_rx_errs_delta)
  - [proc_net_dev_rx_drop_delta](#proc_net_dev_rx_drop_delta)
  - [proc_net_dev_rx_fifo_delta](#proc_net_dev_rx_fifo_delta)
  - [proc_net_dev_rx_frame_delta](#proc_net_dev_rx_frame_delta)
  - [proc_net_dev_rx_compressed_delta](#proc_net_dev_rx_compressed_delta)
  - [proc_net_dev_rx_mcast_delta](#proc_net_dev_rx_mcast_delta)
  - [proc_net_dev_tx_kbps](#proc_net_dev_tx_kbps)
  - [proc_net_dev_tx_pkts_delta](#proc_net_dev_tx_pkts_delta)
  - [proc_net_dev_tx_errs_delta](#proc_net_dev_tx_errs_delta)
  - [proc_net_dev_tx_drop_delta](#proc_net_dev_tx_drop_delta)
  - [proc_net_dev_tx_fifo_delta](#proc_net_dev_tx_fifo_delta)
  - [proc_net_dev_tx_colls_delta](#proc_net_dev_tx_colls_delta)
  - [proc_net_dev_tx_carrier_delta](#proc_net_dev_tx_carrier_delta)
  - [proc_net_dev_tx_compressed_delta](#proc_net_dev_tx_compressed_delta)
  - [proc_net_dev_present](#proc_net_dev_present)
  - [proc_net_dev_metrics_delta_sec](#proc_net_dev_metrics_delta_sec)

<!-- /TOC -->

## General Information

Based on [/proc/net/dev](https://man7.org/linux/man-pages/man5/proc_pid_net.5.html).

## Metrics

Unless otherwise specified, all the metrics have the following label set:

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
| dev | _interface_ |

### proc_net_dev_rx_kbps

The average receive throughput since the last scan, in kbit/sec.

### proc_net_dev_rx_pkts_delta

The number of received packets since the last scan.

### proc_net_dev_rx_errs_delta

The number of receive errors since the last scan.

### proc_net_dev_rx_drop_delta

The number of receive dropped packets since the last scan.

### proc_net_dev_rx_fifo_delta

The number of receive FIFO errors since the last scan.

### proc_net_dev_rx_frame_delta

The number of receive framing errors (length, CRC, etc.) since the last scan.

### proc_net_dev_rx_compressed_delta

The number of receive compressed packets since the last scan.

### proc_net_dev_rx_mcast_delta

The number of receive multicast packets since the last scan.

### proc_net_dev_tx_kbps

The average transmit throughput since the last scan, in kbit/sec.

### proc_net_dev_tx_pkts_delta

The number of transmitted packets since the last scan.

### proc_net_dev_tx_errs_delta

The number of transmit errors since the last scan.

### proc_net_dev_tx_drop_delta

The number of transmit dropped  packets since the last scan.

### proc_net_dev_tx_fifo_delta

The number of transmit FIFO errors since the last scan.

### proc_net_dev_tx_colls_delta

The number of transmit collisions since the last scan.

### proc_net_dev_tx_carrier_delta

The number of transmit no carrier detect since the last scan.

### proc_net_dev_tx_compressed_delta

The number of transmit compressed packets since the last scan.

### proc_net_dev_present

[Pseudo-categorical](internals.md#pseudo-categorical-metrics ) metric indicating the presence (value `1`) or disappearance (value `0`) of an interface. This metric could be used as a qualifier for the values above, since interfaces are hot plug-able. When an interface is unplugged its last metrics may still linger in queries due to lookback interval. For a more precise cutoff point, the metrics can by qualified as follows:

  ```text
  
  proc_net_dev_rx_kbps \
  and on (instance, hostname, dev) \
  (last_over_time(proc_net_dev_present) > 0)

  ```

### proc_net_dev_metrics_delta_sec

Time in seconds since the last scan. The real life counterpart (i.e. measured value) to the desired (configured) `interval`.

| Label Name | Value(s)/Info |
| --- | --- |
| instance | _instance_ |
| hostname | _hostname_ |
