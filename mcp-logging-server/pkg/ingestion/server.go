package ingestion

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/your-org/mcp-logging-server/pkg/auth"
	"github.com/your-org/mcp-logging-server/pkg/buffer"
	"github.com/your-org/mcp-logging-server/pkg/dataprotection"
	"github.com/your-org/mcp-logging-server/pkg/metrics"
	"github.com/your-org/mcp-logging-server/pkg/models"
	"github.com/your-org/mcp-logging-server/pkg/ratelimit"
	"github.com/your-org/mcp-logging-server/pkg/recovery"
	"github.com/your-org/mcp-logging-server/pkg/security"
	"github.com/your-org/mcp-logging-server/pkg/storage"
	tlsconfig "github.com/your-org/mcp-logging-server/pkg/tls"
	"github.com/your-org/mcp-logging-server/pkg/validation"
)

// Server represents the log ingestion HTTP server
type Server struct {
	port               int
	storage            storage.LogStorage
	buffer             *buffer.MessageBuffer
	server             *http.Server
	metrics            *metrics.Metrics
	validator          *validation.LogValidator
	recoveryManager    *recovery.RecoveryManager
	rateLimiter        *ratelimit.RateLimiter
	circuitBreaker     *CircuitBreaker
	authManager        *auth.APIKeyManager
	tlsConfig          *tlsconfig.TLSConfig
	securityConfig     *security.SecurityConfig
	dataProtection     *dataprotection.DataProtectionProcessor
	auditStatsCollector *dataprotection.AuditStatsCollector
}

// NewServer creates a new ingestion server
func NewServer(port int, storage storage.LogStorage, bufferConfig buffer.Config, recoveryDir string, authManager *auth.APIKeyManager, rateLimitConfig *ratelimit.RateLimitConfig, tlsConfig *tlsconfig.TLSConfig, securityConfig *security.SecurityConfig, dataProtectionConfig *dataprotection.DataProtectionConfig) *Server {
	metricsReporter := metrics.NewMetrics()
	recoveryManager := recovery.NewRecoveryManager(recoveryDir)
	
	bufferOptions := buffer.Options{
		RecoveryManager: recoveryManager,
		MetricsReporter: metricsReporter,
	}
	
	messageBuffer := buffer.NewMessageBufferWithOptions(storage, bufferConfig, bufferOptions)
	
	// Use provided configs or defaults
	if rateLimitConfig == nil {
		rateLimitConfig = ratelimit.DefaultRateLimitConfig()
	}
	if tlsConfig == nil {
		tlsConfig = tlsconfig.DefaultTLSConfig()
	}
	if securityConfig == nil {
		securityConfig = security.DefaultSecurityConfig()
	}
	if dataProtectionConfig == nil {
		dataProtectionConfig = dataprotection.DefaultDataProtectionConfig()
	}
	
	// Initialize data protection processor
	dataProtectionProcessor, err := dataprotection.NewDataProtectionProcessor(dataProtectionConfig)
	if err != nil {
		// Log error but continue with disabled data protection
		fmt.Printf("Failed to initialize data protection: %v\n", err)
		dataProtectionProcessor = nil
	}
	
	// Initialize audit stats collector
	var auditStatsCollector *dataprotection.AuditStatsCollector
	if dataProtectionConfig.AuditEnabled {
		auditStatsCollector = dataprotection.NewAuditStatsCollector()
	}
	
	return &Server{
		port:               port,
		storage:            storage,
		buffer:             messageBuffer,
		metrics:            metricsReporter,
		validator:          validation.NewLogValidator(),
		recoveryManager:    recoveryManager,
		rateLimiter:        ratelimit.NewRateLimiter(rateLimitConfig),
		circuitBreaker:     NewCircuitBreaker(5, 30*time.Second, 60*time.Second), // 5 failures, 30s timeout, 60s reset
		authManager:        authManager,
		tlsConfig:          tlsConfig,
		securityConfig:     securityConfig,
		dataProtection:     dataProtectionProcessor,
		auditStatsCollector: auditStatsCollector,
	}
}

