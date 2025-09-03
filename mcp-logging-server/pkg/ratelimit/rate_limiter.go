package ratelimit

import (
	"fmt"
	"net"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// LimitType represents the type of rate limit
type LimitType string

const (
	LimitTypeIP     LimitType = "ip"
	LimitTypeAPIKey LimitType = "api_key"
	LimitTypeGlobal LimitType = "global"
)

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	Enabled           bool          `yaml:"enabled" json:"enabled"`
	RequestsPerMinute int           `yaml:"requests_per_minute" json:"requests_per_minute"`
	BurstSize         int           `yaml:"burst_size" json:"burst_size"`
	CleanupInterval   time.Duration `yaml:"cleanup_interval" json:"cleanup_interval"`
	BlockDuration     time.Duration `yaml:"block_duration" json:"block_duration"`
	MaxViolations     int           `yaml:"max_violations" json:"max_violations"`
}

// DefaultRateLimitConfig returns default rate limiting configuration
func DefaultRateLimitConfig() *RateLimitConfig {
	return &RateLimitConfig{
		Enabled:           true,
		RequestsPerMinute: 1000,
		BurstSize:         100,
		CleanupInterval:   5 * time.Minute,
		BlockDuration:     10 * time.Minute,
		MaxViolations:     5,
	}
}

// RateLimiter implements sophisticated rate limiting with abuse prevention
type RateLimiter struct {
	config     *RateLimitConfig
	limiters   map[string]*rate.Limiter
	violations map[string]*ViolationTracker
	blocked    map[string]time.Time
	mutex      sync.RWMutex
	stopChan   chan struct{}
}

