package auth

import (
	"testing"
	"time"
)

func TestAPIKeyManager_GenerateAPIKey(t *testing.T) {
	manager := NewAPIKeyManager(nil)
	
	key, err := manager.GenerateAPIKey()
	if err != nil {
		t.Fatalf("Failed to generate API key: %v", err)
	}
	
	if len(key) == 0 {
		t.Error("Generated API key is empty")
	}
	
	if key[:4] != "mcp_" {
		t.Error("Generated API key doesn't have correct prefix")
	}
	
	// Generate another key to ensure uniqueness
	key2, err := manager.GenerateAPIKey()
	if err != nil {
		t.Fatalf("Failed to generate second API key: %v", err)
	}
	
	if key == key2 {
		t.Error("Generated API keys are not unique")
	}
}

func TestAPIKeyManager_ValidateAPIKey(t *testing.T) {
	config := &APIKeyConfig{
		RequireAuth: true,
		APIKeys:     make(map[string]APIKeyInfo),
	}
	
	manager := NewAPIKeyManager(config)
	
	// Create a test API key
	apiKey, err := manager.CreateAPIKey("test-key", []Permission{PermissionIngestLogs}, 1000, nil)
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}
	
	// Test valid key
	keyInfo, valid := manager.ValidateAPIKey(apiKey)
	if !valid {
		t.Error("Valid API key was rejected")
	}
	
	if keyInfo.Name != "test-key" {
		t.Errorf("Expected key name 'test-key', got '%s'", keyInfo.Name)
	}
	
	// Test invalid key
	_, valid = manager.ValidateAPIKey("invalid-key")
	if valid {
		t.Error("Invalid API key was accepted")
	}
}

func TestAPIKeyManager_ValidateAPIKey_Expired(t *testing.T) {
	config := &APIKeyConfig{
		RequireAuth: true,
		APIKeys:     make(map[string]APIKeyInfo),
	}
	
	manager := NewAPIKeyManager(config)
	
	// Create an expired API key
	expiredTime := time.Now().Add(-time.Hour)
	apiKey, err := manager.CreateAPIKey("expired-key", []Permission{PermissionIngestLogs}, 1000, &expiredTime)
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}
	
	// Test expired key
	_, valid := manager.ValidateAPIKey(apiKey)
	if valid {
		t.Error("Expired API key was accepted")
	}
}

func TestAPIKeyManager_ValidateAPIKey_Inactive(t *testing.T) {
	config := &APIKeyConfig{
		RequireAuth: true,
		APIKeys:     make(map[string]APIKeyInfo),
	}
	
	manager := NewAPIKeyManager(config)
	
	// Create and then revoke an API key
	apiKey, err := manager.CreateAPIKey("inactive-key", []Permission{PermissionIngestLogs}, 1000, nil)
	if err != nil {
		t.Fatalf("Failed to create API key: %v", err)
	}
	
	// Revoke the key
	if !manager.RevokeAPIKey(apiKey) {
		t.Error("Failed to revoke API key")
	}
	
	// Test inactive key
	_, valid := manager.ValidateAPIKey(apiKey)
	if valid {
		t.Error("Inactive API key was accepted")
	}
}

func TestAPIKeyManager_HasPermission(t *testing.T) {
	keyInfo := &APIKeyInfo{
		Permissions: []Permission{PermissionIngestLogs, PermissionMetrics},
	}
	
	manager := NewAPIKeyManager(nil)
	
	// Test existing permission
	if !manager.HasPermission(keyInfo, PermissionIngestLogs) {
		t.Error("Expected permission was denied")
	}
	
	// Test non-existing permission
	if manager.HasPermission(keyInfo, PermissionAdmin) {
		t.Error("Non-existing permission was granted")
	}
	
	// Test admin permission (should grant all)
	adminKeyInfo := &APIKeyInfo{
		Permissions: []Permission{PermissionAdmin},
	}
	
	if !manager.HasPermission(adminKeyInfo, PermissionIngestLogs) {
		t.Error("Admin permission should grant all permissions")
	}
}

func TestAPIKeyManager_NoAuthRequired(t *testing.T) {
	config := &APIKeyConfig{
		RequireAuth: false,
		APIKeys:     make(map[string]APIKeyInfo),
	}
	
	manager := NewAPIKeyManager(config)
	
	// Any key should be valid when auth is not required
	keyInfo, valid := manager.ValidateAPIKey("any-key")
	if !valid {
		t.Error("Key should be valid when auth is not required")
	}
	
	if keyInfo.Name != "no-auth" {
		t.Error("Expected default key info when auth is not required")
	}
	
	// Should have default permissions
	if !manager.HasPermission(keyInfo, PermissionIngestLogs) {
		t.Error("Default key should have ingest_logs permission")
	}
}

func TestAPIKeyManager_HashAPIKey(t *testing.T) {
	manager := NewAPIKeyManager(nil)
	
	key := "test-key"
	hash1 := manager.HashAPIKey(key)
	hash2 := manager.HashAPIKey(key)
	
	// Same key should produce same hash
	if hash1 != hash2 {
		t.Error("Same key produced different hashes")
	}
	
	// Different keys should produce different hashes
	hash3 := manager.HashAPIKey("different-key")
	if hash1 == hash3 {
		t.Error("Different keys produced same hash")
	}
	
	// Hash should be hex string
	if len(hash1) != 64 { // SHA-256 produces 64 character hex string
		t.Errorf("Expected hash length 64, got %d", len(hash1))
	}
}