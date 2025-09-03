package mcp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/your-org/mcp-logging-server/pkg/models"
)

// IntegrationTestStorage implements a more realistic storage for integration tests
type IntegrationTestStorage struct {
	logs     []models.LogEntry
	services []models.ServiceInfo
}

func (its *IntegrationTestStorage) Store(ctx context.Context, logs []models.LogEntry) error {
	its.logs = append(its.logs, logs...)
	
	// Update services list
	serviceMap := make(map[string]*models.ServiceInfo)
	for _, service := range its.services {
		key := service.ServiceName + ":" + service.AgentID
		serviceMap[key] = &service
	}
	
	for _, log := range logs {
		key := log.ServiceName + ":" + log.AgentID
		if service, exists := serviceMap[key]; exists {
			service.LogCount++
			service.LastSeen = log.Timestamp
		} else {
			its.services = append(its.services, models.ServiceInfo{
				ServiceName: log.ServiceName,
				AgentID:     log.AgentID,
				Platform:    log.Platform,
				LastSeen:    log.Timestamp,
				LogCount:    1,
			})
		}
	}
	
	return nil
}

func (its *IntegrationTestStorage) Query(ctx context.Context, filter models.LogFilter) (*models.LogResult, error) {
	var filteredLogs []models.LogEntry
	
	for _, log := range its.logs {
		if filter.ServiceName != "" && log.ServiceName != filter.ServiceName {
			continue
		}
		if filter.AgentID != "" && log.AgentID != filter.AgentID {
			continue
		}
		if filter.Level != "" && log.Level != filter.Level {
			continue
		}
		if filter.Platform != "" && log.Platform != filter.Platform {
			continue
		}
		if filter.MessageContains != "" && !contains(log.Message, filter.MessageContains) {
			continue
		}
		if !filter.StartTime.IsZero() && log.Timestamp.Before(filter.StartTime) {
			continue
		}
		if !filter.EndTime.IsZero() && log.Timestamp.After(filter.EndTime) {
			continue
		}
		
		filteredLogs = append(filteredLogs, log)
	}
	
	totalCount := len(filteredLogs)
	
	// Apply pagination
	start := filter.Offset
	if start > totalCount {
		start = totalCount
	}
	
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	
	end := start + limit
	if end > totalCount {
		end = totalCount
	}
	
	var resultLogs []models.LogEntry
	if start < totalCount {
		resultLogs = filteredLogs[start:end]
	}
	
	hasMore := end < totalCount
	
	return &models.LogResult{
		Logs:       resultLogs,
		TotalCount: totalCount,
		HasMore:    hasMore,
	}, nil
}

func (its *IntegrationTestStorage) GetByIDs(ctx context.Context, ids []string) ([]models.LogEntry, error) {
	var result []models.LogEntry
	for _, log := range its.logs {
		for _, id := range ids {
			if log.ID == id {
				result = append(result, log)
				break
			}
		}
	}
	return result, nil
}

func (its *IntegrationTestStorage) GetServices(ctx context.Context) ([]models.ServiceInfo, error) {
	return its.services, nil
}

func (its *IntegrationTestStorage) HealthCheck(ctx context.Context) models.HealthStatus {
	return models.HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Details: map[string]string{
			"storage":    "ok",
			"log_count":  string(rune(len(its.logs))),
			"connection": "active",
		},
	}
}

