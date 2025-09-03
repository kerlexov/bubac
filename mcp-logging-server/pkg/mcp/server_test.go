package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/kerlexov/mcp-logging-server/pkg/models"
)

// MockStorage implements storage.LogStorage for testing
type MockStorage struct {
	logs     []models.LogEntry
	services []models.ServiceInfo
}

func (m *MockStorage) Store(ctx context.Context, logs []models.LogEntry) error {
	m.logs = append(m.logs, logs...)
	return nil
}

func (m *MockStorage) Query(ctx context.Context, filter models.LogFilter) (*models.LogResult, error) {
	// Simple mock implementation - return all logs for testing with proper pagination
	totalCount := len(m.logs)

	// Apply offset
	start := filter.Offset
	if start > totalCount {
		start = totalCount
	}

	// Apply limit
	limit := filter.Limit
	if limit <= 0 {
		limit = 100 // default limit
	}

	end := start + limit
	if end > totalCount {
		end = totalCount
	}

	var resultLogs []models.LogEntry
	if start < totalCount {
		resultLogs = m.logs[start:end]
	}

	hasMore := end < totalCount

	return &models.LogResult{
		Logs:       resultLogs,
		TotalCount: totalCount,
		HasMore:    hasMore,
	}, nil
}

func (m *MockStorage) GetByIDs(ctx context.Context, ids []string) ([]models.LogEntry, error) {
	var result []models.LogEntry
	for _, log := range m.logs {
		for _, id := range ids {
			if log.ID == id {
				result = append(result, log)
				break
			}
		}
	}
	return result, nil
}

func (m *MockStorage) GetServices(ctx context.Context) ([]models.ServiceInfo, error) {
	return m.services, nil
}

func (m *MockStorage) HealthCheck(ctx context.Context) models.HealthStatus {
	return models.HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Details:   map[string]string{"storage": "ok"},
	}
}

func (m *MockStorage) Close() error {
	return nil
}

func TestNewServer(t *testing.T) {
	storage := &MockStorage{}
	server := NewServer(8081, storage)

	if server.port != 8081 {
		t.Errorf("Expected port 8081, got %d", server.port)
	}

	if server.storage != storage {
		t.Error("Storage not set correctly")
	}

	// Check that tools are registered
	expectedTools := []string{"query_logs", "get_log_details", "get_service_status", "list_services"}
	for _, toolName := range expectedTools {
		if _, exists := server.tools[toolName]; !exists {
			t.Errorf("Tool %s not registered", toolName)
		}
	}
}

func TestHandleInitialize(t *testing.T) {
	storage := &MockStorage{}
	server := NewServer(8081, storage)

	msg := &MCPMessage{
		JSONRPC: "2.0",
		ID:      "test-1",
		Method:  "initialize",
	}

	response := server.handleInitialize(msg)

	if response.JSONRPC != "2.0" {
		t.Errorf("Expected JSONRPC 2.0, got %s", response.JSONRPC)
	}

	if response.ID != "test-1" {
		t.Errorf("Expected ID test-1, got %v", response.ID)
	}

	if response.Error != nil {
		t.Errorf("Expected no error, got %v", response.Error)
	}

	result, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("Expected protocol version 2024-11-05, got %v", result["protocolVersion"])
	}
}

func TestHandleToolsList(t *testing.T) {
	storage := &MockStorage{}
	server := NewServer(8081, storage)

	msg := &MCPMessage{
		JSONRPC: "2.0",
		ID:      "test-2",
		Method:  "tools/list",
	}

	response := server.handleToolsList(msg)

	if response.Error != nil {
		t.Errorf("Expected no error, got %v", response.Error)
	}

	result, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	tools, ok := result["tools"].([]Tool)
	if !ok {
		t.Fatal("Tools is not a slice of Tool")
	}

	if len(tools) != 4 {
		t.Errorf("Expected 4 tools, got %d", len(tools))
	}

	// Check that all expected tools are present
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolNames[tool.Name] = true
	}

	expectedTools := []string{"query_logs", "get_log_details", "get_service_status", "list_services"}
	for _, expected := range expectedTools {
		if !toolNames[expected] {
			t.Errorf("Expected tool %s not found", expected)
		}
	}
}

