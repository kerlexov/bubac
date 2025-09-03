package logger

import (
	"context"
	"io"
)

type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)

	DebugContext(ctx context.Context, msg string, fields ...Field)
	InfoContext(ctx context.Context, msg string, fields ...Field)
	WarnContext(ctx context.Context, msg string, fields ...Field)
	ErrorContext(ctx context.Context, msg string, fields ...Field)
	FatalContext(ctx context.Context, msg string, fields ...Field)

	WithFields(fields ...Field) Logger
	WithServiceName(serviceName string) Logger
	WithAgentID(agentID string) Logger

	Close() error
}

type Sender interface {
	Send(ctx context.Context, entries []LogEntry) error
	Close() error
}

type Buffer interface {
	Add(entry LogEntry) error
	Flush() ([]LogEntry, error)
	Size() int
	IsFull() bool
	Close() error
}

type Writer interface {
	io.Writer
}

type HealthChecker interface {
	HealthCheck(ctx context.Context) error
}