// Start starts the ingestion server
func (s *Server) Start(ctx context.Context) error {
	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)
	
	router := gin.New()
	
	// Apply security middleware first
	if err := security.ApplySecurityMiddleware(router, s.securityConfig); err != nil {
		return fmt.Errorf("failed to apply security middleware: %w", err)
	}
	
	// Add comprehensive middleware
	router.Use(s.loggingMiddleware())
	router.Use(s.recoveryMiddleware())
	router.Use(auth.AuthMiddleware(s.authManager))
	router.Use(ratelimit.RateLimitMiddleware(s.rateLimiter))
	router.Use(dataprotection.DataProtectionMiddleware(s.dataProtection))
	router.Use(s.corsMiddleware())
	router.Use(s.requestSizeMiddleware())
	router.Use(s.timeoutMiddleware())
	
	// Register routes
	s.registerRoutes(router)
	
	// Create HTTP server
	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	
	// Configure TLS if enabled
	if s.tlsConfig.Enabled {
		tlsConf, err := s.tlsConfig.GetTLSConfig()
		if err != nil {
			return fmt.Errorf("failed to configure TLS: %w", err)
		}
		s.server.TLSConfig = tlsConf
	}
	
	// Recover any pending logs from previous session
	if pendingLogs, err := s.recoveryManager.RecoverPendingLogs(ctx); err != nil {
		fmt.Printf("Failed to recover pending logs: %v\n", err)
	} else if len(pendingLogs) > 0 {
		fmt.Printf("Recovered %d pending logs from previous session\n", len(pendingLogs))
		if err := s.buffer.Add(pendingLogs); err != nil {
			fmt.Printf("Failed to add recovered logs to buffer: %v\n", err)
		}
	}
	
	// Start message buffer
	s.buffer.Start(ctx)
	
	// Start cleanup routine for old recovery files
	go s.cleanupRoutine(ctx)
	
	// Start server in a goroutine
	go func() {
		var err error
		if s.tlsConfig.Enabled {
			fmt.Printf("Starting HTTPS ingestion server on port %d\n", s.port)
			err = s.server.ListenAndServeTLS(s.tlsConfig.CertFile, s.tlsConfig.KeyFile)
		} else {
			fmt.Printf("Starting HTTP ingestion server on port %d\n", s.port)
			err = s.server.ListenAndServe()
		}
		
		if err != nil && err != http.ErrServerClosed {
			fmt.Printf("Failed to start ingestion server: %v\n", err)
		}
	}()
	
	// Wait for context cancellation
	<-ctx.Done()
	
	// Stop message buffer first
	if err := s.buffer.Stop(); err != nil {
		fmt.Printf("Error stopping message buffer: %v\n", err)
	}
	
	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	return s.server.Shutdown(shutdownCtx)
}

// Stop stops the ingestion server
func (s *Server) Stop() error {
	// Stop buffer first
	if s.buffer != nil {
		if err := s.buffer.Stop(); err != nil {
			fmt.Printf("Error stopping message buffer: %v\n", err)
		}
	}
	
	if s.server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return s.server.Shutdown(ctx)
	}
	return nil
}

// registerRoutes registers all HTTP routes
func (s *Server) registerRoutes(router *gin.Engine) {
	// Health check endpoint (public)
	router.GET("/health", s.handleHealthCheck)
	
	// Metrics and stats endpoints (require metrics permission)
	metricsGroup := router.Group("/")
	metricsGroup.Use(auth.RequirePermission(s.authManager, auth.PermissionMetrics))
	{
		metricsGroup.GET("/metrics", s.handleMetrics)
		metricsGroup.GET("/stats", s.handleBufferStats)
		metricsGroup.GET("/recovery/stats", s.handleRecoveryStats)
		metricsGroup.GET("/circuit-breaker/stats", s.handleCircuitBreakerStats)
	}
	
	// Admin endpoints (require admin permission)
	adminGroup := router.Group("/admin")
	adminGroup.Use(auth.RequirePermission(s.authManager, auth.PermissionAdmin))
	adminGroup.Use(ratelimit.AdminRateLimitMiddleware(s.rateLimiter))
	adminGroup.Use(dataprotection.AdminDataProtectionMiddleware(s.dataProtection, s.auditStatsCollector))
	{
		adminGroup.POST("/circuit-breaker/reset", s.handleCircuitBreakerReset)
		adminGroup.POST("/flush", s.handleFlushBuffer)
		// Rate limit management endpoints are handled by AdminRateLimitMiddleware
		// Data protection management endpoints are handled by AdminDataProtectionMiddleware
	}
	
	// Log ingestion endpoints (require ingest_logs permission)
	v1 := router.Group("/v1")
	v1.Use(auth.RequirePermission(s.authManager, auth.PermissionIngestLogs))
	{
		v1.POST("/logs", s.handleIngestLogs)
		v1.POST("/logs/batch", s.handleIngestLogsBatch)
	}
}

