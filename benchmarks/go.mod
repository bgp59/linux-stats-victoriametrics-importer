module github.com/eparparita/linux-stats-victoriametrics-importer/benchmarks

replace github.com/eparparita/linux-stats-victoriametrics-importer => ../

go 1.20

require (
	github.com/eparparita/linux-stats-victoriametrics-importer v0.0.0-00010101000000-000000000000
	github.com/prometheus/procfs v0.11.1
)

require golang.org/x/sys v0.9.0 // indirect
