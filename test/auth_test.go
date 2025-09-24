package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gherlein/go-netgear/pkg/netgear"
)

// TestBasicAuthentication implements Test 2.1: Basic Authentication
func TestBasicAuthentication(t *testing.T) {
	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			// Create unique cache directory for this test
			testCacheDir := filepath.Join(config.TestOptions.CacheDir, "basic_auth_test", switchConfig.Name)

			// Clean up any existing cache
			os.RemoveAll(testCacheDir)

			// Step 1: Create client with custom cache directory
			client, err := netgear.NewClient(switchConfig.Address,
				netgear.WithTokenCache(testCacheDir),
				netgear.WithVerbose(config.TestOptions.Verbose))
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			// Step 2: Login with valid credentials
			ctx := context.Background()
			err = client.Login(ctx, switchConfig.Password)
			if err != nil {
				t.Fatalf("Login failed: %v", err)
			}

			// Step 3: Verify IsAuthenticated() returns true
			if !client.IsAuthenticated() {
				t.Error("Expected client to be authenticated after login")
			}

			// Step 4: Check token file exists in cache directory
			tokenMgr := client.GetTokenManager()
			if ftm, ok := tokenMgr.(*netgear.FileTokenManager); ok {
				cacheDir := ftm.GetCacheDir()
				pattern := filepath.Join(cacheDir, "netgear-token-*.cache")
				files, err := filepath.Glob(pattern)
				if err != nil {
					t.Errorf("Failed to check for token files: %v", err)
				}
				if len(files) == 0 {
					t.Error("Expected token file to exist after login")
				}
			}

			// Step 5: Logout
			err = client.Logout(ctx)
			if err != nil {
				t.Errorf("Logout failed: %v", err)
			}

			// Step 6: Verify token file is removed
			if ftm, ok := tokenMgr.(*netgear.FileTokenManager); ok {
				cacheDir := ftm.GetCacheDir()
				pattern := filepath.Join(cacheDir, "netgear-token-*.cache")
				files, err := filepath.Glob(pattern)
				if err != nil {
					t.Errorf("Failed to check for token files after logout: %v", err)
				}
				if len(files) > 0 {
					t.Error("Expected token file to be removed after logout")
				}
			}

			// Cleanup
			os.RemoveAll(testCacheDir)
		})
	}
}

// TestTokenCachePersistence implements Test 2.2: Token Cache Persistence
func TestTokenCachePersistence(t *testing.T) {
	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			testCacheDir := filepath.Join(config.TestOptions.CacheDir, "cache_persistence_test", switchConfig.Name)

			// Clean up
			os.RemoveAll(testCacheDir)
			defer os.RemoveAll(testCacheDir)

			// Step 1: Create first client and login
			client1, err := netgear.NewClient(switchConfig.Address,
				netgear.WithTokenCache(testCacheDir),
				netgear.WithVerbose(config.TestOptions.Verbose))
			if err != nil {
				t.Fatalf("Failed to create first client: %v", err)
			}

			ctx := context.Background()
			err = client1.Login(ctx, switchConfig.Password)
			if err != nil {
				t.Fatalf("First client login failed: %v", err)
			}

			// Step 2: Create second client with same cache directory
			client2, err := netgear.NewClient(switchConfig.Address,
				netgear.WithTokenCache(testCacheDir),
				netgear.WithVerbose(config.TestOptions.Verbose))
			if err != nil {
				t.Fatalf("Failed to create second client: %v", err)
			}

			// Step 3: Verify second client uses cached token (should be authenticated)
			if !client2.IsAuthenticated() {
				t.Error("Expected second client to be authenticated using cached token")
			}

			// Step 4: Perform operation with second client
			status, err := client2.POE().GetStatus(ctx)
			if err != nil {
				t.Errorf("Operation with second client failed: %v", err)
			}

			// Step 5: Verify operation succeeds
			if len(status) == 0 {
				t.Error("Expected POE status data from second client")
			}

			// Cleanup
			client1.Logout(ctx)
		})
	}
}