// handleHealthCheck handles health check requests
func (s *Server) handleHealthCheck(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()
	
	// Check storage health with circuit breaker protection
	var healthStatus models.HealthStatus
	err := s.circuitBreaker.Execute(func() error {
		healthStatus = s.storage.HealthCheck(ctx)
		if healthStatus.Status != "healthy" {
			return errors.New("storage unhealthy")
		}
		return nil
	})
	
	// Get additional health information
	bufferStats := s.buffer.GetStats()
	metricsSnapshot := s.metrics.GetSnapshot()
	circuitBreakerStats := s.circuitBreaker.GetStats()
	
	overallStatus := "healthy"
	statusCode := http.StatusOK
	
	// Determine overall health status
	if err != nil || healthStatus.Status != "healthy" {
		overallStatus = "unhealthy"
		statusCode = http.StatusServiceUnavailable
	} else if circuitBreakerStats.State == StateOpen {
		overallStatus = "degraded"
		statusCode = http.StatusServiceUnavailable
	} else if bufferStats.Size > int(float64(bufferStats.Capacity)*0.9) {
		overallStatus = "degraded" // Buffer is nearly full
	}
	
	response := gin.H{
		"status":    overallStatus,
		"timestamp": time.Now().UTC(),
		"service":   "ingestion-server",
		"storage":   healthStatus,
		"buffer": gin.H{
			"size":     bufferStats.Size,
			"capacity": bufferStats.Capacity,
			"usage":    float64(bufferStats.Size) / float64(bufferStats.Capacity) * 100,
		},
		"circuit_breaker": gin.H{
			"state":         circuitBreakerStats.State,
			"failure_count": circuitBreakerStats.FailureCount,
		},
		"metrics": gin.H{
			"requests_total":      metricsSnapshot.RequestsTotal,
			"success_rate":        metricsSnapshot.SuccessRate,
			"error_rate":          metricsSnapshot.ErrorRate,
			"uptime_seconds":      metricsSnapshot.UptimeSeconds,
			"logs_ingested":       metricsSnapshot.LogsIngested,
			"validation_errors":   metricsSnapshot.ValidationErrors,
			"storage_errors":      metricsSnapshot.StorageErrors,
		},
	}
	
	c.JSON(statusCode, response)
}

// handleIngestLogs handles single log entry ingestion
func (s *Server) handleIngestLogs(c *gin.Context) {
	s.metrics.IncrementRequestsTotal()
	
	var logEntry models.LogEntry
	
	// Parse JSON request body
	if err := c.ShouldBindJSON(&logEntry); err != nil {
		s.metrics.IncrementRequestsFailed()
		s.metrics.IncrementValidationErrors()
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_JSON",
				"message": "Invalid JSON format",
				"details": err.Error(),
			},
		})
		return
	}
	
	// Generate ID if not provided
	if logEntry.ID == "" {
		logEntry.ID = uuid.New().String()
	}
	
	// Set timestamp if not provided
	if logEntry.Timestamp.IsZero() {
		logEntry.Timestamp = time.Now().UTC()
	}
	
	// Enhanced validation
	validationResult := s.validator.ValidateLogEntry(&logEntry)
	if !validationResult.IsValid {
		s.metrics.IncrementRequestsFailed()
		s.metrics.IncrementValidationErrors()
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": "Log entry validation failed",
				"details": validationResult.Errors,
			},
		})
		return
	}
	
	// Apply data protection
	if s.dataProtection != nil {
		if err := s.dataProtection.ProcessLogEntry(&logEntry); err != nil {
			s.metrics.IncrementRequestsFailed()
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "DATA_PROTECTION_ERROR",
					"message": "Failed to apply data protection",
					"details": err.Error(),
				},
			})
			return
		}
	}
	
	// Add to buffer
	if err := s.buffer.Add([]models.LogEntry{logEntry}); err != nil {
		s.metrics.IncrementRequestsFailed()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "BUFFER_ERROR",
				"message": "Failed to buffer log entry",
				"details": err.Error(),
			},
		})
		return
	}
	
	s.metrics.IncrementRequestsSuccessful()
	s.metrics.IncrementLogsIngested(1)
	s.metrics.IncrementLogsBuffered(1)
	
	c.JSON(http.StatusCreated, gin.H{
		"message": "Log entry buffered successfully",
		"id":      logEntry.ID,
	})
}

