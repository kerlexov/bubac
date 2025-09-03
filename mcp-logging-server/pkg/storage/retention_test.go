package storage

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kerlexov/mcp-logging-server/pkg/models"
)

func TestRetentionService_GetRetentionDate(t *testing.T) {
	policy := RetentionPolicy{
		DefaultDays: 30,
		ByLevel: map[models.LogLevel]int{
			models.LogLevelDebug: 7,
			models.LogLevelError: 90,
		},
	}

	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	retentionService := NewRetentionService(storage, policy)

	now := time.Now()

	// Test default retention
	infoDate := retentionService.GetRetentionDate(models.LogLevelInfo)
	expectedInfo := now.AddDate(0, 0, -30)
	if infoDate.Day() != expectedInfo.Day() {
		t.Errorf("Expected INFO retention date around %v, got %v", expectedInfo, infoDate)
	}

	// Test level-specific retention
	debugDate := retentionService.GetRetentionDate(models.LogLevelDebug)
	expectedDebug := now.AddDate(0, 0, -7)
	if debugDate.Day() != expectedDebug.Day() {
		t.Errorf("Expected DEBUG retention date around %v, got %v", expectedDebug, debugDate)
	}

	errorDate := retentionService.GetRetentionDate(models.LogLevelError)
	expectedError := now.AddDate(0, 0, -90)
	if errorDate.Day() != expectedError.Day() {
		t.Errorf("Expected ERROR retention date around %v, got %v", expectedError, errorDate)
	}
}

func TestRetentionService_CleanupExpiredLogs(t *testing.T) {
	policy := RetentionPolicy{
		DefaultDays: 30,
		ByLevel: map[models.LogLevel]int{
			models.LogLevelDebug: 7,
		},
	}

	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	retentionService := NewRetentionService(storage, policy)
	ctx := context.Background()

	now := time.Now()

	// Create test logs with different ages
	logs := []models.LogEntry{
		{
			ID:          uuid.New().String(),
			Timestamp:   now.AddDate(0, 0, -5), // 5 days old - should be kept
			Level:       models.LogLevelDebug,
			Message:     "Recent debug log",
			ServiceName: "test-service",
			AgentID:     "test-agent",
			Platform:    models.PlatformGo,
		},
		{
			ID:          uuid.New().String(),
			Timestamp:   now.AddDate(0, 0, -10), // 10 days old - should be deleted (debug retention is 7 days)
			Level:       models.LogLevelDebug,
			Message:     "Old debug log",
			ServiceName: "test-service",
			AgentID:     "test-agent",
			Platform:    models.PlatformGo,
		},
		{
			ID:          uuid.New().String(),
			Timestamp:   now.AddDate(0, 0, -20), // 20 days old - should be kept (info retention is 30 days)
			Level:       models.LogLevelInfo,
			Message:     "Recent info log",
			ServiceName: "test-service",
			AgentID:     "test-agent",
			Platform:    models.PlatformGo,
		},
		{
			ID:          uuid.New().String(),
			Timestamp:   now.AddDate(0, 0, -40), // 40 days old - should be deleted (info retention is 30 days)
			Level:       models.LogLevelInfo,
			Message:     "Old info log",
			ServiceName: "test-service",
			AgentID:     "test-agent",
			Platform:    models.PlatformGo,
		},
	}

	// Store the logs
	if err := storage.Store(ctx, logs); err != nil {
		t.Fatalf("Failed to store logs: %v", err)
	}

	// Verify all logs are stored
	result, err := storage.Query(ctx, models.LogFilter{})
	if err != nil {
		t.Fatalf("Failed to query logs: %v", err)
	}
	if len(result.Logs) != 4 {
		t.Errorf("Expected 4 logs before cleanup, got %d", len(result.Logs))
	}

	// Run cleanup
	cleanupResult, err := retentionService.CleanupExpiredLogs(ctx)
	if err != nil {
		t.Fatalf("Failed to cleanup expired logs: %v", err)
	}

	if cleanupResult.TotalDeleted != 2 {
		t.Errorf("Expected 2 logs to be deleted, got %d", cleanupResult.TotalDeleted)
	}

	// Verify remaining logs
	result, err = storage.Query(ctx, models.LogFilter{})
	if err != nil {
		t.Fatalf("Failed to query logs after cleanup: %v", err)
	}
	if len(result.Logs) != 2 {
		t.Errorf("Expected 2 logs after cleanup, got %d", len(result.Logs))
	}

	// Verify the correct logs remain
	for _, log := range result.Logs {
		if log.Level == models.LogLevelDebug && log.Timestamp.Before(now.AddDate(0, 0, -7)) {
			t.Errorf("Old debug log should have been deleted: %v", log)
		}
		if log.Level == models.LogLevelInfo && log.Timestamp.Before(now.AddDate(0, 0, -30)) {
			t.Errorf("Old info log should have been deleted: %v", log)
		}
	}
}

