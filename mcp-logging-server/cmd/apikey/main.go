package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/your-org/mcp-logging-server/pkg/auth"
)

func main() {
	var (
		configPath = flag.String("config", "./config/api-keys.yaml", "Path to API keys configuration file")
		action     = flag.String("action", "", "Action to perform: create, list, revoke, rotate")
		name       = flag.String("name", "", "Name for the API key")
		permissions = flag.String("permissions", "ingest_logs", "Comma-separated list of permissions")
		rateLimit  = flag.Int("rate-limit", 1000, "Rate limit for the API key (requests per minute)")
		expiresIn  = flag.String("expires-in", "", "Expiration duration (e.g., '30d', '1y', '6m')")
		apiKey     = flag.String("key", "", "API key to operate on (for revoke/rotate)")
	)
	flag.Parse()

	if *action == "" {
		fmt.Println("Usage: apikey -action=<create|list|revoke|rotate> [options]")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Load existing configuration
	config, err := auth.LoadAPIKeyConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	manager := auth.NewAPIKeyManager(config)

	switch *action {
	case "create":
		if *name == "" {
			log.Fatal("Name is required for creating API keys")
		}
		
		// Parse permissions
		perms := parsePermissions(*permissions)
		
		// Parse expiration
		var expiresAt *time.Time
		if *expiresIn != "" {
			exp, err := parseExpiration(*expiresIn)
			if err != nil {
				log.Fatalf("Invalid expiration format: %v", err)
			}
			expiresAt = &exp
		}
		
		// Create API key
		key, err := manager.CreateAPIKey(*name, perms, *rateLimit, expiresAt)
		if err != nil {
			log.Fatalf("Failed to create API key: %v", err)
		}
		
		fmt.Printf("Created API key: %s\n", key)
		fmt.Printf("Name: %s\n", *name)
		fmt.Printf("Permissions: %v\n", perms)
		fmt.Printf("Rate Limit: %d requests/minute\n", *rateLimit)
		if expiresAt != nil {
			fmt.Printf("Expires: %s\n", expiresAt.Format(time.RFC3339))
		}
		
		// Save configuration
		if err := auth.SaveAPIKeyConfig(*configPath, config); err != nil {
			log.Fatalf("Failed to save config: %v", err)
		}
		
		fmt.Printf("\nConfiguration saved to: %s\n", *configPath)
		fmt.Println("\n⚠️  IMPORTANT: Store this API key securely. It cannot be retrieved again.")

	case "list":
		keys := manager.ListAPIKeys()
		if len(keys) == 0 {
			fmt.Println("No API keys found")
			return
		}
		
		fmt.Printf("%-20s %-15s %-30s %-20s %-10s\n", "Name", "Permissions", "Created", "Expires", "Active")
		fmt.Println(strings.Repeat("-", 95))
		
		for _, keyInfo := range keys {
			permsStr := strings.Join(permissionsToStrings(keyInfo.Permissions), ",")
			if len(permsStr) > 15 {
				permsStr = permsStr[:12] + "..."
			}
			
			expiresStr := "Never"
			if keyInfo.ExpiresAt != nil {
				expiresStr = keyInfo.ExpiresAt.Format("2006-01-02")
			}
			
			activeStr := "Yes"
			if !keyInfo.IsActive {
				activeStr = "No"
			}
			
			fmt.Printf("%-20s %-15s %-30s %-20s %-10s\n",
				keyInfo.Name,
				permsStr,
				keyInfo.CreatedAt.Format("2006-01-02 15:04:05"),
				expiresStr,
				activeStr,
			)
		}

	case "revoke":
		if *apiKey == "" {
			log.Fatal("API key is required for revocation")
		}
		
		if manager.RevokeAPIKey(*apiKey) {
			fmt.Printf("API key revoked successfully\n")
			
			// Save configuration
			if err := auth.SaveAPIKeyConfig(*configPath, config); err != nil {
				log.Fatalf("Failed to save config: %v", err)
			}
		} else {
			fmt.Printf("API key not found\n")
			os.Exit(1)
		}

	case "rotate":
		if *apiKey == "" {
			log.Fatal("API key is required for rotation")
		}
		
		// Get existing key info
		keyInfo, valid := manager.ValidateAPIKey(*apiKey)
		if !valid {
			log.Fatal("API key not found or invalid")
		}
		
		// Revoke old key
		manager.RevokeAPIKey(*apiKey)
		
		// Create new key with same properties
		newKey, err := manager.CreateAPIKey(keyInfo.Name+"_rotated", keyInfo.Permissions, keyInfo.RateLimit, keyInfo.ExpiresAt)
		if err != nil {
			log.Fatalf("Failed to create new API key: %v", err)
		}
		
		fmt.Printf("Old API key revoked\n")
		fmt.Printf("New API key: %s\n", newKey)
		
		// Save configuration
		if err := auth.SaveAPIKeyConfig(*configPath, config); err != nil {
			log.Fatalf("Failed to save config: %v", err)
		}

	default:
		log.Fatalf("Unknown action: %s", *action)
	}
}

func parsePermissions(permsStr string) []auth.Permission {
	parts := strings.Split(permsStr, ",")
	perms := make([]auth.Permission, 0, len(parts))
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		switch part {
		case "ingest_logs":
			perms = append(perms, auth.PermissionIngestLogs)
		case "query_logs":
			perms = append(perms, auth.PermissionQueryLogs)
		case "admin":
			perms = append(perms, auth.PermissionAdmin)
		case "metrics":
			perms = append(perms, auth.PermissionMetrics)
		default:
			log.Fatalf("Unknown permission: %s", part)
		}
	}
	
	return perms
}

func permissionsToStrings(perms []auth.Permission) []string {
	strs := make([]string, len(perms))
	for i, perm := range perms {
		strs[i] = string(perm)
	}
	return strs
}

func parseExpiration(expiresIn string) (time.Time, error) {
	now := time.Now()
	
	if strings.HasSuffix(expiresIn, "d") {
		days := strings.TrimSuffix(expiresIn, "d")
		var d int
		if _, err := fmt.Sscanf(days, "%d", &d); err != nil {
			return time.Time{}, err
		}
		return now.AddDate(0, 0, d), nil
	}
	
	if strings.HasSuffix(expiresIn, "m") {
		months := strings.TrimSuffix(expiresIn, "m")
		var m int
		if _, err := fmt.Sscanf(months, "%d", &m); err != nil {
			return time.Time{}, err
		}
		return now.AddDate(0, m, 0), nil
	}
	
	if strings.HasSuffix(expiresIn, "y") {
		years := strings.TrimSuffix(expiresIn, "y")
		var y int
		if _, err := fmt.Sscanf(years, "%d", &y); err != nil {
			return time.Time{}, err
		}
		return now.AddDate(y, 0, 0), nil
	}
	
	return time.Time{}, fmt.Errorf("invalid expiration format, use: 30d, 6m, 1y")
}