// handleIngestLogsBatch handles batch log entry ingestion
func (s *Server) handleIngestLogsBatch(c *gin.Context) {
	s.metrics.IncrementRequestsTotal()
	
	var logEntries []models.LogEntry
	
	// Parse JSON request body
	if err := c.ShouldBindJSON(&logEntries); err != nil {
		s.metrics.IncrementRequestsFailed()
		s.metrics.IncrementValidationErrors()
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "INVALID_JSON",
				"message": "Invalid JSON format",
				"details": err.Error(),
			},
		})
		return
	}
	
	// Validate batch size
	if len(logEntries) == 0 {
		s.metrics.IncrementRequestsFailed()
		s.metrics.IncrementValidationErrors()
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "EMPTY_BATCH",
				"message": "Batch cannot be empty",
			},
		})
		return
	}
	
	if len(logEntries) > 1000 {
		s.metrics.IncrementRequestsFailed()
		s.metrics.IncrementValidationErrors()
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "BATCH_TOO_LARGE",
				"message": "Batch size cannot exceed 1000 entries",
				"details": fmt.Sprintf("Received %d entries, maximum allowed is 1000", len(logEntries)),
			},
		})
		return
	}
	
	// Process each log entry with enhanced validation
	for i := range logEntries {
		// Generate ID if not provided
		if logEntries[i].ID == "" {
			logEntries[i].ID = uuid.New().String()
		}
		
		// Set timestamp if not provided
		if logEntries[i].Timestamp.IsZero() {
			logEntries[i].Timestamp = time.Now().UTC()
		}
	}
	
	// Batch validation
	batchResult := s.validator.ValidateLogBatch(logEntries)
	
	// Return validation errors if any invalid entries
	if batchResult.InvalidCount > 0 {
		s.metrics.IncrementRequestsFailed()
		s.metrics.IncrementValidationErrors()
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"code":    "VALIDATION_ERROR",
				"message": fmt.Sprintf("%d out of %d entries failed validation", batchResult.InvalidCount, batchResult.TotalEntries),
				"details": batchResult.InvalidEntries,
			},
		})
		return
	}
	
	// Apply data protection to valid entries
	if s.dataProtection != nil {
		if err := dataprotection.ProcessLogEntries(s.dataProtection, batchResult.ValidEntries); err != nil {
			s.metrics.IncrementRequestsFailed()
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{
					"code":    "DATA_PROTECTION_ERROR",
					"message": "Failed to apply data protection",
					"details": err.Error(),
				},
			})
			return
		}
	}
	
	// Add to buffer
	if err := s.buffer.Add(batchResult.ValidEntries); err != nil {
		s.metrics.IncrementRequestsFailed()
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "BUFFER_ERROR",
				"message": "Failed to buffer log entries",
				"details": err.Error(),
			},
		})
		return
	}
	
	s.metrics.IncrementRequestsSuccessful()
	s.metrics.IncrementLogsIngested(int64(len(batchResult.ValidEntries)))
	s.metrics.IncrementLogsBuffered(int64(len(batchResult.ValidEntries)))
	
	c.JSON(http.StatusCreated, gin.H{
		"message":        "Log entries buffered successfully",
		"buffered_count": batchResult.ValidCount,
		"total_count":    batchResult.TotalEntries,
	})
}

// handleBufferStats handles buffer statistics requests
func (s *Server) handleBufferStats(c *gin.Context) {
	stats := s.buffer.GetStats()
	
	c.JSON(http.StatusOK, gin.H{
		"buffer_stats": stats,
		"timestamp":    time.Now().UTC(),
	})
}

