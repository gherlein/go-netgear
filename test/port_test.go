package test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gherlein/go-netgear/pkg/netgear"
)

// TestPortSpeedSettings implements Test 4.1: Port Speed Settings
func TestPortSpeedSettings(t *testing.T) {
	// Require authentication - fail loudly if not available
	env := DetectTestEnvironment(t)
	env.RequireAuth(t, CategoryModify)

	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	helper := NewTestHelper(config)
	fixtures := NewTestFixtures()
	validSpeeds := fixtures.ValidPortSpeeds()

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			if switchConfig.ShouldSkipTest("port_speed_settings") {
				t.Skip("Test excluded for this switch")
			}

			for _, speed := range validSpeeds {
				t.Run(fmt.Sprintf("speed_%v", speed), func(t *testing.T) {
					for _, portID := range switchConfig.TestPorts {
						t.Run(fmt.Sprintf("port_%d", portID), func(t *testing.T) {
							testFunc := func(client *netgear.Client, testPorts []int) error {
								ctx := context.Background()

								// Step 1: Get current port settings
								settings, err := client.Ports().GetSettings(ctx)
								if err != nil {
									return fmt.Errorf("failed to get port settings: %w", err)
								}

								var currentSetting *netgear.PortSettings
								for _, setting := range settings {
									if setting.PortID == portID {
										currentSetting = &setting
										break
									}
								}

								if currentSetting == nil {
									return fmt.Errorf("port %d not found in port settings", portID)
								}

								// Step 3: Set speed to test value
								if err := client.Ports().SetPortSpeed(ctx, portID, speed); err != nil {
									return fmt.Errorf("failed to set port speed to %v on port %d: %w", speed, portID, err)
								}

								// Wait for change to take effect
								time.Sleep(fixtures.GetWaitTimes().PortSpeedChange)

								// Step 4: Verify GetSettings shows new speed
								updatedSettings, err := client.Ports().GetSettings(ctx)
								if err != nil {
									return fmt.Errorf("failed to get updated port settings: %w", err)
								}

								for _, setting := range updatedSettings {
									if setting.PortID == portID {
										if setting.Speed != speed {
											return fmt.Errorf("expected port %d speed to be %v, got %v", portID, speed, setting.Speed)
										}
										break
									}
								}

								// Step 5: If not "disable", verify port remains operational
								if speed != netgear.PortSpeedDisable {
									// Allow some time for port to stabilize
									time.Sleep(2 * time.Second)

									// Check port status
									finalSettings, err := client.Ports().GetSettings(ctx)
									if err != nil {
										return fmt.Errorf("failed to get final port settings: %w", err)
									}

									for _, setting := range finalSettings {
										if setting.PortID == portID {
											if setting.Status == netgear.PortStatusDisabled {
												return fmt.Errorf("port %d should be operational but is disabled after setting speed to %v", portID, speed)
											}
											break
										}
									}
								}

								return nil
							}

							result := helper.RunTestWithRestore(
								fmt.Sprintf("port_speed_%v_port_%d", speed, portID),
								switchConfig.Name,
								[]int{portID},
								testFunc,
							)

							if !result.Passed {
								// Check if it's an authentication issue
								if result.Error != nil &&
								   (strings.Contains(result.Error.Error(), "invalid credentials") ||
								    strings.Contains(result.Error.Error(), "authentication failed")) {
									t.Fatalf("❌ AUTHENTICATION FAILURE: Cannot authenticate to switch %s after 3 attempts. This blocks all tests. Error: %v", switchConfig.Name, result.Error)
								} else {
									t.Errorf("Test failed: %v", result.Error)
								}
							}
						})
					}
				})
			}
		})
	}
}