// TestSessionExpiration implements Test 2.3: Session Expiration
func TestSessionExpiration(t *testing.T) {
	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			testCacheDir := filepath.Join(config.TestOptions.CacheDir, "session_expiration_test", switchConfig.Name)

			// Clean up
			os.RemoveAll(testCacheDir)
			defer os.RemoveAll(testCacheDir)

			// Step 1: Create client and login
			client, err := netgear.NewClient(switchConfig.Address,
				netgear.WithTokenCache(testCacheDir),
				netgear.WithVerbose(config.TestOptions.Verbose))
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			ctx := context.Background()
			err = client.Login(ctx, switchConfig.Password)
			if err != nil {
				t.Fatalf("Login failed: %v", err)
			}

			// Step 2: Manually corrupt token in cache
			if ftm, ok := client.GetTokenManager().(*netgear.FileTokenManager); ok {
				cacheDir := ftm.GetCacheDir()
				pattern := filepath.Join(cacheDir, "netgear-token-*.cache")
				files, err := filepath.Glob(pattern)
				if err != nil || len(files) == 0 {
					t.Fatalf("Failed to find token file to corrupt: %v", err)
				}

				// Corrupt the token file
				err = os.WriteFile(files[0], []byte("CORRUPTED:invalid_token"), 0600)
				if err != nil {
					t.Fatalf("Failed to corrupt token file: %v", err)
				}
			}

			// Step 3: Create new client with corrupted cache
			client2, err := netgear.NewClient(switchConfig.Address,
				netgear.WithTokenCache(testCacheDir),
				netgear.WithVerbose(config.TestOptions.Verbose))
			if err != nil {
				t.Fatalf("Failed to create second client: %v", err)
			}

			// Step 4: Attempt operation (should fail)
			_, err = client2.POE().GetStatus(ctx)
			if err == nil {
				t.Error("Expected operation to fail with corrupted token")
			}

			// Step 5: Re-login
			err = client2.Login(ctx, switchConfig.Password)
			if err != nil {
				t.Errorf("Re-login failed: %v", err)
			}

			// Step 6: Verify operation now succeeds
			status, err := client2.POE().GetStatus(ctx)
			if err != nil {
				t.Errorf("Operation after re-login failed: %v", err)
			}
			if len(status) == 0 {
				t.Error("Expected POE status data after re-login")
			}
		})
	}
}

// TestEnvironmentAuthentication implements Test 2.4: Environment Authentication
func TestEnvironmentAuthentication(t *testing.T) {
	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			testCacheDir := filepath.Join(config.TestOptions.CacheDir, "env_auth_test", switchConfig.Name)

			// Clean up
			os.RemoveAll(testCacheDir)
			defer os.RemoveAll(testCacheDir)

			// Step 1: Set NETGEAR_PASSWORD_<host> environment variable
			envVar := fmt.Sprintf("NETGEAR_PASSWORD_%s", normalizeHostForEnv(switchConfig.Address))
			oldValue := os.Getenv(envVar)
			defer func() {
				if oldValue == "" {
					os.Unsetenv(envVar)
				} else {
					os.Setenv(envVar, oldValue)
				}
			}()

			os.Setenv(envVar, switchConfig.Password)

			// Step 2: Create client
			client, err := netgear.NewClient(switchConfig.Address,
				netgear.WithTokenCache(testCacheDir),
				netgear.WithVerbose(config.TestOptions.Verbose))
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			// Step 3: Verify auto-authentication occurs
			if !client.IsAuthenticated() {
				// Try explicit auto-login
				ctx := context.Background()
				err = client.LoginAuto(ctx)
				if err != nil {
					t.Fatalf("Auto-authentication failed: %v", err)
				}
			}

			// Step 4: Perform operation without explicit login
			ctx := context.Background()
			status, err := client.POE().GetStatus(ctx)
			if err != nil {
				t.Errorf("Operation with environment auth failed: %v", err)
			}

			// Step 5: Verify operation succeeds
			if len(status) == 0 {
				t.Error("Expected POE status data with environment auth")
			}
		})
	}
}

// TestInvalidCredentials implements Test 6.1: Invalid Credentials
func TestInvalidCredentials(t *testing.T) {
	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			testCacheDir := filepath.Join(config.TestOptions.CacheDir, "invalid_creds_test", switchConfig.Name)

			// Clean up
			os.RemoveAll(testCacheDir)
			defer os.RemoveAll(testCacheDir)

			// Step 1: Attempt login with wrong password
			client, err := netgear.NewClient(switchConfig.Address,
				netgear.WithTokenCache(testCacheDir),
				netgear.WithVerbose(config.TestOptions.Verbose))
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}

			ctx := context.Background()
			wrongPassword := "definitely_wrong_password_12345"
			err = client.Login(ctx, wrongPassword)

			// Step 2: Verify appropriate error is returned
			if err == nil {
				t.Error("Expected error for invalid credentials")
			}

			// Step 3: Verify no token is cached
			if ftm, ok := client.GetTokenManager().(*netgear.FileTokenManager); ok {
				cacheDir := ftm.GetCacheDir()
				pattern := filepath.Join(cacheDir, "netgear-token-*.cache")
				files, err := filepath.Glob(pattern)
				if err == nil && len(files) > 0 {
					t.Error("Expected no token file after failed login")
				}
			}

			// Step 4: Verify IsAuthenticated() returns false
			if client.IsAuthenticated() {
				t.Error("Expected client to not be authenticated after failed login")
			}
		})
	}
}