func TestHandleQueryLogs(t *testing.T) {
	storage := &MockStorage{
		logs: []models.LogEntry{
			{
				ID:          "log-1",
				Timestamp:   time.Now(),
				Level:       models.LogLevelInfo,
				Message:     "Test log message",
				ServiceName: "test-service",
				AgentID:     "agent-1",
				Platform:    models.PlatformGo,
				Metadata:    map[string]interface{}{"user_id": "sensitive-user-123"},
			},
		},
	}
	server := NewServer(8081, storage)

	arguments := map[string]interface{}{
		"service_name": "test-service",
		"level":        "INFO",
		"limit":        float64(10),
	}

	result, err := server.handleQueryLogs(context.Background(), arguments)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Content) != 1 {
		t.Errorf("Expected 1 content block, got %d", len(result.Content))
	}

	if result.Content[0].Type != "text" {
		t.Errorf("Expected content type 'text', got %s", result.Content[0].Type)
	}

	// Verify the JSON content can be parsed and includes pagination
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &response); err != nil {
		t.Errorf("Failed to parse result JSON: %v", err)
	}

	logs, ok := response["logs"].([]interface{})
	if !ok {
		t.Fatal("Expected logs array in response")
	}

	if len(logs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logs))
	}

	pagination, ok := response["pagination"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected pagination object in response")
	}

	if pagination["limit"] != float64(10) {
		t.Errorf("Expected limit 10, got %v", pagination["limit"])
	}
}

func TestHandleQueryLogsWithFieldMasking(t *testing.T) {
	storage := &MockStorage{
		logs: []models.LogEntry{
			{
				ID:          "log-1",
				Timestamp:   time.Now(),
				Level:       models.LogLevelInfo,
				Message:     "Sensitive message content",
				ServiceName: "test-service",
				AgentID:     "sensitive-agent-123",
				Platform:    models.PlatformGo,
				Metadata:    map[string]interface{}{"user_id": "sensitive-user-123"},
			},
		},
	}
	server := NewServer(8081, storage)

	arguments := map[string]interface{}{
		"service_name": "test-service",
		"mask_fields":  []interface{}{"message", "agent_id", "user_id"},
	}

	result, err := server.handleQueryLogs(context.Background(), arguments)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Parse the response
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &response); err != nil {
		t.Errorf("Failed to parse result JSON: %v", err)
	}

	logs, ok := response["logs"].([]interface{})
	if !ok {
		t.Fatal("Expected logs array in response")
	}

	if len(logs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logs))
	}

	logEntry := logs[0].(map[string]interface{})

	// Check that message is masked
	message := logEntry["message"].(string)
	if message == "Sensitive message content" {
		t.Error("Expected message to be masked")
	}
	if message != "Se[MASKED]nt" {
		t.Errorf("Expected masked message 'Se[MASKED]nt', got '%s'", message)
	}

	// Check that agent_id is masked
	agentID := logEntry["agent_id"].(string)
	if agentID == "sensitive-agent-123" {
		t.Error("Expected agent_id to be masked")
	}
	if agentID != "se[MASKED]23" {
		t.Errorf("Expected masked agent_id 'se[MASKED]23', got '%s'", agentID)
	}

	// Check that metadata field is masked
	metadata := logEntry["metadata"].(map[string]interface{})
	userID := metadata["user_id"].(string)
	if userID == "sensitive-user-123" {
		t.Error("Expected user_id metadata to be masked")
	}
	if userID != "se[MASKED]23" {
		t.Errorf("Expected masked user_id 'se[MASKED]23', got '%s'", userID)
	}
}

func TestHandleGetLogDetails(t *testing.T) {
	storage := &MockStorage{
		logs: []models.LogEntry{
			{
				ID:          "log-1",
				Timestamp:   time.Now(),
				Level:       models.LogLevelInfo,
				Message:     "Test log message",
				ServiceName: "test-service",
				AgentID:     "agent-1",
				Platform:    models.PlatformGo,
			},
		},
	}
	server := NewServer(8081, storage)

	arguments := map[string]interface{}{
		"ids": []interface{}{"log-1"},
	}

	result, err := server.handleGetLogDetails(context.Background(), arguments)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Content) != 1 {
		t.Errorf("Expected 1 content block, got %d", len(result.Content))
	}

	// Verify the JSON content can be parsed
	var logs []models.LogEntry
	if err := json.Unmarshal([]byte(result.Content[0].Text), &logs); err != nil {
		t.Errorf("Failed to parse result JSON: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logs))
	}

	if logs[0].ID != "log-1" {
		t.Errorf("Expected log ID 'log-1', got %s", logs[0].ID)
	}
}