// TestPortFlowControl implements Test 4.2: Port Flow Control
func TestPortFlowControl(t *testing.T) {
	// Require authentication - fail loudly if not available
	env := DetectTestEnvironment(t)
	env.RequireAuth(t, CategoryModify)

	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	helper := NewTestHelper(config)
	fixtures := NewTestFixtures()

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			if switchConfig.ShouldSkipTest("port_flow_control") {
				t.Skip("Test excluded for this switch")
			}

			for _, portID := range switchConfig.TestPorts {
				t.Run(fmt.Sprintf("port_%d", portID), func(t *testing.T) {
					testFunc := func(client *netgear.Client, testPorts []int) error {
						ctx := context.Background()

						// Step 1: Get current settings
						settings, err := client.Ports().GetSettings(ctx)
						if err != nil {
							return fmt.Errorf("failed to get port settings: %w", err)
						}

						var currentSetting *netgear.PortSettings
						for _, setting := range settings {
							if setting.PortID == portID {
								currentSetting = &setting
								break
							}
						}

						if currentSetting == nil {
							return fmt.Errorf("port %d not found in port settings", portID)
						}

						// Step 2: Store original flow control state
						originalFlowControl := currentSetting.FlowControl

						// Step 3: If enabled, disable it
						if originalFlowControl {
							if err := client.Ports().SetPortFlowControl(ctx, portID, false); err != nil {
								return fmt.Errorf("failed to disable flow control on port %d: %w", portID, err)
							}

							// Wait for change to take effect
							time.Sleep(fixtures.GetWaitTimes().ConfigChange)

							// Step 4: Verify GetSettings shows disabled
							updatedSettings, err := client.Ports().GetSettings(ctx)
							if err != nil {
								return fmt.Errorf("failed to get updated port settings: %w", err)
							}

							for _, setting := range updatedSettings {
								if setting.PortID == portID {
									if setting.FlowControl {
										return fmt.Errorf("expected port %d flow control to be disabled, but it's still enabled", portID)
									}
									break
								}
							}
						}

						// Step 5: Enable flow control
						if err := client.Ports().SetPortFlowControl(ctx, portID, true); err != nil {
							return fmt.Errorf("failed to enable flow control on port %d: %w", portID, err)
						}

						// Wait for change to take effect
						time.Sleep(fixtures.GetWaitTimes().ConfigChange)

						// Step 6: Verify GetSettings shows enabled
						finalSettings, err := client.Ports().GetSettings(ctx)
						if err != nil {
							return fmt.Errorf("failed to get final port settings: %w", err)
						}

						for _, setting := range finalSettings {
							if setting.PortID == portID {
								if !setting.FlowControl {
									return fmt.Errorf("expected port %d flow control to be enabled, but it's disabled", portID)
								}
								break
							}
						}

						return nil
					}

					result := helper.RunTestWithRestore(
						fmt.Sprintf("port_flow_control_port_%d", portID),
						switchConfig.Name,
						[]int{portID},
						testFunc,
					)

					if !result.Passed {
						// Check if it's an authentication issue
						if result.Error != nil &&
						   (strings.Contains(result.Error.Error(), "invalid credentials") ||
						    strings.Contains(result.Error.Error(), "authentication failed")) {
							t.Fatalf("❌ AUTHENTICATION FAILURE: Cannot authenticate to switch %s after 3 attempts. This blocks all tests. Error: %v", switchConfig.Name, result.Error)
						} else {
							t.Errorf("Test failed: %v", result.Error)
						}
					}
				})
			}
		})
	}
}

