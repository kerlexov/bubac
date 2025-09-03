package recovery

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/your-org/mcp-logging-server/pkg/models"
)

func createTestLogEntry(id string) models.LogEntry {
	return models.LogEntry{
		ID:          id,
		Timestamp:   time.Now(),
		Level:       models.LogLevelInfo,
		Message:     "Test message",
		ServiceName: "test-service",
		AgentID:     "test-agent",
		Platform:    models.PlatformGo,
	}
}

func TestRecoveryManager_SaveAndRecoverPendingLogs(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "recovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	rm := NewRecoveryManager(tempDir)
	
	// Create test logs
	logs := []models.LogEntry{
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440001"),
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440002"),
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440003"),
	}
	
	// Save logs
	err = rm.SavePendingLogs(logs)
	if err != nil {
		t.Fatalf("Failed to save pending logs: %v", err)
	}
	
	// Verify file was created
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read recovery directory: %v", err)
	}
	
	if len(files) != 1 {
		t.Errorf("Expected 1 recovery file, got %d", len(files))
	}
	
	// Recover logs
	ctx := context.Background()
	recoveredLogs, err := rm.RecoverPendingLogs(ctx)
	if err != nil {
		t.Fatalf("Failed to recover pending logs: %v", err)
	}
	
	// Verify recovered logs
	if len(recoveredLogs) != len(logs) {
		t.Errorf("Expected %d recovered logs, got %d", len(logs), len(recoveredLogs))
	}
	
	for i, log := range recoveredLogs {
		if log.ID != logs[i].ID {
			t.Errorf("Expected log ID %s, got %s", logs[i].ID, log.ID)
		}
		if log.Message != logs[i].Message {
			t.Errorf("Expected log message %s, got %s", logs[i].Message, log.Message)
		}
	}
	
	// Verify recovery file was removed after successful recovery
	files, err = os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read recovery directory after recovery: %v", err)
	}
	
	if len(files) != 0 {
		t.Errorf("Expected 0 recovery files after recovery, got %d", len(files))
	}
}

func TestRecoveryManager_EmptyDirectory(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "recovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	rm := NewRecoveryManager(tempDir)
	
	// Try to recover from empty directory
	ctx := context.Background()
	recoveredLogs, err := rm.RecoverPendingLogs(ctx)
	if err != nil {
		t.Fatalf("Failed to recover from empty directory: %v", err)
	}
	
	if len(recoveredLogs) != 0 {
		t.Errorf("Expected 0 recovered logs from empty directory, got %d", len(recoveredLogs))
	}
}

func TestRecoveryManager_NonExistentDirectory(t *testing.T) {
	// Use non-existent directory
	nonExistentDir := "/tmp/non_existent_recovery_dir_12345"
	rm := NewRecoveryManager(nonExistentDir)
	
	// Try to recover from non-existent directory
	ctx := context.Background()
	recoveredLogs, err := rm.RecoverPendingLogs(ctx)
	if err != nil {
		t.Fatalf("Failed to recover from non-existent directory: %v", err)
	}
	
	if len(recoveredLogs) != 0 {
		t.Errorf("Expected 0 recovered logs from non-existent directory, got %d", len(recoveredLogs))
	}
}

func TestRecoveryManager_MultipleRecoveryFiles(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "recovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	rm := NewRecoveryManager(tempDir)
	
	// Create multiple batches of logs
	batch1 := []models.LogEntry{
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440001"),
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440002"),
	}
	
	batch2 := []models.LogEntry{
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440003"),
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440004"),
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440005"),
	}
	
	// Save both batches
	err = rm.SavePendingLogs(batch1)
	if err != nil {
		t.Fatalf("Failed to save batch1: %v", err)
	}
	
	// Wait a bit to ensure different timestamps
	time.Sleep(1100 * time.Millisecond)
	
	err = rm.SavePendingLogs(batch2)
	if err != nil {
		t.Fatalf("Failed to save batch2: %v", err)
	}
	
	// Verify multiple files were created
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read recovery directory: %v", err)
	}
	
	if len(files) != 2 {
		t.Errorf("Expected 2 recovery files, got %d", len(files))
	}
	
	// Recover all logs
	ctx := context.Background()
	recoveredLogs, err := rm.RecoverPendingLogs(ctx)
	if err != nil {
		t.Fatalf("Failed to recover pending logs: %v", err)
	}
	
	// Verify all logs were recovered
	expectedTotal := len(batch1) + len(batch2)
	if len(recoveredLogs) != expectedTotal {
		t.Errorf("Expected %d recovered logs, got %d", expectedTotal, len(recoveredLogs))
	}
	
	// Verify all recovery files were removed
	files, err = os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read recovery directory after recovery: %v", err)
	}
	
	if len(files) != 0 {
		t.Errorf("Expected 0 recovery files after recovery, got %d", len(files))
	}
}

