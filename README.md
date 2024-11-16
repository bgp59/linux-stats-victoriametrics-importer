# linux-stats-victoriametrics-importer (AKA LSVMI)

An utility for importing granular Linux stats, such as those provided by [procfs](https://linux.die.net/man/5/proc") into [VictoriaMetrics](https://docs.victoriametrics.com/)

## Motivation And Solution

Financial institutions use so called Market Data Platforms for disseminating live financial information. Such platforms may be latency sensitive, in that the time in transit between producers (typically external feeds) and consumers (typically an automated trading systems) has to be less than a given threshold at all times, typically < 1 millisecond. Latency spikes are usually created by resource bound conditions, leading to queuing, or by errors/discards, leading to retransmissions. Given the low threshold of latency, the telemetry data for the systems have to be sufficiently granular, time wise, to be of any use. For instance a 100% CPU condition for a thread that lasts 1 second could explain a 20 millisecond latency jump. If the sampling period were 5 seconds, the same thread would show 20% CPU utilization, thus masking the resource bound condition.

[VictoriaMetrics](https://docs.victoriametrics.com/) does en excellent job, based on our experience at [OpenAI](https://openai.com), in handling large numbers of time series and given its integration w/ [Grafana](https://grafana.com/grafana/) and its query language, [MetricsQL](https://docs.victoriametrics.com/MetricsQL.html), a superset of [PromQL](https://prometheus.io/docs/prometheus/latest/querying/basics/), it is a perfect candidate for storing the metrics.

The widely used approach of scraping for collecting metrics would be suboptimal in this case, given the 100 millisecond .. 1 second time granularity of the latter.

Since [VictoriaMetrics](https://docs.victoriametrics.com/) supports the [import](https://docs.victoriametrics.com/Cluster-VictoriaMetrics.html#url-format) paradigm, it is more efficient to collect the granular stats, with the timestamps of the actual collection, into larger batches and to push the latter, in compressed format, to import end points.

## Additional Information

- [The List Of Metrics](docs/metrics.md)
- [User Guide](docs/user_guide.md)
- [Running The Proof Of Concept](docs/poc.md)
- [Developer Guide](docs/dev_guide.md)
- [Internals](docs/internals.md)

## License

See [LICENSE](LICENSE.txt) for rights and limitations (MIT).