func TestHandleGetLogDetailsWithFieldMasking(t *testing.T) {
	storage := &MockStorage{
		logs: []models.LogEntry{
			{
				ID:          "log-1",
				Timestamp:   time.Now(),
				Level:       models.LogLevelInfo,
				Message:     "Sensitive log message",
				ServiceName: "test-service",
				AgentID:     "sensitive-agent-456",
				Platform:    models.PlatformGo,
				Metadata:    map[string]interface{}{"api_key": "secret-key-789"},
			},
		},
	}
	server := NewServer(8081, storage)

	arguments := map[string]interface{}{
		"ids":         []interface{}{"log-1"},
		"mask_fields": []interface{}{"message", "api_key"},
	}

	result, err := server.handleGetLogDetails(context.Background(), arguments)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Parse the response
	var logs []models.LogEntry
	if err := json.Unmarshal([]byte(result.Content[0].Text), &logs); err != nil {
		t.Errorf("Failed to parse result JSON: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("Expected 1 log entry, got %d", len(logs))
	}

	log := logs[0]

	// Check that message is masked
	if log.Message == "Sensitive log message" {
		t.Error("Expected message to be masked")
	}
	if log.Message != "Se[MASKED]ge" {
		t.Errorf("Expected masked message 'Se[MASKED]ge', got '%s'", log.Message)
	}

	// Check that agent_id is NOT masked (not in mask_fields)
	if log.AgentID != "sensitive-agent-456" {
		t.Errorf("Expected agent_id to NOT be masked, got '%s'", log.AgentID)
	}

	// Check that metadata field is masked
	apiKey := log.Metadata["api_key"].(string)
	if apiKey == "secret-key-789" {
		t.Error("Expected api_key metadata to be masked")
	}
	if apiKey != "se[MASKED]89" {
		t.Errorf("Expected masked api_key 'se[MASKED]89', got '%s'", apiKey)
	}
}

func TestHandleGetServiceStatus(t *testing.T) {
	storage := &MockStorage{}
	server := NewServer(8081, storage)

	result, err := server.handleGetServiceStatus(context.Background(), nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Content) != 1 {
		t.Errorf("Expected 1 content block, got %d", len(result.Content))
	}

	// Verify the JSON content can be parsed as enhanced system health
	var systemHealth map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &systemHealth); err != nil {
		t.Errorf("Failed to parse result JSON: %v", err)
	}

	if systemHealth["overall_status"] != "healthy" {
		t.Errorf("Expected overall_status 'healthy', got %v", systemHealth["overall_status"])
	}

	// Check components structure
	if components, ok := systemHealth["components"].(map[string]interface{}); ok {
		if components["storage"] == nil {
			t.Error("Expected storage component in system health")
		}
		if components["mcp_server"] == nil {
			t.Error("Expected mcp_server component in system health")
		}
	} else {
		t.Error("Expected components in system health")
	}
}

func TestHandleListServices(t *testing.T) {
	storage := &MockStorage{
		services: []models.ServiceInfo{
			{
				ServiceName: "test-service",
				AgentID:     "agent-1",
				Platform:    models.PlatformGo,
				LastSeen:    time.Now(),
				LogCount:    10,
			},
		},
	}
	server := NewServer(8081, storage)

	result, err := server.handleListServices(context.Background(), nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(result.Content) != 1 {
		t.Errorf("Expected 1 content block, got %d", len(result.Content))
	}

	// Verify the JSON content can be parsed as enhanced service list
	var serviceList map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &serviceList); err != nil {
		t.Errorf("Failed to parse result JSON: %v", err)
	}

	// Check services array
	if services, ok := serviceList["services"].([]interface{}); ok {
		if len(services) != 1 {
			t.Errorf("Expected 1 service, got %d", len(services))
		}

		service := services[0].(map[string]interface{})
		if service["service_name"] != "test-service" {
			t.Errorf("Expected service name 'test-service', got %v", service["service_name"])
		}
	} else {
		t.Error("Expected services array in response")
	}

	// Check summary
	if summary, ok := serviceList["summary"].(map[string]interface{}); ok {
		if summary["total_services"] != float64(1) {
			t.Errorf("Expected total_services 1, got %v", summary["total_services"])
		}
	} else {
		t.Error("Expected summary in response")
	}
}

