package main

import (
	"log"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/your-org/mcp-logging-go-sdk/pkg/adapters"
	"github.com/your-org/mcp-logging-go-sdk/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// Create MCP logger
	config := logger.DefaultConfig()
	config.ServiceName = "adapter-example"
	config.AgentID = "adapter-001"
	config.ServerURL = "http://localhost:8080"

	mcpLogger, err := logger.New(config)
	if err != nil {
		log.Fatalf("Failed to create MCP logger: %v", err)
	}
	defer mcpLogger.Close()

	log.Println("=== MCP Logger Adapter Examples ===")

	// 1. Standard log adapter
	log.Println("\n1. Standard Log Adapter:")
	standardAdapter := adapters.NewStandardLogAdapter(mcpLogger)
	log.SetOutput(standardAdapter.GetWriter())
	
	log.Println("This message goes through standard log adapter")
	log.Printf("Formatted message with value: %d", 42)

	// 2. Logrus adapter
	log.Println("\n2. Logrus Adapter:")
	logrusLogger := logrus.New()
	logrusHook := adapters.NewLogrusHook(mcpLogger)
	logrusLogger.AddHook(logrusHook)
	
	logrusLogger.WithFields(logrus.Fields{
		"user_id": "123",
		"action":  "login",
	}).Info("User logged in via logrus")
	
	logrusLogger.WithField("error_code", "E001").Error("Error occurred via logrus")

	// 3. Zap adapter
	log.Println("\n3. Zap Adapter:")
	zapLogger := adapters.NewZapLogger(mcpLogger)
	
	zapLogger.Info("Info message via zap",
		zap.String("component", "auth"),
		zap.Int("user_id", 456),
	)
	
	zapLogger.Error("Error message via zap",
		zap.String("error", "connection failed"),
		zap.Duration("timeout", 30*time.Second),
	)

	// 4. Zap sugared logger
	log.Println("\n4. Zap Sugared Logger:")
	sugaredLogger := adapters.NewZapSugaredLogger(mcpLogger)
	
	sugaredLogger.Infow("Sugared info message",
		"key1", "value1",
		"key2", 789,
	)
	
	sugaredLogger.Errorw("Sugared error message",
		"error", "database connection failed",
		"retry_count", 3,
	)

	// 5. Global logrus hook installation
	log.Println("\n5. Global Logrus Hook:")
	adapters.InstallLogrusHook(mcpLogger)
	
	// Now any logrus usage will automatically send to MCP
	logrus.WithField("global", true).Info("This uses the global logrus hook")

	// 6. Logrus formatter
	log.Println("\n6. Logrus Formatter:")
	formatterLogger := logrus.New()
	formatter := adapters.NewLogrusFormatter(mcpLogger, &logrus.TextFormatter{})
	formatterLogger.SetFormatter(formatter)
	
	formatterLogger.Info("Message via logrus formatter")

	// Wait for logs to be sent
	time.Sleep(2 * time.Second)
	
	log.Println("\n=== All adapter examples completed ===")
}