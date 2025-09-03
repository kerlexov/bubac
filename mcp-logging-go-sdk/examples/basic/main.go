package main

import (
	"log"
	"time"

	"github.com/your-org/mcp-logging-go-sdk/pkg/logger"
)

func main() {
	config := logger.DefaultConfig()
	config.ServiceName = "example-service"
	config.AgentID = "agent-001"
	config.ServerURL = "http://localhost:8080"

	mcpLogger, err := logger.New(config)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer mcpLogger.Close()

	mcpLogger.Info("Application started",
		logger.Field{Key: "version", Value: "1.0.0"},
		logger.Field{Key: "environment", Value: "development"},
	)

	mcpLogger.Debug("Debug message with metadata",
		logger.Field{Key: "request_id", Value: "req-123"},
		logger.Field{Key: "user_id", Value: "user-456"},
	)

	mcpLogger.Warn("Warning message")

	mcpLogger.Error("Error occurred",
		logger.Field{Key: "error_code", Value: "E001"},
		logger.Field{Key: "stack_trace", Value: "stacktrace here..."},
	)

	contextLogger := mcpLogger.WithFields(
		logger.Field{Key: "module", Value: "auth"},
		logger.Field{Key: "operation", Value: "login"},
	)

	contextLogger.Info("User login attempt",
		logger.Field{Key: "username", Value: "john_doe"},
	)

	time.Sleep(6 * time.Second)

	log.Println("Example completed")
}
