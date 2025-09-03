package ratelimit

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kerlexov/mcp-logging-server/pkg/auth"
)

// RateLimitMiddleware creates a Gin middleware for rate limiting
func RateLimitMiddleware(rateLimiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip rate limiting for health checks
		if c.Request.URL.Path == "/health" {
			c.Next()
			return
		}

		// Get client IP
		clientIP := c.ClientIP()

		// Check IP-based rate limit first
		ipAllowed, ipInfo := rateLimiter.AllowIP(clientIP)
		if !ipAllowed {
			handleRateLimitExceeded(c, ipInfo, "IP")
			return
		}

		// Check API key-based rate limit if authenticated
		if keyInfo, exists := auth.GetAPIKeyInfo(c); exists {
			if apiKey, hasKey := auth.GetAPIKey(c); hasKey {
				// Use custom rate limit from API key info, or default
				customLimit := keyInfo.RateLimit
				if customLimit <= 0 {
					customLimit = 1000 // Default
				}

				keyAllowed, keyInfo := rateLimiter.AllowAPIKey(apiKey, customLimit)
				if !keyAllowed {
					handleRateLimitExceeded(c, keyInfo, "API_KEY")
					return
				}

				// Add API key rate limit headers
				addRateLimitHeaders(c, keyInfo, "API-Key")
			}
		}

		// Add IP rate limit headers
		addRateLimitHeaders(c, ipInfo, "IP")

		c.Next()
	}
}

// handleRateLimitExceeded handles rate limit exceeded responses
func handleRateLimitExceeded(c *gin.Context, info *RateLimitInfo, limitType string) {
	// Add rate limit headers
	addRateLimitHeaders(c, info, limitType)

	// Calculate retry after
	var retryAfter int
	if info.Blocked {
		retryAfter = int(time.Until(info.BlockedUntil).Seconds())
	} else {
		retryAfter = int(time.Until(info.ResetTime).Seconds())
	}

	if retryAfter < 0 {
		retryAfter = 60 // Default to 1 minute
	}

	c.Header("Retry-After", strconv.Itoa(retryAfter))

	response := gin.H{
		"error": "Rate limit exceeded",
		"code":  "RATE_LIMIT_EXCEEDED",
		"details": gin.H{
			"limit_type":  limitType,
			"retry_after": retryAfter,
			"blocked":     info.Blocked,
		},
	}

	if info.Blocked {
		response["message"] = "Too many violations. Temporarily blocked."
		response["details"].(gin.H)["blocked_until"] = info.BlockedUntil
		c.JSON(http.StatusTooManyRequests, response)
	} else {
		response["message"] = "Rate limit exceeded. Please slow down."
		c.JSON(http.StatusTooManyRequests, response)
	}

	c.Abort()
}

// addRateLimitHeaders adds rate limiting headers to the response
func addRateLimitHeaders(c *gin.Context, info *RateLimitInfo, prefix string) {
	c.Header("X-RateLimit-"+prefix+"-Remaining", strconv.Itoa(info.Remaining))
	c.Header("X-RateLimit-"+prefix+"-Reset", strconv.FormatInt(info.ResetTime.Unix(), 10))

	if info.Blocked {
		c.Header("X-RateLimit-"+prefix+"-Blocked", "true")
		c.Header("X-RateLimit-"+prefix+"-Blocked-Until", strconv.FormatInt(info.BlockedUntil.Unix(), 10))
	}
}

// AdminRateLimitMiddleware creates middleware for admin endpoints to manage rate limiting
func AdminRateLimitMiddleware(rateLimiter *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Add rate limit management endpoints
		switch c.Request.URL.Path {
		case "/admin/rate-limit/stats":
			if c.Request.Method == "GET" {
				handleGetRateLimitStats(c, rateLimiter)
				return
			}
		case "/admin/rate-limit/violations":
			if c.Request.Method == "GET" {
				handleGetViolations(c, rateLimiter)
				return
			}
		case "/admin/rate-limit/blocked":
			if c.Request.Method == "GET" {
				handleGetBlocked(c, rateLimiter)
				return
			}
		case "/admin/rate-limit/unblock":
			if c.Request.Method == "POST" {
				handleUnblock(c, rateLimiter)
				return
			}
		}

		c.Next()
	}
}

// handleGetRateLimitStats returns rate limiting statistics
func handleGetRateLimitStats(c *gin.Context, rateLimiter *RateLimiter) {
	stats := rateLimiter.GetStats()
	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

// handleGetViolations returns current violations
func handleGetViolations(c *gin.Context, rateLimiter *RateLimiter) {
	violations := rateLimiter.GetViolations()
	c.JSON(http.StatusOK, gin.H{
		"violations": violations,
	})
}

// handleGetBlocked returns currently blocked keys
func handleGetBlocked(c *gin.Context, rateLimiter *RateLimiter) {
	blocked := rateLimiter.GetBlocked()
	c.JSON(http.StatusOK, gin.H{
		"blocked": blocked,
	})
}

// handleUnblock unblocks a key
func handleUnblock(c *gin.Context, rateLimiter *RateLimiter) {
	var request struct {
		Key string `json:"key" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request",
			"details": err.Error(),
		})
		return
	}

	success := rateLimiter.UnblockKey(request.Key)
	if success {
		c.JSON(http.StatusOK, gin.H{
			"message": "Key unblocked successfully",
			"key":     request.Key,
		})
	} else {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Key not found in blocked list",
			"key":   request.Key,
		})
	}
}
