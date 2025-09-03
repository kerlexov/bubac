package dataprotection

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/your-org/mcp-logging-server/pkg/models"
)

// ActionType represents the type of data protection action
type ActionType string

const (
	ActionMask ActionType = "mask"
	ActionHash ActionType = "hash"
	ActionDrop ActionType = "drop"
)

// FieldRule represents a rule for protecting a specific field
type FieldRule struct {
	Field   string     `yaml:"field" json:"field"`
	Action  ActionType `yaml:"action" json:"action"`
	Pattern string     `yaml:"pattern,omitempty" json:"pattern,omitempty"` // Regex pattern for partial matching
}

// DataProtectionConfig represents data protection configuration
type DataProtectionConfig struct {
	Enabled      bool        `yaml:"enabled" json:"enabled"`
	FieldRules   []FieldRule `yaml:"field_rules" json:"field_rules"`
	MaskFields   []string    `yaml:"mask_fields" json:"mask_fields"`     // Deprecated: use FieldRules
	HashFields   []string    `yaml:"hash_fields" json:"hash_fields"`     // Deprecated: use FieldRules
	DropFields   []string    `yaml:"drop_fields" json:"drop_fields"`     // Deprecated: use FieldRules
	MaskChar     string      `yaml:"mask_char" json:"mask_char"`
	HashSalt     string      `yaml:"hash_salt" json:"hash_salt"`
	AuditEnabled bool        `yaml:"audit_enabled" json:"audit_enabled"`
}

// DefaultDataProtectionConfig returns default data protection configuration
func DefaultDataProtectionConfig() *DataProtectionConfig {
	return &DataProtectionConfig{
		Enabled:      true,
		MaskChar:     "*",
		HashSalt:     "mcp-logging-default-salt", // Should be changed in production
		AuditEnabled: true,
		FieldRules: []FieldRule{
			{Field: "password", Action: ActionMask},
			{Field: "token", Action: ActionMask},
			{Field: "secret", Action: ActionMask},
			{Field: "key", Action: ActionMask},
			{Field: "authorization", Action: ActionMask},
			{Field: "credit_card", Action: ActionHash},
			{Field: "ssn", Action: ActionHash},
			{Field: "email", Action: ActionMask, Pattern: `([^@]+)@(.+)`}, // Mask username part
		},
		// Backward compatibility
		MaskFields: []string{"password", "token", "secret", "key", "authorization"},
		HashFields: []string{"credit_card", "ssn"},
		DropFields: []string{},
	}
}

// DataProtectionProcessor handles data protection operations
type DataProtectionProcessor struct {
	config      *DataProtectionConfig
	auditLogger *AuditLogger
	patterns    map[string]*regexp.Regexp
}

// NewDataProtectionProcessor creates a new data protection processor
func NewDataProtectionProcessor(config *DataProtectionConfig) (*DataProtectionProcessor, error) {
	if config == nil {
		config = DefaultDataProtectionConfig()
	}
	
	processor := &DataProtectionProcessor{
		config:   config,
		patterns: make(map[string]*regexp.Regexp),
	}
	
	// Compile regex patterns
	for _, rule := range config.FieldRules {
		if rule.Pattern != "" {
			pattern, err := regexp.Compile(rule.Pattern)
			if err != nil {
				return nil, fmt.Errorf("invalid regex pattern for field %s: %w", rule.Field, err)
			}
			processor.patterns[rule.Field] = pattern
		}
	}
	
	// Initialize audit logger if enabled
	if config.AuditEnabled {
		processor.auditLogger = NewAuditLogger()
	}
	
	return processor, nil
}

// ProcessLogEntry processes a log entry according to data protection rules
func (p *DataProtectionProcessor) ProcessLogEntry(entry *models.LogEntry) error {
	if !p.config.Enabled {
		return nil
	}
	
	originalEntry := *entry // Copy for audit
	actionsPerformed := make([]AuditAction, 0)
	
	// Process metadata fields
	if entry.Metadata != nil {
		for field, value := range entry.Metadata {
			action := p.getActionForField(field)
			if action == "" {
				continue
			}
			
			originalValue := fmt.Sprintf("%v", value)
			newValue, err := p.applyAction(field, originalValue, action)
			if err != nil {
				return fmt.Errorf("failed to apply action %s to field %s: %w", action, field, err)
			}
			
			if action == ActionDrop {
				delete(entry.Metadata, field)
			} else {
				entry.Metadata[field] = newValue
			}
			
			// Record audit action
			if p.auditLogger != nil {
				actionsPerformed = append(actionsPerformed, AuditAction{
					Field:         field,
					Action:        action,
					OriginalValue: originalValue,
					NewValue:      fmt.Sprintf("%v", newValue),
				})
			}
		}
	}
	
	// Process message field for sensitive patterns
	if entry.Message != "" {
		processedMessage, messageActions := p.processMessageContent(entry.Message)
		if processedMessage != entry.Message {
			entry.Message = processedMessage
			actionsPerformed = append(actionsPerformed, messageActions...)
		}
	}
	
	// Log audit information
	if p.auditLogger != nil && len(actionsPerformed) > 0 {
		auditEntry := AuditEntry{
			Timestamp:        time.Now(),
			LogEntryID:       entry.ID,
			ServiceName:      entry.ServiceName,
			AgentID:          entry.AgentID,
			ActionsPerformed: actionsPerformed,
		}
		p.auditLogger.LogAuditEntry(auditEntry)
	}
	
	return nil
}

