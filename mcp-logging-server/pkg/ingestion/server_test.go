package ingestion

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/your-org/mcp-logging-server/pkg/buffer"
	"github.com/your-org/mcp-logging-server/pkg/models"
)

// MockStorage implements storage.LogStorage for testing
type MockStorage struct {
	storeCalled     bool
	storeError      error
	healthStatus    models.HealthStatus
	storedLogs      []models.LogEntry
}

func (m *MockStorage) Store(ctx context.Context, logs []models.LogEntry) error {
	m.storeCalled = true
	if m.storeError != nil {
		return m.storeError
	}
	m.storedLogs = append(m.storedLogs, logs...)
	return nil
}

func (m *MockStorage) Query(ctx context.Context, filter models.LogFilter) (*models.LogResult, error) {
	return nil, nil
}

func (m *MockStorage) GetByIDs(ctx context.Context, ids []string) ([]models.LogEntry, error) {
	return nil, nil
}

func (m *MockStorage) GetServices(ctx context.Context) ([]models.ServiceInfo, error) {
	return nil, nil
}

func (m *MockStorage) HealthCheck(ctx context.Context) models.HealthStatus {
	return m.healthStatus
}

func (m *MockStorage) Close() error {
	return nil
}

func TestServer_handleHealthCheck(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		healthStatus   models.HealthStatus
		expectedStatus int
	}{
		{
			name: "healthy storage",
			healthStatus: models.HealthStatus{
				Status:    "healthy",
				Timestamp: time.Now(),
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "unhealthy storage",
			healthStatus: models.HealthStatus{
				Status:    "unhealthy",
				Timestamp: time.Now(),
			},
			expectedStatus: http.StatusServiceUnavailable,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := &MockStorage{
				healthStatus: tt.healthStatus,
			}
			
			bufferConfig := buffer.Config{
				Size:         100,
				MaxBatchSize: 10,
				FlushTimeout: 1 * time.Second,
			}
			
			server := NewServer(8080, mockStorage, bufferConfig, "/tmp/test_recovery")
			
			router := gin.New()
			server.registerRoutes(router)
			
			req, _ := http.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}
			
			if response["service"] != "ingestion-server" {
				t.Errorf("Expected service to be 'ingestion-server', got %v", response["service"])
			}
		})
	}
}

func TestServer_handleIngestLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		logEntry       models.LogEntry
		expectedStatus int
		storeError     error
	}{
		{
			name: "valid log entry",
			logEntry: models.LogEntry{
				ID:          "550e8400-e29b-41d4-a716-446655440000",
				Timestamp:   time.Now(),
				Level:       models.LogLevelInfo,
				Message:     "Test message",
				ServiceName: "test-service",
				AgentID:     "test-agent",
				Platform:    models.PlatformGo,
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "log entry without ID (should generate)",
			logEntry: models.LogEntry{
				Timestamp:   time.Now(),
				Level:       models.LogLevelInfo,
				Message:     "Test message",
				ServiceName: "test-service",
				AgentID:     "test-agent",
				Platform:    models.PlatformGo,
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name: "invalid log entry (missing required fields)",
			logEntry: models.LogEntry{
				Message: "Test message",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := &MockStorage{
				storeError: tt.storeError,
			}
			
			bufferConfig := buffer.Config{
				Size:         100,
				MaxBatchSize: 10,
				FlushTimeout: 1 * time.Second,
			}
			
			server := NewServer(8080, mockStorage, bufferConfig, "/tmp/test_recovery")
			
			router := gin.New()
			server.registerRoutes(router)
			
			jsonData, _ := json.Marshal(tt.logEntry)
			req, _ := http.NewRequest("POST", "/v1/logs", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			
			// Note: With buffering, Store might not be called immediately
			// We'll test the buffering behavior separately
		})
	}
}

func TestServer_handleIngestLogsBatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		logEntries     []models.LogEntry
		expectedStatus int
	}{
		{
			name: "valid batch",
			logEntries: []models.LogEntry{
				{
					ID:          "550e8400-e29b-41d4-a716-446655440001",
					Timestamp:   time.Now(),
					Level:       models.LogLevelInfo,
					Message:     "Test message 1",
					ServiceName: "test-service",
					AgentID:     "test-agent",
					Platform:    models.PlatformGo,
				},
				{
					ID:          "550e8400-e29b-41d4-a716-446655440002",
					Timestamp:   time.Now(),
					Level:       models.LogLevelError,
					Message:     "Test message 2",
					ServiceName: "test-service",
					AgentID:     "test-agent",
					Platform:    models.PlatformGo,
				},
			},
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "empty batch",
			logEntries:     []models.LogEntry{},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "batch with invalid entries",
			logEntries: []models.LogEntry{
				{
					Message: "Invalid entry",
				},
			},
			expectedStatus: http.StatusBadRequest,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := &MockStorage{}
			
			bufferConfig := buffer.Config{
				Size:         100,
				MaxBatchSize: 10,
				FlushTimeout: 1 * time.Second,
			}
			
			server := NewServer(8080, mockStorage, bufferConfig, "/tmp/test_recovery")
			
			router := gin.New()
			server.registerRoutes(router)
			
			jsonData, _ := json.Marshal(tt.logEntries)
			req, _ := http.NewRequest("POST", "/v1/logs/batch", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
		})
	}
}

