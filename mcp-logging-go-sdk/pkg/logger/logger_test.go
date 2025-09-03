package logger

import (
	"context"
	"testing"
	"time"
)

func TestNewLogger(t *testing.T) {
	config := DefaultConfig()
	config.ServiceName = "test-service"
	config.AgentID = "test-agent"

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if logger == nil {
		t.Fatal("Expected logger to be created")
	}

	defer logger.Close()
}

func TestLoggerBasicLogging(t *testing.T) {
	config := DefaultConfig()
	config.ServiceName = "test-service"
	config.AgentID = "test-agent"
	config.BufferSize = 10
	config.FlushInterval = 100 * time.Millisecond

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer logger.Close()

	logger.Debug("Debug message")
	logger.Info("Info message")
	logger.Warn("Warn message")
	logger.Error("Error message")

	time.Sleep(200 * time.Millisecond)
}

func TestLoggerWithFields(t *testing.T) {
	config := DefaultConfig()
	config.ServiceName = "test-service"
	config.AgentID = "test-agent"

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer logger.Close()

	logger.Info("Test message",
		Field{Key: "key1", Value: "value1"},
		Field{Key: "key2", Value: 42},
	)

	contextLogger := logger.WithFields(
		Field{Key: "module", Value: "test"},
		Field{Key: "function", Value: "TestLoggerWithFields"},
	)

	contextLogger.Info("Context message")
}

func TestLoggerContextMethods(t *testing.T) {
	config := DefaultConfig()
	config.ServiceName = "test-service"
	config.AgentID = "test-agent"

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer logger.Close()

	ctx := context.Background()

	logger.DebugContext(ctx, "Debug with context")
	logger.InfoContext(ctx, "Info with context")
	logger.WarnContext(ctx, "Warn with context")
	logger.ErrorContext(ctx, "Error with context")
}

func TestLoggerServiceAndAgentOverrides(t *testing.T) {
	config := DefaultConfig()
	config.ServiceName = "test-service"
	config.AgentID = "test-agent"

	logger, err := New(config)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer logger.Close()

	serviceLogger := logger.WithServiceName("override-service")
	serviceLogger.Info("Message with service override")

	agentLogger := logger.WithAgentID("override-agent")
	agentLogger.Info("Message with agent override")
}
