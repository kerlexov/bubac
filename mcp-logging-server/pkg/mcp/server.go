package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/kerlexov/mcp-logging-server/pkg/models"
	"github.com/kerlexov/mcp-logging-server/pkg/storage"
)

// MCPMessage represents a generic MCP message
type MCPMessage struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Method  string      `json:"method,omitempty"`
	Params  interface{} `json:"params,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP error response
type MCPError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// ToolCallParams represents parameters for a tool call
type ToolCallParams struct {
	Name      string      `json:"name"`
	Arguments interface{} `json:"arguments,omitempty"`
}

// ToolResult represents the result of a tool call
type ToolResult struct {
	Content []ContentBlock `json:"content"`
	IsError bool           `json:"isError,omitempty"`
}

// ContentBlock represents a content block in MCP responses
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// Server represents the MCP server
type Server struct {
	port    int
	storage storage.LogStorage
	tools   map[string]Tool
}

// NewServer creates a new MCP server
func NewServer(port int, storage storage.LogStorage) *Server {
	s := &Server{
		port:    port,
		storage: storage,
		tools:   make(map[string]Tool),
	}

	// Register available tools
	s.registerTools()

	return s
}

// registerTools registers all available MCP tools
func (s *Server) registerTools() {
	// query_logs tool
	s.tools["query_logs"] = Tool{
		Name:        "query_logs",
		Description: "Query logs with filtering options and pagination support",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"service_name": map[string]interface{}{
					"type":        "string",
					"description": "Filter by service name",
				},
				"agent_id": map[string]interface{}{
					"type":        "string",
					"description": "Filter by agent ID",
				},
				"level": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"DEBUG", "INFO", "WARN", "ERROR", "FATAL"},
					"description": "Filter by log level",
				},
				"start_time": map[string]interface{}{
					"type":        "string",
					"format":      "date-time",
					"description": "Start time for log query (RFC3339 format)",
				},
				"end_time": map[string]interface{}{
					"type":        "string",
					"format":      "date-time",
					"description": "End time for log query (RFC3339 format)",
				},
				"message_contains": map[string]interface{}{
					"type":        "string",
					"description": "Filter logs containing this text in the message",
				},
				"platform": map[string]interface{}{
					"type":        "string",
					"enum":        []string{"go", "swift", "express", "react", "react-native", "kotlin"},
					"description": "Filter by platform",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"default":     100,
					"minimum":     1,
					"maximum":     1000,
					"description": "Maximum number of logs to return",
				},
				"offset": map[string]interface{}{
					"type":        "integer",
					"default":     0,
					"minimum":     0,
					"description": "Number of logs to skip",
				},
				"mask_fields": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Array of field names to mask for sensitive data protection (e.g., ['message', 'agent_id', 'custom_field'])",
				},
			},
		},
	}

	// get_log_details tool
	s.tools["get_log_details"] = Tool{
		Name:        "get_log_details",
		Description: "Retrieve specific log entries by their IDs with optional field masking",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"ids": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Array of log entry IDs to retrieve",
					"minItems":    1,
					"maxItems":    100,
				},
				"mask_fields": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "Array of field names to mask for sensitive data protection (e.g., ['message', 'agent_id', 'custom_field'])",
				},
			},
			"required": []string{"ids"},
		},
	}

	// get_service_status tool
	s.tools["get_service_status"] = Tool{
		Name:        "get_service_status",
		Description: "Get health status of the logging service",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}

	// list_services tool
	s.tools["list_services"] = Tool{
		Name:        "list_services",
		Description: "List all available services and agents that have logged entries",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}
}

// Start starts the MCP server
func (s *Server) Start(ctx context.Context) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.port, err)
	}
	defer listener.Close()

	log.Printf("MCP server listening on port %d", s.port)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			conn, err := listener.Accept()
			if err != nil {
				log.Printf("Failed to accept connection: %v", err)
				continue
			}

			go s.handleConnection(ctx, conn)
		}
	}
}

// handleConnection handles a single MCP connection
func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	encoder := json.NewEncoder(conn)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			var msg MCPMessage
			if err := decoder.Decode(&msg); err != nil {
				if err == io.EOF {
					return
				}
				log.Printf("Failed to decode message: %v", err)
				continue
			}

			response := s.handleMessage(ctx, &msg)
			if response != nil {
				if err := encoder.Encode(response); err != nil {
					log.Printf("Failed to encode response: %v", err)
					return
				}
			}
		}
	}
}

// handleMessage processes an MCP message and returns a response
func (s *Server) handleMessage(ctx context.Context, msg *MCPMessage) *MCPMessage {
	switch msg.Method {
	case "initialize":
		return s.handleInitialize(msg)
	case "tools/list":
		return s.handleToolsList(msg)
	case "tools/call":
		return s.handleToolCall(ctx, msg)
	default:
		return &MCPMessage{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: "Method not found",
			},
		}
	}
}

// handleInitialize handles the MCP initialize request
func (s *Server) handleInitialize(msg *MCPMessage) *MCPMessage {
	return &MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
			"serverInfo": map[string]interface{}{
				"name":    "mcp-logging-server",
				"version": "1.0.0",
			},
		},
	}
}

// handleToolsList handles the tools/list request
func (s *Server) handleToolsList(msg *MCPMessage) *MCPMessage {
	tools := make([]Tool, 0, len(s.tools))
	for _, tool := range s.tools {
		tools = append(tools, tool)
	}

	return &MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

// handleToolCall handles the tools/call request
func (s *Server) handleToolCall(ctx context.Context, msg *MCPMessage) *MCPMessage {
	params, ok := msg.Params.(map[string]interface{})
	if !ok {
		return &MCPMessage{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "Invalid params",
			},
		}
	}

	toolName, ok := params["name"].(string)
	if !ok {
		return &MCPMessage{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &MCPError{
				Code:    -32602,
				Message: "Missing tool name",
			},
		}
	}

	arguments := params["arguments"]

	var result *ToolResult
	var err error

	switch toolName {
	case "query_logs":
		result, err = s.handleQueryLogs(ctx, arguments)
	case "get_log_details":
		result, err = s.handleGetLogDetails(ctx, arguments)
	case "get_service_status":
		result, err = s.handleGetServiceStatus(ctx, arguments)
	case "list_services":
		result, err = s.handleListServices(ctx, arguments)
	default:
		return &MCPMessage{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: "Tool not found",
			},
		}
	}

	if err != nil {
		return &MCPMessage{
			JSONRPC: "2.0",
			ID:      msg.ID,
			Error: &MCPError{
				Code:    -32603,
				Message: err.Error(),
			},
		}
	}

	return &MCPMessage{
		JSONRPC: "2.0",
		ID:      msg.ID,
		Result:  result,
	}
}

// handleQueryLogs handles the query_logs tool call
func (s *Server) handleQueryLogs(ctx context.Context, arguments interface{}) (*ToolResult, error) {
	args, ok := arguments.(map[string]interface{})
	if !ok {
		args = make(map[string]interface{})
	}

	filter := models.LogFilter{}

	if serviceName, ok := args["service_name"].(string); ok {
		filter.ServiceName = serviceName
	}
	if agentID, ok := args["agent_id"].(string); ok {
		filter.AgentID = agentID
	}
	if level, ok := args["level"].(string); ok {
		filter.Level = models.LogLevel(level)
	}
	if platform, ok := args["platform"].(string); ok {
		filter.Platform = models.Platform(platform)
	}
	if messageContains, ok := args["message_contains"].(string); ok {
		filter.MessageContains = messageContains
	}
	if limit, ok := args["limit"].(float64); ok {
		filter.Limit = int(limit)
	} else {
		filter.Limit = 100
	}
	if offset, ok := args["offset"].(float64); ok {
		filter.Offset = int(offset)
	}

	// Parse time strings
	if startTimeStr, ok := args["start_time"].(string); ok {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartTime = startTime
		}
	}
	if endTimeStr, ok := args["end_time"].(string); ok {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndTime = endTime
		}
	}

	result, err := s.storage.Query(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}

	// Apply field masking for sensitive data protection
	maskedFields := s.getMaskedFields(args)
	if len(maskedFields) > 0 {
		result = s.applyFieldMasking(result, maskedFields)
	}

	// Add pagination information to the response
	actualLimit := filter.Limit
	if actualLimit == 0 {
		actualLimit = 100 // default limit
	}

	paginationInfo := map[string]interface{}{
		"total_count": result.TotalCount,
		"has_more":    result.HasMore,
		"limit":       actualLimit,
		"offset":      filter.Offset,
	}

	response := map[string]interface{}{
		"logs":       result.Logs,
		"pagination": paginationInfo,
	}

	// Format result as JSON text
	resultJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &ToolResult{
		Content: []ContentBlock{
			{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}

// getMaskedFields extracts field masking configuration from arguments
func (s *Server) getMaskedFields(args map[string]interface{}) []string {
	var maskedFields []string

	if maskFields, ok := args["mask_fields"].([]interface{}); ok {
		for _, field := range maskFields {
			if fieldStr, ok := field.(string); ok {
				maskedFields = append(maskedFields, fieldStr)
			}
		}
	}

	return maskedFields
}

// applyFieldMasking applies field masking to sensitive data
func (s *Server) applyFieldMasking(result *models.LogResult, maskedFields []string) *models.LogResult {
	if len(maskedFields) == 0 {
		return result
	}

	maskedResult := &models.LogResult{
		TotalCount: result.TotalCount,
		HasMore:    result.HasMore,
		Logs:       make([]models.LogEntry, len(result.Logs)),
	}

	for i, log := range result.Logs {
		maskedLog := log

		// Create a copy of metadata to avoid modifying original
		if log.Metadata != nil {
			maskedLog.Metadata = make(map[string]interface{})
			for k, v := range log.Metadata {
				maskedLog.Metadata[k] = v
			}
		}

		// Apply masking to specified fields
		for _, field := range maskedFields {
			switch field {
			case "message":
				maskedLog.Message = s.maskString(maskedLog.Message)
			case "agent_id":
				maskedLog.AgentID = s.maskString(maskedLog.AgentID)
			case "service_name":
				maskedLog.ServiceName = s.maskString(maskedLog.ServiceName)
			case "stack_trace":
				maskedLog.StackTrace = s.maskString(maskedLog.StackTrace)
			default:
				// Handle metadata fields
				if maskedLog.Metadata != nil {
					if _, exists := maskedLog.Metadata[field]; exists {
						if strVal, ok := maskedLog.Metadata[field].(string); ok {
							maskedLog.Metadata[field] = s.maskString(strVal)
						} else {
							maskedLog.Metadata[field] = "[MASKED]"
						}
					}
				}
			}
		}

		maskedResult.Logs[i] = maskedLog
	}

	return maskedResult
}

// maskString masks a string value for sensitive data protection
func (s *Server) maskString(value string) string {
	if len(value) <= 4 {
		return "[MASKED]"
	}

	// Show first 2 and last 2 characters, mask the middle
	return value[:2] + "[MASKED]" + value[len(value)-2:]
}

// handleGetLogDetails handles the get_log_details tool call
func (s *Server) handleGetLogDetails(ctx context.Context, arguments interface{}) (*ToolResult, error) {
	args, ok := arguments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid arguments")
	}

	idsInterface, ok := args["ids"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("missing or invalid ids parameter")
	}

	ids := make([]string, len(idsInterface))
	for i, id := range idsInterface {
		idStr, ok := id.(string)
		if !ok {
			return nil, fmt.Errorf("invalid id at index %d", i)
		}
		ids[i] = idStr
	}

	logs, err := s.storage.GetByIDs(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to get log details: %w", err)
	}

	// Apply field masking for sensitive data protection
	maskedFields := s.getMaskedFields(args)
	if len(maskedFields) > 0 {
		// Create a temporary LogResult to use the existing masking function
		tempResult := &models.LogResult{
			Logs:       logs,
			TotalCount: len(logs),
			HasMore:    false,
		}
		maskedResult := s.applyFieldMasking(tempResult, maskedFields)
		logs = maskedResult.Logs
	}

	// Format result as JSON text
	resultJSON, err := json.MarshalIndent(logs, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &ToolResult{
		Content: []ContentBlock{
			{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}

// handleGetServiceStatus handles the get_service_status tool call
func (s *Server) handleGetServiceStatus(ctx context.Context, arguments interface{}) (*ToolResult, error) {
	// Get storage health status
	storageStatus := s.storage.HealthCheck(ctx)

	// Create comprehensive system health report
	systemHealth := map[string]interface{}{
		"overall_status": "healthy",
		"timestamp":      time.Now(),
		"components": map[string]interface{}{
			"storage": map[string]interface{}{
				"status":    storageStatus.Status,
				"timestamp": storageStatus.Timestamp,
				"details":   storageStatus.Details,
			},
			"mcp_server": map[string]interface{}{
				"status":      "healthy",
				"port":        s.port,
				"tools_count": len(s.tools),
				"tools":       s.getToolNames(),
			},
		},
		"metrics": s.getSystemMetrics(ctx),
	}

	// Determine overall status based on components
	if storageStatus.Status != "healthy" {
		systemHealth["overall_status"] = "degraded"
	}

	// Format result as JSON text
	resultJSON, err := json.MarshalIndent(systemHealth, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &ToolResult{
		Content: []ContentBlock{
			{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}

// getToolNames returns a list of available tool names
func (s *Server) getToolNames() []string {
	names := make([]string, 0, len(s.tools))
	for name := range s.tools {
		names = append(names, name)
	}
	return names
}

// getSystemMetrics returns basic system metrics
func (s *Server) getSystemMetrics(ctx context.Context) map[string]interface{} {
	// Get basic metrics from storage
	services, err := s.storage.GetServices(ctx)
	if err != nil {
		return map[string]interface{}{
			"error": "failed to get metrics",
		}
	}

	totalLogCount := 0
	platformCounts := make(map[string]int)

	for _, service := range services {
		totalLogCount += service.LogCount
		platformCounts[string(service.Platform)]++
	}

	return map[string]interface{}{
		"total_services":  len(services),
		"total_log_count": totalLogCount,
		"platform_counts": platformCounts,
		"uptime_seconds":  time.Since(time.Now().Add(-time.Hour)).Seconds(), // Mock uptime
	}
}

// handleListServices handles the list_services tool call
func (s *Server) handleListServices(ctx context.Context, arguments interface{}) (*ToolResult, error) {
	services, err := s.storage.GetServices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}

	// Create enhanced service listing with summary
	serviceList := map[string]interface{}{
		"services": services,
		"summary": map[string]interface{}{
			"total_services": len(services),
			"platforms":      s.getPlatformSummary(services),
			"last_updated":   time.Now(),
		},
	}

	// Format result as JSON text
	resultJSON, err := json.MarshalIndent(serviceList, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return &ToolResult{
		Content: []ContentBlock{
			{
				Type: "text",
				Text: string(resultJSON),
			},
		},
	}, nil
}

// getPlatformSummary creates a summary of services by platform
func (s *Server) getPlatformSummary(services []models.ServiceInfo) map[string]interface{} {
	platformCounts := make(map[string]int)
	platformLogCounts := make(map[string]int)

	for _, service := range services {
		platform := string(service.Platform)
		platformCounts[platform]++
		platformLogCounts[platform] += service.LogCount
	}

	return map[string]interface{}{
		"service_counts": platformCounts,
		"log_counts":     platformLogCounts,
	}
}
