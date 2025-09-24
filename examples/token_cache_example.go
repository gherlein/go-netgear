package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gherlein/go-netgear/pkg/netgear"
)

func main() {
	// Example 1: Use default cache location (~/.cache/go-netgear)
	defaultCacheExample()

	// Example 2: Use custom cache directory
	customCacheDirExample()

	// Example 3: Use application-specific cache
	appSpecificCacheExample()

	// Example 4: Use in-memory cache only (no persistence)
	memoryOnlyCacheExample()

	// Example 5: Manage cache programmatically
	cacheManagementExample()
}

// Example 1: Default cache location following XDG Base Directory Specification
func defaultCacheExample() {
	fmt.Println("=== Default Cache Location Example ===")

	// Creates client with default cache location:
	// - $XDG_CACHE_HOME/go-netgear/ (if set)
	// - ~/.cache/go-netgear/ (fallback)
	client, err := netgear.NewClient("192.168.1.10")
	if err != nil {
		log.Printf("Failed to create client: %v", err)
		return
	}

	ctx := context.Background()
	err = client.Login(ctx, "admin_password")
	if err != nil {
		log.Printf("Login failed: %v", err)
		return
	}

	fmt.Println("Successfully authenticated with default cache location")
	fmt.Println("Token cached for future use")
}

// Example 2: Custom cache directory
func customCacheDirExample() {
	fmt.Println("\n=== Custom Cache Directory Example ===")

	// Use a custom directory for token caching
	cacheDir := "/var/lib/myapp/cache"

	client, err := netgear.NewClient("192.168.1.11",
		netgear.WithTokenCache(cacheDir))
	if err != nil {
		log.Printf("Failed to create client: %v", err)
		return
	}

	ctx := context.Background()
	err = client.Login(ctx, "admin_password")
	if err != nil {
		log.Printf("Login failed: %v", err)
		return
	}

	fmt.Printf("Token cached in: %s\n", cacheDir)
}

// Example 3: Application-specific cache directory
func appSpecificCacheExample() {
	fmt.Println("\n=== Application-Specific Cache Example ===")

	// Follow XDG spec but use app-specific subdirectory
	var cacheDir string

	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		cacheDir = filepath.Join(xdgCache, "myapp", "netgear-tokens")
	} else if home, err := os.UserHomeDir(); err == nil {
		cacheDir = filepath.Join(home, ".cache", "myapp", "netgear-tokens")
	} else {
		cacheDir = filepath.Join("/tmp", "myapp", "netgear-tokens")
	}

	client, err := netgear.NewClient("192.168.1.12",
		netgear.WithTokenCache(cacheDir))
	if err != nil {
		log.Printf("Failed to create client: %v", err)
		return
	}

	fmt.Printf("Using app-specific cache: %s\n", cacheDir)

	// Set up environment auth and let it auto-authenticate
	os.Setenv("NETGEAR_PASSWORD_192_168_1_12", "admin_password")

	// Client will auto-authenticate if environment password is found
	if !client.IsAuthenticated() {
		ctx := context.Background()
		err = client.LoginAuto(ctx)
		if err != nil {
			log.Printf("Auto-login failed: %v", err)
			return
		}
	}

	fmt.Println("Token cached in application-specific directory")
}

// Example 4: Memory-only cache (no disk persistence)
func memoryOnlyCacheExample() {
	fmt.Println("\n=== Memory-Only Cache Example ===")

	// Use memory-only token manager for sensitive environments
	client, err := netgear.NewClient("192.168.1.13",
		netgear.WithTokenManager(netgear.NewMemoryTokenManager()))
	if err != nil {
		log.Printf("Failed to create client: %v", err)
		return
	}

	ctx := context.Background()
	err = client.Login(ctx, "admin_password")
	if err != nil {
		log.Printf("Login failed: %v", err)
		return
	}

	fmt.Println("Token stored in memory only (not persisted to disk)")
	fmt.Println("Token will be lost when program exits")
}

// Example 5: Cache management
func cacheManagementExample() {
	fmt.Println("\n=== Cache Management Example ===")

	// Create client with custom cache
	cacheDir := "/tmp/netgear-demo-cache"
	client, err := netgear.NewClient("192.168.1.14",
		netgear.WithTokenCache(cacheDir))
	if err != nil {
		log.Printf("Failed to create client: %v", err)
		return
	}

	ctx := context.Background()

	// Login to create a cached token
	err = client.Login(ctx, "admin_password")
	if err != nil {
		log.Printf("Login failed: %v", err)
		return
	}

	// Get cache directory information
	tokenMgr := client.GetTokenManager()
	if ftm, ok := tokenMgr.(*netgear.FileTokenManager); ok {
		fmt.Printf("Cache directory: %s\n", ftm.GetCacheDir())

		// List cached tokens (for demonstration)
		pattern := filepath.Join(ftm.GetCacheDir(), "netgear-token-*.cache")
		files, _ := filepath.Glob(pattern)
		fmt.Printf("Cached tokens: %d\n", len(files))
		for _, file := range files {
			fmt.Printf("  - %s\n", filepath.Base(file))
		}

		// Clear specific token
		fmt.Println("\nClearing token for current switch...")
		err = client.Logout(ctx)
		if err != nil {
			log.Printf("Logout failed: %v", err)
		}

		// Clear all tokens
		fmt.Println("Clearing all cached tokens...")
		err = ftm.ClearAllTokens()
		if err != nil {
			log.Printf("Failed to clear all tokens: %v", err)
		} else {
			fmt.Println("All tokens cleared successfully")
		}
	}
}

// Example of production-ready initialization with error handling
func productionExample() (*netgear.Client, error) {
	// Determine cache directory with fallbacks
	cacheDir := os.Getenv("NETGEAR_CACHE_DIR")
	if cacheDir == "" {
		// Use XDG_CACHE_HOME if available
		if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
			cacheDir = filepath.Join(xdgCache, "myapp")
		} else {
			// Fall back to user cache directory
			home, err := os.UserHomeDir()
			if err != nil {
				// Final fallback to temp directory
				cacheDir = filepath.Join(os.TempDir(), "myapp-cache")
			} else {
				cacheDir = filepath.Join(home, ".cache", "myapp")
			}
		}
	}

	// Create client with production settings
	client, err := netgear.NewClient("switch.example.com",
		netgear.WithTokenCache(cacheDir),
		netgear.WithTimeout(30*time.Second),
		netgear.WithVerbose(os.Getenv("DEBUG") == "true"))

	if err != nil {
		return nil, fmt.Errorf("failed to create Netgear client: %w", err)
	}

	// Log cache location for debugging
	if verbose := os.Getenv("DEBUG"); verbose == "true" {
		if ftm, ok := client.GetTokenManager().(*netgear.FileTokenManager); ok {
			log.Printf("Token cache directory: %s", ftm.GetCacheDir())
		}
	}

	return client, nil
}