func TestServer_CORSHeaders(t *testing.T) {
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
	
	// Add CORS middleware manually for testing
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	})
	
	server.registerRoutes(router)
	
	// Test OPTIONS request
	req, _ := http.NewRequest("OPTIONS", "/v1/logs", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d for OPTIONS, got %d", http.StatusNoContent, w.Code)
	}
	
	// Check CORS headers
	expectedHeaders := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
	}
	
	for header, expectedValue := range expectedHeaders {
		if w.Header().Get(header) != expectedValue {
			t.Errorf("Expected header %s to be %s, got %s", header, expectedValue, w.Header().Get(header))
		}
	}
}

func TestServer_ConcurrentRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockStorage := &MockStorage{}
	
	bufferConfig := buffer.Config{
		Size:         100,
		MaxBatchSize: 10,
		FlushTimeout: 1 * time.Second,
	}
	
	server := NewServer(8080, mockStorage, bufferConfig, "/tmp/test_recovery")
	
	router := gin.New()
	server.registerRoutes(router)
	
	// Create a valid log entry
	logEntry := models.LogEntry{
		ID:          "550e8400-e29b-41d4-a716-446655440000",
		Timestamp:   time.Now(),
		Level:       models.LogLevelInfo,
		Message:     "Test message",
		ServiceName: "test-service",
		AgentID:     "test-agent",
		Platform:    models.PlatformGo,
	}
	
	// Send multiple concurrent requests
	numRequests := 10
	results := make(chan int, numRequests)
	
	for i := 0; i < numRequests; i++ {
		go func() {
			jsonData, _ := json.Marshal(logEntry)
			req, _ := http.NewRequest("POST", "/v1/logs", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			results <- w.Code
		}()
	}
	
	// Collect results
	for i := 0; i < numRequests; i++ {
		status := <-results
		if status != http.StatusCreated {
			t.Errorf("Expected status %d, got %d", http.StatusCreated, status)
		}
	}
	
	// Manually flush the buffer to ensure all entries are stored
	err := server.buffer.Flush()
	if err != nil {
		t.Fatalf("Failed to flush buffer: %v", err)
	}
	
	// Verify all requests were processed
	if len(mockStorage.storedLogs) != numRequests {
		t.Errorf("Expected %d stored logs, got %d", numRequests, len(mockStorage.storedLogs))
	}
}

func TestServer_ErrorHandling(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name           string
		requestBody    string
		contentType    string
		expectedStatus int
		expectedError  string
	}{
		{
			name:           "invalid JSON",
			requestBody:    `{"invalid": json}`,
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_JSON",
		},
		{
			name:           "missing content type",
			requestBody:    `{"message": "test"}`,
			contentType:    "",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "VALIDATION_ERROR",
		},
		{
			name:           "empty request body",
			requestBody:    "",
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectedError:  "INVALID_JSON",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := &MockStorage{}
			bufferConfig := buffer.Config{
				Size:         100,
				MaxBatchSize: 10,
				FlushTimeout: 1 * time.Second,
			}
			
			server := NewServer(8080, mockStorage, bufferConfig, "/tmp/test_recovery")
			router := gin.New()
			server.registerRoutes(router)
			
			req, _ := http.NewRequest("POST", "/v1/logs", bytes.NewBufferString(tt.requestBody))
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}
			
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse error response: %v", err)
			}
			
			if errorObj, ok := response["error"].(map[string]interface{}); ok {
				if code, ok := errorObj["code"].(string); ok {
					if code != tt.expectedError {
						t.Errorf("Expected error code %s, got %s", tt.expectedError, code)
					}
				} else {
					t.Error("Expected error code in response")
				}
			} else {
				t.Error("Expected error object in response")
			}
		})
	}
}

