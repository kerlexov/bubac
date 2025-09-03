package dataprotection

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditAction represents a single data protection action
type AuditAction struct {
	Field         string     `json:"field"`
	Action        ActionType `json:"action"`
	OriginalValue string     `json:"original_value,omitempty"`
	NewValue      string     `json:"new_value,omitempty"`
}

// AuditEntry represents a complete audit log entry
type AuditEntry struct {
	Timestamp        time.Time     `json:"timestamp"`
	LogEntryID       string        `json:"log_entry_id"`
	ServiceName      string        `json:"service_name"`
	AgentID          string        `json:"agent_id"`
	ActionsPerformed []AuditAction `json:"actions_performed"`
}

// AuditLogger handles audit logging for data protection actions
type AuditLogger struct {
	logFile *os.File
	encoder *json.Encoder
	mutex   sync.Mutex
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger() *AuditLogger {
	// Create audit log directory
	auditDir := "./audit"
	if err := os.MkdirAll(auditDir, 0755); err != nil {
		log.Printf("Failed to create audit directory: %v", err)
		return &AuditLogger{} // Return logger without file
	}
	
	// Create audit log file with timestamp
	timestamp := time.Now().Format("2006-01-02")
	filename := filepath.Join(auditDir, fmt.Sprintf("data-protection-%s.log", timestamp))
	
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Printf("Failed to open audit log file: %v", err)
		return &AuditLogger{} // Return logger without file
	}
	
	return &AuditLogger{
		logFile: file,
		encoder: json.NewEncoder(file),
	}
}

// LogAuditEntry logs an audit entry
func (al *AuditLogger) LogAuditEntry(entry AuditEntry) {
	if al.logFile == nil {
		// Log to standard logger if file is not available
		log.Printf("DATA_PROTECTION_AUDIT: %+v", entry)
		return
	}
	
	al.mutex.Lock()
	defer al.mutex.Unlock()
	
	if err := al.encoder.Encode(entry); err != nil {
		log.Printf("Failed to write audit entry: %v", err)
	}
}

// Close closes the audit logger
func (al *AuditLogger) Close() error {
	al.mutex.Lock()
	defer al.mutex.Unlock()
	
	if al.logFile != nil {
		return al.logFile.Close()
	}
	return nil
}

// AuditStats represents audit statistics
type AuditStats struct {
	TotalEntries      int                    `json:"total_entries"`
	ActionCounts      map[ActionType]int     `json:"action_counts"`
	FieldCounts       map[string]int         `json:"field_counts"`
	ServiceCounts     map[string]int         `json:"service_counts"`
	LastAuditTime     time.Time              `json:"last_audit_time"`
}

// AuditStatsCollector collects audit statistics
type AuditStatsCollector struct {
	stats *AuditStats
	mutex sync.RWMutex
}

// NewAuditStatsCollector creates a new audit stats collector
func NewAuditStatsCollector() *AuditStatsCollector {
	return &AuditStatsCollector{
		stats: &AuditStats{
			ActionCounts:  make(map[ActionType]int),
			FieldCounts:   make(map[string]int),
			ServiceCounts: make(map[string]int),
		},
	}
}

// RecordAuditEntry records statistics for an audit entry
func (asc *AuditStatsCollector) RecordAuditEntry(entry AuditEntry) {
	asc.mutex.Lock()
	defer asc.mutex.Unlock()
	
	asc.stats.TotalEntries++
	asc.stats.LastAuditTime = entry.Timestamp
	asc.stats.ServiceCounts[entry.ServiceName]++
	
	for _, action := range entry.ActionsPerformed {
		asc.stats.ActionCounts[action.Action]++
		asc.stats.FieldCounts[action.Field]++
	}
}

// GetStats returns current audit statistics
func (asc *AuditStatsCollector) GetStats() AuditStats {
	asc.mutex.RLock()
	defer asc.mutex.RUnlock()
	
	// Create a copy to avoid race conditions
	statsCopy := AuditStats{
		TotalEntries:  asc.stats.TotalEntries,
		LastAuditTime: asc.stats.LastAuditTime,
		ActionCounts:  make(map[ActionType]int),
		FieldCounts:   make(map[string]int),
		ServiceCounts: make(map[string]int),
	}
	
	for k, v := range asc.stats.ActionCounts {
		statsCopy.ActionCounts[k] = v
	}
	
	for k, v := range asc.stats.FieldCounts {
		statsCopy.FieldCounts[k] = v
	}
	
	for k, v := range asc.stats.ServiceCounts {
		statsCopy.ServiceCounts[k] = v
	}
	
	return statsCopy
}

// ResetStats resets audit statistics
func (asc *AuditStatsCollector) ResetStats() {
	asc.mutex.Lock()
	defer asc.mutex.Unlock()
	
	asc.stats = &AuditStats{
		ActionCounts:  make(map[ActionType]int),
		FieldCounts:   make(map[string]int),
		ServiceCounts: make(map[string]int),
	}
}