// handleFlushBuffer handles manual buffer flush requests
func (s *Server) handleFlushBuffer(c *gin.Context) {
	if err := s.buffer.Flush(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "FLUSH_ERROR",
				"message": "Failed to flush buffer",
				"details": err.Error(),
			},
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message":   "Buffer flushed successfully",
		"timestamp": time.Now().UTC(),
	})
}

// handleMetrics handles metrics requests
func (s *Server) handleMetrics(c *gin.Context) {
	snapshot := s.metrics.GetSnapshot()
	
	c.JSON(http.StatusOK, gin.H{
		"metrics":   snapshot,
		"timestamp": time.Now().UTC(),
	})
}

// handleRecoveryStats handles recovery statistics requests
func (s *Server) handleRecoveryStats(c *gin.Context) {
	stats, err := s.recoveryManager.GetRecoveryStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "RECOVERY_STATS_ERROR",
				"message": "Failed to get recovery statistics",
				"details": err.Error(),
			},
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"recovery_stats": stats,
		"timestamp":      time.Now().UTC(),
	})
}

// handleCircuitBreakerStats handles circuit breaker statistics requests
func (s *Server) handleCircuitBreakerStats(c *gin.Context) {
	stats := s.circuitBreaker.GetStats()
	
	c.JSON(http.StatusOK, gin.H{
		"circuit_breaker_stats": stats,
		"timestamp":             time.Now().UTC(),
	})
}

// handleCircuitBreakerReset handles circuit breaker reset requests
func (s *Server) handleCircuitBreakerReset(c *gin.Context) {
	s.circuitBreaker.Reset()
	
	c.JSON(http.StatusOK, gin.H{
		"message":   "Circuit breaker reset successfully",
		"timestamp": time.Now().UTC(),
	})
}

// cleanupRoutine runs periodic cleanup of old recovery files
func (s *Server) cleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Clean up recovery files older than 24 hours
			if err := s.recoveryManager.CleanupOldRecoveryFiles(24 * time.Hour); err != nil {
				fmt.Printf("Failed to cleanup old recovery files: %v\n", err)
			}
		}
	}
}

// Middleware functions for comprehensive error handling and resilience

// loggingMiddleware provides structured logging for all requests
func (s *Server) loggingMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[%s] %s %s %d %s %s\n",
			param.TimeStamp.Format("2006-01-02 15:04:05"),
			param.Method,
			param.Path,
			param.StatusCode,
			param.Latency,
			param.ClientIP,
		)
	})
}

// recoveryMiddleware provides panic recovery with proper error responses
func (s *Server) recoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		s.metrics.IncrementRequestsFailed()
		
		fmt.Printf("Panic recovered: %v\n", recovered)
		
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": "An internal server error occurred",
				"details": "The server encountered an unexpected error and has recovered",
			},
		})
		c.Abort()
	})
}



// corsMiddleware handles CORS headers for cross-origin requests
func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	}
}

// requestSizeMiddleware limits the size of request bodies
func (s *Server) requestSizeMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		const maxRequestSize = 10 * 1024 * 1024 // 10MB
		
		if c.Request.ContentLength > maxRequestSize {
			s.metrics.IncrementRequestsFailed()
			s.metrics.IncrementValidationErrors()
			
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": gin.H{
					"code":    "REQUEST_TOO_LARGE",
					"message": "Request body too large",
					"details": fmt.Sprintf("Request body cannot exceed %d bytes", maxRequestSize),
				},
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// timeoutMiddleware adds request timeout handling
func (s *Server) timeoutMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
		defer cancel()
		
		c.Request = c.Request.WithContext(ctx)
		
		done := make(chan struct{})
		go func() {
			c.Next()
			close(done)
		}()
		
		select {
		case <-done:
			// Request completed normally
		case <-ctx.Done():
			// Request timed out
			s.metrics.IncrementRequestsFailed()
			
			c.JSON(http.StatusRequestTimeout, gin.H{
				"error": gin.H{
					"code":    "REQUEST_TIMEOUT",
					"message": "Request timeout",
					"details": "Request took too long to process",
				},
			})
			c.Abort()
		}
	}
}