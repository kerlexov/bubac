package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/your-org/mcp-logging-server/pkg/buffer"
	"github.com/your-org/mcp-logging-server/pkg/config"
	"github.com/your-org/mcp-logging-server/pkg/ingestion"
	"github.com/your-org/mcp-logging-server/pkg/mcp"
	"github.com/your-org/mcp-logging-server/pkg/storage"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize storage
	store, err := storage.NewSQLiteStorage(cfg.Storage.ConnectionString)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Initialize ingestion server
	bufferConfig := buffer.Config{
		Size:         cfg.Buffer.Size,
		MaxBatchSize: cfg.Buffer.MaxBatchSize,
		FlushTimeout: cfg.Buffer.FlushTimeout,
	}
	recoveryDir := "./recovery"
	ingestionServer := ingestion.NewServer(cfg.Server.IngestionPort, store, bufferConfig, recoveryDir)

	// Initialize MCP server
	mcpServer := mcp.NewServer(cfg.Server.MCPPort, store)

	// Start servers
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := ingestionServer.Start(ctx); err != nil {
			log.Printf("Ingestion server error: %v", err)
		}
	}()

	go func() {
		if err := mcpServer.Start(ctx); err != nil {
			log.Printf("MCP server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down servers...")
	cancel()
}