// Definitions common to all metrics and generators.

package lsvmi

import "bytes"

// A metrics generator satisfies the TaskActivity interface to be able to
// register with the scheduler.

// The generated metrics are written into *bytes.Buffer's which then queued into
// the metrics queue for transmission.

type MetricsQueue interface {
	GetBuf() *bytes.Buffer
	QueueBuf(b *bytes.Buffer)
	GetTargetSize() int
}

// The general flow of the TaskActivity implementation:
//  repeat until no more metrics
//  - buf <- MetricsQueue.GetBuf()
//  - fill buf it with metrics until it reaches MetricsQueue.GetTargetSize() or
//    there are no more metrics
//  - MetricsQueue.QueueBuf(buf)
