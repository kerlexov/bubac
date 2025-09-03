package logger

import (
	"context"
	"runtime"
	"strings"
	"sync"
	"time"
)

type mcpLogger struct {
	config        Config
	sender        Sender
	buffer        *memoryBuffer
	defaultFields map[string]interface{}
	mu            sync.RWMutex
	closed        bool
	stopCh        chan struct{}
	wg            sync.WaitGroup
}

func New(config Config) (Logger, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	sender := NewHTTPSender(config.ServerURL, config.HTTPTimeout)
	buffer := newMemoryBuffer(config.BufferSize)

	logger := &mcpLogger{
		config:        config,
		sender:        sender,
		buffer:        buffer,
		defaultFields: make(map[string]interface{}),
		stopCh:        make(chan struct{}),
	}

	logger.startFlushWorker()

	if config.EnableHealthCheck {
		logger.startHealthChecker()
	}

	return logger, nil
}

func (l *mcpLogger) Debug(msg string, fields ...Field) {
	l.log(LogLevelDebug, msg, fields...)
}

func (l *mcpLogger) Info(msg string, fields ...Field) {
	l.log(LogLevelInfo, msg, fields...)
}

func (l *mcpLogger) Warn(msg string, fields ...Field) {
	l.log(LogLevelWarn, msg, fields...)
}

func (l *mcpLogger) Error(msg string, fields ...Field) {
	l.log(LogLevelError, msg, fields...)
}

func (l *mcpLogger) Fatal(msg string, fields ...Field) {
	l.log(LogLevelFatal, msg, fields...)
}

func (l *mcpLogger) DebugContext(ctx context.Context, msg string, fields ...Field) {
	l.logContext(ctx, LogLevelDebug, msg, fields...)
}

func (l *mcpLogger) InfoContext(ctx context.Context, msg string, fields ...Field) {
	l.logContext(ctx, LogLevelInfo, msg, fields...)
}

func (l *mcpLogger) WarnContext(ctx context.Context, msg string, fields ...Field) {
	l.logContext(ctx, LogLevelWarn, msg, fields...)
}

func (l *mcpLogger) ErrorContext(ctx context.Context, msg string, fields ...Field) {
	l.logContext(ctx, LogLevelError, msg, fields...)
}

func (l *mcpLogger) FatalContext(ctx context.Context, msg string, fields ...Field) {
	l.logContext(ctx, LogLevelFatal, msg, fields...)
}

func (l *mcpLogger) WithFields(fields ...Field) Logger {
	newLogger := &mcpLogger{
		config:        l.config,
		sender:        l.sender,
		buffer:        l.buffer,
		defaultFields: make(map[string]interface{}),
		stopCh:        l.stopCh,
	}

	l.mu.RLock()
	for k, v := range l.defaultFields {
		newLogger.defaultFields[k] = v
	}
	l.mu.RUnlock()

	for _, field := range fields {
		newLogger.defaultFields[field.Key] = field.Value
	}

	return newLogger
}

func (l *mcpLogger) WithServiceName(serviceName string) Logger {
	return l.WithFields(Field{Key: "service_name_override", Value: serviceName})
}

func (l *mcpLogger) WithAgentID(agentID string) Logger {
	return l.WithFields(Field{Key: "agent_id_override", Value: agentID})
}

func (l *mcpLogger) Close() error {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil
	}
	l.closed = true
	l.mu.Unlock()

	close(l.stopCh)
	l.wg.Wait()

	if l.buffer != nil {
		entries, err := l.buffer.Flush()
		if err == nil && len(entries) > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			l.sender.Send(ctx, entries)
		}
		l.buffer.Close()
	}

	if l.sender != nil {
		l.sender.Close()
	}

	return nil
}

func (l *mcpLogger) log(level LogLevel, msg string, fields ...Field) {
	l.logContext(context.Background(), level, msg, fields...)
}

func (l *mcpLogger) logContext(ctx context.Context, level LogLevel, msg string, fields ...Field) {
	l.mu.RLock()
	if l.closed {
		l.mu.RUnlock()
		return
	}
	l.mu.RUnlock()

	metadata := make(map[string]interface{})

	l.mu.RLock()
	for k, v := range l.defaultFields {
		metadata[k] = v
	}
	l.mu.RUnlock()

	for _, field := range fields {
		metadata[field.Key] = field.Value
	}

	serviceName := l.config.ServiceName
	if override, ok := metadata["service_name_override"].(string); ok {
		serviceName = override
		delete(metadata, "service_name_override")
	}

	agentID := l.config.AgentID
	if override, ok := metadata["agent_id_override"].(string); ok {
		agentID = override
		delete(metadata, "agent_id_override")
	}

	entry := LogEntry{
		ID:             generateID(),
		Timestamp:      time.Now().UTC(),
		Level:          level,
		Message:        msg,
		ServiceName:    serviceName,
		AgentID:        agentID,
		Platform:       "go",
		Metadata:       metadata,
		SourceLocation: l.getSourceLocation(),
	}

	if err := l.buffer.Add(entry); err != nil {
		return
	}
}

func (l *mcpLogger) getSourceLocation() *SourceLocation {
	pc, file, line, ok := runtime.Caller(4)
	if !ok {
		return nil
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return nil
	}

	parts := strings.Split(file, "/")
	if len(parts) > 0 {
		file = parts[len(parts)-1]
	}

	return &SourceLocation{
		File:     file,
		Line:     line,
		Function: fn.Name(),
	}
}

func (l *mcpLogger) startFlushWorker() {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		ticker := time.NewTicker(l.config.FlushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				l.flush()
			case <-l.stopCh:
				return
			}
		}
	}()
}

func (l *mcpLogger) startHealthChecker() {
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		ticker := time.NewTicker(l.config.HealthCheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if hc, ok := l.sender.(HealthChecker); ok {
					hc.HealthCheck(ctx)
				}
				cancel()
			case <-l.stopCh:
				return
			}
		}
	}()
}

func (l *mcpLogger) flush() {
	entries, err := l.buffer.Flush()
	if err != nil || len(entries) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), l.config.HTTPTimeout)
	defer cancel()

	if err := l.sender.Send(ctx, entries); err != nil {
		for _, entry := range entries {
			l.buffer.Add(entry)
		}
	}
}
