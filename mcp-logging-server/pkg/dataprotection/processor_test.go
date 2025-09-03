package dataprotection

import (
	"testing"
	"time"

	"github.com/kerlexov/mcp-logging-server/pkg/models"
)

func TestDataProtectionProcessor_ProcessLogEntry(t *testing.T) {
	config := &DataProtectionConfig{
		Enabled:  true,
		MaskChar: "*",
		HashSalt: "test-salt",
		FieldRules: []FieldRule{
			{Field: "password", Action: ActionMask},
			{Field: "credit_card", Action: ActionHash},
			{Field: "internal_id", Action: ActionDrop},
		},
		AuditEnabled: false, // Disable for testing
	}

	processor, err := NewDataProtectionProcessor(config)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	logEntry := &models.LogEntry{
		ID:          "test-id",
		Timestamp:   time.Now(),
		Level:       models.LogLevelInfo,
		Message:     "User login attempt",
		ServiceName: "auth-service",
		AgentID:     "agent-001",
		Platform:    models.PlatformGo,
		Metadata: map[string]interface{}{
			"password":    "secret123",
			"credit_card": "4111111111111111",
			"internal_id": "internal-12345",
			"username":    "john.doe",
		},
	}

	err = processor.ProcessLogEntry(logEntry)
	if err != nil {
		t.Fatalf("Failed to process log entry: %v", err)
	}

	// Check that password was masked
	if password, exists := logEntry.Metadata["password"]; exists {
		if password == "secret123" {
			t.Error("Password should have been masked")
		}
		if passwordStr, ok := password.(string); ok {
			if passwordStr != "se****23" {
				t.Errorf("Expected password to be 'se****23', got '%s'", passwordStr)
			}
		}
	}

	// Check that credit card was hashed
	if creditCard, exists := logEntry.Metadata["credit_card"]; exists {
		if creditCard == "4111111111111111" {
			t.Error("Credit card should have been hashed")
		}
		if ccStr, ok := creditCard.(string); ok {
			if ccStr[:7] != "sha256:" {
				t.Error("Credit card should be hashed with sha256 prefix")
			}
		}
	}

	// Check that internal_id was dropped
	if _, exists := logEntry.Metadata["internal_id"]; exists {
		t.Error("internal_id should have been dropped")
	}

	// Check that username was not modified
	if username, exists := logEntry.Metadata["username"]; exists {
		if username != "john.doe" {
			t.Error("Username should not have been modified")
		}
	}
}

func TestDataProtectionProcessor_MaskValue(t *testing.T) {
	config := &DataProtectionConfig{
		Enabled:  true,
		MaskChar: "*",
	}

	processor, err := NewDataProtectionProcessor(config)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	testCases := []struct {
		input    string
		expected string
	}{
		{"a", "*"},
		{"ab", "**"},
		{"abc", "***"},
		{"abcd", "****"},
		{"abcde", "ab*de"},
		{"password123", "pa******23"},
		{"verylongpassword", "ve************rd"},
	}

	for _, tc := range testCases {
		result := processor.maskValue("test", tc.input)
		if result != tc.expected {
			t.Errorf("Expected '%s' to be masked as '%s', got '%s'", tc.input, tc.expected, result)
		}
	}
}

func TestDataProtectionProcessor_HashValue(t *testing.T) {
	config := &DataProtectionConfig{
		Enabled:  true,
		HashSalt: "test-salt",
	}

	processor, err := NewDataProtectionProcessor(config)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	value := "sensitive-data"
	hash1 := processor.hashValue(value)
	hash2 := processor.hashValue(value)

	// Same value should produce same hash
	if hash1 != hash2 {
		t.Error("Same value should produce same hash")
	}

	// Hash should have sha256 prefix
	if hash1[:7] != "sha256:" {
		t.Error("Hash should have sha256 prefix")
	}

	// Different values should produce different hashes
	hash3 := processor.hashValue("different-data")
	if hash1 == hash3 {
		t.Error("Different values should produce different hashes")
	}
}

func TestDataProtectionProcessor_ProcessMessageContent(t *testing.T) {
	config := &DataProtectionConfig{
		Enabled:      true,
		MaskChar:     "*",
		AuditEnabled: false,
	}

	processor, err := NewDataProtectionProcessor(config)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	message := "User john.doe@example.com logged in with credit card 4111-1111-1111-1111 from IP 192.168.1.1"
	processedMessage, actions := processor.processMessageContent(message)

	// Check that sensitive data was masked
	if processedMessage == message {
		t.Error("Message should have been processed")
	}

	// Should have detected and masked email, credit card, and IP
	if len(actions) == 0 {
		t.Error("Should have detected sensitive patterns")
	}

	// Check that email was masked
	if processedMessage == message {
		t.Error("Email should have been masked in message")
	}
}

func TestDataProtectionProcessor_Disabled(t *testing.T) {
	config := &DataProtectionConfig{
		Enabled: false,
	}

	processor, err := NewDataProtectionProcessor(config)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	logEntry := &models.LogEntry{
		ID:          "test-id",
		Timestamp:   time.Now(),
		Level:       models.LogLevelInfo,
		Message:     "Test message",
		ServiceName: "test-service",
		AgentID:     "agent-001",
		Platform:    models.PlatformGo,
		Metadata: map[string]interface{}{
			"password": "secret123",
		},
	}

	originalPassword := logEntry.Metadata["password"]

	err = processor.ProcessLogEntry(logEntry)
	if err != nil {
		t.Fatalf("Failed to process log entry: %v", err)
	}

	// Password should not have been modified when disabled
	if logEntry.Metadata["password"] != originalPassword {
		t.Error("Password should not have been modified when data protection is disabled")
	}
}

func TestDataProtectionProcessor_PatternMatching(t *testing.T) {
	config := &DataProtectionConfig{
		Enabled:  true,
		MaskChar: "*",
		FieldRules: []FieldRule{
			{
				Field:   "email",
				Action:  ActionMask,
				Pattern: `([^@]+)@(.+)`, // Mask username part of email
			},
		},
		AuditEnabled: false,
	}

	processor, err := NewDataProtectionProcessor(config)
	if err != nil {
		t.Fatalf("Failed to create processor: %v", err)
	}

	logEntry := &models.LogEntry{
		ID:          "test-id",
		Timestamp:   time.Now(),
		Level:       models.LogLevelInfo,
		Message:     "Test message",
		ServiceName: "test-service",
		AgentID:     "agent-001",
		Platform:    models.PlatformGo,
		Metadata: map[string]interface{}{
			"email": "john.doe@example.com",
		},
	}

	err = processor.ProcessLogEntry(logEntry)
	if err != nil {
		t.Fatalf("Failed to process log entry: %v", err)
	}

	// Email should have username part masked
	if email, exists := logEntry.Metadata["email"]; exists {
		if emailStr, ok := email.(string); ok {
			if emailStr == "john.doe@example.com" {
				t.Error("Email should have been partially masked")
			}
			// Should still contain @example.com
			if emailStr[len(emailStr)-12:] != "@example.com" {
				t.Error("Email domain should be preserved")
			}
		}
	}
}
