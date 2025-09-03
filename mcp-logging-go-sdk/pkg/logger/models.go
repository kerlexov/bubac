package logger

import (
	"time"
)

type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
	LogLevelFatal LogLevel = "FATAL"
)

type LogEntry struct {
	ID             string                 `json:"id"`
	Timestamp      time.Time              `json:"timestamp"`
	Level          LogLevel               `json:"level"`
	Message        string                 `json:"message"`
	ServiceName    string                 `json:"service_name"`
	AgentID        string                 `json:"agent_id"`
	Platform       string                 `json:"platform"`
	Metadata       map[string]interface{} `json:"metadata"`
	DeviceInfo     *DeviceInfo            `json:"device_info,omitempty"`
	StackTrace     string                 `json:"stack_trace,omitempty"`
	SourceLocation *SourceLocation        `json:"source_location,omitempty"`
}

type DeviceInfo struct {
	Platform   string `json:"platform"`
	Version    string `json:"version"`
	Model      string `json:"model"`
	AppVersion string `json:"app_version"`
}

type SourceLocation struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Function string `json:"function"`
}

type Field struct {
	Key   string      `json:"key"`
	Value interface{} `json:"value"`
}
