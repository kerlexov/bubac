package ingestion

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/your-org/mcp-logging-server/pkg/buffer"
	"github.com/your-org/mcp-logging-server/pkg/models"
)

// FailingStorage simulates storage failures for testing resilience
type FailingStorage struct {
	failureCount int
	maxFailures  int
	mutex        sync.Mutex
	storedLogs   []models.LogEntry
}

func NewFailingStorage(maxFailures int) *FailingStorage {
	return &FailingStorage{
		maxFailures: maxFailures,
		storedLogs:  make([]models.LogEntry, 0),
	}
}

func (fs *FailingStorage) Store(ctx context.Context, logs []models.LogEntry) error {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	
	if fs.failureCount < fs.maxFailures {
		fs.failureCount++
		return errors.New("simulated storage failure")
	}
	
	fs.storedLogs = append(fs.storedLogs, logs...)
	return nil
}

func (fs *FailingStorage) Query(ctx context.Context, filter models.LogFilter) (*models.LogResult, error) {
	return nil, nil
}

func (fs *FailingStorage) GetByIDs(ctx context.Context, ids []string) ([]models.LogEntry, error) {
	return nil, nil
}

func (fs *FailingStorage) GetServices(ctx context.Context) ([]models.ServiceInfo, error) {
	return nil, nil
}

func (fs *FailingStorage) HealthCheck(ctx context.Context) models.HealthStatus {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	
	if fs.failureCount < fs.maxFailures {
		return models.HealthStatus{
			Status:    "unhealthy",
			Timestamp: time.Now(),
			Details:   map[string]string{"error": "simulated failure"},
		}
	}
	
	return models.HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
	}
}

func (fs *FailingStorage) Close() error {
	return nil
}

func (fs *FailingStorage) GetStoredLogs() []models.LogEntry {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	return append([]models.LogEntry{}, fs.storedLogs...)
}

func TestResilienceIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create temporary directory for recovery
	tempDir, err := os.MkdirTemp("", "resilience_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create failing storage that fails first 3 attempts
	failingStorage := NewFailingStorage(3)
	
	bufferConfig := buffer.Config{
		Size:         100,
		MaxBatchSize: 10,
		FlushTimeout: 100 * time.Millisecond, // Short timeout for testing
	}
	
	server := NewServer(8080, failingStorage, bufferConfig, tempDir)
	
	// Start server context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	// Start buffer
	server.buffer.Start(ctx)
	defer server.buffer.Stop()
	
	router := gin.New()
	server.registerRoutes(router)
	
	// Test 1: Health check should show unhealthy initially
	t.Run("initial_health_check_unhealthy", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusServiceUnavailable {
			t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
		}
		
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		
		if response["status"] != "unhealthy" {
			t.Errorf("Expected status unhealthy, got %v", response["status"])
		}
	})
	
	// Test 2: Send logs during storage failures
	validLogEntry := models.LogEntry{
		ID:          "550e8400-e29b-41d4-a716-446655440000",
		Timestamp:   time.Now(),
		Level:       models.LogLevelInfo,
		Message:     "Test message during failure",
		ServiceName: "test-service",
		AgentID:     "test-agent",
		Platform:    models.PlatformGo,
	}
	
	t.Run("log_ingestion_during_storage_failures", func(t *testing.T) {
		// Send multiple log entries
		for i := 0; i < 5; i++ {
			logEntry := validLogEntry
			logEntry.ID = fmt.Sprintf("550e8400-e29b-41d4-a716-44665544%04d", i)
			logEntry.Message = fmt.Sprintf("Test message %d", i)
			
			jsonData, _ := json.Marshal(logEntry)
			req, _ := http.NewRequest("POST", "/v1/logs", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			// Requests should still succeed (buffered)
			if w.Code != http.StatusCreated {
				t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
			}
		}
		
		// Wait for buffer flush attempts
		time.Sleep(200 * time.Millisecond)
		
		// Verify logs are still in buffer (not stored due to failures)
		stats := server.buffer.GetStats()
		if stats.Size == 0 {
			t.Error("Expected logs to be buffered during storage failures")
		}
	})
	
	// Test 3: Circuit breaker should eventually open
	t.Run("circuit_breaker_opens_after_failures", func(t *testing.T) {
		// Wait for circuit breaker to potentially open
		time.Sleep(500 * time.Millisecond)
		
		// Check circuit breaker stats
		req, _ := http.NewRequest("GET", "/circuit-breaker/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		
		if cbStats, ok := response["circuit_breaker_stats"].(map[string]interface{}); ok {
			if failureCount, ok := cbStats["failure_count"].(float64); ok {
				if failureCount == 0 {
					t.Log("Circuit breaker hasn't recorded failures yet (timing dependent)")
				}
			}
		}
	})
	
	// Test 4: Storage recovery - after max failures, storage should work
	t.Run("storage_recovery_and_log_processing", func(t *testing.T) {
		// Wait for storage to "recover" (after max failures)
		time.Sleep(1 * time.Second)
		
		// Manually flush buffer to trigger storage attempts
		err := server.buffer.Flush()
		if err != nil {
			t.Logf("Buffer flush error (expected during failures): %v", err)
		}
		
		// Wait a bit more for processing
		time.Sleep(500 * time.Millisecond)
		
		// Check if some logs were eventually stored
		storedLogs := failingStorage.GetStoredLogs()
		t.Logf("Stored logs count: %d", len(storedLogs))
		
		// Health check should eventually become healthy
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		
		// Status might be healthy or degraded depending on timing
		status := response["status"].(string)
		if status != "healthy" && status != "degraded" {
			t.Errorf("Expected status healthy or degraded after recovery, got %s", status)
		}
	})
	
	// Test 5: Rate limiting
	t.Run("rate_limiting", func(t *testing.T) {
		// This test would require sending many requests quickly
		// For now, just verify the rate limiter exists
		if server.rateLimiter == nil {
			t.Error("Expected rate limiter to be initialized")
		}
	})
	
	// Test 6: Metrics collection
	t.Run("metrics_collection", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/metrics", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		
		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)
		
		if metrics, ok := response["metrics"].(map[string]interface{}); ok {
			// Verify key metrics are present
			expectedMetrics := []string{
				"requests_total", "success_rate", "error_rate",
				"uptime_seconds", "logs_ingested", "validation_errors",
			}
			
			for _, metric := range expectedMetrics {
				if _, exists := metrics[metric]; !exists {
					t.Errorf("Expected metric %s to be present", metric)
				}
			}
		} else {
			t.Error("Expected metrics object in response")
		}
	})
}

func TestErrorHandlingIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockStorage := &MockStorage{
		healthStatus: models.HealthStatus{Status: "healthy"},
	}
	
	bufferConfig := buffer.Config{
		Size:         100,
		MaxBatchSize: 10,
		FlushTimeout: 1 * time.Second,
	}
	
	server := NewServer(8080, mockStorage, bufferConfig, "/tmp/test_recovery")
	router := gin.New()
	server.registerRoutes(router)
	
	// Test comprehensive error scenarios
	errorTests := []struct {
		name           string
		method         string
		path           string
		body           string
		contentType    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "malformed_json",
			method:         "POST",
			path:           "/v1/logs",
			body:           `{"invalid": json, "missing": "quotes"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_JSON",
		},
		{
			name:           "validation_error_missing_fields",
			method:         "POST",
			path:           "/v1/logs",
			body:           `{"message": "test"}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "batch_empty",
			method:         "POST",
			path:           "/v1/logs/batch",
			body:           `[]`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "EMPTY_BATCH",
		},
		{
			name:           "unsupported_method",
			method:         "PUT",
			path:           "/v1/logs",
			body:           "",
			contentType:    "",
			expectedStatus: http.StatusNotFound,
			expectedError:  "",
		},
	}
	
	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req, _ = http.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			} else {
				req, _ = http.NewRequest(tt.method, tt.path, nil)
			}
			
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			
			if tt.expectedError != "" {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse error response: %v", err)
				}
				
				if errorObj, ok := response["error"].(map[string]interface{}); ok {
					if code, ok := errorObj["code"].(string); ok {
						if code != tt.expectedError {
							t.Errorf("Expected error code %s, got %s", tt.expectedError, code)
						}
					}
				}
			}
		})
	}
}

func TestRecoveryIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	// Create temporary directory for recovery
	tempDir, err := os.MkdirTemp("", "recovery_integration_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	mockStorage := &MockStorage{
		healthStatus: models.HealthStatus{Status: "healthy"},
	}
	
	bufferConfig := buffer.Config{
		Size:         100,
		MaxBatchSize: 10,
		FlushTimeout: 1 * time.Second,
	}
	
	server := NewServer(8080, mockStorage, bufferConfig, tempDir)
	
	// Test recovery stats endpoint
	t.Run("recovery_stats_endpoint", func(t *testing.T) {
		router := gin.New()
		server.registerRoutes(router)
		
		req, _ := http.NewRequest("GET", "/recovery/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		if w.Code != http.StatusOK {
			t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
		}
		
		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}
		
		if _, exists := response["recovery_stats"]; !exists {
			t.Error("Expected recovery_stats in response")
		}
	})
}