func TestHandleMessage_UnknownMethod(t *testing.T) {
	storage := &MockStorage{}
	server := NewServer(8081, storage)

	msg := &MCPMessage{
		JSONRPC: "2.0",
		ID:      "test-3",
		Method:  "unknown_method",
	}

	response := server.handleMessage(context.Background(), msg)

	if response.Error == nil {
		t.Error("Expected error for unknown method")
	}

	if response.Error.Code != -32601 {
		t.Errorf("Expected error code -32601, got %d", response.Error.Code)
	}

	if response.Error.Message != "Method not found" {
		t.Errorf("Expected error message 'Method not found', got %s", response.Error.Message)
	}
}

func TestHandleToolCall_UnknownTool(t *testing.T) {
	storage := &MockStorage{}
	server := NewServer(8081, storage)

	msg := &MCPMessage{
		JSONRPC: "2.0",
		ID:      "test-4",
		Method:  "tools/call",
		Params: map[string]interface{}{
			"name": "unknown_tool",
		},
	}

	response := server.handleToolCall(context.Background(), msg)

	if response.Error == nil {
		t.Error("Expected error for unknown tool")
	}

	if response.Error.Code != -32601 {
		t.Errorf("Expected error code -32601, got %d", response.Error.Code)
	}

	if response.Error.Message != "Tool not found" {
		t.Errorf("Expected error message 'Tool not found', got %s", response.Error.Message)
	}
}

func TestHandleToolCall_InvalidParams(t *testing.T) {
	storage := &MockStorage{}
	server := NewServer(8081, storage)

	msg := &MCPMessage{
		JSONRPC: "2.0",
		ID:      "test-5",
		Method:  "tools/call",
		Params:  "invalid params",
	}

	response := server.handleToolCall(context.Background(), msg)

	if response.Error == nil {
		t.Error("Expected error for invalid params")
	}

	if response.Error.Code != -32602 {
		t.Errorf("Expected error code -32602, got %d", response.Error.Code)
	}
}

func TestMCPProtocolCompliance(t *testing.T) {
	// Test that all responses follow MCP protocol structure
	storage := &MockStorage{}
	server := NewServer(8081, storage)

	testCases := []struct {
		name   string
		method string
		params interface{}
	}{
		{"initialize", "initialize", nil},
		{"tools/list", "tools/list", nil},
		{"tools/call", "tools/call", map[string]interface{}{
			"name": "get_service_status",
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msg := &MCPMessage{
				JSONRPC: "2.0",
				ID:      "test-" + tc.name,
				Method:  tc.method,
				Params:  tc.params,
			}

			response := server.handleMessage(context.Background(), msg)

			// All responses should have JSONRPC version
			if response.JSONRPC != "2.0" {
				t.Errorf("Expected JSONRPC 2.0, got %s", response.JSONRPC)
			}

			// All responses should have matching ID
			if response.ID != msg.ID {
				t.Errorf("Expected ID %v, got %v", msg.ID, response.ID)
			}

			// Response should have either Result or Error, but not both
			if response.Result != nil && response.Error != nil {
				t.Error("Response has both Result and Error")
			}

			if response.Result == nil && response.Error == nil {
				t.Error("Response has neither Result nor Error")
			}
		})
	}
}

func TestMaskString(t *testing.T) {
	storage := &MockStorage{}
	server := NewServer(8081, storage)

	testCases := []struct {
		input    string
		expected string
	}{
		{"", "[MASKED]"},
		{"a", "[MASKED]"},
		{"ab", "[MASKED]"},
		{"abc", "[MASKED]"},
		{"abcd", "[MASKED]"},
		{"abcde", "ab[MASKED]de"},
		{"sensitive-data-123", "se[MASKED]23"},
		{"very-long-sensitive-string", "ve[MASKED]ng"},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := server.maskString(tc.input)
			if result != tc.expected {
				t.Errorf("Expected '%s', got '%s'", tc.expected, result)
			}
		})
	}
}