// TestMultiSwitchAuthentication tests authentication with multiple switches
func TestMultiSwitchAuthentication(t *testing.T) {
	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	if len(config.Switches) < 2 {
		t.Skip("Skipping multi-switch test - need at least 2 switches configured")
	}

	testCacheDir := filepath.Join(config.TestOptions.CacheDir, "multi_switch_test")

	// Clean up
	os.RemoveAll(testCacheDir)
	defer os.RemoveAll(testCacheDir)

	// Create clients for all switches
	var clients []*netgear.Client
	ctx := context.Background()

	for i, switchConfig := range config.Switches {
		client, err := netgear.NewClient(switchConfig.Address,
			netgear.WithTokenCache(testCacheDir),
			netgear.WithVerbose(config.TestOptions.Verbose))
		if err != nil {
			t.Fatalf("Failed to create client %d: %v", i, err)
		}

		err = client.Login(ctx, switchConfig.Password)
		if err != nil {
			t.Fatalf("Failed to login to switch %d: %v", i, err)
		}

		clients = append(clients, client)
	}

	// Verify each client maintains separate authentication
	for i, client := range clients {
		if !client.IsAuthenticated() {
			t.Errorf("Client %d should be authenticated", i)
		}

		// Perform operation to verify session works
		status, err := client.POE().GetStatus(ctx)
		if err != nil {
			t.Errorf("Operation failed for client %d: %v", i, err)
		}
		if len(status) == 0 {
			t.Errorf("Expected POE status data for client %d", i)
		}
	}

	// Verify separate token files exist
	if len(clients) > 0 {
		if ftm, ok := clients[0].GetTokenManager().(*netgear.FileTokenManager); ok {
			cacheDir := ftm.GetCacheDir()
			pattern := filepath.Join(cacheDir, "netgear-token-*.cache")
			files, err := filepath.Glob(pattern)
			if err != nil {
				t.Errorf("Failed to check token files: %v", err)
			}
			if len(files) != len(clients) {
				t.Errorf("Expected %d token files, found %d", len(clients), len(files))
			}
		}
	}

	// Cleanup
	for _, client := range clients {
		client.Logout(ctx)
	}
}

// Helper function to normalize host for environment variable
func normalizeHostForEnv(host string) string {
	// Convert dots and colons to underscores, make uppercase
	normalized := ""
	for _, char := range host {
		if char == '.' || char == ':' {
			normalized += "_"
		} else {
			normalized += string(char)
		}
	}
	return strings.ToUpper(normalized)
}

// Benchmark test for authentication performance
func BenchmarkAuthentication(b *testing.B) {
	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		b.Skipf("Skipping benchmark - no config file: %v", err)
	}

	if len(config.Switches) == 0 {
		b.Skip("No switches configured for benchmark")
	}

	switchConfig := config.Switches[0]
	testCacheDir := filepath.Join(config.TestOptions.CacheDir, "benchmark_test")

	// Clean up
	os.RemoveAll(testCacheDir)
	defer os.RemoveAll(testCacheDir)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		client, err := netgear.NewClient(switchConfig.Address,
			netgear.WithTokenCache(testCacheDir))
		if err != nil {
			b.Fatalf("Failed to create client: %v", err)
		}

		ctx := context.Background()
		err = client.Login(ctx, switchConfig.Password)
		if err != nil {
			b.Fatalf("Login failed: %v", err)
		}

		client.Logout(ctx)
	}
}

// Test authentication timeout
func TestAuthenticationTimeout(t *testing.T) {
	// Test with invalid IP address to trigger timeout
	start := time.Now()

	_, err := netgear.NewClient("192.0.2.1", // RFC5737 test address
		netgear.WithTimeout(2*time.Second))

	duration := time.Since(start)

	// Should fail due to timeout during client creation/model detection
	if err == nil {
		t.Error("Expected client creation to fail due to timeout")
		return
	}

	// Should timeout within reasonable time (allow some buffer)
	if duration > 5*time.Second {
		t.Errorf("Timeout took too long: %v", duration)
	}

	// Test passed - timeout occurred as expected
	t.Logf("Client creation properly timed out after %v with error: %v", duration, err)
}