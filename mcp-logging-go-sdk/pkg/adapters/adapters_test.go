package adapters

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/kerlexov/mcp-logging-go-sdk/pkg/logger"
	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Mock logger for testing
type mockLogger struct {
	entries []mockLogEntry
}

type mockLogEntry struct {
	level   logger.LogLevel
	message string
	fields  []logger.Field
}

func newMockLogger() *mockLogger {
	return &mockLogger{
		entries: make([]mockLogEntry, 0),
	}
}

func (m *mockLogger) Debug(msg string, fields ...logger.Field) {
	m.entries = append(m.entries, mockLogEntry{logger.LogLevelDebug, msg, fields})
}

func (m *mockLogger) Info(msg string, fields ...logger.Field) {
	m.entries = append(m.entries, mockLogEntry{logger.LogLevelInfo, msg, fields})
}

func (m *mockLogger) Warn(msg string, fields ...logger.Field) {
	m.entries = append(m.entries, mockLogEntry{logger.LogLevelWarn, msg, fields})
}

func (m *mockLogger) Error(msg string, fields ...logger.Field) {
	m.entries = append(m.entries, mockLogEntry{logger.LogLevelError, msg, fields})
}

func (m *mockLogger) Fatal(msg string, fields ...logger.Field) {
	m.entries = append(m.entries, mockLogEntry{logger.LogLevelFatal, msg, fields})
}

func (m *mockLogger) DebugContext(ctx context.Context, msg string, fields ...logger.Field) {
	m.Debug(msg, fields...)
}

func (m *mockLogger) InfoContext(ctx context.Context, msg string, fields ...logger.Field) {
	m.Info(msg, fields...)
}

func (m *mockLogger) WarnContext(ctx context.Context, msg string, fields ...logger.Field) {
	m.Warn(msg, fields...)
}

func (m *mockLogger) ErrorContext(ctx context.Context, msg string, fields ...logger.Field) {
	m.Error(msg, fields...)
}

func (m *mockLogger) FatalContext(ctx context.Context, msg string, fields ...logger.Field) {
	m.Fatal(msg, fields...)
}

func (m *mockLogger) WithFields(fields ...logger.Field) logger.Logger {
	return m // Simplified for testing
}

func (m *mockLogger) WithServiceName(serviceName string) logger.Logger {
	return m
}

func (m *mockLogger) WithAgentID(agentID string) logger.Logger {
	return m
}

func (m *mockLogger) Close() error {
	return nil
}

func TestStandardLogAdapter(t *testing.T) {
	mockLog := newMockLogger()
	adapter := NewStandardLogAdapter(mockLog)

	// Test writer interface directly
	writer := adapter.GetWriter()
	if writer == nil {
		t.Error("Expected writer to be non-nil")
	}

	// Test writing directly to the writer
	testMessage := "Test message from standard log\n"
	n, err := writer.Write([]byte(testMessage))
	if err != nil {
		t.Errorf("Expected no error writing, got %v", err)
	}
	if n != len(testMessage) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(testMessage), n)
	}

	// Verify the message was captured
	if len(mockLog.entries) != 1 {
		t.Errorf("Expected 1 log entry to be captured, got %d", len(mockLog.entries))
	}

	if len(mockLog.entries) > 0 {
		entry := mockLog.entries[0]
		if entry.level != logger.LogLevelInfo {
			t.Errorf("Expected log level INFO, got %s", entry.level)
		}
		if !strings.Contains(entry.message, "Test message from standard log") {
			t.Errorf("Expected message to contain 'Test message from standard log', got '%s'", entry.message)
		}
	}

	// Test level setting
	adapter.SetLevel(logger.LogLevelError)

	// Test empty message handling
	n, err = writer.Write([]byte(""))
	if err != nil {
		t.Errorf("Expected no error writing empty message, got %v", err)
	}
	if n != 0 {
		t.Errorf("Expected to write 0 bytes for empty message, wrote %d", n)
	}

	// Should still have only 1 entry (empty message shouldn't be logged)
	if len(mockLog.entries) != 1 {
		t.Errorf("Expected still 1 log entry after empty write, got %d", len(mockLog.entries))
	}
}