func TestRetentionService_CleanupByCount(t *testing.T) {
	policy := RetentionPolicy{
		DefaultDays:       0, // No time-based retention
		MaxTotalLogs:      3,
		MaxLogsPerService: 2,
	}

	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	retentionService := NewRetentionService(storage, policy)
	ctx := context.Background()

	now := time.Now()

	// Create test logs
	logs := []models.LogEntry{
		{
			ID:          uuid.New().String(),
			Timestamp:   now.Add(-4 * time.Hour),
			Level:       models.LogLevelInfo,
			Message:     "Service 1 log 1",
			ServiceName: "service-1",
			AgentID:     "agent-1",
			Platform:    models.PlatformGo,
		},
		{
			ID:          uuid.New().String(),
			Timestamp:   now.Add(-3 * time.Hour),
			Level:       models.LogLevelInfo,
			Message:     "Service 1 log 2",
			ServiceName: "service-1",
			AgentID:     "agent-1",
			Platform:    models.PlatformGo,
		},
		{
			ID:          uuid.New().String(),
			Timestamp:   now.Add(-2 * time.Hour),
			Level:       models.LogLevelInfo,
			Message:     "Service 1 log 3",
			ServiceName: "service-1",
			AgentID:     "agent-1",
			Platform:    models.PlatformGo,
		},
		{
			ID:          uuid.New().String(),
			Timestamp:   now.Add(-1 * time.Hour),
			Level:       models.LogLevelInfo,
			Message:     "Service 2 log 1",
			ServiceName: "service-2",
			AgentID:     "agent-2",
			Platform:    models.PlatformGo,
		},
		{
			ID:          uuid.New().String(),
			Timestamp:   now,
			Level:       models.LogLevelInfo,
			Message:     "Service 2 log 2",
			ServiceName: "service-2",
			AgentID:     "agent-2",
			Platform:    models.PlatformGo,
		},
	}

	// Store the logs
	if err := storage.Store(ctx, logs); err != nil {
		t.Fatalf("Failed to store logs: %v", err)
	}

	// Verify all logs are stored
	result, err := storage.Query(ctx, models.LogFilter{})
	if err != nil {
		t.Fatalf("Failed to query logs: %v", err)
	}
	if len(result.Logs) != 5 {
		t.Errorf("Expected 5 logs before cleanup, got %d", len(result.Logs))
	}

	// Run count-based cleanup
	cleanupResult, err := retentionService.CleanupByCount(ctx)
	if err != nil {
		t.Fatalf("Failed to cleanup by count: %v", err)
	}

	// Should delete logs to meet both total and per-service limits
	if cleanupResult.TotalDeleted == 0 {
		t.Error("Expected some logs to be deleted")
	}

	// Verify remaining logs don't exceed limits
	result, err = storage.Query(ctx, models.LogFilter{})
	if err != nil {
		t.Fatalf("Failed to query logs after cleanup: %v", err)
	}

	if len(result.Logs) > policy.MaxTotalLogs {
		t.Errorf("Total logs %d exceeds limit %d", len(result.Logs), policy.MaxTotalLogs)
	}
}

func TestSQLiteStorage_DeleteByIDs(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()

	// Create test logs
	logs := []models.LogEntry{
		{
			ID:          uuid.New().String(),
			Timestamp:   time.Now(),
			Level:       models.LogLevelInfo,
			Message:     "Test log 1",
			ServiceName: "test-service",
			AgentID:     "test-agent",
			Platform:    models.PlatformGo,
		},
		{
			ID:          uuid.New().String(),
			Timestamp:   time.Now(),
			Level:       models.LogLevelInfo,
			Message:     "Test log 2",
			ServiceName: "test-service",
			AgentID:     "test-agent",
			Platform:    models.PlatformGo,
		},
		{
			ID:          uuid.New().String(),
			Timestamp:   time.Now(),
			Level:       models.LogLevelInfo,
			Message:     "Test log 3",
			ServiceName: "test-service",
			AgentID:     "test-agent",
			Platform:    models.PlatformGo,
		},
	}

	// Store the logs
	if err := storage.Store(ctx, logs); err != nil {
		t.Fatalf("Failed to store logs: %v", err)
	}

	// Verify all logs are stored
	result, err := storage.Query(ctx, models.LogFilter{})
	if err != nil {
		t.Fatalf("Failed to query logs: %v", err)
	}
	if len(result.Logs) != 3 {
		t.Errorf("Expected 3 logs before deletion, got %d", len(result.Logs))
	}

	// Delete two logs
	idsToDelete := []string{logs[0].ID, logs[1].ID}
	deleted, err := storage.DeleteByIDs(ctx, idsToDelete)
	if err != nil {
		t.Fatalf("Failed to delete logs: %v", err)
	}

	if deleted != 2 {
		t.Errorf("Expected 2 logs deleted, got %d", deleted)
	}

	// Verify remaining log
	result, err = storage.Query(ctx, models.LogFilter{})
	if err != nil {
		t.Fatalf("Failed to query logs after deletion: %v", err)
	}
	if len(result.Logs) != 1 {
		t.Errorf("Expected 1 log after deletion, got %d", len(result.Logs))
	}

	if result.Logs[0].ID != logs[2].ID {
		t.Errorf("Wrong log remained after deletion: expected %s, got %s", logs[2].ID, result.Logs[0].ID)
	}

	// Test deleting non-existent IDs
	deleted, err = storage.DeleteByIDs(ctx, []string{"non-existent-id"})
	if err != nil {
		t.Fatalf("Failed to delete non-existent logs: %v", err)
	}

	if deleted != 0 {
		t.Errorf("Expected 0 logs deleted for non-existent ID, got %d", deleted)
	}

	// Test deleting empty list
	deleted, err = storage.DeleteByIDs(ctx, []string{})
	if err != nil {
		t.Fatalf("Failed to delete empty list: %v", err)
	}

	if deleted != 0 {
		t.Errorf("Expected 0 logs deleted for empty list, got %d", deleted)
	}
}

