package adapters

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/your-org/mcp-logging-go-sdk/pkg/logger"
)

type LogrusHook struct {
	mcpLogger logger.Logger
}

func NewLogrusHook(mcpLogger logger.Logger) *LogrusHook {
	return &LogrusHook{
		mcpLogger: mcpLogger,
	}
}

func (hook *LogrusHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (hook *LogrusHook) Fire(entry *logrus.Entry) error {
	var level logger.LogLevel
	switch entry.Level {
	case logrus.TraceLevel, logrus.DebugLevel:
		level = logger.LogLevelDebug
	case logrus.InfoLevel:
		level = logger.LogLevelInfo
	case logrus.WarnLevel:
		level = logger.LogLevelWarn
	case logrus.ErrorLevel:
		level = logger.LogLevelError
	case logrus.FatalLevel, logrus.PanicLevel:
		level = logger.LogLevelFatal
	default:
		level = logger.LogLevelInfo
	}

	fields := make([]logger.Field, 0, len(entry.Data))
	for key, value := range entry.Data {
		fields = append(fields, logger.Field{
			Key:   key,
			Value: value,
		})
	}

	switch level {
	case logger.LogLevelDebug:
		hook.mcpLogger.Debug(entry.Message, fields...)
	case logger.LogLevelInfo:
		hook.mcpLogger.Info(entry.Message, fields...)
	case logger.LogLevelWarn:
		hook.mcpLogger.Warn(entry.Message, fields...)
	case logger.LogLevelError:
		hook.mcpLogger.Error(entry.Message, fields...)
	case logger.LogLevelFatal:
		hook.mcpLogger.Fatal(entry.Message, fields...)
	}

	return nil
}

func InstallLogrusHook(mcpLogger logger.Logger) {
	hook := NewLogrusHook(mcpLogger)
	logrus.AddHook(hook)
}

type LogrusFormatter struct {
	mcpLogger logger.Logger
	original  logrus.Formatter
}

func NewLogrusFormatter(mcpLogger logger.Logger, original logrus.Formatter) *LogrusFormatter {
	return &LogrusFormatter{
		mcpLogger: mcpLogger,
		original:  original,
	}
}

func (f *LogrusFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	hook := NewLogrusHook(f.mcpLogger)
	hook.Fire(entry)

	if f.original != nil {
		return f.original.Format(entry)
	}

	return []byte(fmt.Sprintf("[%s] %s\n", entry.Level.String(), entry.Message)), nil
}
