package tls

import (
	"crypto/tls"
	"fmt"
	"os"
)

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled    bool   `yaml:"enabled" json:"enabled"`
	CertFile   string `yaml:"cert_file" json:"cert_file"`
	KeyFile    string `yaml:"key_file" json:"key_file"`
	MinVersion string `yaml:"min_version" json:"min_version"`
	CipherSuites []string `yaml:"cipher_suites" json:"cipher_suites"`
}

// DefaultTLSConfig returns default TLS configuration
func DefaultTLSConfig() *TLSConfig {
	return &TLSConfig{
		Enabled:    false, // Disabled by default, usually handled by reverse proxy
		CertFile:   "/app/certs/server.crt",
		KeyFile:    "/app/certs/server.key",
		MinVersion: "TLS1.2",
		CipherSuites: []string{
			"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
			"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
			"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305",
			"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
		},
	}
}

// LoadTLSConfigFromEnv loads TLS configuration from environment variables
func LoadTLSConfigFromEnv() *TLSConfig {
	config := DefaultTLSConfig()
	
	if os.Getenv("TLS_ENABLED") == "true" {
		config.Enabled = true
	}
	
	if certFile := os.Getenv("TLS_CERT_PATH"); certFile != "" {
		config.CertFile = certFile
	}
	
	if keyFile := os.Getenv("TLS_KEY_PATH"); keyFile != "" {
		config.KeyFile = keyFile
	}
	
	if minVersion := os.Getenv("TLS_MIN_VERSION"); minVersion != "" {
		config.MinVersion = minVersion
	}
	
	return config
}

// GetTLSConfig converts the configuration to Go's tls.Config
func (c *TLSConfig) GetTLSConfig() (*tls.Config, error) {
	if !c.Enabled {
		return nil, nil
	}
	
	// Validate certificate files exist
	if _, err := os.Stat(c.CertFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("certificate file not found: %s", c.CertFile)
	}
	
	if _, err := os.Stat(c.KeyFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("key file not found: %s", c.KeyFile)
	}
	
	// Load certificate
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}
	
	// Parse minimum TLS version
	minVersion, err := c.parseMinVersion()
	if err != nil {
		return nil, fmt.Errorf("invalid min TLS version: %w", err)
	}
	
	// Parse cipher suites
	cipherSuites, err := c.parseCipherSuites()
	if err != nil {
		return nil, fmt.Errorf("invalid cipher suites: %w", err)
	}
	
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   minVersion,
		CipherSuites: cipherSuites,
		// Security best practices
		PreferServerCipherSuites: true,
		CurvePreferences: []tls.CurveID{
			tls.CurveP256,
			tls.CurveP384,
			tls.CurveP521,
			tls.X25519,
		},
	}
	
	return tlsConfig, nil
}

// parseMinVersion converts string to TLS version constant
func (c *TLSConfig) parseMinVersion() (uint16, error) {
	switch c.MinVersion {
	case "TLS1.0":
		return tls.VersionTLS10, nil
	case "TLS1.1":
		return tls.VersionTLS11, nil
	case "TLS1.2":
		return tls.VersionTLS12, nil
	case "TLS1.3":
		return tls.VersionTLS13, nil
	default:
		return tls.VersionTLS12, fmt.Errorf("unsupported TLS version: %s", c.MinVersion)
	}
}

// parseCipherSuites converts string names to cipher suite constants
func (c *TLSConfig) parseCipherSuites() ([]uint16, error) {
	cipherMap := map[string]uint16{
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":     tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305":      tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":     tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384":   tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305":    tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256":   tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"TLS_RSA_WITH_AES_256_GCM_SHA384":           tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_RSA_WITH_AES_128_GCM_SHA256":           tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
	}
	
	var suites []uint16
	for _, suiteName := range c.CipherSuites {
		if suite, exists := cipherMap[suiteName]; exists {
			suites = append(suites, suite)
		} else {
			return nil, fmt.Errorf("unsupported cipher suite: %s", suiteName)
		}
	}
	
	return suites, nil
}

// ValidateConfig validates the TLS configuration
func (c *TLSConfig) ValidateConfig() error {
	if !c.Enabled {
		return nil
	}
	
	if c.CertFile == "" {
		return fmt.Errorf("certificate file path is required when TLS is enabled")
	}
	
	if c.KeyFile == "" {
		return fmt.Errorf("key file path is required when TLS is enabled")
	}
	
	// Validate min version
	if _, err := c.parseMinVersion(); err != nil {
		return err
	}
	
	// Validate cipher suites
	if _, err := c.parseCipherSuites(); err != nil {
		return err
	}
	
	return nil
}