func TestRecoveryManager_GetRecoveryStats(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "recovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	rm := NewRecoveryManager(tempDir)
	
	// Initially, stats should be empty
	stats, err := rm.GetRecoveryStats()
	if err != nil {
		t.Fatalf("Failed to get recovery stats: %v", err)
	}
	
	if stats.FileCount != 0 {
		t.Errorf("Expected FileCount to be 0, got %d", stats.FileCount)
	}
	
	// Save some logs
	logs := []models.LogEntry{
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440001"),
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440002"),
	}
	
	err = rm.SavePendingLogs(logs)
	if err != nil {
		t.Fatalf("Failed to save pending logs: %v", err)
	}
	
	// Check stats after saving
	stats, err = rm.GetRecoveryStats()
	if err != nil {
		t.Fatalf("Failed to get recovery stats after saving: %v", err)
	}
	
	if stats.FileCount != 1 {
		t.Errorf("Expected FileCount to be 1, got %d", stats.FileCount)
	}
	
	if stats.TotalSize <= 0 {
		t.Errorf("Expected TotalSize to be positive, got %d", stats.TotalSize)
	}
	
	if stats.OldestFile.IsZero() {
		t.Error("Expected OldestFile to be set")
	}
	
	if stats.NewestFile.IsZero() {
		t.Error("Expected NewestFile to be set")
	}
}

func TestRecoveryManager_CleanupOldRecoveryFiles(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "recovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	rm := NewRecoveryManager(tempDir)
	
	// Create a recovery file
	logs := []models.LogEntry{
		createTestLogEntry("550e8400-e29b-41d4-a716-446655440001"),
	}
	
	err = rm.SavePendingLogs(logs)
	if err != nil {
		t.Fatalf("Failed to save pending logs: %v", err)
	}
	
	// Verify file exists
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read recovery directory: %v", err)
	}
	
	if len(files) != 1 {
		t.Errorf("Expected 1 recovery file, got %d", len(files))
	}
	
	// Cleanup files older than 1 hour (should not remove the recent file)
	err = rm.CleanupOldRecoveryFiles(1 * time.Hour)
	if err != nil {
		t.Fatalf("Failed to cleanup old recovery files: %v", err)
	}
	
	// Verify file still exists
	files, err = os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read recovery directory after cleanup: %v", err)
	}
	
	if len(files) != 1 {
		t.Errorf("Expected 1 recovery file after cleanup, got %d", len(files))
	}
	
	// Cleanup files older than 0 seconds (should remove all files)
	err = rm.CleanupOldRecoveryFiles(0)
	if err != nil {
		t.Fatalf("Failed to cleanup all recovery files: %v", err)
	}
	
	// Verify file was removed
	files, err = os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read recovery directory after aggressive cleanup: %v", err)
	}
	
	if len(files) != 0 {
		t.Errorf("Expected 0 recovery files after aggressive cleanup, got %d", len(files))
	}
}

func TestRecoveryManager_CorruptedRecoveryFile(t *testing.T) {
	// Create temporary directory for testing
	tempDir, err := os.MkdirTemp("", "recovery_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	rm := NewRecoveryManager(tempDir)
	
	// Create a corrupted recovery file
	corruptedFile := filepath.Join(tempDir, "pending_logs_123456.json")
	err = os.WriteFile(corruptedFile, []byte("invalid json content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create corrupted file: %v", err)
	}
	
	// Try to recover (should handle corrupted file gracefully)
	ctx := context.Background()
	recoveredLogs, err := rm.RecoverPendingLogs(ctx)
	if err != nil {
		t.Fatalf("Failed to recover with corrupted file present: %v", err)
	}
	
	// Should return empty logs (corrupted file should be skipped)
	if len(recoveredLogs) != 0 {
		t.Errorf("Expected 0 recovered logs with corrupted file, got %d", len(recoveredLogs))
	}
	
	// Corrupted file should still exist (not removed due to error)
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read recovery directory: %v", err)
	}
	
	if len(files) != 1 {
		t.Errorf("Expected 1 file (corrupted) to remain, got %d", len(files))
	}
}