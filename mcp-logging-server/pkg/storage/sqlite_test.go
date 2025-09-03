package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kerlexov/mcp-logging-server/pkg/models"
)

func TestSQLiteStorage_NewSQLiteStorage(t *testing.T) {
	// Test with in-memory database
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	// Verify database is initialized
	ctx := context.Background()
	health := storage.HealthCheck(ctx)
	if health.Status != "healthy" {
		t.Errorf("Expected healthy status, got %s", health.Status)
	}
}

func TestSQLiteStorage_Store(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()

	// Test storing single log entry
	logs := []models.LogEntry{
		{
			ID:          uuid.New().String(),
			Timestamp:   time.Now(),
			Level:       models.LogLevelInfo,
			Message:     "Test message",
			ServiceName: "test-service",
			AgentID:     "test-agent",
			Platform:    models.PlatformGo,
			Metadata: map[string]interface{}{
				"key1": "value1",
				"key2": 42,
			},
			DeviceInfo: &models.DeviceInfo{
				Platform:   "linux",
				Version:    "1.0.0",
				Model:      "server",
				AppVersion: "1.0.0",
			},
			SourceLocation: &models.SourceLocation{
				File:     "main.go",
				Line:     42,
				Function: "main",
			},
		},
	}

	if err := storage.Store(ctx, logs); err != nil {
		t.Fatalf("Failed to store logs: %v", err)
	}

	// Verify log was stored
	result, err := storage.Query(ctx, models.LogFilter{})
	if err != nil {
		t.Fatalf("Failed to query logs: %v", err)
	}

	if len(result.Logs) != 1 {
		t.Errorf("Expected 1 log, got %d", len(result.Logs))
	}

	if result.Logs[0].ID != logs[0].ID {
		t.Errorf("Expected log ID '%s', got %s", logs[0].ID, result.Logs[0].ID)
	}
}

func TestSQLiteStorage_StoreBatch(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()

	// Test storing multiple log entries
	logs := []models.LogEntry{
		{
			ID:          uuid.New().String(),
			Timestamp:   time.Now(),
			Level:       models.LogLevelInfo,
			Message:     "Test message 1",
			ServiceName: "test-service",
			AgentID:     "test-agent",
			Platform:    models.PlatformGo,
		},
		{
			ID:          uuid.New().String(),
			Timestamp:   time.Now().Add(time.Minute),
			Level:       models.LogLevelError,
			Message:     "Test message 2",
			ServiceName: "test-service",
			AgentID:     "test-agent",
			Platform:    models.PlatformGo,
		},
	}

	if err := storage.Store(ctx, logs); err != nil {
		t.Fatalf("Failed to store logs: %v", err)
	}

	// Verify logs were stored
	result, err := storage.Query(ctx, models.LogFilter{})
	if err != nil {
		t.Fatalf("Failed to query logs: %v", err)
	}

	if len(result.Logs) != 2 {
		t.Errorf("Expected 2 logs, got %d", len(result.Logs))
	}
}

func TestSQLiteStorage_Query(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()

	// Store test data
	now := time.Now()
	logs := []models.LogEntry{
		{
			ID:          uuid.New().String(),
			Timestamp:   now,
			Level:       models.LogLevelInfo,
			Message:     "Info message",
			ServiceName: "service-1",
			AgentID:     "agent-1",
			Platform:    models.PlatformGo,
		},
		{
			ID:          uuid.New().String(),
			Timestamp:   now.Add(time.Minute),
			Level:       models.LogLevelError,
			Message:     "Error message",
			ServiceName: "service-2",
			AgentID:     "agent-2",
			Platform:    models.PlatformSwift,
		},
		{
			ID:          uuid.New().String(),
			Timestamp:   now.Add(2 * time.Minute),
			Level:       models.LogLevelWarn,
			Message:     "Warning message",
			ServiceName: "service-1",
			AgentID:     "agent-1",
			Platform:    models.PlatformGo,
		},
	}

	if err := storage.Store(ctx, logs); err != nil {
		t.Fatalf("Failed to store logs: %v", err)
	}

	// Test query by service name
	result, err := storage.Query(ctx, models.LogFilter{ServiceName: "service-1"})
	if err != nil {
		t.Fatalf("Failed to query logs by service: %v", err)
	}
	if len(result.Logs) != 2 {
		t.Errorf("Expected 2 logs for service-1, got %d", len(result.Logs))
	}

	// Test query by level
	result, err = storage.Query(ctx, models.LogFilter{Level: models.LogLevelError})
	if err != nil {
		t.Fatalf("Failed to query logs by level: %v", err)
	}
	if len(result.Logs) != 1 {
		t.Errorf("Expected 1 error log, got %d", len(result.Logs))
	}

	// Test query by message contains
	result, err = storage.Query(ctx, models.LogFilter{MessageContains: "Error"})
	if err != nil {
		t.Fatalf("Failed to query logs by message: %v", err)
	}
	if len(result.Logs) != 1 {
		t.Errorf("Expected 1 log containing 'Error', got %d", len(result.Logs))
	}

	// Test query with time range
	result, err = storage.Query(ctx, models.LogFilter{
		StartTime: now.Add(30 * time.Second),
		EndTime:   now.Add(90 * time.Second),
	})
	if err != nil {
		t.Fatalf("Failed to query logs by time range: %v", err)
	}
	if len(result.Logs) != 1 {
		t.Errorf("Expected 1 log in time range, got %d", len(result.Logs))
	}

	// Test pagination
	result, err = storage.Query(ctx, models.LogFilter{Limit: 2})
	if err != nil {
		t.Fatalf("Failed to query logs with limit: %v", err)
	}
	if len(result.Logs) != 2 {
		t.Errorf("Expected 2 logs with limit, got %d", len(result.Logs))
	}
	if !result.HasMore {
		t.Error("Expected HasMore to be true")
	}
}

