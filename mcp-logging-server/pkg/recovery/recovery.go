package recovery

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/your-org/mcp-logging-server/pkg/models"
)

// RecoveryManager handles server restart scenarios and data recovery
type RecoveryManager struct {
	recoveryDir string
	mutex       sync.RWMutex
}

// NewRecoveryManager creates a new recovery manager
func NewRecoveryManager(recoveryDir string) *RecoveryManager {
	return &RecoveryManager{
		recoveryDir: recoveryDir,
	}
}

// SavePendingLogs saves logs to disk for recovery after restart
func (rm *RecoveryManager) SavePendingLogs(logs []models.LogEntry) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	
	// Ensure recovery directory exists
	if err := os.MkdirAll(rm.recoveryDir, 0755); err != nil {
		return fmt.Errorf("failed to create recovery directory: %w", err)
	}
	
	// Create recovery file with timestamp
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("pending_logs_%d.json", timestamp)
	filepath := filepath.Join(rm.recoveryDir, filename)
	
	// Marshal logs to JSON
	data, err := json.Marshal(logs)
	if err != nil {
		return fmt.Errorf("failed to marshal logs: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("failed to write recovery file: %w", err)
	}
	
	return nil
}

// RecoverPendingLogs recovers logs from disk after server restart
func (rm *RecoveryManager) RecoverPendingLogs(ctx context.Context) ([]models.LogEntry, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	var allLogs []models.LogEntry
	
	// Check if recovery directory exists
	if _, err := os.Stat(rm.recoveryDir); os.IsNotExist(err) {
		return allLogs, nil // No recovery files
	}
	
	// Read all recovery files
	files, err := os.ReadDir(rm.recoveryDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read recovery directory: %w", err)
	}
	
	for _, file := range files {
		if file.IsDir() || !isRecoveryFile(file.Name()) {
			continue
		}
		
		filepath := filepath.Join(rm.recoveryDir, file.Name())
		logs, err := rm.loadLogsFromFile(filepath)
		if err != nil {
			// Log error but continue with other files
			fmt.Printf("Failed to load recovery file %s: %v\n", file.Name(), err)
			continue
		}
		
		allLogs = append(allLogs, logs...)
		
		// Remove recovery file after successful loading
		if err := os.Remove(filepath); err != nil {
			fmt.Printf("Failed to remove recovery file %s: %v\n", file.Name(), err)
		}
	}
	
	return allLogs, nil
}

// CleanupOldRecoveryFiles removes recovery files older than the specified duration
func (rm *RecoveryManager) CleanupOldRecoveryFiles(maxAge time.Duration) error {
	rm.mutex.Lock()
	defer rm.mutex.Unlock()
	
	// Check if recovery directory exists
	if _, err := os.Stat(rm.recoveryDir); os.IsNotExist(err) {
		return nil // No recovery directory
	}
	
	files, err := os.ReadDir(rm.recoveryDir)
	if err != nil {
		return fmt.Errorf("failed to read recovery directory: %w", err)
	}
	
	cutoff := time.Now().Add(-maxAge)
	
	for _, file := range files {
		if file.IsDir() || !isRecoveryFile(file.Name()) {
			continue
		}
		
		info, err := file.Info()
		if err != nil {
			continue
		}
		
		if info.ModTime().Before(cutoff) {
			filepath := filepath.Join(rm.recoveryDir, file.Name())
			if err := os.Remove(filepath); err != nil {
				fmt.Printf("Failed to remove old recovery file %s: %v\n", file.Name(), err)
			}
		}
	}
	
	return nil
}

// GetRecoveryStats returns statistics about recovery files
func (rm *RecoveryManager) GetRecoveryStats() (RecoveryStats, error) {
	rm.mutex.RLock()
	defer rm.mutex.RUnlock()
	
	stats := RecoveryStats{}
	
	// Check if recovery directory exists
	if _, err := os.Stat(rm.recoveryDir); os.IsNotExist(err) {
		return stats, nil
	}
	
	files, err := os.ReadDir(rm.recoveryDir)
	if err != nil {
		return stats, fmt.Errorf("failed to read recovery directory: %w", err)
	}
	
	for _, file := range files {
		if file.IsDir() || !isRecoveryFile(file.Name()) {
			continue
		}
		
		info, err := file.Info()
		if err != nil {
			continue
		}
		
		stats.FileCount++
		stats.TotalSize += info.Size()
		
		if stats.OldestFile.IsZero() || info.ModTime().Before(stats.OldestFile) {
			stats.OldestFile = info.ModTime()
		}
		
		if info.ModTime().After(stats.NewestFile) {
			stats.NewestFile = info.ModTime()
		}
	}
	
	return stats, nil
}

// RecoveryStats contains statistics about recovery files
type RecoveryStats struct {
	FileCount   int       `json:"file_count"`
	TotalSize   int64     `json:"total_size_bytes"`
	OldestFile  time.Time `json:"oldest_file"`
	NewestFile  time.Time `json:"newest_file"`
}

// loadLogsFromFile loads logs from a recovery file
func (rm *RecoveryManager) loadLogsFromFile(filepath string) ([]models.LogEntry, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}
	
	var logs []models.LogEntry
	if err := json.Unmarshal(data, &logs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal logs: %w", err)
	}
	
	return logs, nil
}

// isRecoveryFile checks if a filename is a recovery file
func isRecoveryFile(filename string) bool {
	return filepath.Ext(filename) == ".json" && 
		   (filepath.Base(filename)[:12] == "pending_logs" || 
		    filepath.Base(filename)[:13] == "pending_logs_")
}