func (its *IntegrationTestStorage) Close() error {
	return nil
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestMCPServerIntegration(t *testing.T) {
	// Create test data
	testLogs := []models.LogEntry{
		{
			ID:          "log-1",
			Timestamp:   time.Now().Add(-time.Hour),
			Level:       models.LogLevelInfo,
			Message:     "User login successful",
			ServiceName: "auth-service",
			AgentID:     "auth-agent-1",
			Platform:    models.PlatformGo,
			Metadata:    map[string]interface{}{"user_id": "user123", "ip": "192.168.1.1"},
		},
		{
			ID:          "log-2",
			Timestamp:   time.Now().Add(-30 * time.Minute),
			Level:       models.LogLevelError,
			Message:     "Database connection failed",
			ServiceName: "db-service",
			AgentID:     "db-agent-1",
			Platform:    models.PlatformExpress,
			Metadata:    map[string]interface{}{"error_code": "DB_CONN_TIMEOUT"},
		},
		{
			ID:          "log-3",
			Timestamp:   time.Now().Add(-15 * time.Minute),
			Level:       models.LogLevelWarn,
			Message:     "High memory usage detected",
			ServiceName: "monitor-service",
			AgentID:     "monitor-agent-1",
			Platform:    models.PlatformSwift,
			Metadata:    map[string]interface{}{"memory_usage": "85%"},
		},
	}

	// Setup storage with test data
	storage := &IntegrationTestStorage{}
	storage.Store(context.Background(), testLogs)

	// Create MCP server
	server := NewServer(8081, storage)

	// Test all MCP tools through complete workflow
	t.Run("CompleteWorkflow", func(t *testing.T) {
		ctx := context.Background()

		// Test 1: Initialize MCP server
		initMsg := &MCPMessage{
			JSONRPC: "2.0",
			ID:      "init-1",
			Method:  "initialize",
		}
		
		initResponse := server.handleMessage(ctx, initMsg)
		if initResponse.Error != nil {
			t.Fatalf("Initialize failed: %v", initResponse.Error)
		}

		// Test 2: List available tools
		toolsMsg := &MCPMessage{
			JSONRPC: "2.0",
			ID:      "tools-1",
			Method:  "tools/list",
		}
		
		toolsResponse := server.handleMessage(ctx, toolsMsg)
		if toolsResponse.Error != nil {
			t.Fatalf("Tools list failed: %v", toolsResponse.Error)
		}

		// Verify all expected tools are available
		result := toolsResponse.Result.(map[string]interface{})
		tools := result["tools"].([]Tool)
		expectedTools := map[string]bool{
			"query_logs":         false,
			"get_log_details":    false,
			"get_service_status": false,
			"list_services":      false,
		}
		
		for _, tool := range tools {
			if _, exists := expectedTools[tool.Name]; exists {
				expectedTools[tool.Name] = true
			}
		}
		
		for toolName, found := range expectedTools {
			if !found {
				t.Errorf("Expected tool %s not found", toolName)
			}
		}

		// Test 3: Query logs with various filters
		queryTests := []struct {
			name      string
			arguments map[string]interface{}
			expectMin int
		}{
			{
				name:      "all_logs",
				arguments: map[string]interface{}{},
				expectMin: 3,
			},
			{
				name: "filter_by_service",
				arguments: map[string]interface{}{
					"service_name": "auth-service",
				},
				expectMin: 1,
			},
			{
				name: "filter_by_level",
				arguments: map[string]interface{}{
					"level": "ERROR",
				},
				expectMin: 1,
			},
			{
				name: "filter_by_platform",
				arguments: map[string]interface{}{
					"platform": "go",
				},
				expectMin: 1,
			},
			{
				name: "filter_by_message_content",
				arguments: map[string]interface{}{
					"message_contains": "login",
				},
				expectMin: 1,
			},
			{
				name: "pagination_test",
				arguments: map[string]interface{}{
					"limit":  float64(2),
					"offset": float64(0),
				},
				expectMin: 2,
			},
		}

		for _, qt := range queryTests {
			t.Run("query_"+qt.name, func(t *testing.T) {
				queryMsg := &MCPMessage{
					JSONRPC: "2.0",
					ID:      "query-" + qt.name,
					Method:  "tools/call",
					Params: map[string]interface{}{
						"name":      "query_logs",
						"arguments": qt.arguments,
					},
				}

				queryResponse := server.handleMessage(ctx, queryMsg)
				if queryResponse.Error != nil {
					t.Fatalf("Query logs failed: %v", queryResponse.Error)
				}

				toolResult := queryResponse.Result.(*ToolResult)
				var response map[string]interface{}
				if err := json.Unmarshal([]byte(toolResult.Content[0].Text), &response); err != nil {
					t.Fatalf("Failed to parse query response: %v", err)
				}

				logs := response["logs"].([]interface{})
				if len(logs) < qt.expectMin {
					t.Errorf("Expected at least %d logs, got %d", qt.expectMin, len(logs))
				}

				// Verify pagination info is present
				if pagination, ok := response["pagination"].(map[string]interface{}); ok {
					if _, hasTotal := pagination["total_count"]; !hasTotal {
						t.Error("Pagination missing total_count")
					}
					if _, hasMore := pagination["has_more"]; !hasMore {
						t.Error("Pagination missing has_more")
					}
				} else {
					t.Error("Response missing pagination information")
				}
			})
		}

		// Test 4: Get specific log details
		getDetailsMsg := &MCPMessage{
			JSONRPC: "2.0",
			ID:      "details-1",
			Method:  "tools/call",
			Params: map[string]interface{}{
				"name": "get_log_details",
				"arguments": map[string]interface{}{
					"ids": []interface{}{"log-1", "log-2"},
				},
			},
		}

		detailsResponse := server.handleMessage(ctx, getDetailsMsg)
		if detailsResponse.Error != nil {
			t.Fatalf("Get log details failed: %v", detailsResponse.Error)
		}

		toolResult := detailsResponse.Result.(*ToolResult)
		var logs []models.LogEntry
		if err := json.Unmarshal([]byte(toolResult.Content[0].Text), &logs); err != nil {
			t.Fatalf("Failed to parse details response: %v", err)
		}

		if len(logs) != 2 {
			t.Errorf("Expected 2 logs, got %d", len(logs))
		}

		// Test 5: Get service status
		statusMsg := &MCPMessage{
			JSONRPC: "2.0",
			ID:      "status-1",
			Method:  "tools/call",
			Params: map[string]interface{}{
				"name":      "get_service_status",
				"arguments": map[string]interface{}{},
			},
		}

		statusResponse := server.handleMessage(ctx, statusMsg)
		if statusResponse.Error != nil {
			t.Fatalf("Get service status failed: %v", statusResponse.Error)
		}

		toolResult = statusResponse.Result.(*ToolResult)
		var status map[string]interface{}
		if err := json.Unmarshal([]byte(toolResult.Content[0].Text), &status); err != nil {
			t.Fatalf("Failed to parse status response: %v", err)
		}

		// Verify status structure
		if status["overall_status"] == nil {
			t.Error("Status missing overall_status")
		}
		if components, ok := status["components"].(map[string]interface{}); ok {
			if components["storage"] == nil {
				t.Error("Status missing storage component")
			}
			if components["mcp_server"] == nil {
				t.Error("Status missing mcp_server component")
			}
		} else {
			t.Error("Status missing components")
		}

		// Test 6: List services
		servicesMsg := &MCPMessage{
			JSONRPC: "2.0",
			ID:      "services-1",
			Method:  "tools/call",
			Params: map[string]interface{}{
				"name":      "list_services",
				"arguments": map[string]interface{}{},
			},
		}

		servicesResponse := server.handleMessage(ctx, servicesMsg)
		if servicesResponse.Error != nil {
			t.Fatalf("List services failed: %v", servicesResponse.Error)
		}

		toolResult = servicesResponse.Result.(*ToolResult)
		var serviceList map[string]interface{}
		if err := json.Unmarshal([]byte(toolResult.Content[0].Text), &serviceList); err != nil {
			t.Fatalf("Failed to parse services response: %v", err)
		}

		// Verify services structure
		if services, ok := serviceList["services"].([]interface{}); ok {
			if len(services) < 3 {
				t.Errorf("Expected at least 3 services, got %d", len(services))
			}
		} else {
			t.Error("Services response missing services array")
		}

		if summary, ok := serviceList["summary"].(map[string]interface{}); ok {
			if summary["total_services"] == nil {
				t.Error("Services summary missing total_services")
			}
			if summary["platforms"] == nil {
				t.Error("Services summary missing platforms")
			}
		} else {
			t.Error("Services response missing summary")
		}
	})
}

func TestMCPServerFieldMaskingIntegration(t *testing.T) {
	// Test field masking across all tools
	testLogs := []models.LogEntry{
		{
			ID:          "sensitive-log-1",
			Timestamp:   time.Now(),
			Level:       models.LogLevelInfo,
			Message:     "Sensitive user data processed",
			ServiceName: "user-service",
			AgentID:     "sensitive-agent-123",
			Platform:    models.PlatformGo,
			Metadata: map[string]interface{}{
				"user_id":    "user-sensitive-456",
				"api_key":    "secret-key-789",
				"public_info": "this-is-public",
			},
		},
	}

	storage := &IntegrationTestStorage{}
	storage.Store(context.Background(), testLogs)
	server := NewServer(8081, storage)

	ctx := context.Background()

	// Test field masking in query_logs
	t.Run("QueryLogsFieldMasking", func(t *testing.T) {
		queryMsg := &MCPMessage{
			JSONRPC: "2.0",
			ID:      "mask-query-1",
			Method:  "tools/call",
			Params: map[string]interface{}{
				"name": "query_logs",
				"arguments": map[string]interface{}{
					"mask_fields": []interface{}{"message", "agent_id", "user_id"},
				},
			},
		}

		response := server.handleMessage(ctx, queryMsg)
		if response.Error != nil {
			t.Fatalf("Query with masking failed: %v", response.Error)
		}

		toolResult := response.Result.(*ToolResult)
		var queryResponse map[string]interface{}
		if err := json.Unmarshal([]byte(toolResult.Content[0].Text), &queryResponse); err != nil {
			t.Fatalf("Failed to parse masked query response: %v", err)
		}

		logs := queryResponse["logs"].([]interface{})
		if len(logs) != 1 {
			t.Fatalf("Expected 1 log, got %d", len(logs))
		}

		log := logs[0].(map[string]interface{})
		
		// Verify message is masked
		message := log["message"].(string)
		if message == "Sensitive user data processed" {
			t.Error("Message should be masked")
		}

		// Verify agent_id is masked
		agentID := log["agent_id"].(string)
		if agentID == "sensitive-agent-123" {
			t.Error("Agent ID should be masked")
		}

		// Verify metadata user_id is masked
		metadata := log["metadata"].(map[string]interface{})
		userID := metadata["user_id"].(string)
		if userID == "user-sensitive-456" {
			t.Error("User ID in metadata should be masked")
		}

		// Verify non-masked fields are intact
		publicInfo := metadata["public_info"].(string)
		if publicInfo != "this-is-public" {
			t.Error("Public info should not be masked")
		}
	})

	// Test field masking in get_log_details
	t.Run("GetLogDetailsFieldMasking", func(t *testing.T) {
		detailsMsg := &MCPMessage{
			JSONRPC: "2.0",
			ID:      "mask-details-1",
			Method:  "tools/call",
			Params: map[string]interface{}{
				"name": "get_log_details",
				"arguments": map[string]interface{}{
					"ids":         []interface{}{"sensitive-log-1"},
					"mask_fields": []interface{}{"api_key"},
				},
			},
		}

		response := server.handleMessage(ctx, detailsMsg)
		if response.Error != nil {
			t.Fatalf("Get details with masking failed: %v", response.Error)
		}

		toolResult := response.Result.(*ToolResult)
		var logs []models.LogEntry
		if err := json.Unmarshal([]byte(toolResult.Content[0].Text), &logs); err != nil {
			t.Fatalf("Failed to parse masked details response: %v", err)
		}

		if len(logs) != 1 {
			t.Fatalf("Expected 1 log, got %d", len(logs))
		}

		log := logs[0]
		
		// Verify api_key is masked
		apiKey := log.Metadata["api_key"].(string)
		if apiKey == "secret-key-789" {
			t.Error("API key should be masked")
		}

		// Verify message is NOT masked (not in mask_fields)
		if log.Message != "Sensitive user data processed" {
			t.Error("Message should not be masked")
		}
	})
}

func TestMCPServerErrorHandling(t *testing.T) {
	storage := &IntegrationTestStorage{}
	server := NewServer(8081, storage)
	ctx := context.Background()

	errorTests := []struct {
		name           string
		message        *MCPMessage
		expectedError  int
		expectedMsg    string
	}{
		{
			name: "unknown_method",
			message: &MCPMessage{
				JSONRPC: "2.0",
				ID:      "error-1",
				Method:  "unknown_method",
			},
			expectedError: -32601,
			expectedMsg:   "Method not found",
		},
		{
			name: "unknown_tool",
			message: &MCPMessage{
				JSONRPC: "2.0",
				ID:      "error-2",
				Method:  "tools/call",
				Params: map[string]interface{}{
					"name": "unknown_tool",
				},
			},
			expectedError: -32601,
			expectedMsg:   "Tool not found",
		},
		{
			name: "invalid_tool_params",
			message: &MCPMessage{
				JSONRPC: "2.0",
				ID:      "error-3",
				Method:  "tools/call",
				Params:  "invalid",
			},
			expectedError: -32602,
			expectedMsg:   "Invalid params",
		},
		{
			name: "missing_tool_name",
			message: &MCPMessage{
				JSONRPC: "2.0",
				ID:      "error-4",
				Method:  "tools/call",
				Params: map[string]interface{}{
					"arguments": map[string]interface{}{},
				},
			},
			expectedError: -32602,
			expectedMsg:   "Missing tool name",
		},
	}

	for _, et := range errorTests {
		t.Run(et.name, func(t *testing.T) {
			response := server.handleMessage(ctx, et.message)
			
			if response.Error == nil {
				t.Fatal("Expected error but got none")
			}

			if response.Error.Code != et.expectedError {
				t.Errorf("Expected error code %d, got %d", et.expectedError, response.Error.Code)
			}

			if response.Error.Message != et.expectedMsg {
				t.Errorf("Expected error message '%s', got '%s'", et.expectedMsg, response.Error.Message)
			}

			if response.Result != nil {
				t.Error("Expected no result when error is present")
			}
		})
	}
}