func TestRetentionScheduler(t *testing.T) {
	policy := RetentionPolicy{
		DefaultDays: 1, // Very short retention for testing
	}

	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	retentionService := NewRetentionService(storage, policy)
	scheduler := NewRetentionScheduler(retentionService, 100*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// Create an old log that should be cleaned up
	oldLog := models.LogEntry{
		ID:          uuid.New().String(),
		Timestamp:   time.Now().AddDate(0, 0, -2), // 2 days old
		Level:       models.LogLevelInfo,
		Message:     "Old log to be cleaned",
		ServiceName: "test-service",
		AgentID:     "test-agent",
		Platform:    models.PlatformGo,
	}

	if err := storage.Store(ctx, []models.LogEntry{oldLog}); err != nil {
		t.Fatalf("Failed to store old log: %v", err)
	}

	// Verify log is stored
	result, err := storage.Query(ctx, models.LogFilter{})
	if err != nil {
		t.Fatalf("Failed to query logs: %v", err)
	}
	if len(result.Logs) != 1 {
		t.Errorf("Expected 1 log before scheduler, got %d", len(result.Logs))
	}

	// Start scheduler
	if scheduler.IsRunning() {
		t.Error("Scheduler should not be running initially")
	}

	scheduler.Start(ctx)

	if !scheduler.IsRunning() {
		t.Error("Scheduler should be running after start")
	}

	// Wait for cleanup to happen
	time.Sleep(200 * time.Millisecond)

	// Stop scheduler
	scheduler.Stop()

	if scheduler.IsRunning() {
		t.Error("Scheduler should not be running after stop")
	}

	// Wait a bit for scheduler to fully stop
	time.Sleep(50 * time.Millisecond)

	// Verify log was cleaned up
	result, err = storage.Query(ctx, models.LogFilter{})
	if err != nil {
		t.Fatalf("Failed to query logs after cleanup: %v", err)
	}
	if len(result.Logs) != 0 {
		t.Errorf("Expected 0 logs after cleanup, got %d", len(result.Logs))
	}
}

func TestRetentionPolicy_NoRetention(t *testing.T) {
	// Test policy with no retention (keep forever)
	policy := RetentionPolicy{
		DefaultDays: 0, // No retention
	}

	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	defer storage.Close()

	retentionService := NewRetentionService(storage, policy)

	// Test that retention date is zero (no retention)
	retentionDate := retentionService.GetRetentionDate(models.LogLevelInfo)
	if !retentionDate.IsZero() {
		t.Errorf("Expected zero retention date for no retention policy, got %v", retentionDate)
	}

	ctx := context.Background()

	// Create very old log
	oldLog := models.LogEntry{
		ID:          uuid.New().String(),
		Timestamp:   time.Now().AddDate(-1, 0, 0), // 1 year old
		Level:       models.LogLevelInfo,
		Message:     "Very old log",
		ServiceName: "test-service",
		AgentID:     "test-agent",
		Platform:    models.PlatformGo,
	}

	if err := storage.Store(ctx, []models.LogEntry{oldLog}); err != nil {
		t.Fatalf("Failed to store old log: %v", err)
	}

	// Run cleanup
	result, err := retentionService.CleanupExpiredLogs(ctx)
	if err != nil {
		t.Fatalf("Failed to cleanup: %v", err)
	}

	// Should not delete anything
	if result.TotalDeleted != 0 {
		t.Errorf("Expected 0 logs deleted with no retention policy, got %d", result.TotalDeleted)
	}

	// Verify log still exists
	logs, err := storage.Query(ctx, models.LogFilter{})
	if err != nil {
		t.Fatalf("Failed to query logs: %v", err)
	}
	if len(logs.Logs) != 1 {
		t.Errorf("Expected 1 log to remain, got %d", len(logs.Logs))
	}
}