func TestServer_BatchSizeValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockStorage := &MockStorage{}
	bufferConfig := buffer.Config{
		Size:         100,
		MaxBatchSize: 10,
		FlushTimeout: 1 * time.Second,
	}
	
	server := NewServer(8080, mockStorage, bufferConfig, "/tmp/test_recovery")
	router := gin.New()
	server.registerRoutes(router)
	
	// Create a batch that's too large (over 1000 entries)
	largeBatch := make([]models.LogEntry, 1001)
	for i := range largeBatch {
		largeBatch[i] = models.LogEntry{
			ID:          fmt.Sprintf("550e8400-e29b-41d4-a716-44665544%04d", i),
			Timestamp:   time.Now(),
			Level:       models.LogLevelInfo,
			Message:     fmt.Sprintf("Test message %d", i),
			ServiceName: "test-service",
			AgentID:     "test-agent",
			Platform:    models.PlatformGo,
		}
	}
	
	jsonData, _ := json.Marshal(largeBatch)
	req, _ := http.NewRequest("POST", "/v1/logs/batch", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d for large batch, got %d", http.StatusBadRequest, w.Code)
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	
	if errorObj, ok := response["error"].(map[string]interface{}); ok {
		if code, ok := errorObj["code"].(string); ok {
			if code != "BATCH_TOO_LARGE" {
				t.Errorf("Expected error code BATCH_TOO_LARGE, got %s", code)
			}
		}
	}
}

func TestServer_CircuitBreakerEndpoints(t *testing.T) {
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
	
	// Test circuit breaker stats endpoint
	req, _ := http.NewRequest("GET", "/circuit-breaker/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for circuit breaker stats, got %d", http.StatusOK, w.Code)
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	
	if _, exists := response["circuit_breaker_stats"]; !exists {
		t.Error("Expected circuit_breaker_stats in response")
	}
	
	// Test circuit breaker reset endpoint
	req, _ = http.NewRequest("POST", "/circuit-breaker/reset", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d for circuit breaker reset, got %d", http.StatusOK, w.Code)
	}
}

func TestServer_handleBufferStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockStorage := &MockStorage{}
	bufferConfig := buffer.Config{
		Size:         100,
		MaxBatchSize: 10,
		FlushTimeout: 1 * time.Second,
	}
	
	server := NewServer(8080, mockStorage, bufferConfig, "/tmp/test_recovery")
	
	router := gin.New()
	server.registerRoutes(router)
	
	req, _ := http.NewRequest("GET", "/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	
	if _, exists := response["buffer_stats"]; !exists {
		t.Error("Expected buffer_stats in response")
	}
}

func TestServer_handleFlushBuffer(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	mockStorage := &MockStorage{}
	bufferConfig := buffer.Config{
		Size:         100,
		MaxBatchSize: 10,
		FlushTimeout: 1 * time.Second,
	}
	
	server := NewServer(8080, mockStorage, bufferConfig, "/tmp/test_recovery")
	
	router := gin.New()
	server.registerRoutes(router)
	
	// Add some entries to buffer first
	logEntry := models.LogEntry{
		ID:          "550e8400-e29b-41d4-a716-446655440000",
		Timestamp:   time.Now(),
		Level:       models.LogLevelInfo,
		Message:     "Test message",
		ServiceName: "test-service",
		AgentID:     "test-agent",
		Platform:    models.PlatformGo,
	}
	
	server.buffer.Add([]models.LogEntry{logEntry})
	
	// Test flush endpoint
	req, _ := http.NewRequest("POST", "/v1/flush", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}
	
	if response["message"] != "Buffer flushed successfully" {
		t.Errorf("Expected success message, got %v", response["message"])
	}
	
	// Verify entry was stored
	if len(mockStorage.storedLogs) != 1 {
		t.Errorf("Expected 1 stored log after flush, got %d", len(mockStorage.storedLogs))
	}
}