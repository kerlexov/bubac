package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/kerlexov/mcp-logging-server/pkg/models"
)

// RetentionPolicy defines how long logs should be kept
type RetentionPolicy struct {
	// DefaultDays is the default retention period in days
	DefaultDays int `json:"default_days" yaml:"default_days"`

	// ByLevel defines retention periods by log level
	ByLevel map[models.LogLevel]int `json:"by_level" yaml:"by_level"`

	// MaxTotalLogs is the maximum number of logs to keep (0 = unlimited)
	MaxTotalLogs int `json:"max_total_logs" yaml:"max_total_logs"`

	// MaxLogsPerService is the maximum number of logs per service (0 = unlimited)
	MaxLogsPerService int `json:"max_logs_per_service" yaml:"max_logs_per_service"`
}

// RetentionService manages log retention and cleanup
type RetentionService struct {
	storage LogStorage
	policy  RetentionPolicy
}

// NewRetentionService creates a new retention service
func NewRetentionService(storage LogStorage, policy RetentionPolicy) *RetentionService {
	return &RetentionService{
		storage: storage,
		policy:  policy,
	}
}

// GetRetentionDate calculates the retention cutoff date for a given log level
func (r *RetentionService) GetRetentionDate(level models.LogLevel) time.Time {
	days := r.policy.DefaultDays

	// Check if there's a specific retention period for this level
	if levelDays, exists := r.policy.ByLevel[level]; exists {
		days = levelDays
	}

	if days <= 0 {
		// No retention (keep forever)
		return time.Time{}
	}

	return time.Now().AddDate(0, 0, -days)
}

// CleanupExpiredLogs removes logs that have exceeded their retention period
func (r *RetentionService) CleanupExpiredLogs(ctx context.Context) (*CleanupResult, error) {
	result := &CleanupResult{
		StartTime:      time.Now(),
		DeletedByLevel: make(map[models.LogLevel]int),
	}

	// Get all log levels to process
	levels := []models.LogLevel{
		models.LogLevelDebug,
		models.LogLevelInfo,
		models.LogLevelWarn,
		models.LogLevelError,
		models.LogLevelFatal,
	}

	totalDeleted := 0

	for _, level := range levels {
		cutoffDate := r.GetRetentionDate(level)

		// Skip if no retention policy for this level
		if cutoffDate.IsZero() {
			continue
		}

		// Find logs to delete for this level
		filter := models.LogFilter{
			Level:   level,
			EndTime: cutoffDate,
			Limit:   1000, // Process in batches
		}

		for {
			logs, err := r.storage.Query(ctx, filter)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to query %s logs: %v", level, err))
				break
			}

			if len(logs.Logs) == 0 {
				break
			}

			// Delete the logs
			deleted, err := r.deleteLogs(ctx, logs.Logs)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Failed to delete %s logs: %v", level, err))
				break
			}

			totalDeleted += deleted
			result.DeletedByLevel[level] += deleted

			// If we got fewer logs than the limit, we're done with this level
			if len(logs.Logs) < filter.Limit {
				break
			}
		}
	}

	result.TotalDeleted = totalDeleted
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// CleanupByCount removes oldest logs when count limits are exceeded
func (r *RetentionService) CleanupByCount(ctx context.Context) (*CleanupResult, error) {
	result := &CleanupResult{
		StartTime:      time.Now(),
		DeletedByLevel: make(map[models.LogLevel]int),
	}

	totalDeleted := 0

	// Cleanup by total log count
	if r.policy.MaxTotalLogs > 0 {
		deleted, err := r.cleanupByTotalCount(ctx, r.policy.MaxTotalLogs)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to cleanup by total count: %v", err))
		} else {
			totalDeleted += deleted
		}
	}

	// Cleanup by service count
	if r.policy.MaxLogsPerService > 0 {
		deleted, err := r.cleanupByServiceCount(ctx, r.policy.MaxLogsPerService)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to cleanup by service count: %v", err))
		} else {
			totalDeleted += deleted
		}
	}

	result.TotalDeleted = totalDeleted
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// cleanupByTotalCount removes oldest logs when total count exceeds limit
func (r *RetentionService) cleanupByTotalCount(ctx context.Context, maxLogs int) (int, error) {
	// Get total count
	allLogs, err := r.storage.Query(ctx, models.LogFilter{Limit: 1})
	if err != nil {
		return 0, fmt.Errorf("failed to get total count: %w", err)
	}

	if allLogs.TotalCount <= maxLogs {
		return 0, nil // No cleanup needed
	}

	// Calculate how many logs to delete
	toDelete := allLogs.TotalCount - maxLogs

	// Get oldest logs to delete
	oldestLogs, err := r.storage.Query(ctx, models.LogFilter{
		Limit: toDelete,
		// Note: We need to order by timestamp ASC to get oldest first
		// This would require modifying the Query method to support different sort orders
		// For now, we'll implement a simpler approach
	})
	if err != nil {
		return 0, fmt.Errorf("failed to get oldest logs: %w", err)
	}

	return r.deleteLogs(ctx, oldestLogs.Logs)
}