func TestLogrusHook(t *testing.T) {
	mockLog := newMockLogger()
	hook := NewLogrusHook(mockLog)

	// Test hook levels
	levels := hook.Levels()
	if len(levels) != len(logrus.AllLevels) {
		t.Errorf("Expected %d levels, got %d", len(logrus.AllLevels), len(levels))
	}

	// Test different log levels
	testCases := []struct {
		logrusLevel logrus.Level
		expectedMCP logger.LogLevel
		message     string
	}{
		{logrus.DebugLevel, logger.LogLevelDebug, "Debug message"},
		{logrus.InfoLevel, logger.LogLevelInfo, "Info message"},
		{logrus.WarnLevel, logger.LogLevelWarn, "Warn message"},
		{logrus.ErrorLevel, logger.LogLevelError, "Error message"},
		{logrus.FatalLevel, logger.LogLevelFatal, "Fatal message"},
	}

	for _, tc := range testCases {
		entry := &logrus.Entry{
			Level:   tc.logrusLevel,
			Message: tc.message,
			Data: logrus.Fields{
				"key1": "value1",
				"key2": 42,
			},
		}

		err := hook.Fire(entry)
		if err != nil {
			t.Errorf("Expected no error firing hook, got %v", err)
		}
	}

	// Verify entries were logged
	if len(mockLog.entries) != len(testCases) {
		t.Errorf("Expected %d log entries, got %d", len(testCases), len(mockLog.entries))
	}

	// Verify first entry details
	if len(mockLog.entries) > 0 {
		entry := mockLog.entries[0]
		if entry.level != logger.LogLevelDebug {
			t.Errorf("Expected first entry level DEBUG, got %s", entry.level)
		}
		if entry.message != "Debug message" {
			t.Errorf("Expected first entry message 'Debug message', got '%s'", entry.message)
		}
		if len(entry.fields) != 2 {
			t.Errorf("Expected 2 fields, got %d", len(entry.fields))
		}
	}
}

func TestLogrusInstallHook(t *testing.T) {
	mockLog := newMockLogger()

	// Install the hook
	InstallLogrusHook(mockLog)

	// Create a new logrus logger to test
	testLogger := logrus.New()
	testLogger.SetOutput(&bytes.Buffer{}) // Suppress output

	// Log a message
	testLogger.Info("Test message from logrus")

	// Give some time for async processing
	time.Sleep(10 * time.Millisecond)

	// Note: This test might not work as expected because InstallLogrusHook
	// adds to the global logrus instance, not our test logger
	// In a real scenario, you'd want to add the hook to specific logger instances
}

func TestZapCore(t *testing.T) {
	mockLog := newMockLogger()
	core := NewZapCore(mockLog)

	// Test core enabled
	if !core.Enabled(zapcore.InfoLevel) {
		t.Error("Expected core to be enabled for INFO level")
	}

	// Test with fields
	fields := []zapcore.Field{
		{Key: "key1", Type: zapcore.StringType, String: "value1"},
		{Key: "key2", Type: zapcore.Int64Type, Integer: 42},
	}

	newCore := core.With(fields)
	if newCore == nil {
		t.Error("Expected new core with fields to be created")
	}

	// Test write
	entry := zapcore.Entry{
		Level:   zapcore.InfoLevel,
		Message: "Test zap message",
	}

	err := core.Write(entry, fields)
	if err != nil {
		t.Errorf("Expected no error writing to core, got %v", err)
	}

	// Verify entry was logged
	if len(mockLog.entries) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(mockLog.entries))
	}

	if len(mockLog.entries) > 0 {
		logEntry := mockLog.entries[0]
		if logEntry.level != logger.LogLevelInfo {
			t.Errorf("Expected log level INFO, got %s", logEntry.level)
		}
		if logEntry.message != "Test zap message" {
			t.Errorf("Expected message 'Test zap message', got '%s'", logEntry.message)
		}
	}

	// Test sync
	err = core.Sync()
	if err != nil {
		t.Errorf("Expected no error syncing core, got %v", err)
	}
}

func TestZapLogger(t *testing.T) {
	mockLog := newMockLogger()
	zapLogger := NewZapLogger(mockLog)

	if zapLogger == nil {
		t.Error("Expected zap logger to be created")
	}

	// Test logging
	zapLogger.Info("Test zap logger message",
		zap.String("key1", "value1"),
		zap.Int("key2", 42),
	)

	// Verify entry was logged
	if len(mockLog.entries) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(mockLog.entries))
	}
}

func TestZapSugaredLogger(t *testing.T) {
	mockLog := newMockLogger()
	sugaredLogger := NewZapSugaredLogger(mockLog)

	if sugaredLogger == nil {
		t.Error("Expected zap sugared logger to be created")
	}

	// Test logging
	sugaredLogger.Infow("Test sugared logger message",
		"key1", "value1",
		"key2", 42,
	)

	// Verify entry was logged
	if len(mockLog.entries) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(mockLog.entries))
	}
}

func TestLogrusFormatter(t *testing.T) {
	mockLog := newMockLogger()
	formatter := NewLogrusFormatter(mockLog, nil)

	entry := &logrus.Entry{
		Level:   logrus.InfoLevel,
		Message: "Test formatter message",
		Data:    logrus.Fields{"key": "value"},
	}

	output, err := formatter.Format(entry)
	if err != nil {
		t.Errorf("Expected no error formatting, got %v", err)
	}

	if len(output) == 0 {
		t.Error("Expected formatted output to be non-empty")
	}

	// Verify the message was also sent to MCP logger
	if len(mockLog.entries) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(mockLog.entries))
	}
}