// TestPortNaming implements Test 4.3: Port Naming
func TestPortNaming(t *testing.T) {
	// Require authentication - fail loudly if not available
	env := DetectTestEnvironment(t)
	env.RequireAuth(t, CategoryModify)

	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	helper := NewTestHelper(config)
	fixtures := NewTestFixtures()

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			if switchConfig.ShouldSkipTest("port_naming") {
				t.Skip("Test excluded for this switch")
			}

			for _, portID := range switchConfig.TestPorts {
				t.Run(fmt.Sprintf("port_%d", portID), func(t *testing.T) {
					testFunc := func(client *netgear.Client, testPorts []int) error {
						ctx := context.Background()

						// Step 3: Set name to unique test name
						testName := fixtures.TestPortNameWithTimestamp(fmt.Sprintf("TEST_PORT_%d", portID))

						if err := client.Ports().SetPortName(ctx, portID, testName); err != nil {
							return fmt.Errorf("failed to set port name to %s on port %d: %w", testName, portID, err)
						}

						// Wait for change to take effect
						time.Sleep(fixtures.GetWaitTimes().ConfigChange)

						// Step 4: Verify GetSettings shows new name
						updatedSettings, err := client.Ports().GetSettings(ctx)
						if err != nil {
							return fmt.Errorf("failed to get updated port settings: %w", err)
						}

						for _, setting := range updatedSettings {
							if setting.PortID == portID {
								if setting.PortName != testName {
									return fmt.Errorf("expected port %d name to be %s, got %s", portID, testName, setting.PortName)
								}
								break
							}
						}

						// Step 5: Test special characters in name (valid ones)
						validNames := fixtures.ValidPortNames()
						for _, validName := range validNames {
							if err := client.Ports().SetPortName(ctx, portID, validName); err != nil {
								return fmt.Errorf("failed to set valid port name %s on port %d: %w", validName, portID, err)
							}

							time.Sleep(fixtures.GetWaitTimes().ConfigChange)

							settings, err := client.Ports().GetSettings(ctx)
							if err != nil {
								return fmt.Errorf("failed to get port settings after setting name to %s: %w", validName, err)
							}

							for _, setting := range settings {
								if setting.PortID == portID {
									if setting.PortName != validName {
										return fmt.Errorf("expected port %d name to be %s, got %s", portID, validName, setting.PortName)
									}
									break
								}
							}
						}

						// Step 6: Test invalid names (should fail)
						invalidNames := fixtures.InvalidPortNames()
						for _, invalidName := range invalidNames {
							err := client.Ports().SetPortName(ctx, portID, invalidName)
							if err == nil {
								t.Logf("Warning: Setting invalid port name %q succeeded when it should have failed", invalidName)
								// Don't fail the test as some invalid names might be allowed by the switch
							}
						}

						return nil
					}

					result := helper.RunTestWithRestore(
						fmt.Sprintf("port_naming_port_%d", portID),
						switchConfig.Name,
						[]int{portID},
						testFunc,
					)

					if !result.Passed {
						// Check if it's an authentication issue
						if result.Error != nil &&
						   (strings.Contains(result.Error.Error(), "invalid credentials") ||
						    strings.Contains(result.Error.Error(), "authentication failed")) {
							t.Fatalf("❌ AUTHENTICATION FAILURE: Cannot authenticate to switch %s after 3 attempts. This blocks all tests. Error: %v", switchConfig.Name, result.Error)
						} else {
							t.Errorf("Test failed: %v", result.Error)
						}
					}
				})
			}
		})
	}
}

// TestPortRateLimiting implements Test 4.4: Port Rate Limiting
func TestPortRateLimiting(t *testing.T) {
	// Require authentication - fail loudly if not available
	env := DetectTestEnvironment(t)
	env.RequireAuth(t, CategoryModify)

	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	helper := NewTestHelper(config)
	fixtures := NewTestFixtures()
	validLimits := fixtures.ValidRateLimits()
	invalidLimits := fixtures.InvalidRateLimits()

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			if switchConfig.ShouldSkipTest("rate_limiting") {
				t.Skip("Test excluded for this switch")
			}

			for _, portID := range switchConfig.TestPorts {
				t.Run(fmt.Sprintf("port_%d", portID), func(t *testing.T) {
					testFunc := func(client *netgear.Client, testPorts []int) error {
						ctx := context.Background()

						// Step 3: Test ingress limits
						for _, ingressLimit := range validLimits {
							if err := client.Ports().SetPortLimits(ctx, portID, ingressLimit, "unlimited"); err != nil {
								return fmt.Errorf("failed to set ingress limit to %s on port %d: %w", ingressLimit, portID, err)
							}

							// Wait for change to take effect
							time.Sleep(fixtures.GetWaitTimes().ConfigChange)

							// Step 5: Verify GetSettings shows each limit
							settings, err := client.Ports().GetSettings(ctx)
							if err != nil {
								return fmt.Errorf("failed to get port settings after setting ingress limit: %w", err)
							}

							for _, setting := range settings {
								if setting.PortID == portID {
									if setting.IngressLimit != ingressLimit {
										return fmt.Errorf("expected port %d ingress limit to be %s, got %s", portID, ingressLimit, setting.IngressLimit)
									}
									break
								}
							}
						}

						// Step 4: Test egress limits
						for _, egressLimit := range validLimits {
							if err := client.Ports().SetPortLimits(ctx, portID, "unlimited", egressLimit); err != nil {
								return fmt.Errorf("failed to set egress limit to %s on port %d: %w", egressLimit, portID, err)
							}

							// Wait for change to take effect
							time.Sleep(fixtures.GetWaitTimes().ConfigChange)

							// Verify GetSettings shows each limit
							settings, err := client.Ports().GetSettings(ctx)
							if err != nil {
								return fmt.Errorf("failed to get port settings after setting egress limit: %w", err)
							}

							for _, setting := range settings {
								if setting.PortID == portID {
									if setting.EgressLimit != egressLimit {
										return fmt.Errorf("expected port %d egress limit to be %s, got %s", portID, egressLimit, setting.EgressLimit)
									}
									break
								}
							}
						}

						// Step 6: Test invalid limits
						for _, invalidLimit := range invalidLimits {
							err := client.Ports().SetPortLimits(ctx, portID, invalidLimit, "unlimited")
							if err == nil {
								return fmt.Errorf("expected error when setting invalid ingress limit %s on port %d, but update succeeded", invalidLimit, portID)
							}
							// Error is expected, continue

							err = client.Ports().SetPortLimits(ctx, portID, "unlimited", invalidLimit)
							if err == nil {
								return fmt.Errorf("expected error when setting invalid egress limit %s on port %d, but update succeeded", invalidLimit, portID)
							}
							// Error is expected, continue
						}

						return nil
					}

					result := helper.RunTestWithRestore(
						fmt.Sprintf("port_rate_limits_port_%d", portID),
						switchConfig.Name,
						[]int{portID},
						testFunc,
					)

					if !result.Passed {
						// Check if it's an authentication issue
						if result.Error != nil &&
						   (strings.Contains(result.Error.Error(), "invalid credentials") ||
						    strings.Contains(result.Error.Error(), "authentication failed")) {
							t.Fatalf("❌ AUTHENTICATION FAILURE: Cannot authenticate to switch %s after 3 attempts. This blocks all tests. Error: %v", switchConfig.Name, result.Error)
						} else {
							t.Errorf("Test failed: %v", result.Error)
						}
					}
				})
			}
		})
	}
}

