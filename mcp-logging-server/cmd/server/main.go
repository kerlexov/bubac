package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/your-org/mcp-logging-server/pkg/auth"
	"github.com/your-org/mcp-logging-server/pkg/buffer"
	"github.com/your-org/mcp-logging-server/pkg/config"
	"github.com/your-org/mcp-logging-server/pkg/dataprotection"
	"github.com/your-org/mcp-logging-server/pkg/ingestion"
	"github.com/your-org/mcp-logging-server/pkg/mcp"
	"github.com/your-org/mcp-logging-server/pkg/ratelimit"
	"github.com/your-org/mcp-logging-server/pkg/security"
	"github.com/your-org/mcp-logging-server/pkg/storage"
	tlsconfig "github.com/your-org/mcp-logging-server/pkg/tls"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Load authentication configuration
	apiKeyConfigPath := os.Getenv("API_KEYS_CONFIG_PATH")
	if apiKeyConfigPath == "" {
		apiKeyConfigPath = "./config/api-keys.yaml"
	}
	
	authConfig, err := auth.LoadAPIKeyConfig(apiKeyConfigPath)
	if err != nil {
		log.Fatalf("Failed to load API key configuration: %v", err)
	}
	
	// Merge with environment configuration
	envAuthConfig := auth.LoadAPIKeyConfigFromEnv()
	authConfig = auth.MergeConfigs(authConfig, envAuthConfig)
	
	authManager := auth.NewAPIKeyManager(authConfig)

	// Load rate limiting configuration
	rateLimitConfig := ratelimit.DefaultRateLimitConfig()
	if os.Getenv("RATE_LIMIT_ENABLED") == "false" {
		rateLimitConfig.Enabled = false
	}
	if requestsPerMinute := os.Getenv("RATE_LIMIT_REQUESTS_PER_MINUTE"); requestsPerMinute != "" {
		if rpm, err := strconv.Atoi(requestsPerMinute); err == nil {
			rateLimitConfig.RequestsPerMinute = rpm
		}
	}
	if burstSize := os.Getenv("RATE_LIMIT_BURST"); burstSize != "" {
		if burst, err := strconv.Atoi(burstSize); err == nil {
			rateLimitConfig.BurstSize = burst
		}
	}

	// Load TLS configuration
	tlsConfig := tlsconfig.LoadTLSConfigFromEnv()
	if err := tlsConfig.ValidateConfig(); err != nil {
		log.Fatalf("Invalid TLS configuration: %v", err)
	}

	// Load security configuration
	securityConfig := security.DefaultSecurityConfig()
	if os.Getenv("HTTPS_REDIRECT") == "true" {
		securityConfig.HTTPSRedirect = true
	}

	// Load data protection configuration
	dataProtectionConfig := dataprotection.DefaultDataProtectionConfig()
	if os.Getenv("MASK_SENSITIVE_FIELDS") == "false" {
		dataProtectionConfig.Enabled = false
	}
	if sensitiveFields := os.Getenv("SENSITIVE_FIELDS"); sensitiveFields != "" {
		fields := strings.Split(sensitiveFields, ",")
		dataProtectionConfig.MaskFields = fields
		// Update field rules as well
		dataProtectionConfig.FieldRules = make([]dataprotection.FieldRule, len(fields))
		for i, field := range fields {
			dataProtectionConfig.FieldRules[i] = dataprotection.FieldRule{
				Field:  strings.TrimSpace(field),
				Action: dataprotection.ActionMask,
			}
		}
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
	recoveryDir := os.Getenv("MCP_LOGGING_RECOVERY_DIR")
	if recoveryDir == "" {
		recoveryDir = "./recovery"
	}
	ingestionServer := ingestion.NewServer(cfg.Server.IngestionPort, store, bufferConfig, recoveryDir, authManager, rateLimitConfig, tlsConfig, securityConfig, dataProtectionConfig)

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