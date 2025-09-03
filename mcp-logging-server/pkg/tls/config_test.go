package tls

import (
	"crypto/tls"
	"os"
	"testing"
)

func TestDefaultTLSConfig(t *testing.T) {
	config := DefaultTLSConfig()
	
	if config.Enabled {
		t.Error("TLS should be disabled by default")
	}
	
	if config.MinVersion != "TLS1.2" {
		t.Errorf("Expected min version TLS1.2, got %s", config.MinVersion)
	}
	
	if len(config.CipherSuites) == 0 {
		t.Error("Expected cipher suites to be configured")
	}
}

func TestLoadTLSConfigFromEnv(t *testing.T) {
	// Set environment variables
	os.Setenv("TLS_ENABLED", "true")
	os.Setenv("TLS_CERT_PATH", "/custom/cert.pem")
	os.Setenv("TLS_KEY_PATH", "/custom/key.pem")
	os.Setenv("TLS_MIN_VERSION", "TLS1.3")
	
	defer func() {
		os.Unsetenv("TLS_ENABLED")
		os.Unsetenv("TLS_CERT_PATH")
		os.Unsetenv("TLS_KEY_PATH")
		os.Unsetenv("TLS_MIN_VERSION")
	}()
	
	config := LoadTLSConfigFromEnv()
	
	if !config.Enabled {
		t.Error("TLS should be enabled")
	}
	
	if config.CertFile != "/custom/cert.pem" {
		t.Errorf("Expected cert file /custom/cert.pem, got %s", config.CertFile)
	}
	
	if config.KeyFile != "/custom/key.pem" {
		t.Errorf("Expected key file /custom/key.pem, got %s", config.KeyFile)
	}
	
	if config.MinVersion != "TLS1.3" {
		t.Errorf("Expected min version TLS1.3, got %s", config.MinVersion)
	}
}

func TestParseMinVersion(t *testing.T) {
	config := &TLSConfig{}
	
	testCases := []struct {
		version  string
		expected uint16
		hasError bool
	}{
		{"TLS1.0", tls.VersionTLS10, false},
		{"TLS1.1", tls.VersionTLS11, false},
		{"TLS1.2", tls.VersionTLS12, false},
		{"TLS1.3", tls.VersionTLS13, false},
		{"invalid", tls.VersionTLS12, true},
	}
	
	for _, tc := range testCases {
		config.MinVersion = tc.version
		version, err := config.parseMinVersion()
		
		if tc.hasError {
			if err == nil {
				t.Errorf("Expected error for version %s", tc.version)
			}
		} else {
			if err != nil {
				t.Errorf("Unexpected error for version %s: %v", tc.version, err)
			}
			if version != tc.expected {
				t.Errorf("Expected version %d for %s, got %d", tc.expected, tc.version, version)
			}
		}
	}
}

func TestParseCipherSuites(t *testing.T) {
	config := &TLSConfig{
		CipherSuites: []string{
			"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305",
		},
	}
	
	suites, err := config.parseCipherSuites()
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(suites) != 2 {
		t.Errorf("Expected 2 cipher suites, got %d", len(suites))
	}
	
	expectedSuites := []uint16{
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
	}
	
	for i, expected := range expectedSuites {
		if suites[i] != expected {
			t.Errorf("Expected cipher suite %d, got %d", expected, suites[i])
		}
	}
}

func TestParseCipherSuites_Invalid(t *testing.T) {
	config := &TLSConfig{
		CipherSuites: []string{
			"INVALID_CIPHER_SUITE",
		},
	}
	
	_, err := config.parseCipherSuites()
	if err == nil {
		t.Error("Expected error for invalid cipher suite")
	}
}

func TestValidateConfig(t *testing.T) {
	// Test disabled config (should be valid)
	config := &TLSConfig{Enabled: false}
	if err := config.ValidateConfig(); err != nil {
		t.Errorf("Disabled config should be valid: %v", err)
	}
	
	// Test enabled config without cert file
	config = &TLSConfig{
		Enabled:  true,
		CertFile: "",
		KeyFile:  "/path/to/key",
	}
	if err := config.ValidateConfig(); err == nil {
		t.Error("Expected error for missing cert file")
	}
	
	// Test enabled config without key file
	config = &TLSConfig{
		Enabled:  true,
		CertFile: "/path/to/cert",
		KeyFile:  "",
	}
	if err := config.ValidateConfig(); err == nil {
		t.Error("Expected error for missing key file")
	}
	
	// Test enabled config with invalid min version
	config = &TLSConfig{
		Enabled:    true,
		CertFile:   "/path/to/cert",
		KeyFile:    "/path/to/key",
		MinVersion: "invalid",
	}
	if err := config.ValidateConfig(); err == nil {
		t.Error("Expected error for invalid min version")
	}
}