func TestGetMaskedFields(t *testing.T) {
	storage := &MockStorage{}
	server := NewServer(8081, storage)

	testCases := []struct {
		name     string
		args     map[string]interface{}
		expected []string
	}{
		{
			name:     "no mask_fields",
			args:     map[string]interface{}{},
			expected: []string{},
		},
		{
			name: "single field",
			args: map[string]interface{}{
				"mask_fields": []interface{}{"message"},
			},
			expected: []string{"message"},
		},
		{
			name: "multiple fields",
			args: map[string]interface{}{
				"mask_fields": []interface{}{"message", "agent_id", "user_id"},
			},
			expected: []string{"message", "agent_id", "user_id"},
		},
		{
			name: "invalid mask_fields type",
			args: map[string]interface{}{
				"mask_fields": "not an array",
			},
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := server.getMaskedFields(tc.args)
			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d fields, got %d", len(tc.expected), len(result))
				return
			}
			for i, expected := range tc.expected {
				if result[i] != expected {
					t.Errorf("Expected field '%s' at index %d, got '%s'", expected, i, result[i])
				}
			}
		})
	}
}

func TestApplyFieldMasking(t *testing.T) {
	storage := &MockStorage{}
	server := NewServer(8081, storage)

	originalResult := &models.LogResult{
		Logs: []models.LogEntry{
			{
				ID:          "log-1",
				Message:     "Original message",
				AgentID:     "original-agent",
				ServiceName: "original-service",
				Metadata: map[string]interface{}{
					"user_id": "user-123",
					"api_key": "secret-key",
					"count":   42,
				},
			},
		},
		TotalCount: 1,
		HasMore:    false,
	}

	maskedFields := []string{"message", "user_id"}
	result := server.applyFieldMasking(originalResult, maskedFields)

	// Check that original is not modified
	if originalResult.Logs[0].Message != "Original message" {
		t.Error("Original result was modified")
	}

	// Check that result is properly masked
	maskedLog := result.Logs[0]
	if maskedLog.Message == "Original message" {
		t.Error("Message was not masked")
	}
	if maskedLog.Message != "Or[MASKED]ge" {
		t.Errorf("Expected masked message 'Or[MASKED]ge', got '%s'", maskedLog.Message)
	}

	// Check that agent_id is NOT masked (not in maskedFields)
	if maskedLog.AgentID != "original-agent" {
		t.Errorf("AgentID should not be masked, got '%s'", maskedLog.AgentID)
	}

	// Check that metadata user_id is masked
	userID := maskedLog.Metadata["user_id"].(string)
	if userID == "user-123" {
		t.Error("user_id metadata was not masked")
	}
	if userID != "us[MASKED]23" {
		t.Errorf("Expected masked user_id 'us[MASKED]23', got '%s'", userID)
	}

	// Check that api_key is NOT masked (not in maskedFields)
	apiKey := maskedLog.Metadata["api_key"].(string)
	if apiKey != "secret-key" {
		t.Errorf("api_key should not be masked, got '%s'", apiKey)
	}

	// Check that non-string metadata is handled correctly
	count := maskedLog.Metadata["count"]
	if count != 42 {
		t.Errorf("Non-string metadata should not be affected, got %v", count)
	}
}

func TestPaginationInformation(t *testing.T) {
	storage := &MockStorage{
		logs: []models.LogEntry{
			{ID: "log-1", Message: "Message 1", Level: models.LogLevelInfo, ServiceName: "service", AgentID: "agent", Platform: models.PlatformGo, Timestamp: time.Now()},
			{ID: "log-2", Message: "Message 2", Level: models.LogLevelInfo, ServiceName: "service", AgentID: "agent", Platform: models.PlatformGo, Timestamp: time.Now()},
		},
	}
	server := NewServer(8081, storage)

	arguments := map[string]interface{}{
		"limit":  float64(5),
		"offset": float64(10),
	}

	result, err := server.handleQueryLogs(context.Background(), arguments)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Parse the response
	var response map[string]interface{}
	if err := json.Unmarshal([]byte(result.Content[0].Text), &response); err != nil {
		t.Errorf("Failed to parse result JSON: %v", err)
	}

	pagination, ok := response["pagination"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected pagination object in response")
	}

	if pagination["limit"] != float64(5) {
		t.Errorf("Expected limit 5, got %v", pagination["limit"])
	}

	if pagination["offset"] != float64(10) {
		t.Errorf("Expected offset 10, got %v", pagination["offset"])
	}

	if pagination["total_count"] != float64(2) {
		t.Errorf("Expected total_count 2, got %v", pagination["total_count"])
	}

	if pagination["has_more"] != false {
		t.Errorf("Expected has_more false, got %v", pagination["has_more"])
	}
}
