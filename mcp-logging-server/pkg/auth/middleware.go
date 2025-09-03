package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AuthMiddleware creates a Gin middleware for API key authentication
func AuthMiddleware(keyManager *APIKeyManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip authentication for health check and public endpoints
		if isPublicEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}
		
		// If authentication is not required, continue
		if !keyManager.GetConfig().RequireAuth {
			c.Next()
			return
		}
		
		// Extract API key from header
		apiKey := extractAPIKey(c)
		if apiKey == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "API key required",
				"code":  "MISSING_API_KEY",
			})
			c.Abort()
			return
		}
		
		// Validate API key
		keyInfo, valid := keyManager.ValidateAPIKey(apiKey)
		if !valid {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid or expired API key",
				"code":  "INVALID_API_KEY",
			})
			c.Abort()
			return
		}
		
		// Update last used timestamp
		keyManager.UpdateLastUsed(apiKey)
		
		// Store key info in context for later use
		c.Set("api_key_info", keyInfo)
		c.Set("api_key", apiKey)
		
		c.Next()
	}
}

// RequirePermission creates a middleware that requires a specific permission
func RequirePermission(keyManager *APIKeyManager, permission Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		// If authentication is not required, allow all
		if !keyManager.GetConfig().RequireAuth {
			c.Next()
			return
		}
		
		// Get key info from context (set by AuthMiddleware)
		keyInfoInterface, exists := c.Get("api_key_info")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": "Authentication required",
				"code":  "AUTHENTICATION_REQUIRED",
			})
			c.Abort()
			return
		}
		
		keyInfo, ok := keyInfoInterface.(*APIKeyInfo)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Invalid authentication context",
				"code":  "INVALID_AUTH_CONTEXT",
			})
			c.Abort()
			return
		}
		
		// Check permission
		if !keyManager.HasPermission(keyInfo, permission) {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Insufficient permissions",
				"code":  "INSUFFICIENT_PERMISSIONS",
				"required_permission": permission,
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// extractAPIKey extracts the API key from various sources
func extractAPIKey(c *gin.Context) string {
	// Try X-API-Key header first
	if apiKey := c.GetHeader("X-API-Key"); apiKey != "" {
		return apiKey
	}
	
	// Try Authorization header with Bearer token
	if auth := c.GetHeader("Authorization"); auth != "" {
		if strings.HasPrefix(auth, "Bearer ") {
			return strings.TrimPrefix(auth, "Bearer ")
		}
		if strings.HasPrefix(auth, "ApiKey ") {
			return strings.TrimPrefix(auth, "ApiKey ")
		}
	}
	
	// Try query parameter as fallback (less secure)
	if apiKey := c.Query("api_key"); apiKey != "" {
		return apiKey
	}
	
	return ""
}

// isPublicEndpoint checks if an endpoint should be publicly accessible
func isPublicEndpoint(path string) bool {
	publicEndpoints := []string{
		"/health",
		"/ping",
		"/version",
	}
	
	for _, endpoint := range publicEndpoints {
		if path == endpoint {
			return true
		}
	}
	
	return false
}

// GetAPIKeyInfo retrieves API key info from the Gin context
func GetAPIKeyInfo(c *gin.Context) (*APIKeyInfo, bool) {
	keyInfoInterface, exists := c.Get("api_key_info")
	if !exists {
		return nil, false
	}
	
	keyInfo, ok := keyInfoInterface.(*APIKeyInfo)
	return keyInfo, ok
}

// GetAPIKey retrieves the API key from the Gin context
func GetAPIKey(c *gin.Context) (string, bool) {
	apiKeyInterface, exists := c.Get("api_key")
	if !exists {
		return "", false
	}
	
	apiKey, ok := apiKeyInterface.(string)
	return apiKey, ok
}