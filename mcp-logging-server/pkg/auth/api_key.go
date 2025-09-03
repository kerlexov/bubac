package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

// Permission represents a permission that can be granted to an API key
type Permission string

const (
	PermissionIngestLogs Permission = "ingest_logs"
	PermissionQueryLogs  Permission = "query_logs"
	PermissionAdmin      Permission = "admin"
	PermissionMetrics    Permission = "metrics"
)

// APIKeyInfo contains information about an API key
type APIKeyInfo struct {
	Name        string       `yaml:"name" json:"name"`
	Permissions []Permission `yaml:"permissions" json:"permissions"`
	RateLimit   int          `yaml:"rate_limit" json:"rate_limit"`
	ExpiresAt   *time.Time   `yaml:"expires_at,omitempty" json:"expires_at,omitempty"`
	CreatedAt   time.Time    `yaml:"created_at" json:"created_at"`
	LastUsed    *time.Time   `yaml:"last_used,omitempty" json:"last_used,omitempty"`
	IsActive    bool         `yaml:"is_active" json:"is_active"`
}

// APIKeyConfig represents the configuration for API key authentication
type APIKeyConfig struct {
	RequireAuth bool                  `yaml:"require_auth" json:"require_auth"`
	APIKeys     map[string]APIKeyInfo `yaml:"api_keys" json:"api_keys"`
}

// APIKeyManager manages API keys and their validation
type APIKeyManager struct {
	config *APIKeyConfig
}

// NewAPIKeyManager creates a new API key manager
func NewAPIKeyManager(config *APIKeyConfig) *APIKeyManager {
	if config == nil {
		config = &APIKeyConfig{
			RequireAuth: false,
			APIKeys:     make(map[string]APIKeyInfo),
		}
	}
	return &APIKeyManager{
		config: config,
	}
}

// GenerateAPIKey generates a new secure API key
func (m *APIKeyManager) GenerateAPIKey() (string, error) {
	// Generate 32 random bytes
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	
	// Convert to hex string with prefix
	apiKey := "mcp_" + hex.EncodeToString(bytes)
	return apiKey, nil
}

// HashAPIKey creates a SHA-256 hash of the API key for secure storage
func (m *APIKeyManager) HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return hex.EncodeToString(hash[:])
}

// ValidateAPIKey validates an API key and returns its information
func (m *APIKeyManager) ValidateAPIKey(apiKey string) (*APIKeyInfo, bool) {
	if !m.config.RequireAuth {
		// If auth is not required, return a default key info with all permissions
		return &APIKeyInfo{
			Name:        "no-auth",
			Permissions: []Permission{PermissionIngestLogs, PermissionQueryLogs, PermissionMetrics},
			RateLimit:   1000,
			IsActive:    true,
		}, true
	}
	
	// Hash the provided API key to compare with stored hashes
	hashedKey := m.HashAPIKey(apiKey)
	
	keyInfo, exists := m.config.APIKeys[hashedKey]
	if !exists {
		return nil, false
	}
	
	// Check if key is active
	if !keyInfo.IsActive {
		return nil, false
	}
	
	// Check if key has expired
	if keyInfo.ExpiresAt != nil && keyInfo.ExpiresAt.Before(time.Now()) {
		return nil, false
	}
	
	return &keyInfo, true
}

// HasPermission checks if an API key has a specific permission
func (m *APIKeyManager) HasPermission(keyInfo *APIKeyInfo, permission Permission) bool {
	if keyInfo == nil {
		return false
	}
	
	// Admin permission grants all permissions
	for _, p := range keyInfo.Permissions {
		if p == PermissionAdmin || p == permission {
			return true
		}
	}
	
	return false
}

// UpdateLastUsed updates the last used timestamp for an API key
func (m *APIKeyManager) UpdateLastUsed(apiKey string) {
	if !m.config.RequireAuth {
		return
	}
	
	hashedKey := m.HashAPIKey(apiKey)
	if keyInfo, exists := m.config.APIKeys[hashedKey]; exists {
		now := time.Now()
		keyInfo.LastUsed = &now
		m.config.APIKeys[hashedKey] = keyInfo
	}
}

// CreateAPIKey creates a new API key with the specified configuration
func (m *APIKeyManager) CreateAPIKey(name string, permissions []Permission, rateLimit int, expiresAt *time.Time) (string, error) {
	apiKey, err := m.GenerateAPIKey()
	if err != nil {
		return "", err
	}
	
	hashedKey := m.HashAPIKey(apiKey)
	
	keyInfo := APIKeyInfo{
		Name:        name,
		Permissions: permissions,
		RateLimit:   rateLimit,
		ExpiresAt:   expiresAt,
		CreatedAt:   time.Now(),
		IsActive:    true,
	}
	
	m.config.APIKeys[hashedKey] = keyInfo
	
	return apiKey, nil
}

// RevokeAPIKey revokes an API key by setting it as inactive
func (m *APIKeyManager) RevokeAPIKey(apiKey string) bool {
	hashedKey := m.HashAPIKey(apiKey)
	if keyInfo, exists := m.config.APIKeys[hashedKey]; exists {
		keyInfo.IsActive = false
		m.config.APIKeys[hashedKey] = keyInfo
		return true
	}
	return false
}

// ListAPIKeys returns a list of all API keys (without the actual key values)
func (m *APIKeyManager) ListAPIKeys() []APIKeyInfo {
	keys := make([]APIKeyInfo, 0, len(m.config.APIKeys))
	for _, keyInfo := range m.config.APIKeys {
		keys = append(keys, keyInfo)
	}
	return keys
}

// GetConfig returns the current API key configuration
func (m *APIKeyManager) GetConfig() *APIKeyConfig {
	return m.config
}

// SetConfig updates the API key configuration
func (m *APIKeyManager) SetConfig(config *APIKeyConfig) {
	m.config = config
}