package metrics

import (
	"sync"
	"time"
)

// Metrics holds operational metrics for the server
type Metrics struct {
	mutex                sync.RWMutex
	requestsTotal        int64
	requestsSuccessful   int64
	requestsFailed       int64
	logsIngested         int64
	logsBuffered         int64
	bufferFlushes        int64
	bufferFlushErrors    int64
	storageErrors        int64
	validationErrors     int64
	lastRequestTime      time.Time
	serverStartTime      time.Time
	bufferOverflows      int64
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		serverStartTime: time.Now(),
	}
}

// IncrementRequestsTotal increments the total requests counter
func (m *Metrics) IncrementRequestsTotal() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.requestsTotal++
	m.lastRequestTime = time.Now()
}

// IncrementRequestsSuccessful increments the successful requests counter
func (m *Metrics) IncrementRequestsSuccessful() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.requestsSuccessful++
}

// IncrementRequestsFailed increments the failed requests counter
func (m *Metrics) IncrementRequestsFailed() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.requestsFailed++
}

// IncrementLogsIngested increments the logs ingested counter
func (m *Metrics) IncrementLogsIngested(count int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.logsIngested += count
}

// IncrementLogsBuffered increments the logs buffered counter
func (m *Metrics) IncrementLogsBuffered(count int64) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.logsBuffered += count
}

// IncrementBufferFlushes increments the buffer flushes counter
func (m *Metrics) IncrementBufferFlushes() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.bufferFlushes++
}

// IncrementBufferFlushErrors increments the buffer flush errors counter
func (m *Metrics) IncrementBufferFlushErrors() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.bufferFlushErrors++
}

// IncrementStorageErrors increments the storage errors counter
func (m *Metrics) IncrementStorageErrors() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.storageErrors++
}

// IncrementValidationErrors increments the validation errors counter
func (m *Metrics) IncrementValidationErrors() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.validationErrors++
}

// IncrementBufferOverflows increments the buffer overflows counter
func (m *Metrics) IncrementBufferOverflows() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.bufferOverflows++
}

// GetSnapshot returns a snapshot of current metrics
func (m *Metrics) GetSnapshot() MetricsSnapshot {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	uptime := time.Since(m.serverStartTime)
	
	return MetricsSnapshot{
		RequestsTotal:        m.requestsTotal,
		RequestsSuccessful:   m.requestsSuccessful,
		RequestsFailed:       m.requestsFailed,
		LogsIngested:         m.logsIngested,
		LogsBuffered:         m.logsBuffered,
		BufferFlushes:        m.bufferFlushes,
		BufferFlushErrors:    m.bufferFlushErrors,
		StorageErrors:        m.storageErrors,
		ValidationErrors:     m.validationErrors,
		BufferOverflows:      m.bufferOverflows,
		LastRequestTime:      m.lastRequestTime,
		ServerStartTime:      m.serverStartTime,
		UptimeSeconds:        int64(uptime.Seconds()),
		SuccessRate:          m.calculateSuccessRate(),
		ErrorRate:            m.calculateErrorRate(),
	}
}

// MetricsSnapshot represents a point-in-time snapshot of metrics
type MetricsSnapshot struct {
	RequestsTotal        int64     `json:"requests_total"`
	RequestsSuccessful   int64     `json:"requests_successful"`
	RequestsFailed       int64     `json:"requests_failed"`
	LogsIngested         int64     `json:"logs_ingested"`
	LogsBuffered         int64     `json:"logs_buffered"`
	BufferFlushes        int64     `json:"buffer_flushes"`
	BufferFlushErrors    int64     `json:"buffer_flush_errors"`
	StorageErrors        int64     `json:"storage_errors"`
	ValidationErrors     int64     `json:"validation_errors"`
	BufferOverflows      int64     `json:"buffer_overflows"`
	LastRequestTime      time.Time `json:"last_request_time"`
	ServerStartTime      time.Time `json:"server_start_time"`
	UptimeSeconds        int64     `json:"uptime_seconds"`
	SuccessRate          float64   `json:"success_rate"`
	ErrorRate            float64   `json:"error_rate"`
}

// calculateSuccessRate calculates the success rate as a percentage
func (m *Metrics) calculateSuccessRate() float64 {
	if m.requestsTotal == 0 {
		return 0.0
	}
	return float64(m.requestsSuccessful) / float64(m.requestsTotal) * 100.0
}

// calculateErrorRate calculates the error rate as a percentage
func (m *Metrics) calculateErrorRate() float64 {
	if m.requestsTotal == 0 {
		return 0.0
	}
	return float64(m.requestsFailed) / float64(m.requestsTotal) * 100.0
}

// Reset resets all metrics (useful for testing)
func (m *Metrics) Reset() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	m.requestsTotal = 0
	m.requestsSuccessful = 0
	m.requestsFailed = 0
	m.logsIngested = 0
	m.logsBuffered = 0
	m.bufferFlushes = 0
	m.bufferFlushErrors = 0
	m.storageErrors = 0
	m.validationErrors = 0
	m.bufferOverflows = 0
	m.lastRequestTime = time.Time{}
	m.serverStartTime = time.Now()
}