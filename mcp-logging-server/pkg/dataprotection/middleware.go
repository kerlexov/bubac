package dataprotection

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/kerlexov/mcp-logging-server/pkg/models"
)

// DataProtectionMiddleware creates middleware for data protection
func DataProtectionMiddleware(processor *DataProtectionProcessor) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Store processor in context for use by handlers
		c.Set("data_protection_processor", processor)
		c.Next()
	}
}

// ProcessLogEntries processes a slice of log entries for data protection
func ProcessLogEntries(processor *DataProtectionProcessor, entries []models.LogEntry) error {
	if processor == nil || !processor.GetConfig().Enabled {
		return nil
	}

	for i := range entries {
		if err := processor.ProcessLogEntry(&entries[i]); err != nil {
			return err
		}
	}

	return nil
}

// GetProcessorFromContext retrieves the data protection processor from Gin context
func GetProcessorFromContext(c *gin.Context) *DataProtectionProcessor {
	if processor, exists := c.Get("data_protection_processor"); exists {
		if dp, ok := processor.(*DataProtectionProcessor); ok {
			return dp
		}
	}
	return nil
}

// AdminDataProtectionMiddleware creates middleware for data protection admin endpoints
func AdminDataProtectionMiddleware(processor *DataProtectionProcessor, statsCollector *AuditStatsCollector) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.URL.Path {
		case "/admin/data-protection/config":
			if c.Request.Method == "GET" {
				handleGetDataProtectionConfig(c, processor)
				return
			} else if c.Request.Method == "PUT" {
				handleUpdateDataProtectionConfig(c, processor)
				return
			}
		case "/admin/data-protection/stats":
			if c.Request.Method == "GET" {
				handleGetDataProtectionStats(c, statsCollector)
				return
			}
		case "/admin/data-protection/test":
			if c.Request.Method == "POST" {
				handleTestDataProtection(c, processor)
				return
			}
		}

		c.Next()
	}
}

// handleGetDataProtectionConfig returns current data protection configuration
func handleGetDataProtectionConfig(c *gin.Context, processor *DataProtectionProcessor) {
	config := processor.GetConfig()
	c.JSON(http.StatusOK, gin.H{
		"config": config,
	})
}

// handleUpdateDataProtectionConfig updates data protection configuration
func handleUpdateDataProtectionConfig(c *gin.Context, processor *DataProtectionProcessor) {
	var newConfig DataProtectionConfig
	if err := c.ShouldBindJSON(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid configuration",
			"details": err.Error(),
		})
		return
	}

	if err := processor.UpdateConfig(&newConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Failed to update configuration",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Configuration updated successfully",
		"config":  newConfig,
	})
}

// handleGetDataProtectionStats returns data protection statistics
func handleGetDataProtectionStats(c *gin.Context, statsCollector *AuditStatsCollector) {
	if statsCollector == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Statistics collection not enabled",
		})
		return
	}

	stats := statsCollector.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

// handleTestDataProtection tests data protection on sample data
func handleTestDataProtection(c *gin.Context, processor *DataProtectionProcessor) {
	var request struct {
		LogEntry models.LogEntry `json:"log_entry" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	// Create a copy for testing
	originalEntry := request.LogEntry
	testEntry := request.LogEntry

	// Process the test entry
	if err := processor.ProcessLogEntry(&testEntry); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to process log entry",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"original":  originalEntry,
		"processed": testEntry,
		"config":    processor.GetConfig(),
	})
}