// ViolationTracker tracks rate limit violations for abuse prevention
type ViolationTracker struct {
	Count     int       `json:"count"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(config *RateLimitConfig) *RateLimiter {
	if config == nil {
		config = DefaultRateLimitConfig()
	}
	
	rl := &RateLimiter{
		config:     config,
		limiters:   make(map[string]*rate.Limiter),
		violations: make(map[string]*ViolationTracker),
		blocked:    make(map[string]time.Time),
		stopChan:   make(chan struct{}),
	}
	
	// Start cleanup routine
	go rl.cleanupRoutine()
	
	return rl
}

// Allow checks if a request is allowed for the given key
func (rl *RateLimiter) Allow(key string, customLimit ...int) (bool, *RateLimitInfo) {
	if !rl.config.Enabled {
		return true, &RateLimitInfo{
			Allowed:   true,
			Remaining: -1,
			ResetTime: time.Time{},
		}
	}
	
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	// Check if key is currently blocked
	if blockedUntil, isBlocked := rl.blocked[key]; isBlocked {
		if time.Now().Before(blockedUntil) {
			return false, &RateLimitInfo{
				Allowed:     false,
				Remaining:   0,
				ResetTime:   blockedUntil,
				Blocked:     true,
				BlockedUntil: blockedUntil,
			}
		}
		// Block has expired, remove it
		delete(rl.blocked, key)
	}
	
	// Get or create limiter for this key
	limiter := rl.getLimiter(key, customLimit...)
	
	// Check if request is allowed
	allowed := limiter.Allow()
	
	info := &RateLimitInfo{
		Allowed:   allowed,
		Remaining: int(limiter.Tokens()),
		ResetTime: time.Now().Add(time.Minute),
	}
	
	if !allowed {
		// Track violation
		rl.trackViolation(key)
		
		// Check if we should block this key
		if rl.shouldBlock(key) {
			blockUntil := time.Now().Add(rl.config.BlockDuration)
			rl.blocked[key] = blockUntil
			info.Blocked = true
			info.BlockedUntil = blockUntil
		}
	}
	
	return allowed, info
}

// AllowIP checks if a request is allowed for the given IP address
func (rl *RateLimiter) AllowIP(ip string) (bool, *RateLimitInfo) {
	// Normalize IP address
	if parsedIP := net.ParseIP(ip); parsedIP != nil {
		ip = parsedIP.String()
	}
	
	return rl.Allow(fmt.Sprintf("ip:%s", ip))
}

// AllowAPIKey checks if a request is allowed for the given API key
func (rl *RateLimiter) AllowAPIKey(apiKey string, customLimit int) (bool, *RateLimitInfo) {
	return rl.Allow(fmt.Sprintf("api_key:%s", apiKey), customLimit)
}

// getLimiter gets or creates a rate limiter for the given key
func (rl *RateLimiter) getLimiter(key string, customLimit ...int) *rate.Limiter {
	limiter, exists := rl.limiters[key]
	if !exists {
		requestsPerMinute := rl.config.RequestsPerMinute
		if len(customLimit) > 0 && customLimit[0] > 0 {
			requestsPerMinute = customLimit[0]
		}
		
		// Convert requests per minute to requests per second
		rps := rate.Limit(float64(requestsPerMinute) / 60.0)
		limiter = rate.NewLimiter(rps, rl.config.BurstSize)
		rl.limiters[key] = limiter
	}
	return limiter
}

// trackViolation tracks a rate limit violation
func (rl *RateLimiter) trackViolation(key string) {
	now := time.Now()
	
	if tracker, exists := rl.violations[key]; exists {
		tracker.Count++
		tracker.LastSeen = now
	} else {
		rl.violations[key] = &ViolationTracker{
			Count:     1,
			FirstSeen: now,
			LastSeen:  now,
		}
	}
}

// shouldBlock determines if a key should be temporarily blocked
func (rl *RateLimiter) shouldBlock(key string) bool {
	tracker, exists := rl.violations[key]
	if !exists {
		return false
	}
	
	// Block if violations exceed threshold within the block duration
	if tracker.Count >= rl.config.MaxViolations {
		timeSinceFirst := time.Since(tracker.FirstSeen)
		return timeSinceFirst <= rl.config.BlockDuration
	}
	
	return false
}

// cleanupRoutine periodically cleans up old limiters and violations
func (rl *RateLimiter) cleanupRoutine() {
	ticker := time.NewTicker(rl.config.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			rl.cleanup()
		case <-rl.stopChan:
			return
		}
	}
}

// cleanup removes old limiters, violations, and expired blocks
func (rl *RateLimiter) cleanup() {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	now := time.Now()
	cutoff := now.Add(-rl.config.CleanupInterval)
	
	// Clean up expired blocks
	for key, blockedUntil := range rl.blocked {
		if now.After(blockedUntil) {
			delete(rl.blocked, key)
		}
	}
	
	// Clean up old violations
	for key, tracker := range rl.violations {
		if tracker.LastSeen.Before(cutoff) {
			delete(rl.violations, key)
		}
	}
	
	// Clean up unused limiters (keep them for a while in case they're needed again)
	// This is a simple cleanup - in production, you might want more sophisticated logic
	if len(rl.limiters) > 10000 { // Arbitrary threshold
		// Remove half of the limiters (simple strategy)
		count := 0
		for key := range rl.limiters {
			if count > len(rl.limiters)/2 {
				break
			}
			delete(rl.limiters, key)
			count++
		}
	}
}

// GetStats returns rate limiting statistics
func (rl *RateLimiter) GetStats() *RateLimitStats {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()
	
	return &RateLimitStats{
		ActiveLimiters:  len(rl.limiters),
		ActiveViolators: len(rl.violations),
		BlockedKeys:     len(rl.blocked),
		Config:          *rl.config,
	}
}

// GetViolations returns current violations
func (rl *RateLimiter) GetViolations() map[string]ViolationTracker {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()
	
	violations := make(map[string]ViolationTracker)
	for key, tracker := range rl.violations {
		violations[key] = *tracker
	}
	
	return violations
}

// GetBlocked returns currently blocked keys
func (rl *RateLimiter) GetBlocked() map[string]time.Time {
	rl.mutex.RLock()
	defer rl.mutex.RUnlock()
	
	blocked := make(map[string]time.Time)
	for key, blockedUntil := range rl.blocked {
		blocked[key] = blockedUntil
	}
	
	return blocked
}

// UnblockKey manually unblocks a key
func (rl *RateLimiter) UnblockKey(key string) bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()
	
	if _, exists := rl.blocked[key]; exists {
		delete(rl.blocked, key)
		// Also clear violations for this key
		delete(rl.violations, key)
		return true
	}
	
	return false
}

// Stop stops the rate limiter and cleanup routine
func (rl *RateLimiter) Stop() {
	close(rl.stopChan)
}

// RateLimitInfo contains information about a rate limit check
type RateLimitInfo struct {
	Allowed      bool      `json:"allowed"`
	Remaining    int       `json:"remaining"`
	ResetTime    time.Time `json:"reset_time"`
	Blocked      bool      `json:"blocked"`
	BlockedUntil time.Time `json:"blocked_until,omitempty"`
}

// RateLimitStats contains rate limiting statistics
type RateLimitStats struct {
	ActiveLimiters  int               `json:"active_limiters"`
	ActiveViolators int               `json:"active_violators"`
	BlockedKeys     int               `json:"blocked_keys"`
	Config          RateLimitConfig   `json:"config"`
}