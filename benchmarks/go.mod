module github.com/emypar/linux-stats-victoriametrics-importer/benchmarks

replace github.com/emypar/linux-stats-victoriametrics-importer => ../

go 1.21

toolchain go1.21.6

require (
	github.com/emypar/linux-stats-victoriametrics-importer v0.0.0-00010101000000-000000000000
	github.com/prometheus/procfs v0.11.1
)

require (
	github.com/capnm/sysinfo v0.0.0-20130621111458-5909a53897f3 // indirect
	github.com/tklauser/go-sysconf v0.3.12 // indirect
	github.com/tklauser/numcpus v0.6.1 // indirect
	golang.org/x/sys v0.15.0 // indirect
)
