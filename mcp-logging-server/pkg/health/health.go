package health

import (
	"time"

	"github.com/kerlexov/mcp-logging-server/pkg/models"
)

// CheckSystem performs a system health check
func CheckSystem() models.HealthStatus {
	// TODO: Implement comprehensive health checks
	// This will be expanded in later tasks
	return models.HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Details: map[string]string{
			"version": "1.0.0",
		},
	}
}
