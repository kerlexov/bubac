package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/your-org/mcp-logging-server/pkg/models"
	_ "github.com/mattn/go-sqlite3"
)

// SQLiteStorage implements LogStorage using SQLite
type SQLiteStorage struct {
	db     *sql.DB
	search *SearchService
}

// NewSQLiteStorage creates a new SQLite storage instance
func NewSQLiteStorage(connectionString string) (*SQLiteStorage, error) {
	return NewSQLiteStorageWithSearch(connectionString, "")
}

// NewSQLiteStorageWithSearch creates a new SQLite storage instance with search capabilities
func NewSQLiteStorageWithSearch(connectionString, searchIndexPath string) (*SQLiteStorage, error) {
	db, err := sql.Open("sqlite3", connectionString)
	if err != nil {
		return nil, err
	}
	
	// Enable foreign keys and WAL mode for better performance
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}
	
	if _, err := db.Exec("PRAGMA journal_mode = WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}
	
	storage := &SQLiteStorage{db: db}
	
	// Initialize database schema
	if err := storage.migrate(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}
	
	// Initialize search service if path is provided
	if searchIndexPath != "" {
		searchService, err := NewSearchService(searchIndexPath)
		if err != nil {
			db.Close()
			return nil, fmt.Errorf("failed to initialize search service: %w", err)
		}
		storage.search = searchService
	}
	
	return storage, nil
}

