package storage

import (
	"context"

	"github.com/your-org/mcp-logging-server/pkg/models"
)

// LogStorage defines the interface for log storage operations
type LogStorage interface {
	// Store stores a batch of log entries
	Store(ctx context.Context, logs []models.LogEntry) error
	
	// Query retrieves logs based on filter criteria
	Query(ctx context.Context, filter models.LogFilter) (*models.LogResult, error)
	
	// GetByIDs retrieves specific log entries by their IDs
	GetByIDs(ctx context.Context, ids []string) ([]models.LogEntry, error)
	
	// GetServices returns a list of services that have logged entries
	GetServices(ctx context.Context) ([]models.ServiceInfo, error)
	
	// HealthCheck returns the health status of the storage system
	HealthCheck(ctx context.Context) models.HealthStatus
	
	// Close closes the storage connection
	Close() error
}