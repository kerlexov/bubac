package storage

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
	"github.com/blevesearch/bleve/v2/search/query"
	"github.com/kerlexov/mcp-logging-server/pkg/models"
)

// SearchableLogEntry represents a log entry optimized for search indexing
type SearchableLogEntry struct {
	ID             string                 `json:"id"`
	Timestamp      time.Time              `json:"timestamp"`
	Level          string                 `json:"level"`
	Message        string                 `json:"message"`
	ServiceName    string                 `json:"service_name"`
	AgentID        string                 `json:"agent_id"`
	Platform       string                 `json:"platform"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
	StackTrace     string                 `json:"stack_trace,omitempty"`
	DevicePlatform string                 `json:"device_platform,omitempty"`
	DeviceModel    string                 `json:"device_model,omitempty"`
	SourceFile     string                 `json:"source_file,omitempty"`
	SourceFunction string                 `json:"source_function,omitempty"`
}

// SearchService provides full-text search capabilities for log entries
type SearchService struct {
	index bleve.Index
}

// NewSearchService creates a new search service with Bleve index
func NewSearchService(indexPath string) (*SearchService, error) {
	var index bleve.Index
	var err error

	// Check if index already exists
	if _, statErr := os.Stat(indexPath); os.IsNotExist(statErr) {
		// Create new index
		mapping := buildIndexMapping()
		index, err = bleve.New(indexPath, mapping)
		if err != nil {
			return nil, fmt.Errorf("failed to create search index: %w", err)
		}
	} else {
		// Open existing index
		index, err = bleve.Open(indexPath)
		if err != nil {
			return nil, fmt.Errorf("failed to open search index: %w", err)
		}
	}

	return &SearchService{index: index}, nil
}

// buildIndexMapping creates the Bleve index mapping for log entries
func buildIndexMapping() mapping.IndexMapping {
	// Create a mapping
	logMapping := bleve.NewDocumentMapping()

	// ID field - keyword (exact match)
	idFieldMapping := bleve.NewTextFieldMapping()
	idFieldMapping.Analyzer = "keyword"
	logMapping.AddFieldMappingsAt("id", idFieldMapping)

	// Timestamp field - datetime
	timestampFieldMapping := bleve.NewDateTimeFieldMapping()
	logMapping.AddFieldMappingsAt("timestamp", timestampFieldMapping)

	// Level field - keyword (exact match)
	levelFieldMapping := bleve.NewTextFieldMapping()
	levelFieldMapping.Analyzer = "keyword"
	logMapping.AddFieldMappingsAt("level", levelFieldMapping)

	// Message field - full text search
	messageFieldMapping := bleve.NewTextFieldMapping()
	messageFieldMapping.Analyzer = "standard"
	logMapping.AddFieldMappingsAt("message", messageFieldMapping)

	// Service name field - keyword (exact match)
	serviceFieldMapping := bleve.NewTextFieldMapping()
	serviceFieldMapping.Analyzer = "keyword"
	logMapping.AddFieldMappingsAt("service_name", serviceFieldMapping)

	// Agent ID field - keyword (exact match)
	agentFieldMapping := bleve.NewTextFieldMapping()
	agentFieldMapping.Analyzer = "keyword"
	logMapping.AddFieldMappingsAt("agent_id", agentFieldMapping)

	// Platform field - keyword (exact match)
	platformFieldMapping := bleve.NewTextFieldMapping()
	platformFieldMapping.Analyzer = "keyword"
	logMapping.AddFieldMappingsAt("platform", platformFieldMapping)

	// Stack trace field - full text search
	stackTraceFieldMapping := bleve.NewTextFieldMapping()
	stackTraceFieldMapping.Analyzer = "standard"
	logMapping.AddFieldMappingsAt("stack_trace", stackTraceFieldMapping)

	// Device fields - keyword (exact match)
	devicePlatformFieldMapping := bleve.NewTextFieldMapping()
	devicePlatformFieldMapping.Analyzer = "keyword"
	logMapping.AddFieldMappingsAt("device_platform", devicePlatformFieldMapping)

	deviceModelFieldMapping := bleve.NewTextFieldMapping()
	deviceModelFieldMapping.Analyzer = "keyword"
	logMapping.AddFieldMappingsAt("device_model", deviceModelFieldMapping)

	// Source location fields - keyword (exact match)
	sourceFileFieldMapping := bleve.NewTextFieldMapping()
	sourceFileFieldMapping.Analyzer = "keyword"
	logMapping.AddFieldMappingsAt("source_file", sourceFileFieldMapping)

	sourceFunctionFieldMapping := bleve.NewTextFieldMapping()
	sourceFunctionFieldMapping.Analyzer = "keyword"
	logMapping.AddFieldMappingsAt("source_function", sourceFunctionFieldMapping)

	// Create index mapping
	indexMapping := bleve.NewIndexMapping()
	indexMapping.AddDocumentMapping("log", logMapping)
	indexMapping.DefaultMapping = logMapping

	return indexMapping
}

// IndexLogEntry adds or updates a log entry in the search index
func (s *SearchService) IndexLogEntry(logEntry models.LogEntry) error {
	searchableEntry := s.convertToSearchable(logEntry)
	return s.index.Index(logEntry.ID, searchableEntry)
}

// IndexLogEntries adds or updates multiple log entries in the search index
func (s *SearchService) IndexLogEntries(logEntries []models.LogEntry) error {
	batch := s.index.NewBatch()

	for _, logEntry := range logEntries {
		searchableEntry := s.convertToSearchable(logEntry)
		if err := batch.Index(logEntry.ID, searchableEntry); err != nil {
			return fmt.Errorf("failed to add log entry %s to batch: %w", logEntry.ID, err)
		}
	}

	return s.index.Batch(batch)
}

// SearchLogs performs a full-text search on log entries
func (s *SearchService) SearchLogs(ctx context.Context, query string, filter models.LogFilter) ([]string, error) {
	// Build search query
	searchQuery := s.buildSearchQuery(query, filter)

	// Create search request
	searchRequest := bleve.NewSearchRequest(searchQuery)
	searchRequest.Size = filter.Limit
	if filter.Limit <= 0 {
		searchRequest.Size = 100
	}
	searchRequest.From = filter.Offset
	if filter.Offset < 0 {
		searchRequest.From = 0
	}

	// Sort by timestamp descending
	searchRequest.SortBy([]string{"-timestamp"})

	// Execute search
	searchResult, err := s.index.Search(searchRequest)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	// Extract log IDs from search results
	var logIDs []string
	for _, hit := range searchResult.Hits {
		logIDs = append(logIDs, hit.ID)
	}

	return logIDs, nil
}

// buildSearchQuery constructs a Bleve query based on search text and filters
func (s *SearchService) buildSearchQuery(queryText string, filter models.LogFilter) query.Query {
	var queries []query.Query

	// Full-text search query
	if queryText != "" {
		// Search in message and stack trace fields
		messageQuery := bleve.NewMatchQuery(queryText)
		messageQuery.SetField("message")

		stackTraceQuery := bleve.NewMatchQuery(queryText)
		stackTraceQuery.SetField("stack_trace")

		textQuery := bleve.NewDisjunctionQuery(messageQuery, stackTraceQuery)
		queries = append(queries, textQuery)
	}

	// Filter by service name
	if filter.ServiceName != "" {
		serviceQuery := bleve.NewTermQuery(filter.ServiceName)
		serviceQuery.SetField("service_name")
		queries = append(queries, serviceQuery)
	}

	// Filter by agent ID
	if filter.AgentID != "" {
		agentQuery := bleve.NewTermQuery(filter.AgentID)
		agentQuery.SetField("agent_id")
		queries = append(queries, agentQuery)
	}

	// Filter by level
	if filter.Level != "" {
		levelQuery := bleve.NewTermQuery(string(filter.Level))
		levelQuery.SetField("level")
		queries = append(queries, levelQuery)
	}

	// Filter by platform
	if filter.Platform != "" {
		platformQuery := bleve.NewTermQuery(string(filter.Platform))
		platformQuery.SetField("platform")
		queries = append(queries, platformQuery)
	}

	// Filter by time range
	if !filter.StartTime.IsZero() || !filter.EndTime.IsZero() {
		var timeQuery *query.DateRangeQuery

		if !filter.StartTime.IsZero() && !filter.EndTime.IsZero() {
			// Both start and end times specified
			timeQuery = bleve.NewDateRangeQuery(filter.StartTime, filter.EndTime)
		} else if !filter.StartTime.IsZero() {
			// Only start time specified - use a very far future date as end
			farFuture := time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC)
			timeQuery = bleve.NewDateRangeQuery(filter.StartTime, farFuture)
		} else {
			// Only end time specified - use epoch as start
			epoch := time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC)
			timeQuery = bleve.NewDateRangeQuery(epoch, filter.EndTime)
		}

		timeQuery.SetField("timestamp")
		queries = append(queries, timeQuery)
	}

	// If no queries, return match all
	if len(queries) == 0 {
		return bleve.NewMatchAllQuery()
	}

	// If only one query, return it directly
	if len(queries) == 1 {
		return queries[0]
	}

	// Combine all queries with AND
	return bleve.NewConjunctionQuery(queries...)
}

// convertToSearchable converts a LogEntry to SearchableLogEntry
func (s *SearchService) convertToSearchable(logEntry models.LogEntry) SearchableLogEntry {
	searchable := SearchableLogEntry{
		ID:          logEntry.ID,
		Timestamp:   logEntry.Timestamp,
		Level:       string(logEntry.Level),
		Message:     logEntry.Message,
		ServiceName: logEntry.ServiceName,
		AgentID:     logEntry.AgentID,
		Platform:    string(logEntry.Platform),
		Metadata:    logEntry.Metadata,
		StackTrace:  logEntry.StackTrace,
	}

	// Extract device information
	if logEntry.DeviceInfo != nil {
		searchable.DevicePlatform = logEntry.DeviceInfo.Platform
		searchable.DeviceModel = logEntry.DeviceInfo.Model
	}

	// Extract source location information
	if logEntry.SourceLocation != nil {
		searchable.SourceFile = logEntry.SourceLocation.File
		searchable.SourceFunction = logEntry.SourceLocation.Function
	}

	return searchable
}

// DeleteLogEntry removes a log entry from the search index
func (s *SearchService) DeleteLogEntry(id string) error {
	return s.index.Delete(id)
}

// GetIndexStats returns statistics about the search index
func (s *SearchService) GetIndexStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get document count
	docCount, err := s.index.DocCount()
	if err != nil {
		return nil, fmt.Errorf("failed to get document count: %w", err)
	}
	stats["document_count"] = docCount

	// Get index size (approximate)
	if indexImpl, ok := s.index.(interface{ StatsMap() map[string]interface{} }); ok {
		indexStats := indexImpl.StatsMap()
		stats["index_stats"] = indexStats
	}

	return stats, nil
}

// Close closes the search index
func (s *SearchService) Close() error {
	return s.index.Close()
}

// HealthCheck returns the health status of the search service
func (s *SearchService) HealthCheck(ctx context.Context) models.HealthStatus {
	status := models.HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Details:   make(map[string]string),
	}

	// Check if index is accessible
	docCount, err := s.index.DocCount()
	if err != nil {
		status.Status = "unhealthy"
		status.Details["index"] = fmt.Sprintf("failed to get document count: %v", err)
		return status
	}

	status.Details["index"] = "accessible"
	status.Details["document_count"] = strconv.FormatUint(docCount, 10)

	return status
}
