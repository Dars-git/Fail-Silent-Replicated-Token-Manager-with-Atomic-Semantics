package main

import (
	"sync/atomic"
	"time"
)

type MetricsSnapshot struct {
	WriteTokenCalls             uint64
	ReadTokenCalls              uint64
	WriteBroadcastCalls         uint64
	ReadBroadcastCalls          uint64
	WriteTokenErrors            uint64
	ReadTokenErrors             uint64
	WriteBroadcastErrors        uint64
	ReadBroadcastErrors         uint64
	WriteTokenAuthFailures      uint64
	ReadTokenAuthFailures       uint64
	OutboundBroadcastErrors     uint64
	OutboundReadBroadcastErrors uint64
	QuorumFailures              uint64
	WriteTokenLatencyAvgMicros  float64
	ReadTokenLatencyAvgMicros   float64
}

type serverMetrics struct {
	writeTokenCalls             atomic.Uint64
	readTokenCalls              atomic.Uint64
	writeBroadcastCalls         atomic.Uint64
	readBroadcastCalls          atomic.Uint64
	writeTokenErrors            atomic.Uint64
	readTokenErrors             atomic.Uint64
	writeBroadcastErrors        atomic.Uint64
	readBroadcastErrors         atomic.Uint64
	writeTokenAuthFailures      atomic.Uint64
	readTokenAuthFailures       atomic.Uint64
	outboundBroadcastErrors     atomic.Uint64
	outboundReadBroadcastErrors atomic.Uint64
	quorumFailures              atomic.Uint64
	writeTokenLatencyCount      atomic.Uint64
	readTokenLatencyCount       atomic.Uint64
	writeTokenLatencyNanos      atomic.Int64
	readTokenLatencyNanos       atomic.Int64
	writeBroadcastLatencyNanos  atomic.Int64
	readBroadcastLatencyNanos   atomic.Int64
}

func newServerMetrics() *serverMetrics { return &serverMetrics{} }

func (m *serverMetrics) incWriteTokenCalls()             { m.writeTokenCalls.Add(1) }
func (m *serverMetrics) incReadTokenCalls()              { m.readTokenCalls.Add(1) }
func (m *serverMetrics) incWriteBroadcastCalls()         { m.writeBroadcastCalls.Add(1) }
func (m *serverMetrics) incReadBroadcastCalls()          { m.readBroadcastCalls.Add(1) }
func (m *serverMetrics) incWriteTokenErrors()            { m.writeTokenErrors.Add(1) }
func (m *serverMetrics) incReadTokenErrors()             { m.readTokenErrors.Add(1) }
func (m *serverMetrics) incWriteBroadcastErrors()        { m.writeBroadcastErrors.Add(1) }
func (m *serverMetrics) incReadBroadcastErrors()         { m.readBroadcastErrors.Add(1) }
func (m *serverMetrics) incWriteTokenAuthFailures()      { m.writeTokenAuthFailures.Add(1) }
func (m *serverMetrics) incReadTokenAuthFailures()       { m.readTokenAuthFailures.Add(1) }
func (m *serverMetrics) incOutboundBroadcastErrors()     { m.outboundBroadcastErrors.Add(1) }
func (m *serverMetrics) incOutboundReadBroadcastErrors() { m.outboundReadBroadcastErrors.Add(1) }
func (m *serverMetrics) incQuorumFailures()              { m.quorumFailures.Add(1) }
func (m *serverMetrics) observeWriteTokenLatency(d time.Duration) {
	m.writeTokenLatencyCount.Add(1)
	m.writeTokenLatencyNanos.Add(d.Nanoseconds())
}
func (m *serverMetrics) observeReadTokenLatency(d time.Duration) {
	m.readTokenLatencyCount.Add(1)
	m.readTokenLatencyNanos.Add(d.Nanoseconds())
}
func (m *serverMetrics) observeWriteBroadcastLatency(d time.Duration) {
	m.writeBroadcastLatencyNanos.Add(d.Nanoseconds())
}
func (m *serverMetrics) observeReadBroadcastLatency(d time.Duration) {
	m.readBroadcastLatencyNanos.Add(d.Nanoseconds())
}

func (s *TokenManagerServer) MetricsSnapshot() MetricsSnapshot {
	s.ensureDefaults()
	out := MetricsSnapshot{
		WriteTokenCalls:             s.metrics.writeTokenCalls.Load(),
		ReadTokenCalls:              s.metrics.readTokenCalls.Load(),
		WriteBroadcastCalls:         s.metrics.writeBroadcastCalls.Load(),
		ReadBroadcastCalls:          s.metrics.readBroadcastCalls.Load(),
		WriteTokenErrors:            s.metrics.writeTokenErrors.Load(),
		ReadTokenErrors:             s.metrics.readTokenErrors.Load(),
		WriteBroadcastErrors:        s.metrics.writeBroadcastErrors.Load(),
		ReadBroadcastErrors:         s.metrics.readBroadcastErrors.Load(),
		WriteTokenAuthFailures:      s.metrics.writeTokenAuthFailures.Load(),
		ReadTokenAuthFailures:       s.metrics.readTokenAuthFailures.Load(),
		OutboundBroadcastErrors:     s.metrics.outboundBroadcastErrors.Load(),
		OutboundReadBroadcastErrors: s.metrics.outboundReadBroadcastErrors.Load(),
		QuorumFailures:              s.metrics.quorumFailures.Load(),
	}
	if c := s.metrics.writeTokenLatencyCount.Load(); c > 0 {
		out.WriteTokenLatencyAvgMicros = float64(s.metrics.writeTokenLatencyNanos.Load()) / float64(c) / 1000.0
	}
	if c := s.metrics.readTokenLatencyCount.Load(); c > 0 {
		out.ReadTokenLatencyAvgMicros = float64(s.metrics.readTokenLatencyNanos.Load()) / float64(c) / 1000.0
	}
	return out
}