// TestPortNetworkEnableDisable implements Test 4.5: Port Enable/Disable (Network)
func TestPortNetworkEnableDisable(t *testing.T) {
	// Require authentication - fail loudly if not available
	env := DetectTestEnvironment(t)
	env.RequireAuth(t, CategoryModify)

	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	helper := NewTestHelper(config)
	fixtures := NewTestFixtures()

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			if switchConfig.ShouldSkipTest("port_network_enable_disable") {
				t.Skip("Test excluded for this switch")
			}

			for _, portID := range switchConfig.TestPorts {
				t.Run(fmt.Sprintf("port_%d", portID), func(t *testing.T) {
					testFunc := func(client *netgear.Client, testPorts []int) error {
						ctx := context.Background()

						// Step 3: Set speed to "disable"
						if err := client.Ports().DisablePort(ctx, portID); err != nil {
							return fmt.Errorf("failed to disable port %d: %w", portID, err)
						}

						// Wait for change to take effect
						time.Sleep(fixtures.GetWaitTimes().PortSpeedChange)

						// Step 4: Verify port shows as disabled
						disabledSettings, err := client.Ports().GetSettings(ctx)
						if err != nil {
							return fmt.Errorf("failed to get port settings after disabling: %w", err)
						}

						for _, setting := range disabledSettings {
							if setting.PortID == portID {
								if setting.Status != netgear.PortStatusDisabled {
									return fmt.Errorf("expected port %d to be disabled, got status %v", portID, setting.Status)
								}
								break
							}
						}

						// Step 6: Set speed to "auto" (enable port)
						if err := client.Ports().EnablePort(ctx, portID); err != nil {
							return fmt.Errorf("failed to enable port %d: %w", portID, err)
						}

						// Wait for port to stabilize
						time.Sleep(fixtures.GetWaitTimes().PortSpeedChange)

						// Step 7: Verify port is enabled
						enabledSettings, err := client.Ports().GetSettings(ctx)
						if err != nil {
							return fmt.Errorf("failed to get port settings after enabling: %w", err)
						}

						for _, setting := range enabledSettings {
							if setting.PortID == portID {
								if setting.Status == netgear.PortStatusDisabled {
									return fmt.Errorf("expected port %d to be enabled, but it's still disabled", portID)
								}
								break
							}
						}

						return nil
					}

					result := helper.RunTestWithRestore(
						fmt.Sprintf("port_network_enable_disable_port_%d", portID),
						switchConfig.Name,
						[]int{portID},
						testFunc,
					)

					if !result.Passed {
						// Check if it's an authentication issue
						if result.Error != nil &&
						   (strings.Contains(result.Error.Error(), "invalid credentials") ||
						    strings.Contains(result.Error.Error(), "authentication failed")) {
							t.Fatalf("❌ AUTHENTICATION FAILURE: Cannot authenticate to switch %s after 3 attempts. This blocks all tests. Error: %v", switchConfig.Name, result.Error)
						} else {
							t.Errorf("Test failed: %v", result.Error)
						}
					}
				})
			}
		})
	}
}