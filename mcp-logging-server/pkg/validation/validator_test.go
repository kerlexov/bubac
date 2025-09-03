package validation

import (
	"testing"
	"time"

	"github.com/your-org/mcp-logging-server/pkg/models"
)

func createValidLogEntry() models.LogEntry {
	return models.LogEntry{
		ID:          "550e8400-e29b-41d4-a716-446655440000",
		Timestamp:   time.Now(),
		Level:       models.LogLevelInfo,
		Message:     "Test message",
		ServiceName: "test-service",
		AgentID:     "test-agent",
		Platform:    models.PlatformGo,
	}
}

func TestLogValidator_ValidateLogEntry(t *testing.T) {
	validator := NewLogValidator()
	
	tests := []struct {
		name        string
		entry       models.LogEntry
		expectValid bool
		expectError string
	}{
		{
			name:        "valid entry",
			entry:       createValidLogEntry(),
			expectValid: true,
		},
		{
			name: "missing required field",
			entry: models.LogEntry{
				ID:        "550e8400-e29b-41d4-a716-446655440000",
				Timestamp: time.Now(),
				Level:     models.LogLevelInfo,
				Message:   "Test message",
				// Missing ServiceName
				AgentID:  "test-agent",
				Platform: models.PlatformGo,
			},
			expectValid: false,
			expectError: "ServiceName",
		},
		{
			name: "invalid UUID",
			entry: models.LogEntry{
				ID:          "invalid-uuid",
				Timestamp:   time.Now(),
				Level:       models.LogLevelInfo,
				Message:     "Test message",
				ServiceName: "test-service",
				AgentID:     "test-agent",
				Platform:    models.PlatformGo,
			},
			expectValid: false,
			expectError: "ID",
		},
		{
			name: "invalid service name",
			entry: models.LogEntry{
				ID:          "550e8400-e29b-41d4-a716-446655440000",
				Timestamp:   time.Now(),
				Level:       models.LogLevelInfo,
				Message:     "Test message",
				ServiceName: "test service!", // Invalid characters
				AgentID:     "test-agent",
				Platform:    models.PlatformGo,
			},
			expectValid: false,
			expectError: "ServiceName",
		},
		{
			name: "future timestamp",
			entry: models.LogEntry{
				ID:          "550e8400-e29b-41d4-a716-446655440000",
				Timestamp:   time.Now().Add(10 * time.Minute), // Too far in future
				Level:       models.LogLevelInfo,
				Message:     "Test message",
				ServiceName: "test-service",
				AgentID:     "test-agent",
				Platform:    models.PlatformGo,
			},
			expectValid: false,
			expectError: "timestamp",
		},
		{
			name: "very old timestamp",
			entry: models.LogEntry{
				ID:          "550e8400-e29b-41d4-a716-446655440000",
				Timestamp:   time.Now().Add(-400 * 24 * time.Hour), // More than 1 year old
				Level:       models.LogLevelInfo,
				Message:     "Test message",
				ServiceName: "test-service",
				AgentID:     "test-agent",
				Platform:    models.PlatformGo,
			},
			expectValid: false,
			expectError: "timestamp",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateLogEntry(&tt.entry)
			
			if result.IsValid != tt.expectValid {
				t.Errorf("Expected IsValid=%v, got %v", tt.expectValid, result.IsValid)
			}
			
			if !tt.expectValid && tt.expectError != "" {
				found := false
				for _, err := range result.Errors {
					if err.Field == tt.expectError {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error for field %s, but not found in errors: %v", tt.expectError, result.Errors)
				}
			}
		})
	}
}

func TestLogValidator_ValidateLogBatch(t *testing.T) {
	validator := NewLogValidator()
	
	validEntry1 := createValidLogEntry()
	validEntry1.ID = "550e8400-e29b-41d4-a716-446655440001"
	
	validEntry2 := createValidLogEntry()
	validEntry2.ID = "550e8400-e29b-41d4-a716-446655440002"
	
	invalidEntry := models.LogEntry{
		ID:        "invalid-uuid",
		Timestamp: time.Now(),
		Level:     models.LogLevelInfo,
		Message:   "Test message",
		// Missing required fields
	}
	
	tests := []struct {
		name            string
		entries         []models.LogEntry
		expectedValid   int
		expectedInvalid int
	}{
		{
			name:            "all valid entries",
			entries:         []models.LogEntry{validEntry1, validEntry2},
			expectedValid:   2,
			expectedInvalid: 0,
		},
		{
			name:            "mixed valid and invalid entries",
			entries:         []models.LogEntry{validEntry1, invalidEntry, validEntry2},
			expectedValid:   2,
			expectedInvalid: 1,
		},
		{
			name:            "all invalid entries",
			entries:         []models.LogEntry{invalidEntry},
			expectedValid:   0,
			expectedInvalid: 1,
		},
		{
			name:            "empty batch",
			entries:         []models.LogEntry{},
			expectedValid:   0,
			expectedInvalid: 0,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateLogBatch(tt.entries)
			
			if result.ValidCount != tt.expectedValid {
				t.Errorf("Expected %d valid entries, got %d", tt.expectedValid, result.ValidCount)
			}
			
			if result.InvalidCount != tt.expectedInvalid {
				t.Errorf("Expected %d invalid entries, got %d", tt.expectedInvalid, result.InvalidCount)
			}
			
			if result.TotalEntries != len(tt.entries) {
				t.Errorf("Expected total entries %d, got %d", len(tt.entries), result.TotalEntries)
			}
		})
	}
}

func TestCustomValidators(t *testing.T) {
	validator := NewLogValidator()
	
	tests := []struct {
		name        string
		entry       models.LogEntry
		expectValid bool
	}{
		{
			name: "valid service name with hyphens",
			entry: models.LogEntry{
				ID:          "550e8400-e29b-41d4-a716-446655440000",
				Timestamp:   time.Now(),
				Level:       models.LogLevelInfo,
				Message:     "Test message",
				ServiceName: "test-service-name",
				AgentID:     "test-agent",
				Platform:    models.PlatformGo,
			},
			expectValid: true,
		},
		{
			name: "valid service name with underscores",
			entry: models.LogEntry{
				ID:          "550e8400-e29b-41d4-a716-446655440000",
				Timestamp:   time.Now(),
				Level:       models.LogLevelInfo,
				Message:     "Test message",
				ServiceName: "test_service_name",
				AgentID:     "test-agent",
				Platform:    models.PlatformGo,
			},
			expectValid: true,
		},
		{
			name: "invalid service name with spaces",
			entry: models.LogEntry{
				ID:          "550e8400-e29b-41d4-a716-446655440000",
				Timestamp:   time.Now(),
				Level:       models.LogLevelInfo,
				Message:     "Test message",
				ServiceName: "test service name",
				AgentID:     "test-agent",
				Platform:    models.PlatformGo,
			},
			expectValid: false,
		},
		{
			name: "invalid service name with special characters",
			entry: models.LogEntry{
				ID:          "550e8400-e29b-41d4-a716-446655440000",
				Timestamp:   time.Now(),
				Level:       models.LogLevelInfo,
				Message:     "Test message",
				ServiceName: "test@service#name",
				AgentID:     "test-agent",
				Platform:    models.PlatformGo,
			},
			expectValid: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validator.ValidateLogEntry(&tt.entry)
			
			if result.IsValid != tt.expectValid {
				t.Errorf("Expected IsValid=%v, got %v. Errors: %v", tt.expectValid, result.IsValid, result.Errors)
			}
		})
	}
}