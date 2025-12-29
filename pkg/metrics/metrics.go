package metrics

import (
	"sync"
	"time"
)

// Metrics provides basic in-memory metrics tracking
type Metrics struct {
	mu              sync.RWMutex
	requestCount    int64
	errorCount      int64
	latencySum      time.Duration
	requestCountMap map[string]int64
}

// New creates a new metrics instance
func New() *Metrics {
	return &Metrics{
		requestCountMap: make(map[string]int64),
	}
}

// RecordRequest records a successful request
func (m *Metrics) RecordRequest(endpoint string, latency time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requestCount++
	m.latencySum += latency
	m.requestCountMap[endpoint]++
}

// RecordError records an error
func (m *Metrics) RecordError(endpoint string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.errorCount++
}

// GetStats returns current metrics statistics
func (m *Metrics) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	avgLatency := time.Duration(0)
	if m.requestCount > 0 {
		avgLatency = m.latencySum / time.Duration(m.requestCount)
	}

	return map[string]interface{}{
		"total_requests":   m.requestCount,
		"total_errors":     m.errorCount,
		"average_latency":  avgLatency,
		"requests_by_path": m.requestCountMap,
	}
}
