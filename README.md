# linux-stats-victoriametrics-importer

An utility for importing granular Linux stats, such as those provided by <a href="https://linux.die.net/man/5/proc" target="_blank">procfs</a> into <a href="https://docs.victoriametrics.com/Cluster-VictoriaMetrics.html" target="_blank">VictoriaMetrics</a>

# Motivation And Solution

Financial institutions use so called Market Data Platforms for disseminating live financial information. Such platforms may be latency sensitive, in that the data transition time between producers (typically external feeds) and consumers (typically an automated trading systems) has to be less than a given threshold at all times, typically < 1 millisecond. Latency spikes are usually created by resource bound conditions, leading to queuing, or by errors/discards, leading to retransmissions. Given the low threshold of latency, the telemetry data for the systems have to be sufficiently granular, time wise, to be of any use. For instance a 100% CPU condition for a thread that lasts 1 second could explain a 20 millisecond latency jump. If the sampling period were 5 seconds, the same thread would show 20% CPU utilization, thus masking the resource bound condition.

[VictoriaMetrics](https://docs.victoriametrics.com/Cluster-VictoriaMetrics.html) does en excellent job based on our experience at [OpenAI](https://openai.com) in handling large numbers of time series and given its integration w/ [Grafana](https://grafana.com/grafana/) and its query language, [MetricsQL](https://docs.victoriametrics.com/MetricsQL.html), a superset of [PromQL](https://prometheus.io/docs/prometheus/latest/querying/basics/), it is a perfect candidate for storing the metrics.

The widely used approach of scraping for collecting metrics would be suboptimal in this case, given the 100 millisecond .. 1 second time granularity of the latter. 

Since **VictoriaMetrics** supports the [import](https://docs.victoriametrics.com/Cluster-VictoriaMetrics.html#url-format) paradigm, it is more efficient to collect the granular stats, with the timestamps of the actual collection, into larger batches and to push the latter, in compressed format, to import end points.

# TL;DR: Quick Start For PoC

The **PoC** requires an instance of **VictoriaMetrics**, **Grafana** and **LSVMI** running on the same Linux server or container.

## Using A Linux Server

* choose a suitable location for installing **VictoriaMetrics** and **Grafana**, e.g. under `$HOME/lsvmi-poc`:/
  ```
  cd /tmp
  curl -s -L \
      https://github.com/emypar/linux-stats-victoriametrics-importer/releases/download/poc_infra/lsvmi-infra-install.tgz | \
  tar xfz -

  ./lsvmi-infra-install/install-lsvmi-infra.sh # it will install under $HOME/lsvmi-poc

  rm -rf ./lsvmi-infra-install                 # optional cleanup
  ```
	If a different location 

## Using A Containerized Solution

* have  [Docker](https://docs.docker.com/get-started/get-docker/) installed
* invoke:
	```
	docker run \
		-it \
		--rm \
		--detach \
		--publish 3000:3000 \
		--name lsvmi-demo \
		emypar/linux-stats-victoriametrics-importer:demo 
	```
* point a, preferably Chrome (may be in Incognito mode), browser to http://localhost:3000 for **Grafana** UI, user: `admin`, password: `lsvmi`



# Architecture

## Diagram

![architecture](docs/images/architecture.jpg)

## Components
### Scheduler
The scheduler is responsible for determining the next (the  nearest in time, that is) task that needs to be done. A task is an encapsulation of a metrics generator, responsible for metrics that are configured as a group. The metrics generators are generally grouped by source, e.g. `/proc/stat`, `/proc/PID/{stat,status,cmdline}`, etc.

### TODO Queue
A Golang channel storing the tasks, written by the **Scheduler** and read by workers. This allows the parallelization of metrics generation.

### Task, TaskActivity And Metrics Generators

The **Task** is the abstraction used for scheduling. It contains a **TaskActivity** which, for scheduling purposes, is an interface with an `Execute` method.

In its actual implementation the **TaskActivity** is a metrics generator with its context, most notably the cache of previous values used for deltas.

Each generator uses parsers for reading [/proc](https://man7.org/linux/man-pages/man5/proc.5.html) or other source of information which it later formats into [Prometheus exposition text format](https://github.com/prometheus/docs/blob/main/content/docs/instrumenting/exposition_formats.md#text-based-format). 

The generated metrics are packed into buffers, until the latter reach ~ 64k in size (the last buffer of the scan may be shorter, of course). The buffers are written into the **Compressor Queue**

### Compressor Queue

A Golang channel with metrics holding buffers from all metrics generator functions, which are its writers. The readers are gzip compressor workers. This approach has 2 benefits:
* it supports the parallelization of compression
* it allows more efficient packing by consolidating metrics across all generator functions, compared to individual compression inside the latter.

### Compressor Workers

They perform gzip compression until either the compressed buffer reaches ~ 64k in size, or the partially compressed data becomes older than N seconds (time based flush, that is). Once a compressed buffer is ready to be sent, the compressor uses **SendBuffer**, the sender method of the **HTTP Sender Pool**, to ship it to an import end point.

### HTTP Sender Pool

The **HTTP Sender Pool** holds information and state about all the configured VictoriaMetrics end points. The end points can be either healthy or unhealthy. If a send operation fails, the used end point is moved to the unhealthy list. The latter is periodically checked by health checkers and end points that pass the check are moved back to the healthy list. **SendBuffer** is a method of the **HTTP Sender Pool** and it works with the latter to maintain the healthy / unhealthy lists. The **Compressor Workers** that actually invoke **SendBuffer** are unaware of these details, they are simply informed that the compressed buffer was successfully sent or that it was discarded (after a number of attempts). The healthy end points are used in a round robin fashion to spread the load across all of the VictoriaMetrics import end points.

### Bandwidth Control

The **Bandwidth Control** implements a credit based mechanism to ensure that the egress traffic across all **SendBuffer** invocations does not exceed a certain limit. This is useful in smoothing bursts when all metrics are generated at the same time, e.g. at start.

# Implementation Considerations

## The Three Laws Of Stats Collection

1. **First Do No Harm:** The collectors should have a light footprint in terms of resources: no CPU or memory hogs, no I/O blasters, no DDoS attack on the metrics database,  etc. Anyone who has had the computer rendered irresponsive by a "lightweight" virus scanner, will intuitively understand and relate.
1. **Be Useful:** Collect only data that might have a use case.
1. **Be Comprehensive:** Collect **all** potentially useful data, even if it may be needed once in the lifetime of the system; that single use may save the day.

## Resource Utilization Mitigation Techniques

### Custom Parsers
#### Minimal Parsing

Most stats are presented in text format via <a href="https://linux.die.net/man/5/proc" target="_blank">procfs</a> file system. The generated metrics are also in text format, stored in `[]byte` buffers. For parsed data used as-is, either as label or metric values, the most efficient parsing is none whatsoever, in that the file is read into a `[]byte` buffer and the parser simply splits it into `[][]byte` fields.


#### Reusable Objects And The Double Buffer

Typical stats parsers will create and return a new object w/ the parsed data for every invocation. However most of the stats have a fixed structure [^1] so the data could be stored in a previously created object, thus avoiding the pressure on the garbage collector.

Additionally certain metrics generators may need to refer the previous scan values. The double buffer approach will rely on a `parser [2]*ParserType` array in the generator context together with a `currentIndex` integer that's toggled between `0` and `1` at every scan. `parser[currentIndex]` will be passed to the parser to retrieve the latest data and `parser[1 - currentIndex]` will represent the previous scan.

For the reason above custom parsers were created under the [procfs](procfs).

[^1]: The fixed structure applies for a given kernel version, i.e. it is fixed for the uptime of a given host. 

### Handling Counters

Counters are typically used for deltas and rates, rather than as-is. While the time series database can compute those in the query, there are 2 issues with storing counters directly:
* most counters are `uint64` while Prometheus values are `float64`. Converting the former to the latter results in a loss of precision for large values that may generate misleading 0 deltas.
* counters may roll over resulting in unrealistic, large in absolute value, negative deltas and rates, due to the `float64` arithmetic.

`Golang` native integer arithmetic handles correctly the rollover, e.g.
```
package main

import (
	"fmt"
	"math"
)

func main() {
	var crt, prev uint64 = 0, math.MaxUint64
	fmt.Println(crt - prev)
}
```
correctly prints `1`. For those reasons metrics based on counters are published as deltas or rates.


### Reducing The Number Of Data Points

#### Partial V. Full Metrics

In order to reduce the traffic between the importer and the import endpoints, only the metrics whose values have changed from the previous scan are being generated and sent. Pedantically that would require that queries be made using the [last_over_time(METRIC[RANGE_INTERVAL])](https://prometheus.io/docs/prometheus/latest/querying/functions/#aggregation_over_time) function; in practice **Grafana** and **VictoriaMetrics** query interface will have a look-back interval > _RANGE_INTERVAL_ so in most cases `last_over_time` is not needed.

To make the range interval predictable, all metrics generator are configured with 2 parameters: `interval` and `full_metrics_factor` and each metric is guaranteed to be sent over a `interval` x `full_metrics_factor`, regardless of its lack of change from the previous scan. 

For each metric or small group of metrics there is a `cycle#`, incremented modulo `full_metrics_factor` after each generation. When the counter reaches 0, a full metrics cycle will ensue. In order to spread the full metrics cycles evenly across all metrics (to avoid bursts), the `cycle#` is initialized from an auto-incrementing global counter.

It should be noted that for delta values the partial approach is implemented as no-zero-after-zero, i.e. if the current and previous deltas are both 0 then the current metric is skipped, except for full cycle of course.

**Note:** The partial approach can be disabled by setting the `full_metrics_factor` to 0.

#### Active Processes/Threads

In addition to the change only approach, process/thread metrics use the concept of active process to further reduce the number of metrics. PIDs/TIDs are classified into active/inactive based upon whether they used and CPU since the previous scan. Inactive processes/threads are ignored for partial cycles.



