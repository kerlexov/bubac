package adapters

import (
	"io"
	"log"
	"strings"

	"github.com/kerlexov/mcp-logging-go-sdk/pkg/logger"
)

type StandardLogAdapter struct {
	mcpLogger logger.Logger
	writer    *logWriter
}

type logWriter struct {
	mcpLogger logger.Logger
	level     logger.LogLevel
}

func NewStandardLogAdapter(mcpLogger logger.Logger) *StandardLogAdapter {
	writer := &logWriter{
		mcpLogger: mcpLogger,
		level:     logger.LogLevelInfo,
	}

	adapter := &StandardLogAdapter{
		mcpLogger: mcpLogger,
		writer:    writer,
	}

	log.SetOutput(writer)

	return adapter
}

func (w *logWriter) Write(p []byte) (n int, err error) {
	message := strings.TrimSpace(string(p))
	if message == "" {
		return len(p), nil
	}

	message = strings.TrimPrefix(message, log.Prefix())

	w.mcpLogger.Info(message)

	return len(p), nil
}

func (a *StandardLogAdapter) SetLevel(level logger.LogLevel) {
	a.writer.level = level
}

func (a *StandardLogAdapter) GetWriter() io.Writer {
	return a.writer
}
