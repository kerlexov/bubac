package security

import (
	"github.com/gin-gonic/gin"
)

// SecurityHeadersConfig represents security headers configuration
type SecurityHeadersConfig struct {
	Enabled                   bool   `yaml:"enabled" json:"enabled"`
	ContentTypeOptions        string `yaml:"content_type_options" json:"content_type_options"`
	FrameOptions              string `yaml:"frame_options" json:"frame_options"`
	XSSProtection             string `yaml:"xss_protection" json:"xss_protection"`
	StrictTransportSecurity   string `yaml:"strict_transport_security" json:"strict_transport_security"`
	ContentSecurityPolicy     string `yaml:"content_security_policy" json:"content_security_policy"`
	ReferrerPolicy            string `yaml:"referrer_policy" json:"referrer_policy"`
	PermissionsPolicy         string `yaml:"permissions_policy" json:"permissions_policy"`
}

// DefaultSecurityHeadersConfig returns default security headers configuration
func DefaultSecurityHeadersConfig() *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		Enabled:                   true,
		ContentTypeOptions:        "nosniff",
		FrameOptions:              "DENY",
		XSSProtection:             "1; mode=block",
		StrictTransportSecurity:   "max-age=31536000; includeSubDomains; preload",
		ContentSecurityPolicy:     "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none';",
		ReferrerPolicy:            "strict-origin-when-cross-origin",
		PermissionsPolicy:         "geolocation=(), microphone=(), camera=(), payment=(), usb=(), magnetometer=(), gyroscope=(), accelerometer=()",
	}
}

// SecurityHeadersMiddleware creates a Gin middleware for security headers
func SecurityHeadersMiddleware(config *SecurityHeadersConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultSecurityHeadersConfig()
	}
	
	return func(c *gin.Context) {
		if !config.Enabled {
			c.Next()
			return
		}
		
		// X-Content-Type-Options
		if config.ContentTypeOptions != "" {
			c.Header("X-Content-Type-Options", config.ContentTypeOptions)
		}
		
		// X-Frame-Options
		if config.FrameOptions != "" {
			c.Header("X-Frame-Options", config.FrameOptions)
		}
		
		// X-XSS-Protection
		if config.XSSProtection != "" {
			c.Header("X-XSS-Protection", config.XSSProtection)
		}
		
		// Strict-Transport-Security (only add if HTTPS)
		if config.StrictTransportSecurity != "" && c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", config.StrictTransportSecurity)
		}
		
		// Content-Security-Policy
		if config.ContentSecurityPolicy != "" {
			c.Header("Content-Security-Policy", config.ContentSecurityPolicy)
		}
		
		// Referrer-Policy
		if config.ReferrerPolicy != "" {
			c.Header("Referrer-Policy", config.ReferrerPolicy)
		}
		
		// Permissions-Policy
		if config.PermissionsPolicy != "" {
			c.Header("Permissions-Policy", config.PermissionsPolicy)
		}
		
		// Additional security headers
		c.Header("X-Robots-Tag", "noindex, nofollow, nosnippet, noarchive")
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		
		// Remove server information
		c.Header("Server", "")
		
		c.Next()
	}
}

// HTTPSRedirectMiddleware redirects HTTP requests to HTTPS
func HTTPSRedirectMiddleware(enabled bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !enabled {
			c.Next()
			return
		}
		
		// Check if request is HTTP and should be redirected
		if c.Request.Header.Get("X-Forwarded-Proto") == "http" || 
		   (c.Request.TLS == nil && c.Request.Header.Get("X-Forwarded-Proto") == "") {
			
			// Skip redirect for health checks
			if c.Request.URL.Path == "/health" {
				c.Next()
				return
			}
			
			// Redirect to HTTPS
			httpsURL := "https://" + c.Request.Host + c.Request.RequestURI
			c.Redirect(301, httpsURL)
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// SecurityConfig represents overall security configuration
type SecurityConfig struct {
	Headers       *SecurityHeadersConfig `yaml:"headers" json:"headers"`
	HTTPSRedirect bool                   `yaml:"https_redirect" json:"https_redirect"`
	TrustedProxies []string              `yaml:"trusted_proxies" json:"trusted_proxies"`
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		Headers:       DefaultSecurityHeadersConfig(),
		HTTPSRedirect: false, // Usually handled by reverse proxy
		TrustedProxies: []string{
			"10.0.0.0/8",
			"172.16.0.0/12",
			"192.168.0.0/16",
			"127.0.0.1/32",
		},
	}
}

// ApplySecurityMiddleware applies all security middleware to a Gin engine
func ApplySecurityMiddleware(router *gin.Engine, config *SecurityConfig) error {
	if config == nil {
		config = DefaultSecurityConfig()
	}
	
	// Set trusted proxies
	if len(config.TrustedProxies) > 0 {
		if err := router.SetTrustedProxies(config.TrustedProxies); err != nil {
			return err
		}
	}
	
	// Apply HTTPS redirect middleware
	router.Use(HTTPSRedirectMiddleware(config.HTTPSRedirect))
	
	// Apply security headers middleware
	router.Use(SecurityHeadersMiddleware(config.Headers))
	
	return nil
}