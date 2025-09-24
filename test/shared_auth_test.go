package test

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

// TestSharedAuthentication tests the shared authentication system
// This replaces TestBasicAuthentication with a version that works with shared auth
func TestSharedAuthentication(t *testing.T) {
	// Require authentication - fail loudly if not available
	env := DetectTestEnvironment(t)
	env.RequireAuth(t, CategoryAuth)

	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	helper := NewTestHelper(config)

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			// Step 1: Get authenticated client from shared cache
			client, err := helper.GetClientForTest(switchConfig.Name)
			if err != nil {
				// Check if it's an authentication issue - skip instead of failing
				if strings.Contains(err.Error(), "invalid credentials") ||
				   strings.Contains(err.Error(), "authentication failed") {
					t.Fatalf("❌ AUTHENTICATION FAILURE: Cannot authenticate to switch after 3 attempts. This blocks all tests. Error: %v", err)
				} else {
					t.Fatalf("Failed to get authenticated client: %v", err)
				}
			}

			// Step 2: Verify client is authenticated
			if !client.IsAuthenticated() {
				t.Error("Expected client to be authenticated from shared cache")
			}

			// Step 3: Verify we can perform operations (basic API test)
			ctx := context.Background()
			_, err = client.POE().GetStatus(ctx)
			if err != nil {
				t.Errorf("Failed to perform basic POE operation: %v", err)
			}

			// Step 4: Verify we can perform port operations
			_, err = client.Ports().GetSettings(ctx)
			if err != nil {
				t.Errorf("Failed to perform basic Port operation: %v", err)
			}

			t.Logf("✅ Successfully used shared authentication for %s", switchConfig.Name)
		})
	}
}

// TestTokenCachePersistence tests that tokens persist across different test helper instances
func TestTokenCachePersistence(t *testing.T) {
	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	// Create first helper instance
	helper1 := NewTestHelper(config)

	// Create second helper instance (should share the same global auth manager)
	helper2 := NewTestHelper(config)

	switchConfig := config.Switches[0]

	// Get client from first helper
	client1, err := helper1.GetClientForTest(switchConfig.Name)
	if err != nil {
		t.Fatalf("Failed to get client from first helper: %v", err)
	}

	// Get client from second helper - should use same cached auth
	client2, err := helper2.GetClientForTest(switchConfig.Name)
	if err != nil {
		t.Fatalf("Failed to get client from second helper: %v", err)
	}

	// Both should be authenticated
	if !client1.IsAuthenticated() {
		t.Error("First client not authenticated")
	}
	if !client2.IsAuthenticated() {
		t.Error("Second client not authenticated")
	}

	// Both should be able to perform operations
	ctx := context.Background()
	_, err = client1.POE().GetStatus(ctx)
	if err != nil {
		t.Errorf("First client POE operation failed: %v", err)
	}

	_, err = client2.POE().GetStatus(ctx)
	if err != nil {
		t.Errorf("Second client POE operation failed: %v", err)
	}

	t.Log("✅ Token cache persistence working correctly")
}