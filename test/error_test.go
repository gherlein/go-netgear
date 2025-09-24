package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gherlein/go-netgear/pkg/netgear"
)

// TestInvalidPortNumbers implements Test 6.3: Invalid Port Numbers
func TestInvalidPortNumbers(t *testing.T) {
	// Require authentication - fail loudly if not available
	env := DetectTestEnvironment(t)
	env.RequireAuth(t, CategoryBasic)

	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	helper := NewTestHelper(config)
	fixtures := NewTestFixtures()
	invalidPorts := fixtures.GetInvalidPortNumbers()

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			if switchConfig.ShouldSkipTest("invalid_port_numbers") {
				t.Skip("Test excluded for this switch")
			}

			client, err := helper.GetClientForTest(switchConfig.Name)
			if err != nil {
				t.Fatalf("Failed to get authenticated client: %v", err)
			}

			ctx := context.Background()

			for _, invalidPort := range invalidPorts {
				t.Run(fmt.Sprintf("port_%d", invalidPort), func(t *testing.T) {
					// Step 1 & 2: Attempt operations on invalid port numbers

					// Test POE operations
					err := client.POE().UpdatePort(ctx, netgear.POEPortUpdate{
						PortID: invalidPort,
						Enabled: func() *bool { b := true; return &b }(),
					})
					if err == nil {
						t.Errorf("Expected error for POE operation on invalid port %d", invalidPort)
					}

					// Test port operations
					err = client.Ports().SetPortName(ctx, invalidPort, "TEST_PORT")
					if err == nil {
						t.Errorf("Expected error for port name operation on invalid port %d", invalidPort)
					}

					err = client.Ports().SetPortSpeed(ctx, invalidPort, netgear.PortSpeedAuto)
					if err == nil {
						t.Errorf("Expected error for port speed operation on invalid port %d", invalidPort)
					}

					err = client.Ports().SetPortFlowControl(ctx, invalidPort, true)
					if err == nil {
						t.Errorf("Expected error for flow control operation on invalid port %d", invalidPort)
					}

					err = client.Ports().SetPortLimits(ctx, invalidPort, "unlimited", "unlimited")
					if err == nil {
						t.Errorf("Expected error for rate limit operation on invalid port %d", invalidPort)
					}
				})
			}

			// Step 4: Verify switch state is not corrupted
			// Get all settings to ensure switch is still responding correctly
			poeSettings, err := client.POE().GetSettings(ctx)
			if err != nil {
				t.Errorf("Switch POE settings corrupted after invalid port operations: %v", err)
			}

			portSettings, err := client.Ports().GetSettings(ctx)
			if err != nil {
				t.Errorf("Switch port settings corrupted after invalid port operations: %v", err)
			}

			// Verify expected number of ports
			expectedPortCount := fixtures.GetPortCount(switchConfig.Model)
			if len(poeSettings) != expectedPortCount {
				t.Errorf("POE settings count changed after invalid operations: expected %d, got %d", expectedPortCount, len(poeSettings))
			}
			if len(portSettings) != expectedPortCount {
				t.Errorf("Port settings count changed after invalid operations: expected %d, got %d", expectedPortCount, len(portSettings))
			}
		})
	}
}

