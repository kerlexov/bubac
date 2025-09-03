package adapters

import (
	"github.com/your-org/mcp-logging-go-sdk/pkg/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type ZapCore struct {
	mcpLogger logger.Logger
	encoder   zapcore.Encoder
	level     zapcore.LevelEnabler
	output    zapcore.WriteSyncer
}

func NewZapCore(mcpLogger logger.Logger) zapcore.Core {
	return &ZapCore{
		mcpLogger: mcpLogger,
		encoder:   zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		level:     zapcore.InfoLevel,
	}
}

func (zc *ZapCore) Enabled(level zapcore.Level) bool {
	return zc.level.Enabled(level)
}

func (zc *ZapCore) With(fields []zapcore.Field) zapcore.Core {
	mcpFields := make([]logger.Field, len(fields))
	for i, field := range fields {
		mcpFields[i] = logger.Field{
			Key:   field.Key,
			Value: field.Interface,
		}
	}

	return &ZapCore{
		mcpLogger: zc.mcpLogger.WithFields(mcpFields...),
		encoder:   zc.encoder,
		level:     zc.level,
		output:    zc.output,
	}
}

func (zc *ZapCore) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if zc.Enabled(entry.Level) {
		return checked.AddCore(entry, zc)
	}
	return checked
}

func (zc *ZapCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	var mcpLevel logger.LogLevel
	switch entry.Level {
	case zapcore.DebugLevel:
		mcpLevel = logger.LogLevelDebug
	case zapcore.InfoLevel:
		mcpLevel = logger.LogLevelInfo
	case zapcore.WarnLevel:
		mcpLevel = logger.LogLevelWarn
	case zapcore.ErrorLevel:
		mcpLevel = logger.LogLevelError
	case zapcore.FatalLevel, zapcore.PanicLevel:
		mcpLevel = logger.LogLevelFatal
	default:
		mcpLevel = logger.LogLevelInfo
	}

	mcpFields := make([]logger.Field, len(fields))
	for i, field := range fields {
		mcpFields[i] = logger.Field{
			Key:   field.Key,
			Value: field.Interface,
		}
	}

	switch mcpLevel {
	case logger.LogLevelDebug:
		zc.mcpLogger.Debug(entry.Message, mcpFields...)
	case logger.LogLevelInfo:
		zc.mcpLogger.Info(entry.Message, mcpFields...)
	case logger.LogLevelWarn:
		zc.mcpLogger.Warn(entry.Message, mcpFields...)
	case logger.LogLevelError:
		zc.mcpLogger.Error(entry.Message, mcpFields...)
	case logger.LogLevelFatal:
		zc.mcpLogger.Fatal(entry.Message, mcpFields...)
	}

	return nil
}

func (zc *ZapCore) Sync() error {
	return nil
}

func NewZapLogger(mcpLogger logger.Logger) *zap.Logger {
	core := NewZapCore(mcpLogger)
	return zap.New(core)
}

func NewZapSugaredLogger(mcpLogger logger.Logger) *zap.SugaredLogger {
	return NewZapLogger(mcpLogger).Sugar()
}
