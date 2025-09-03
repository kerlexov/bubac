package auth

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadAPIKeyConfig loads API key configuration from a YAML file
func LoadAPIKeyConfig(configPath string) (*APIKeyConfig, error) {
	// If config path is not provided or file doesn't exist, return default config
	if configPath == "" {
		return &APIKeyConfig{
			RequireAuth: false,
			APIKeys:     make(map[string]APIKeyInfo),
		}, nil
	}
	
	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config file
		defaultConfig := &APIKeyConfig{
			RequireAuth: false,
			APIKeys:     make(map[string]APIKeyInfo),
		}
		
		if err := SaveAPIKeyConfig(configPath, defaultConfig); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
		
		return defaultConfig, nil
	}
	
	// Read the file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	// Parse YAML
	var config APIKeyConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}
	
	// Initialize APIKeys map if nil
	if config.APIKeys == nil {
		config.APIKeys = make(map[string]APIKeyInfo)
	}
	
	return &config, nil
}

// SaveAPIKeyConfig saves API key configuration to a YAML file
func SaveAPIKeyConfig(configPath string, config *APIKeyConfig) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	
	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	// Write to file
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// LoadAPIKeyConfigFromEnv loads API key configuration from environment variables
func LoadAPIKeyConfigFromEnv() *APIKeyConfig {
	config := &APIKeyConfig{
		RequireAuth: os.Getenv("API_KEY_REQUIRED") == "true",
		APIKeys:     make(map[string]APIKeyInfo),
	}
	
	return config
}

// MergeConfigs merges two API key configurations, with the second taking precedence
func MergeConfigs(base, override *APIKeyConfig) *APIKeyConfig {
	if base == nil {
		return override
	}
	if override == nil {
		return base
	}
	
	merged := &APIKeyConfig{
		RequireAuth: override.RequireAuth || base.RequireAuth,
		APIKeys:     make(map[string]APIKeyInfo),
	}
	
	// Copy base keys
	for k, v := range base.APIKeys {
		merged.APIKeys[k] = v
	}
	
	// Override with new keys
	for k, v := range override.APIKeys {
		merged.APIKeys[k] = v
	}
	
	return merged
}