// migrate runs database migrations
func (s *SQLiteStorage) migrate() error {
	// Create migrations table if it doesn't exist
	createMigrationsTable := `
	CREATE TABLE IF NOT EXISTS migrations (
		version INTEGER PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	
	if _, err := s.db.Exec(createMigrationsTable); err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}
	
	// Define migrations
	migrations := []struct {
		version int
		sql     string
	}{
		{
			version: 1,
			sql: `
			CREATE TABLE IF NOT EXISTS log_entries (
				id TEXT PRIMARY KEY,
				timestamp DATETIME NOT NULL,
				level TEXT NOT NULL CHECK (level IN ('DEBUG', 'INFO', 'WARN', 'ERROR', 'FATAL')),
				message TEXT NOT NULL,
				service_name TEXT NOT NULL,
				agent_id TEXT NOT NULL,
				platform TEXT NOT NULL CHECK (platform IN ('go', 'swift', 'express', 'react', 'react-native', 'kotlin')),
				metadata TEXT, -- JSON
				device_info TEXT, -- JSON
				stack_trace TEXT,
				source_location TEXT, -- JSON
				created_at DATETIME DEFAULT CURRENT_TIMESTAMP
			);
			
			CREATE INDEX IF NOT EXISTS idx_log_entries_timestamp ON log_entries(timestamp);
			CREATE INDEX IF NOT EXISTS idx_log_entries_level ON log_entries(level);
			CREATE INDEX IF NOT EXISTS idx_log_entries_service_name ON log_entries(service_name);
			CREATE INDEX IF NOT EXISTS idx_log_entries_agent_id ON log_entries(agent_id);
			CREATE INDEX IF NOT EXISTS idx_log_entries_platform ON log_entries(platform);
			CREATE INDEX IF NOT EXISTS idx_log_entries_service_agent ON log_entries(service_name, agent_id);
			`,
		},
	}
	
	// Apply migrations
	for _, migration := range migrations {
		var count int
		err := s.db.QueryRow("SELECT COUNT(*) FROM migrations WHERE version = ?", migration.version).Scan(&count)
		if err != nil {
			return fmt.Errorf("failed to check migration version %d: %w", migration.version, err)
		}
		
		if count == 0 {
			// Apply migration
			if _, err := s.db.Exec(migration.sql); err != nil {
				return fmt.Errorf("failed to apply migration version %d: %w", migration.version, err)
			}
			
			// Record migration
			if _, err := s.db.Exec("INSERT INTO migrations (version) VALUES (?)", migration.version); err != nil {
				return fmt.Errorf("failed to record migration version %d: %w", migration.version, err)
			}
		}
	}
	
	return nil
}

// Store stores a batch of log entries
func (s *SQLiteStorage) Store(ctx context.Context, logs []models.LogEntry) error {
	if len(logs) == 0 {
		return nil
	}
	
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO log_entries (
			id, timestamp, level, message, service_name, agent_id, platform,
			metadata, device_info, stack_trace, source_location
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()
	
	for _, log := range logs {
		// Validate log entry
		if err := log.Validate(); err != nil {
			return fmt.Errorf("invalid log entry %s: %w", log.ID, err)
		}
		
		// Serialize JSON fields
		var metadataJSON, deviceInfoJSON, sourceLocationJSON *string
		
		if log.Metadata != nil {
			if data, err := json.Marshal(log.Metadata); err != nil {
				return fmt.Errorf("failed to marshal metadata for log %s: %w", log.ID, err)
			} else {
				metadataStr := string(data)
				metadataJSON = &metadataStr
			}
		}
		
		if log.DeviceInfo != nil {
			if data, err := json.Marshal(log.DeviceInfo); err != nil {
				return fmt.Errorf("failed to marshal device info for log %s: %w", log.ID, err)
			} else {
				deviceInfoStr := string(data)
				deviceInfoJSON = &deviceInfoStr
			}
		}
		
		if log.SourceLocation != nil {
			if data, err := json.Marshal(log.SourceLocation); err != nil {
				return fmt.Errorf("failed to marshal source location for log %s: %w", log.ID, err)
			} else {
				sourceLocationStr := string(data)
				sourceLocationJSON = &sourceLocationStr
			}
		}
		
		var stackTrace *string
		if log.StackTrace != "" {
			stackTrace = &log.StackTrace
		}
		
		_, err := stmt.ExecContext(ctx,
			log.ID,
			log.Timestamp,
			string(log.Level),
			log.Message,
			log.ServiceName,
			log.AgentID,
			string(log.Platform),
			metadataJSON,
			deviceInfoJSON,
			stackTrace,
			sourceLocationJSON,
		)
		if err != nil {
			return fmt.Errorf("failed to insert log entry %s: %w", log.ID, err)
		}
	}
	
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	// Index logs for search if search service is available
	if s.search != nil {
		if err := s.search.IndexLogEntries(logs); err != nil {
			// Log error but don't fail the storage operation
			fmt.Printf("Warning: failed to index logs for search: %v\n", err)
		}
	}
	
	return nil
}

// Query retrieves logs based on filter criteria
func (s *SQLiteStorage) Query(ctx context.Context, filter models.LogFilter) (*models.LogResult, error) {
	// If search service is available and message contains filter is used, use full-text search
	if s.search != nil && filter.MessageContains != "" {
		return s.queryWithSearch(ctx, filter)
	}
	
	return s.queryWithSQL(ctx, filter)
}

// queryWithSearch performs a search using the Bleve index and then retrieves full records from SQL
func (s *SQLiteStorage) queryWithSearch(ctx context.Context, filter models.LogFilter) (*models.LogResult, error) {
	// Perform search to get log IDs
	logIDs, err := s.search.SearchLogs(ctx, filter.MessageContains, filter)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}
	
	if len(logIDs) == 0 {
		return &models.LogResult{
			Logs:       []models.LogEntry{},
			TotalCount: 0,
			HasMore:    false,
		}, nil
	}
	
	// Get full log entries by IDs
	logs, err := s.GetByIDs(ctx, logIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get logs by IDs: %w", err)
	}
	
	// Apply additional SQL-based filtering if needed (for precise filtering)
	filteredLogs := s.applyAdditionalFiltering(logs, filter)
	
	// Calculate pagination
	totalCount := len(filteredLogs)
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	
	// Apply pagination
	end := offset + limit
	if end > len(filteredLogs) {
		end = len(filteredLogs)
	}
	
	var paginatedLogs []models.LogEntry
	if offset < len(filteredLogs) {
		paginatedLogs = filteredLogs[offset:end]
	}
	
	hasMore := offset+len(paginatedLogs) < totalCount
	
	return &models.LogResult{
		Logs:       paginatedLogs,
		TotalCount: totalCount,
		HasMore:    hasMore,
	}, nil
}

// applyAdditionalFiltering applies filters that weren't handled by the search
func (s *SQLiteStorage) applyAdditionalFiltering(logs []models.LogEntry, filter models.LogFilter) []models.LogEntry {
	var filtered []models.LogEntry
	
	for _, log := range logs {
		// Additional time range filtering (search might be less precise)
		if !filter.StartTime.IsZero() && log.Timestamp.Before(filter.StartTime) {
			continue
		}
		if !filter.EndTime.IsZero() && log.Timestamp.After(filter.EndTime) {
			continue
		}
		
		filtered = append(filtered, log)
	}
	
	return filtered
}

// queryWithSQL performs a traditional SQL-based query
func (s *SQLiteStorage) queryWithSQL(ctx context.Context, filter models.LogFilter) (*models.LogResult, error) {
	// Build WHERE clause and args
	var conditions []string
	var args []interface{}
	argIndex := 0
	
	if filter.ServiceName != "" {
		conditions = append(conditions, "service_name = ?")
		args = append(args, filter.ServiceName)
		argIndex++
	}
	
	if filter.AgentID != "" {
		conditions = append(conditions, "agent_id = ?")
		args = append(args, filter.AgentID)
		argIndex++
	}
	
	if filter.Level != "" {
		conditions = append(conditions, "level = ?")
		args = append(args, string(filter.Level))
		argIndex++
	}
	
	if filter.Platform != "" {
		conditions = append(conditions, "platform = ?")
		args = append(args, string(filter.Platform))
		argIndex++
	}
	
	if !filter.StartTime.IsZero() {
		conditions = append(conditions, "timestamp >= ?")
		args = append(args, filter.StartTime)
		argIndex++
	}
	
	if !filter.EndTime.IsZero() {
		conditions = append(conditions, "timestamp <= ?")
		args = append(args, filter.EndTime)
		argIndex++
	}
	
	if filter.MessageContains != "" {
		conditions = append(conditions, "message LIKE ?")
		args = append(args, "%"+filter.MessageContains+"%")
		argIndex++
	}
	
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}
	
	// Set default limit if not specified
	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	
	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM log_entries %s", whereClause)
	var totalCount int
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, fmt.Errorf("failed to get total count: %w", err)
	}
	
	// Get logs
	query := fmt.Sprintf(`
		SELECT id, timestamp, level, message, service_name, agent_id, platform,
			   metadata, device_info, stack_trace, source_location
		FROM log_entries %s
		ORDER BY timestamp DESC
		LIMIT ? OFFSET ?
	`, whereClause)
	
	args = append(args, limit, offset)
	
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()
	
	var logs []models.LogEntry
	for rows.Next() {
		var log models.LogEntry
		var metadataJSON, deviceInfoJSON, sourceLocationJSON, stackTrace sql.NullString
		
		err := rows.Scan(
			&log.ID,
			&log.Timestamp,
			&log.Level,
			&log.Message,
			&log.ServiceName,
			&log.AgentID,
			&log.Platform,
			&metadataJSON,
			&deviceInfoJSON,
			&stackTrace,
			&sourceLocationJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log entry: %w", err)
		}
		
		// Deserialize JSON fields
		if metadataJSON.Valid {
			if err := json.Unmarshal([]byte(metadataJSON.String), &log.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata for log %s: %w", log.ID, err)
			}
		}
		
		if deviceInfoJSON.Valid {
			log.DeviceInfo = &models.DeviceInfo{}
			if err := json.Unmarshal([]byte(deviceInfoJSON.String), log.DeviceInfo); err != nil {
				return nil, fmt.Errorf("failed to unmarshal device info for log %s: %w", log.ID, err)
			}
		}
		
		if sourceLocationJSON.Valid {
			log.SourceLocation = &models.SourceLocation{}
			if err := json.Unmarshal([]byte(sourceLocationJSON.String), log.SourceLocation); err != nil {
				return nil, fmt.Errorf("failed to unmarshal source location for log %s: %w", log.ID, err)
			}
		}
		
		if stackTrace.Valid {
			log.StackTrace = stackTrace.String
		}
		
		logs = append(logs, log)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	
	hasMore := offset+len(logs) < totalCount
	
	return &models.LogResult{
		Logs:       logs,
		TotalCount: totalCount,
		HasMore:    hasMore,
	}, nil
}

// GetByIDs retrieves specific log entries by their IDs
func (s *SQLiteStorage) GetByIDs(ctx context.Context, ids []string) ([]models.LogEntry, error) {
	if len(ids) == 0 {
		return []models.LogEntry{}, nil
	}
	
	// Build IN clause with placeholders
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	
	query := fmt.Sprintf(`
		SELECT id, timestamp, level, message, service_name, agent_id, platform,
			   metadata, device_info, stack_trace, source_location
		FROM log_entries
		WHERE id IN (%s)
		ORDER BY timestamp DESC
	`, strings.Join(placeholders, ","))
	
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query logs by IDs: %w", err)
	}
	defer rows.Close()
	
	var logs []models.LogEntry
	for rows.Next() {
		var log models.LogEntry
		var metadataJSON, deviceInfoJSON, sourceLocationJSON, stackTrace sql.NullString
		
		err := rows.Scan(
			&log.ID,
			&log.Timestamp,
			&log.Level,
			&log.Message,
			&log.ServiceName,
			&log.AgentID,
			&log.Platform,
			&metadataJSON,
			&deviceInfoJSON,
			&stackTrace,
			&sourceLocationJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log entry: %w", err)
		}
		
		// Deserialize JSON fields
		if metadataJSON.Valid {
			if err := json.Unmarshal([]byte(metadataJSON.String), &log.Metadata); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metadata for log %s: %w", log.ID, err)
			}
		}
		
		if deviceInfoJSON.Valid {
			log.DeviceInfo = &models.DeviceInfo{}
			if err := json.Unmarshal([]byte(deviceInfoJSON.String), log.DeviceInfo); err != nil {
				return nil, fmt.Errorf("failed to unmarshal device info for log %s: %w", log.ID, err)
			}
		}
		
		if sourceLocationJSON.Valid {
			log.SourceLocation = &models.SourceLocation{}
			if err := json.Unmarshal([]byte(sourceLocationJSON.String), log.SourceLocation); err != nil {
				return nil, fmt.Errorf("failed to unmarshal source location for log %s: %w", log.ID, err)
			}
		}
		
		if stackTrace.Valid {
			log.StackTrace = stackTrace.String
		}
		
		logs = append(logs, log)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	
	return logs, nil
}

// GetServices returns a list of services that have logged entries
func (s *SQLiteStorage) GetServices(ctx context.Context) ([]models.ServiceInfo, error) {
	query := `
		SELECT service_name, agent_id, platform, MAX(timestamp) as last_seen, COUNT(*) as log_count
		FROM log_entries
		GROUP BY service_name, agent_id, platform
		ORDER BY last_seen DESC
	`
	
	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query services: %w", err)
	}
	defer rows.Close()
	
	var services []models.ServiceInfo
	for rows.Next() {
		var service models.ServiceInfo
		var platformStr string
		var lastSeenStr string
		
		err := rows.Scan(
			&service.ServiceName,
			&service.AgentID,
			&platformStr,
			&lastSeenStr,
			&service.LogCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan service info: %w", err)
		}
		
		// Parse timestamp string
		lastSeen, err := time.Parse("2006-01-02 15:04:05.999999999-07:00", lastSeenStr)
		if err != nil {
			// Try alternative format
			lastSeen, err = time.Parse("2006-01-02 15:04:05", lastSeenStr)
			if err != nil {
				return nil, fmt.Errorf("failed to parse last_seen timestamp: %w", err)
			}
		}
		
		service.Platform = models.Platform(platformStr)
		service.LastSeen = lastSeen
		services = append(services, service)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}
	
	return services, nil
}

// DeleteByIDs deletes log entries by their IDs and returns the number of deleted entries
func (s *SQLiteStorage) DeleteByIDs(ctx context.Context, ids []string) (int, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Build IN clause with placeholders
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}
	
	query := fmt.Sprintf("DELETE FROM log_entries WHERE id IN (%s)", strings.Join(placeholders, ","))
	
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return 0, fmt.Errorf("failed to delete log entries: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	// Remove from search index if available
	if s.search != nil {
		for _, id := range ids {
			if err := s.search.DeleteLogEntry(id); err != nil {
				// Log error but don't fail the deletion
				fmt.Printf("Warning: failed to delete log %s from search index: %v\n", id, err)
			}
		}
	}
	
	return int(rowsAffected), nil
}

// HealthCheck returns the health status of the storage system
func (s *SQLiteStorage) HealthCheck(ctx context.Context) models.HealthStatus {
	status := models.HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Details:   make(map[string]string),
	}
	
	// Test database connection
	if err := s.db.PingContext(ctx); err != nil {
		status.Status = "unhealthy"
		status.Details["database"] = fmt.Sprintf("ping failed: %v", err)
		return status
	}
	
	// Test basic query
	var count int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM log_entries").Scan(&count); err != nil {
		status.Status = "unhealthy"
		status.Details["query"] = fmt.Sprintf("count query failed: %v", err)
		return status
	}
	
	status.Details["database"] = "connected"
	status.Details["log_count"] = fmt.Sprintf("%d", count)
	
	return status
}

// Close closes the storage connection
func (s *SQLiteStorage) Close() error {
	var err error
	
	if s.search != nil {
		if searchErr := s.search.Close(); searchErr != nil {
			err = fmt.Errorf("failed to close search service: %w", searchErr)
		}
	}
	
	if s.db != nil {
		if dbErr := s.db.Close(); dbErr != nil {
			if err != nil {
				err = fmt.Errorf("%w; failed to close database: %w", err, dbErr)
			} else {
				err = fmt.Errorf("failed to close database: %w", dbErr)
			}
		}
	}
	
	return err
}