// TestInvalidConfigurationValues implements Test 6.4: Invalid Configuration Values
func TestInvalidConfigurationValues(t *testing.T) {
	// Require authentication - fail loudly if not available
	env := DetectTestEnvironment(t)
	env.RequireAuth(t, CategoryBasic)

	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	helper := NewTestHelper(config)
	fixtures := NewTestFixtures()

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			if switchConfig.ShouldSkipTest("invalid_configuration_values") {
				t.Skip("Test excluded for this switch")
			}

			client, err := helper.GetClientForTest(switchConfig.Name)
			if err != nil {
				t.Fatalf("Failed to get authenticated client: %v", err)
			}

			ctx := context.Background()

			// Use the first test port for operations
			if len(switchConfig.TestPorts) == 0 {
				t.Skip("No test ports configured")
			}
			testPort := switchConfig.TestPorts[0]

			// Capture initial state for verification
			initialPOESettings, err := client.POE().GetSettings(ctx)
			if err != nil {
				t.Fatalf("Failed to get initial POE settings: %v", err)
			}

			initialPortSettings, err := client.Ports().GetSettings(ctx)
			if err != nil {
				t.Fatalf("Failed to get initial port settings: %v", err)
			}

			// Step 1: Attempt to set invalid POE mode (create an invalid mode)
			invalidMode := netgear.POEMode("invalid-mode-999")
			err = client.POE().UpdatePort(ctx, netgear.POEPortUpdate{
				PortID: testPort,
				Mode:   &invalidMode,
			})
			if err == nil {
				t.Error("Expected error for invalid POE mode")
			}

			// Step 2: Attempt to set power limit of 1000W (excessive)
			invalidPowerLimits := fixtures.InvalidPowerLimits()
			for _, invalidLimit := range invalidPowerLimits {
				limitType := netgear.POELimitTypeUser
				err = client.POE().UpdatePort(ctx, netgear.POEPortUpdate{
					PortID:         testPort,
					PowerLimitType: &limitType,
					PowerLimitW:    &invalidLimit,
				})
				if err == nil {
					t.Errorf("Expected error for invalid power limit %.1fW", invalidLimit)
				}
			}

			// Step 3: Attempt to set negative/invalid rate limits
			invalidRateLimits := fixtures.InvalidRateLimits()
			for _, invalidLimit := range invalidRateLimits {
				err = client.Ports().SetPortLimits(ctx, testPort, invalidLimit, "unlimited")
				if err == nil {
					t.Errorf("Expected error for invalid ingress rate limit: %s", invalidLimit)
				}

				err = client.Ports().SetPortLimits(ctx, testPort, "unlimited", invalidLimit)
				if err == nil {
					t.Errorf("Expected error for invalid egress rate limit: %s", invalidLimit)
				}
			}

			// Test invalid port speed
			invalidSpeed := netgear.PortSpeed("invalid-speed-999")
			err = client.Ports().SetPortSpeed(ctx, testPort, invalidSpeed)
			if err == nil {
				t.Error("Expected error for invalid port speed")
			}

			// Test invalid port names
			invalidNames := fixtures.InvalidPortNames()
			for _, invalidName := range invalidNames {
				err = client.Ports().SetPortName(ctx, testPort, invalidName)
				if err == nil {
					// Some invalid names might be allowed, just log a warning
					t.Logf("Warning: Invalid port name %q was accepted", invalidName)
				}
			}

			// Step 5: Verify current settings remain unchanged after all invalid operations
			finalPOESettings, err := client.POE().GetSettings(ctx)
			if err != nil {
				t.Fatalf("Failed to get final POE settings: %v", err)
			}

			finalPortSettings, err := client.Ports().GetSettings(ctx)
			if err != nil {
				t.Fatalf("Failed to get final port settings: %v", err)
			}

			// Compare settings for the test port
			var initialPOE, finalPOE *netgear.POEPortSettings
			var initialPort, finalPort *netgear.PortSettings

			for _, setting := range initialPOESettings {
				if setting.PortID == testPort {
					initialPOE = &setting
					break
				}
			}

			for _, setting := range finalPOESettings {
				if setting.PortID == testPort {
					finalPOE = &setting
					break
				}
			}

			for _, setting := range initialPortSettings {
				if setting.PortID == testPort {
					initialPort = &setting
					break
				}
			}

			for _, setting := range finalPortSettings {
				if setting.PortID == testPort {
					finalPort = &setting
					break
				}
			}

			if initialPOE != nil && finalPOE != nil {
				if initialPOE.Mode != finalPOE.Mode {
					t.Errorf("POE mode changed after invalid operations: %v -> %v", initialPOE.Mode, finalPOE.Mode)
				}
				if abs(initialPOE.PowerLimitW - finalPOE.PowerLimitW) > 0.1 {
					t.Errorf("POE power limit changed after invalid operations: %.1f -> %.1f", initialPOE.PowerLimitW, finalPOE.PowerLimitW)
				}
			}

			if initialPort != nil && finalPort != nil {
				if initialPort.Speed != finalPort.Speed {
					t.Errorf("Port speed changed after invalid operations: %v -> %v", initialPort.Speed, finalPort.Speed)
				}
				if initialPort.IngressLimit != finalPort.IngressLimit {
					t.Errorf("Ingress limit changed after invalid operations: %s -> %s", initialPort.IngressLimit, finalPort.IngressLimit)
				}
				if initialPort.EgressLimit != finalPort.EgressLimit {
					t.Errorf("Egress limit changed after invalid operations: %s -> %s", initialPort.EgressLimit, finalPort.EgressLimit)
				}
			}
		})
	}
}

// TestNetworkTimeout implements timeout testing (part of Test 6.2: Network Disconnection)
func TestNetworkTimeout(t *testing.T) {
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

	t.Logf("Network timeout test passed: failed after %v with error: %v", duration, err)
}

// TestConcurrentOperations tests that the client can handle multiple operations safely
func TestConcurrentOperations(t *testing.T) {
	// Require authentication - fail loudly if not available
	env := DetectTestEnvironment(t)
	env.RequireAuth(t, CategoryBasic)

	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	helper := NewTestHelper(config)

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			if switchConfig.ShouldSkipTest("concurrent_operations") {
				t.Skip("Test excluded for this switch")
			}

			client, err := helper.GetClientForTest(switchConfig.Name)
			if err != nil {
				t.Fatalf("Failed to get authenticated client: %v", err)
			}

			ctx := context.Background()

			// Test concurrent read operations
			const numConcurrent = 3
			results := make(chan error, numConcurrent)

			for i := 0; i < numConcurrent; i++ {
				go func() {
					// Perform various read operations concurrently
					_, err1 := client.POE().GetStatus(ctx)
					_, err2 := client.POE().GetSettings(ctx)
					_, err3 := client.Ports().GetSettings(ctx)

					if err1 != nil {
						results <- fmt.Errorf("POE status: %w", err1)
						return
					}
					if err2 != nil {
						results <- fmt.Errorf("POE settings: %w", err2)
						return
					}
					if err3 != nil {
						results <- fmt.Errorf("Port settings: %w", err3)
						return
					}
					results <- nil
				}()
			}

			// Wait for all operations to complete
			for i := 0; i < numConcurrent; i++ {
				select {
				case err := <-results:
					if err != nil {
						t.Errorf("Concurrent operation failed: %v", err)
					}
				case <-time.After(30 * time.Second):
					t.Error("Concurrent operation timed out")
				}
			}
		})
	}
}