package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

// ServerConfig contains server-specific configuration
type ServerConfig struct {
	IngestionPort int `yaml:"ingestion_port" validate:"required,min=1024,max=65535"`
	MCPPort       int `yaml:"mcp_port" validate:"required,min=1024,max=65535"`
}

// StorageConfig contains storage-specific configuration
type StorageConfig struct {
	Type             string `yaml:"type" validate:"required,oneof=sqlite postgres clickhouse"`
	ConnectionString string `yaml:"connection_string" validate:"required"`
	MaxConnections   int    `yaml:"max_connections" validate:"min=1,max=1000"`
}

// RetentionConfig contains log retention policies
type RetentionConfig struct {
	DefaultDays int                `yaml:"default_days" validate:"min=1,max=3650"`
	ByLevel     map[string]int     `yaml:"by_level"`
}

// IndexingConfig contains search indexing configuration
type IndexingConfig struct {
	Enabled        bool `yaml:"enabled"`
	FullTextSearch bool `yaml:"full_text_search"`
}

// BufferConfig contains message buffering configuration
type BufferConfig struct {
	Size         int           `yaml:"size" validate:"min=100,max=1000000"`
	FlushTimeout time.Duration `yaml:"flush_timeout" validate:"min=1s,max=60s"`
	MaxBatchSize int           `yaml:"max_batch_size" validate:"min=1,max=10000"`
}

// Config represents the complete application configuration
type Config struct {
	Server    ServerConfig    `yaml:"server" validate:"required"`
	Storage   StorageConfig   `yaml:"storage" validate:"required"`
	Retention RetentionConfig `yaml:"retention" validate:"required"`
	Indexing  IndexingConfig  `yaml:"indexing"`
	Buffer    BufferConfig    `yaml:"buffer" validate:"required"`
}

// Validate validates the configuration using struct tags
func (c *Config) Validate() error {
	validate := validator.New()
	
	// Custom validation for port conflicts
	if c.Server.IngestionPort == c.Server.MCPPort {
		return fmt.Errorf("ingestion_port and mcp_port cannot be the same")
	}
	
	return validate.Struct(c)
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			IngestionPort: 8080,
			MCPPort:       8081,
		},
		Storage: StorageConfig{
			Type:             "sqlite",
			ConnectionString: "./logs.db",
			MaxConnections:   10,
		},
		Retention: RetentionConfig{
			DefaultDays: 30,
			ByLevel: map[string]int{
				"DEBUG": 7,
				"INFO":  30,
				"WARN":  90,
				"ERROR": 365,
				"FATAL": 365,
			},
		},
		Indexing: IndexingConfig{
			Enabled:        true,
			FullTextSearch: true,
		},
		Buffer: BufferConfig{
			Size:         10000,
			FlushTimeout: 5 * time.Second,
			MaxBatchSize: 100,
		},
	}
}

// Load loads configuration from file or environment variables
func Load() (*Config, error) {
	config := DefaultConfig()
	
	// Try to load from config file
	configPath := os.Getenv("MCP_LOGGING_CONFIG")
	if configPath == "" {
		// Look for config file in common locations
		possiblePaths := []string{
			"./config.yaml",
			"./config.yml",
			"/etc/mcp-logging/config.yaml",
			filepath.Join(os.Getenv("HOME"), ".mcp-logging", "config.yaml"),
		}
		
		for _, path := range possiblePaths {
			if _, err := os.Stat(path); err == nil {
				configPath = path
				break
			}
		}
	}
	
	if configPath != "" {
		if err := loadFromFile(config, configPath); err != nil {
			return nil, fmt.Errorf("failed to load config from file %s: %w", configPath, err)
		}
	}
	
	// Override with environment variables
	loadFromEnv(config)
	
	// Validate the final configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}
	
	return config, nil
}

// loadFromFile loads configuration from a YAML file
func loadFromFile(config *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	
	return yaml.Unmarshal(data, config)
}

// loadFromEnv loads configuration from environment variables
func loadFromEnv(config *Config) {
	if port := os.Getenv("MCP_LOGGING_INGESTION_PORT"); port != "" {
		if p, err := parsePort(port); err == nil {
			config.Server.IngestionPort = p
		}
	}
	
	if port := os.Getenv("MCP_LOGGING_MCP_PORT"); port != "" {
		if p, err := parsePort(port); err == nil {
			config.Server.MCPPort = p
		}
	}
	
	if connStr := os.Getenv("MCP_LOGGING_DB_CONNECTION"); connStr != "" {
		config.Storage.ConnectionString = connStr
	}
	
	if dbType := os.Getenv("MCP_LOGGING_DB_TYPE"); dbType != "" {
		config.Storage.Type = dbType
	}
}

// parsePort parses a port string to int with validation
func parsePort(portStr string) (int, error) {
	var port int
	if _, err := fmt.Sscanf(portStr, "%d", &port); err != nil {
		return 0, err
	}
	if port < 1024 || port > 65535 {
		return 0, fmt.Errorf("port must be between 1024 and 65535")
	}
	return port, nil
}

// SaveToFile saves the configuration to a YAML file
func (c *Config) SaveToFile(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	
	return os.WriteFile(path, data, 0644)
}