package logger

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.ServerURL != "http://localhost:8080" {
		t.Errorf("Expected default server URL, got %s", config.ServerURL)
	}

	if config.BufferSize != 1000 {
		t.Errorf("Expected default buffer size 1000, got %d", config.BufferSize)
	}

	if config.FlushInterval != 5*time.Second {
		t.Errorf("Expected default flush interval 5s, got %v", config.FlushInterval)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name:        "Valid config",
			config:      Config{ServerURL: "http://localhost:8080", ServiceName: "test", AgentID: "agent1"},
			expectError: false,
		},
		{
			name:        "Missing server URL",
			config:      Config{ServiceName: "test", AgentID: "agent1"},
			expectError: true,
		},
		{
			name:        "Missing service name",
			config:      Config{ServerURL: "http://localhost:8080", AgentID: "agent1"},
			expectError: true,
		},
		{
			name:        "Missing agent ID",
			config:      Config{ServerURL: "http://localhost:8080", ServiceName: "test"},
			expectError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.config.Validate()
			if test.expectError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !test.expectError && err != nil {
				t.Errorf("Expected no validation error, got %v", err)
			}
		})
	}
}

func TestConfigDefaults(t *testing.T) {
	config := Config{
		ServerURL:   "http://localhost:8080",
		ServiceName: "test",
		AgentID:     "agent1",
	}

	err := config.Validate()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if config.BufferSize != 1000 {
		t.Errorf("Expected default buffer size to be set to 1000, got %d", config.BufferSize)
	}

	if config.FlushInterval != 5*time.Second {
		t.Errorf("Expected default flush interval to be set to 5s, got %v", config.FlushInterval)
	}

	if config.HTTPTimeout != 10*time.Second {
		t.Errorf("Expected default HTTP timeout to be set to 10s, got %v", config.HTTPTimeout)
	}
}
