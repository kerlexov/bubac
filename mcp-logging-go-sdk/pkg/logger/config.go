package logger

import (
	"errors"
	"time"
)

type Config struct {
	ServerURL           string        `json:"server_url" yaml:"server_url"`
	ServiceName         string        `json:"service_name" yaml:"service_name"`
	AgentID             string        `json:"agent_id" yaml:"agent_id"`
	BufferSize          int           `json:"buffer_size" yaml:"buffer_size"`
	FlushInterval       time.Duration `json:"flush_interval" yaml:"flush_interval"`
	RetryConfig         RetryConfig   `json:"retry_config" yaml:"retry_config"`
	HTTPTimeout         time.Duration `json:"http_timeout" yaml:"http_timeout"`
	EnableHealthCheck   bool          `json:"enable_health_check" yaml:"enable_health_check"`
	HealthCheckInterval time.Duration `json:"health_check_interval" yaml:"health_check_interval"`
	MaxRetries          int           `json:"max_retries" yaml:"max_retries"`
}

type RetryConfig struct {
	InitialInterval     time.Duration `json:"initial_interval" yaml:"initial_interval"`
	MaxInterval         time.Duration `json:"max_interval" yaml:"max_interval"`
	MaxElapsedTime      time.Duration `json:"max_elapsed_time" yaml:"max_elapsed_time"`
	Multiplier          float64       `json:"multiplier" yaml:"multiplier"`
	RandomizationFactor float64       `json:"randomization_factor" yaml:"randomization_factor"`
}

func DefaultConfig() Config {
	return Config{
		ServerURL:           "http://localhost:8080",
		BufferSize:          1000,
		FlushInterval:       5 * time.Second,
		HTTPTimeout:         10 * time.Second,
		EnableHealthCheck:   true,
		HealthCheckInterval: 30 * time.Second,
		MaxRetries:          3,
		RetryConfig: RetryConfig{
			InitialInterval:     1 * time.Second,
			MaxInterval:         30 * time.Second,
			MaxElapsedTime:      5 * time.Minute,
			Multiplier:          2.0,
			RandomizationFactor: 0.1,
		},
	}
}

func (c *Config) Validate() error {
	if c.ServerURL == "" {
		return errors.New("server_url is required")
	}
	if c.ServiceName == "" {
		return errors.New("service_name is required")
	}
	if c.AgentID == "" {
		return errors.New("agent_id is required")
	}
	if c.BufferSize <= 0 {
		c.BufferSize = 1000
	}
	if c.FlushInterval <= 0 {
		c.FlushInterval = 5 * time.Second
	}
	if c.HTTPTimeout <= 0 {
		c.HTTPTimeout = 10 * time.Second
	}
	if c.RetryConfig.InitialInterval <= 0 {
		c.RetryConfig.InitialInterval = 1 * time.Second
	}
	if c.RetryConfig.MaxInterval <= 0 {
		c.RetryConfig.MaxInterval = 30 * time.Second
	}
	if c.RetryConfig.Multiplier <= 1 {
		c.RetryConfig.Multiplier = 2.0
	}
	return nil
}
