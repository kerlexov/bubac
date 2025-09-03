package ratelimit

import (
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 60, // 1 request per second
		BurstSize:         5,
		CleanupInterval:   time.Minute,
		BlockDuration:     time.Minute,
		MaxViolations:     3,
	}
	
	rl := NewRateLimiter(config)
	defer rl.Stop()
	
	key := "test-key"
	
	// Should allow initial requests up to burst size
	for i := 0; i < config.BurstSize; i++ {
		allowed, info := rl.Allow(key)
		if !allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
		if info.Remaining < 0 {
			t.Errorf("Remaining should be non-negative, got %d", info.Remaining)
		}
	}
	
	// Next request should be denied (burst exhausted)
	allowed, info := rl.Allow(key)
	if allowed {
		t.Error("Request should be denied after burst exhausted")
	}
	if info.Remaining != 0 {
		t.Errorf("Expected remaining to be 0, got %d", info.Remaining)
	}
}

func TestRateLimiter_AllowIP(t *testing.T) {
	config := DefaultRateLimitConfig()
	rl := NewRateLimiter(config)
	defer rl.Stop()
	
	ip := "192.168.1.1"
	
	// Should allow requests
	allowed, info := rl.AllowIP(ip)
	if !allowed {
		t.Error("IP request should be allowed")
	}
	if info.Remaining < 0 {
		t.Error("Remaining should be non-negative")
	}
}

func TestRateLimiter_AllowAPIKey(t *testing.T) {
	config := DefaultRateLimitConfig()
	rl := NewRateLimiter(config)
	defer rl.Stop()
	
	apiKey := "test-api-key"
	customLimit := 500
	
	// Should allow requests with custom limit
	allowed, info := rl.AllowAPIKey(apiKey, customLimit)
	if !allowed {
		t.Error("API key request should be allowed")
	}
	if info.Remaining < 0 {
		t.Error("Remaining should be non-negative")
	}
}

func TestRateLimiter_Blocking(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 60,
		BurstSize:         2,
		CleanupInterval:   time.Minute,
		BlockDuration:     time.Second * 2,
		MaxViolations:     2,
	}
	
	rl := NewRateLimiter(config)
	defer rl.Stop()
	
	key := "test-key"
	
	// Exhaust burst
	for i := 0; i < config.BurstSize; i++ {
		rl.Allow(key)
	}
	
	// Generate violations
	for i := 0; i < config.MaxViolations; i++ {
		allowed, _ := rl.Allow(key)
		if allowed {
			t.Error("Request should be denied")
		}
	}
	
	// Next request should be blocked
	allowed, info := rl.Allow(key)
	if allowed {
		t.Error("Request should be blocked")
	}
	if !info.Blocked {
		t.Error("Info should indicate blocked status")
	}
	
	// Wait for block to expire
	time.Sleep(config.BlockDuration + time.Millisecond*100)
	
	// Should be allowed again after block expires
	allowed, info = rl.Allow(key)
	if !allowed {
		t.Error("Request should be allowed after block expires")
	}
	if info.Blocked {
		t.Error("Should not be blocked after expiration")
	}
}

func TestRateLimiter_UnblockKey(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 60,
		BurstSize:         1,
		CleanupInterval:   time.Minute,
		BlockDuration:     time.Minute,
		MaxViolations:     1,
	}
	
	rl := NewRateLimiter(config)
	defer rl.Stop()
	
	key := "test-key"
	
	// Exhaust burst and generate violation
	rl.Allow(key)
	rl.Allow(key) // This should cause blocking
	
	// Verify blocked
	allowed, info := rl.Allow(key)
	if allowed || !info.Blocked {
		t.Error("Key should be blocked")
	}
	
	// Unblock manually
	success := rl.UnblockKey(key)
	if !success {
		t.Error("Unblock should succeed")
	}
	
	// Should be allowed now
	allowed, info = rl.Allow(key)
	if !allowed {
		t.Error("Request should be allowed after manual unblock")
	}
	if info.Blocked {
		t.Error("Should not be blocked after manual unblock")
	}
}

func TestRateLimiter_GetStats(t *testing.T) {
	config := DefaultRateLimitConfig()
	rl := NewRateLimiter(config)
	defer rl.Stop()
	
	// Generate some activity
	rl.Allow("key1")
	rl.Allow("key2")
	
	stats := rl.GetStats()
	if stats.ActiveLimiters != 2 {
		t.Errorf("Expected 2 active limiters, got %d", stats.ActiveLimiters)
	}
	if stats.Config.Enabled != config.Enabled {
		t.Error("Config should match")
	}
}

func TestRateLimiter_Disabled(t *testing.T) {
	config := &RateLimitConfig{
		Enabled: false,
	}
	
	rl := NewRateLimiter(config)
	defer rl.Stop()
	
	// Should always allow when disabled
	for i := 0; i < 1000; i++ {
		allowed, info := rl.Allow("test-key")
		if !allowed {
			t.Error("All requests should be allowed when rate limiting is disabled")
		}
		if info.Remaining != -1 {
			t.Error("Remaining should be -1 when disabled")
		}
	}
}

func TestRateLimiter_Cleanup(t *testing.T) {
	config := &RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 60,
		BurstSize:         1,
		CleanupInterval:   time.Millisecond * 100,
		BlockDuration:     time.Millisecond * 50,
		MaxViolations:     1,
	}
	
	rl := NewRateLimiter(config)
	defer rl.Stop()
	
	// Create some blocked entries
	rl.Allow("key1")
	rl.Allow("key1") // Block key1
	
	// Wait for cleanup
	time.Sleep(config.CleanupInterval + config.BlockDuration + time.Millisecond*50)
	
	// Blocked entries should be cleaned up
	blocked := rl.GetBlocked()
	if len(blocked) > 0 {
		t.Error("Blocked entries should be cleaned up")
	}
}