// getActionForField determines the action to take for a specific field
func (p *DataProtectionProcessor) getActionForField(field string) ActionType {
	fieldLower := strings.ToLower(field)
	
	// Check field rules first
	for _, rule := range p.config.FieldRules {
		if strings.ToLower(rule.Field) == fieldLower {
			return rule.Action
		}
	}
	
	// Check backward compatibility fields
	for _, maskField := range p.config.MaskFields {
		if strings.ToLower(maskField) == fieldLower {
			return ActionMask
		}
	}
	
	for _, hashField := range p.config.HashFields {
		if strings.ToLower(hashField) == fieldLower {
			return ActionHash
		}
	}
	
	for _, dropField := range p.config.DropFields {
		if strings.ToLower(dropField) == fieldLower {
			return ActionDrop
		}
	}
	
	return ""
}

// applyAction applies the specified action to a field value
func (p *DataProtectionProcessor) applyAction(field, value string, action ActionType) (interface{}, error) {
	switch action {
	case ActionMask:
		return p.maskValue(field, value), nil
	case ActionHash:
		return p.hashValue(value), nil
	case ActionDrop:
		return nil, nil
	default:
		return value, nil
	}
}

// maskValue masks a value according to the field's pattern or default masking
func (p *DataProtectionProcessor) maskValue(field, value string) string {
	if pattern, exists := p.patterns[field]; exists {
		// Use regex pattern for partial masking
		return pattern.ReplaceAllStringFunc(value, func(match string) string {
			groups := pattern.FindStringSubmatch(match)
			if len(groups) > 1 {
				// Mask the first capture group, keep the rest
				masked := strings.Repeat(p.config.MaskChar, len(groups[1]))
				result := strings.Replace(match, groups[1], masked, 1)
				return result
			}
			return p.maskString(match)
		})
	}
	
	return p.maskString(value)
}

// maskString masks a string with the configured mask character
func (p *DataProtectionProcessor) maskString(value string) string {
	if len(value) <= 4 {
		return strings.Repeat(p.config.MaskChar, len(value))
	}
	
	// Show first and last 2 characters, mask the middle
	prefix := value[:2]
	suffix := value[len(value)-2:]
	middle := strings.Repeat(p.config.MaskChar, len(value)-4)
	
	return prefix + middle + suffix
}

// hashValue creates a SHA-256 hash of the value with salt
func (p *DataProtectionProcessor) hashValue(value string) string {
	saltedValue := value + p.config.HashSalt
	hash := sha256.Sum256([]byte(saltedValue))
	return "sha256:" + hex.EncodeToString(hash[:])
}

// processMessageContent processes the message content for sensitive patterns
func (p *DataProtectionProcessor) processMessageContent(message string) (string, []AuditAction) {
	actions := make([]AuditAction, 0)
	processedMessage := message
	
	// Common sensitive patterns
	patterns := map[string]*regexp.Regexp{
		"credit_card": regexp.MustCompile(`\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`),
		"ssn":         regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
		"email":       regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
		"phone":       regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`),
		"ip_address":  regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b`),
	}
	
	for patternName, pattern := range patterns {
		matches := pattern.FindAllString(processedMessage, -1)
		for _, match := range matches {
			masked := p.maskString(match)
			processedMessage = strings.Replace(processedMessage, match, masked, -1)
			
			actions = append(actions, AuditAction{
				Field:         "message:" + patternName,
				Action:        ActionMask,
				OriginalValue: match,
				NewValue:      masked,
			})
		}
	}
	
	return processedMessage, actions
}

// GetConfig returns the current configuration
func (p *DataProtectionProcessor) GetConfig() *DataProtectionConfig {
	return p.config
}

// UpdateConfig updates the processor configuration
func (p *DataProtectionProcessor) UpdateConfig(config *DataProtectionConfig) error {
	// Recompile patterns
	patterns := make(map[string]*regexp.Regexp)
	for _, rule := range config.FieldRules {
		if rule.Pattern != "" {
			pattern, err := regexp.Compile(rule.Pattern)
			if err != nil {
				return fmt.Errorf("invalid regex pattern for field %s: %w", rule.Field, err)
			}
			patterns[rule.Field] = pattern
		}
	}
	
	p.config = config
	p.patterns = patterns
	
	// Update audit logger
	if config.AuditEnabled && p.auditLogger == nil {
		p.auditLogger = NewAuditLogger()
	} else if !config.AuditEnabled {
		p.auditLogger = nil
	}
	
	return nil
}