package validation

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/your-org/mcp-logging-server/pkg/models"
)

// LogValidator provides comprehensive validation for log entries
type LogValidator struct {
	validator *validator.Validate
}

// NewLogValidator creates a new log validator
func NewLogValidator() *LogValidator {
	v := validator.New()
	
	// Register custom validators
	v.RegisterValidation("service_name", validateServiceName)
	v.RegisterValidation("agent_id", validateAgentID)
	v.RegisterValidation("log_message", validateLogMessage)
	v.RegisterValidation("metadata_size", validateMetadataSize)
	
	return &LogValidator{
		validator: v,
	}
}

// ValidateLogEntry validates a single log entry with detailed error reporting
func (lv *LogValidator) ValidateLogEntry(entry *models.LogEntry) *ValidationResult {
	result := &ValidationResult{
		IsValid: true,
		Errors:  make([]ValidationError, 0),
	}
	
	// Basic struct validation
	if err := lv.validator.Struct(entry); err != nil {
		if validationErrors, ok := err.(validator.ValidationErrors); ok {
			for _, fieldError := range validationErrors {
				result.Errors = append(result.Errors, ValidationError{
					Field:   fieldError.Field(),
					Value:   fmt.Sprintf("%v", fieldError.Value()),
					Message: getValidationMessage(fieldError),
				})
			}
		}
	}
	
	// Custom business logic validation
	lv.validateBusinessRules(entry, result)
	
	result.IsValid = len(result.Errors) == 0
	return result
}

// ValidateLogBatch validates a batch of log entries
func (lv *LogValidator) ValidateLogBatch(entries []models.LogEntry) *BatchValidationResult {
	result := &BatchValidationResult{
		TotalEntries:   len(entries),
		ValidEntries:   make([]models.LogEntry, 0),
		InvalidEntries: make([]InvalidEntry, 0),
	}
	
	for i, entry := range entries {
		validationResult := lv.ValidateLogEntry(&entry)
		if validationResult.IsValid {
			result.ValidEntries = append(result.ValidEntries, entry)
		} else {
			result.InvalidEntries = append(result.InvalidEntries, InvalidEntry{
				Index:  i,
				Entry:  entry,
				Errors: validationResult.Errors,
			})
		}
	}
	
	result.ValidCount = len(result.ValidEntries)
	result.InvalidCount = len(result.InvalidEntries)
	
	return result
}

// ValidationResult represents the result of validating a single log entry
type ValidationResult struct {
	IsValid bool              `json:"is_valid"`
	Errors  []ValidationError `json:"errors,omitempty"`
}

// ValidationError represents a single validation error
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// BatchValidationResult represents the result of validating a batch of log entries
type BatchValidationResult struct {
	TotalEntries   int            `json:"total_entries"`
	ValidCount     int            `json:"valid_count"`
	InvalidCount   int            `json:"invalid_count"`
	ValidEntries   []models.LogEntry `json:"valid_entries"`
	InvalidEntries []InvalidEntry `json:"invalid_entries"`
}

// InvalidEntry represents an invalid log entry with its errors
type InvalidEntry struct {
	Index  int               `json:"index"`
	Entry  models.LogEntry   `json:"entry"`
	Errors []ValidationError `json:"errors"`
}

// validateBusinessRules applies custom business logic validation
func (lv *LogValidator) validateBusinessRules(entry *models.LogEntry, result *ValidationResult) {
	// Validate timestamp is not too far in the future
	if entry.Timestamp.After(time.Now().Add(5 * time.Minute)) {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "timestamp",
			Value:   entry.Timestamp.String(),
			Message: "Timestamp cannot be more than 5 minutes in the future",
		})
	}
	
	// Validate timestamp is not too old (more than 1 year)
	if entry.Timestamp.Before(time.Now().Add(-365 * 24 * time.Hour)) {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "timestamp",
			Value:   entry.Timestamp.String(),
			Message: "Timestamp cannot be more than 1 year in the past",
		})
	}
	
	// Validate metadata size
	if entry.Metadata != nil && len(entry.Metadata) > 50 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "metadata",
			Value:   fmt.Sprintf("%d keys", len(entry.Metadata)),
			Message: "Metadata cannot have more than 50 keys",
		})
	}
	
	// Validate stack trace size
	if len(entry.StackTrace) > 50000 {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "stack_trace",
			Value:   fmt.Sprintf("%d characters", len(entry.StackTrace)),
			Message: "Stack trace cannot exceed 50,000 characters",
		})
	}
}

// Custom validator functions
func validateServiceName(fl validator.FieldLevel) bool {
	serviceName := fl.Field().String()
	// Service name should contain only alphanumeric characters, hyphens, and underscores
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, serviceName)
	return matched
}

func validateAgentID(fl validator.FieldLevel) bool {
	agentID := fl.Field().String()
	// Agent ID should contain only alphanumeric characters, hyphens, and underscores
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, agentID)
	return matched
}

func validateLogMessage(fl validator.FieldLevel) bool {
	message := fl.Field().String()
	// Message should not be empty after trimming whitespace
	return len(strings.TrimSpace(message)) > 0
}

func validateMetadataSize(fl validator.FieldLevel) bool {
	// This is handled in business rules validation
	return true
}

// getValidationMessage returns a human-readable validation error message
func getValidationMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Field())
	case "uuid4":
		return fmt.Sprintf("%s must be a valid UUID v4", fe.Field())
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", fe.Field(), fe.Param())
	case "max":
		return fmt.Sprintf("%s cannot exceed %s characters", fe.Field(), fe.Param())
	case "min":
		return fmt.Sprintf("%s must be at least %s characters", fe.Field(), fe.Param())
	case "service_name":
		return fmt.Sprintf("%s can only contain alphanumeric characters, hyphens, and underscores", fe.Field())
	case "agent_id":
		return fmt.Sprintf("%s can only contain alphanumeric characters, hyphens, and underscores", fe.Field())
	case "log_message":
		return fmt.Sprintf("%s cannot be empty", fe.Field())
	default:
		return fmt.Sprintf("%s is invalid", fe.Field())
	}
}