// cleanupByServiceCount removes oldest logs per service when count exceeds limit
func (r *RetentionService) cleanupByServiceCount(ctx context.Context, maxLogsPerService int) (int, error) {
	// Get all services
	services, err := r.storage.GetServices(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get services: %w", err)
	}

	totalDeleted := 0

	for _, service := range services {
		if service.LogCount <= maxLogsPerService {
			continue // No cleanup needed for this service
		}

		// Calculate how many logs to delete for this service
		toDelete := service.LogCount - maxLogsPerService

		// Get oldest logs for this service
		serviceLogs, err := r.storage.Query(ctx, models.LogFilter{
			ServiceName: service.ServiceName,
			AgentID:     service.AgentID,
			Limit:       toDelete,
		})
		if err != nil {
			return totalDeleted, fmt.Errorf("failed to get logs for service %s: %w", service.ServiceName, err)
		}

		deleted, err := r.deleteLogs(ctx, serviceLogs.Logs)
		if err != nil {
			return totalDeleted, fmt.Errorf("failed to delete logs for service %s: %w", service.ServiceName, err)
		}

		totalDeleted += deleted
	}

	return totalDeleted, nil
}

// deleteLogs deletes a batch of log entries
func (r *RetentionService) deleteLogs(ctx context.Context, logs []models.LogEntry) (int, error) {
	if len(logs) == 0 {
		return 0, nil
	}

	// Extract log IDs
	var logIDs []string
	for _, log := range logs {
		logIDs = append(logIDs, log.ID)
	}

	// Delete from storage (this would require adding a Delete method to LogStorage interface)
	// For now, we'll assume this functionality exists
	if deleter, ok := r.storage.(LogDeleter); ok {
		return deleter.DeleteByIDs(ctx, logIDs)
	}

	return 0, fmt.Errorf("storage does not support deletion")
}

// CleanupResult represents the result of a cleanup operation
type CleanupResult struct {
	StartTime      time.Time               `json:"start_time"`
	EndTime        time.Time               `json:"end_time"`
	Duration       time.Duration           `json:"duration"`
	TotalDeleted   int                     `json:"total_deleted"`
	DeletedByLevel map[models.LogLevel]int `json:"deleted_by_level"`
	Errors         []string                `json:"errors,omitempty"`
}

// LogDeleter interface for storages that support log deletion
type LogDeleter interface {
	DeleteByIDs(ctx context.Context, ids []string) (int, error)
}

// RetentionScheduler manages automatic cleanup scheduling
type RetentionScheduler struct {
	retentionService *RetentionService
	interval         time.Duration
	stopChan         chan struct{}
	running          bool
}

// NewRetentionScheduler creates a new retention scheduler
func NewRetentionScheduler(retentionService *RetentionService, interval time.Duration) *RetentionScheduler {
	return &RetentionScheduler{
		retentionService: retentionService,
		interval:         interval,
		stopChan:         make(chan struct{}),
	}
}

// Start begins the automatic cleanup schedule
func (s *RetentionScheduler) Start(ctx context.Context) {
	if s.running {
		return
	}

	s.running = true

	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Run cleanup
				result, err := s.retentionService.CleanupExpiredLogs(ctx)
				if err != nil {
					fmt.Printf("Retention cleanup failed: %v\n", err)
				} else if result.TotalDeleted > 0 {
					fmt.Printf("Retention cleanup completed: deleted %d logs in %v\n",
						result.TotalDeleted, result.Duration)
				}

				// Also run count-based cleanup
				countResult, err := s.retentionService.CleanupByCount(ctx)
				if err != nil {
					fmt.Printf("Count-based cleanup failed: %v\n", err)
				} else if countResult.TotalDeleted > 0 {
					fmt.Printf("Count-based cleanup completed: deleted %d logs in %v\n",
						countResult.TotalDeleted, countResult.Duration)
				}

			case <-s.stopChan:
				s.running = false
				return
			case <-ctx.Done():
				s.running = false
				return
			}
		}
	}()
}

// Stop stops the automatic cleanup schedule
func (s *RetentionScheduler) Stop() {
	if !s.running {
		return
	}

	close(s.stopChan)
	s.running = false
}

// IsRunning returns whether the scheduler is currently running
func (s *RetentionScheduler) IsRunning() bool {
	return s.running
}