func TestSQLiteStorage_GetByIDs(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()

	// Store test data
	logs := []models.LogEntry{
		{
			ID:          uuid.New().String(),
			Timestamp:   time.Now(),
			Level:       models.LogLevelInfo,
			Message:     "Test message 1",
			ServiceName: "test-service",
			AgentID:     "test-agent",
			Platform:    models.PlatformGo,
		},
		{
			ID:          uuid.New().String(),
			Timestamp:   time.Now(),
			Level:       models.LogLevelError,
			Message:     "Test message 2",
			ServiceName: "test-service",
			AgentID:     "test-agent",
			Platform:    models.PlatformGo,
		},
	}

	if err := storage.Store(ctx, logs); err != nil {
		t.Fatalf("Failed to store logs: %v", err)
	}

	// Test getting by IDs
	result, err := storage.GetByIDs(ctx, []string{logs[0].ID, logs[1].ID})
	if err != nil {
		t.Fatalf("Failed to get logs by IDs: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 logs, got %d", len(result))
	}

	// Test getting non-existent ID
	result, err = storage.GetByIDs(ctx, []string{"non-existent"})
	if err != nil {
		t.Fatalf("Failed to get logs by non-existent ID: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected 0 logs for non-existent ID, got %d", len(result))
	}
}

func TestSQLiteStorage_GetServices(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()

	// Store test data
	now := time.Now()
	logs := []models.LogEntry{
		{
			ID:          uuid.New().String(),
			Timestamp:   now,
			Level:       models.LogLevelInfo,
			Message:     "Test message 1",
			ServiceName: "service-1",
			AgentID:     "agent-1",
			Platform:    models.PlatformGo,
		},
		{
			ID:          uuid.New().String(),
			Timestamp:   now.Add(time.Minute),
			Level:       models.LogLevelError,
			Message:     "Test message 2",
			ServiceName: "service-1",
			AgentID:     "agent-1",
			Platform:    models.PlatformGo,
		},
		{
			ID:          uuid.New().String(),
			Timestamp:   now.Add(2 * time.Minute),
			Level:       models.LogLevelWarn,
			Message:     "Test message 3",
			ServiceName: "service-2",
			AgentID:     "agent-2",
			Platform:    models.PlatformSwift,
		},
	}

	if err := storage.Store(ctx, logs); err != nil {
		t.Fatalf("Failed to store logs: %v", err)
	}

	// Test getting services
	services, err := storage.GetServices(ctx)
	if err != nil {
		t.Fatalf("Failed to get services: %v", err)
	}

	if len(services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(services))
	}

	// Verify service info
	for _, service := range services {
		if service.ServiceName == "service-1" {
			if service.LogCount != 2 {
				t.Errorf("Expected 2 logs for service-1, got %d", service.LogCount)
			}
			if service.Platform != models.PlatformGo {
				t.Errorf("Expected platform go for service-1, got %s", service.Platform)
			}
		} else if service.ServiceName == "service-2" {
			if service.LogCount != 1 {
				t.Errorf("Expected 1 log for service-2, got %d", service.LogCount)
			}
			if service.Platform != models.PlatformSwift {
				t.Errorf("Expected platform swift for service-2, got %s", service.Platform)
			}
		}
	}
}

func TestSQLiteStorage_HealthCheck(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()

	health := storage.HealthCheck(ctx)
	if health.Status != "healthy" {
		t.Errorf("Expected healthy status, got %s", health.Status)
	}

	if health.Details["database"] != "connected" {
		t.Errorf("Expected database connected, got %s", health.Details["database"])
	}
}

func TestSQLiteStorage_InvalidData(t *testing.T) {
	storage, err := NewSQLiteStorage(":memory:")
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()

	// Test storing invalid log entry (missing required fields)
	logs := []models.LogEntry{
		{
			ID:      uuid.New().String(),
			Message: "Test message",
			// Missing required fields
		},
	}

	if err := storage.Store(ctx, logs); err == nil {
		t.Error("Expected error when storing invalid log entry")
	}
}

func TestSQLiteStorage_Migration(t *testing.T) {
	// Create temporary file for testing migration
	tmpFile, err := os.CreateTemp("", "test_migration_*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Create storage with file database
	storage, err := NewSQLiteStorage(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create SQLite storage: %v", err)
	}
	storage.Close()

	// Reopen storage to test migration idempotency
	storage, err = NewSQLiteStorage(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to reopen SQLite storage: %v", err)
	}
	defer storage.Close()

	// Verify database is still healthy
	ctx := context.Background()
	health := storage.HealthCheck(ctx)
	if health.Status != "healthy" {
		t.Errorf("Expected healthy status after migration, got %s", health.Status)
	}
}
