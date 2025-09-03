package storage

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/your-org/mcp-logging-server/pkg/models"
)

func TestSearchService_NewSearchService(t *testing.T) {
	// Create temporary directory for index
	tmpDir, err := os.MkdirTemp("", "search_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	indexPath := filepath.Join(tmpDir, "test_index")

	// Create new search service
	searchService, err := NewSearchService(indexPath)
	if err != nil {
		t.Fatalf("Failed to create search service: %v", err)
	}
	defer searchService.Close()

	// Verify health check
	ctx := context.Background()
	health := searchService.HealthCheck(ctx)
	if health.Status != "healthy" {
		t.Errorf("Expected healthy status, got %s", health.Status)
	}
}

func TestSearchService_IndexLogEntry(t *testing.T) {
	// Create temporary directory for index
	tmpDir, err := os.MkdirTemp("", "search_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	indexPath := filepath.Join(tmpDir, "test_index")
	searchService, err := NewSearchService(indexPath)
	if err != nil {
		t.Fatalf("Failed to create search service: %v", err)
	}
	defer searchService.Close()

	// Create test log entry
	logEntry := models.LogEntry{
		ID:          uuid.New().String(),
		Timestamp:   time.Now(),
		Level:       models.LogLevelInfo,
		Message:     "Test message for indexing",
		ServiceName: "test-service",
		AgentID:     "test-agent",
		Platform:    models.PlatformGo,
		Metadata: map[string]interface{}{
			"key1": "value1",
		},
		DeviceInfo: &models.DeviceInfo{
			Platform: "linux",
			Model:    "server",
		},
		SourceLocation: &models.SourceLocation{
			File:     "main.go",
			Function: "main",
		},
	}

	// Index the log entry
	if err := searchService.IndexLogEntry(logEntry); err != nil {
		t.Fatalf("Failed to index log entry: %v", err)
	}

	// Verify document count
	stats, err := searchService.GetIndexStats()
	if err != nil {
		t.Fatalf("Failed to get index stats: %v", err)
	}

	if docCount, ok := stats["document_count"].(uint64); !ok || docCount != 1 {
		t.Errorf("Expected document count 1, got %v", stats["document_count"])
	}
}

func TestSearchService_SearchLogs(t *testing.T) {
	// Create temporary directory for index
	tmpDir, err := os.MkdirTemp("", "search_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	indexPath := filepath.Join(tmpDir, "test_index")
	searchService, err := NewSearchService(indexPath)
	if err != nil {
		t.Fatalf("Failed to create search service: %v", err)
	}
	defer searchService.Close()

	ctx := context.Background()
	now := time.Now()

	// Create test log entries
	logEntries := []models.LogEntry{
		{
			ID:          uuid.New().String(),
			Timestamp:   now,
			Level:       models.LogLevelInfo,
			Message:     "User authentication successful",
			ServiceName: "auth-service",
			AgentID:     "auth-agent",
			Platform:    models.PlatformGo,
		},
		{
			ID:          uuid.New().String(),
			Timestamp:   now.Add(time.Minute),
			Level:       models.LogLevelError,
			Message:     "Database connection failed",
			ServiceName: "db-service",
			AgentID:     "db-agent",
			Platform:    models.PlatformSwift,
		},
		{
			ID:          uuid.New().String(),
			Timestamp:   now.Add(2 * time.Minute),
			Level:       models.LogLevelWarn,
			Message:     "High memory usage detected",
			ServiceName: "monitor-service",
			AgentID:     "monitor-agent",
			Platform:    models.PlatformGo,
		},
	}

	// Index all log entries
	if err := searchService.IndexLogEntries(logEntries); err != nil {
		t.Fatalf("Failed to index log entries: %v", err)
	}

	// Test search by message content
	logIDs, err := searchService.SearchLogs(ctx, "authentication", models.LogFilter{})
	if err != nil {
		t.Fatalf("Failed to search logs: %v", err)
	}
	if len(logIDs) != 1 {
		t.Errorf("Expected 1 result for 'authentication', got %d", len(logIDs))
	}
	if logIDs[0] != logEntries[0].ID {
		t.Errorf("Expected log ID %s, got %s", logEntries[0].ID, logIDs[0])
	}

	// Test search by partial message
	logIDs, err = searchService.SearchLogs(ctx, "connection", models.LogFilter{})
	if err != nil {
		t.Fatalf("Failed to search logs: %v", err)
	}
	if len(logIDs) != 1 {
		t.Errorf("Expected 1 result for 'connection', got %d", len(logIDs))
	}

	// Test search with service filter
	logIDs, err = searchService.SearchLogs(ctx, "", models.LogFilter{
		ServiceName: "auth-service",
	})
	if err != nil {
		t.Fatalf("Failed to search logs with service filter: %v", err)
	}
	if len(logIDs) != 1 {
		t.Errorf("Expected 1 result for auth-service, got %d", len(logIDs))
	}

	// Test search with level filter
	logIDs, err = searchService.SearchLogs(ctx, "", models.LogFilter{
		Level: models.LogLevelError,
	})
	if err != nil {
		t.Fatalf("Failed to search logs with level filter: %v", err)
	}
	if len(logIDs) != 1 {
		t.Errorf("Expected 1 result for ERROR level, got %d", len(logIDs))
	}

	// Test search with time range
	logIDs, err = searchService.SearchLogs(ctx, "", models.LogFilter{
		StartTime: now.Add(30 * time.Second),
		EndTime:   now.Add(90 * time.Second),
	})
	if err != nil {
		t.Fatalf("Failed to search logs with time range: %v", err)
	}
	if len(logIDs) != 1 {
		t.Errorf("Expected 1 result in time range, got %d", len(logIDs))
	}

	// Test search with pagination
	logIDs, err = searchService.SearchLogs(ctx, "", models.LogFilter{
		Limit: 2,
	})
	if err != nil {
		t.Fatalf("Failed to search logs with pagination: %v", err)
	}
	if len(logIDs) != 2 {
		t.Errorf("Expected 2 results with limit, got %d", len(logIDs))
	}
}

func TestSearchService_DeleteLogEntry(t *testing.T) {
	// Create temporary directory for index
	tmpDir, err := os.MkdirTemp("", "search_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	indexPath := filepath.Join(tmpDir, "test_index")
	searchService, err := NewSearchService(indexPath)
	if err != nil {
		t.Fatalf("Failed to create search service: %v", err)
	}
	defer searchService.Close()

	// Create and index test log entry
	logEntry := models.LogEntry{
		ID:          uuid.New().String(),
		Timestamp:   time.Now(),
		Level:       models.LogLevelInfo,
		Message:     "Test message for deletion",
		ServiceName: "test-service",
		AgentID:     "test-agent",
		Platform:    models.PlatformGo,
	}

	if err := searchService.IndexLogEntry(logEntry); err != nil {
		t.Fatalf("Failed to index log entry: %v", err)
	}

	// Verify entry exists
	ctx := context.Background()
	logIDs, err := searchService.SearchLogs(ctx, "deletion", models.LogFilter{})
	if err != nil {
		t.Fatalf("Failed to search logs: %v", err)
	}
	if len(logIDs) != 1 {
		t.Errorf("Expected 1 result before deletion, got %d", len(logIDs))
	}

	// Delete the entry
	if err := searchService.DeleteLogEntry(logEntry.ID); err != nil {
		t.Fatalf("Failed to delete log entry: %v", err)
	}

	// Verify entry is deleted
	logIDs, err = searchService.SearchLogs(ctx, "deletion", models.LogFilter{})
	if err != nil {
		t.Fatalf("Failed to search logs after deletion: %v", err)
	}
	if len(logIDs) != 0 {
		t.Errorf("Expected 0 results after deletion, got %d", len(logIDs))
	}
}

func TestSQLiteStorageWithSearch_Integration(t *testing.T) {
	// Create temporary directory for both database and search index
	tmpDir, err := os.MkdirTemp("", "integration_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	dbPath := filepath.Join(tmpDir, "test.db")
	indexPath := filepath.Join(tmpDir, "search_index")

	// Create storage with search
	storage, err := NewSQLiteStorageWithSearch(dbPath, indexPath)
	if err != nil {
		t.Fatalf("Failed to create storage with search: %v", err)
	}
	defer storage.Close()

	ctx := context.Background()
	now := time.Now()

	// Store test data
	logs := []models.LogEntry{
		{
			ID:          uuid.New().String(),
			Timestamp:   now,
			Level:       models.LogLevelInfo,
			Message:     "User login successful with authentication token",
			ServiceName: "auth-service",
			AgentID:     "auth-agent",
			Platform:    models.PlatformGo,
		},
		{
			ID:          uuid.New().String(),
			Timestamp:   now.Add(time.Minute),
			Level:       models.LogLevelError,
			Message:     "Database connection timeout occurred",
			ServiceName: "db-service",
			AgentID:     "db-agent",
			Platform:    models.PlatformSwift,
		},
		{
			ID:          uuid.New().String(),
			Timestamp:   now.Add(2 * time.Minute),
			Level:       models.LogLevelWarn,
			Message:     "Memory usage is approaching threshold limits",
			ServiceName: "monitor-service",
			AgentID:     "monitor-agent",
			Platform:    models.PlatformGo,
		},
	}

	if err := storage.Store(ctx, logs); err != nil {
		t.Fatalf("Failed to store logs: %v", err)
	}

	// Test search functionality through storage interface
	result, err := storage.Query(ctx, models.LogFilter{
		MessageContains: "authentication",
	})
	if err != nil {
		t.Fatalf("Failed to query logs with search: %v", err)
	}

	if len(result.Logs) != 1 {
		t.Errorf("Expected 1 result for 'authentication' search, got %d", len(result.Logs))
	}

	if result.Logs[0].Message != "User login successful with authentication token" {
		t.Errorf("Unexpected message in search result: %s", result.Logs[0].Message)
	}

	// Test search with additional filters
	result, err = storage.Query(ctx, models.LogFilter{
		MessageContains: "connection",
		Level:           models.LogLevelError,
	})
	if err != nil {
		t.Fatalf("Failed to query logs with search and level filter: %v", err)
	}

	if len(result.Logs) != 1 {
		t.Errorf("Expected 1 result for 'connection' + ERROR level, got %d", len(result.Logs))
	}

	// Test fallback to SQL when no search term
	result, err = storage.Query(ctx, models.LogFilter{
		ServiceName: "monitor-service",
	})
	if err != nil {
		t.Fatalf("Failed to query logs with SQL fallback: %v", err)
	}

	if len(result.Logs) != 1 {
		t.Errorf("Expected 1 result for monitor-service, got %d", len(result.Logs))
	}
}

func TestSearchService_ReopenIndex(t *testing.T) {
	// Create temporary directory for index
	tmpDir, err := os.MkdirTemp("", "search_reopen_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	indexPath := filepath.Join(tmpDir, "test_index")

	// Create search service and index a log entry
	searchService, err := NewSearchService(indexPath)
	if err != nil {
		t.Fatalf("Failed to create search service: %v", err)
	}

	logEntry := models.LogEntry{
		ID:          uuid.New().String(),
		Timestamp:   time.Now(),
		Level:       models.LogLevelInfo,
		Message:     "Persistent test message",
		ServiceName: "test-service",
		AgentID:     "test-agent",
		Platform:    models.PlatformGo,
	}

	if err := searchService.IndexLogEntry(logEntry); err != nil {
		t.Fatalf("Failed to index log entry: %v", err)
	}

	searchService.Close()

	// Reopen the search service
	searchService, err = NewSearchService(indexPath)
	if err != nil {
		t.Fatalf("Failed to reopen search service: %v", err)
	}
	defer searchService.Close()

	// Verify the indexed entry is still there
	ctx := context.Background()
	logIDs, err := searchService.SearchLogs(ctx, "persistent", models.LogFilter{})
	if err != nil {
		t.Fatalf("Failed to search logs after reopen: %v", err)
	}

	if len(logIDs) != 1 {
		t.Errorf("Expected 1 result after reopen, got %d", len(logIDs))
	}

	if logIDs[0] != logEntry.ID {
		t.Errorf("Expected log ID %s after reopen, got %s", logEntry.ID, logIDs[0])
	}
}