package test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gherlein/go-netgear/pkg/netgear"
)

// TestPOEEnableDisable implements Test 3.1: POE Enable/Disable
func TestPOEEnableDisable(t *testing.T) {
	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	helper := NewTestHelper(config)
	fixtures := NewTestFixtures()

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			if switchConfig.ShouldSkipTest("poe_enable_disable") {
				t.Skip("Test excluded for this switch")
			}

			for _, portID := range switchConfig.TestPorts {
				t.Run(fmt.Sprintf("port_%d", portID), func(t *testing.T) {
					testFunc := func(client *netgear.Client, testPorts []int) error {
						ctx := context.Background()

						// Step 1: Get current POE settings
						settings, err := client.POE().GetSettings(ctx)
						if err != nil {
							return fmt.Errorf("failed to get POE settings: %w", err)
						}

						var currentSetting *netgear.POEPortSettings
						for _, setting := range settings {
							if setting.PortID == portID {
								currentSetting = &setting
								break
							}
						}

						if currentSetting == nil {
							return fmt.Errorf("port %d not found in POE settings", portID)
						}

						// Step 2: Store original enabled state
						originalEnabled := currentSetting.Enabled

						// Step 3: If enabled, disable it
						if originalEnabled {
							disabled := false
							update := netgear.POEPortUpdate{
								PortID:  portID,
								Enabled: &disabled,
							}

							if err := client.POE().UpdatePort(ctx, update); err != nil {
								return fmt.Errorf("failed to disable POE on port %d: %w", portID, err)
							}

							// Wait for change to take effect
							time.Sleep(fixtures.GetWaitTimes().ConfigChange)

							// Step 4: Verify GetSettings shows disabled
							updatedSettings, err := client.POE().GetSettings(ctx)
							if err != nil {
								return fmt.Errorf("failed to get updated POE settings: %w", err)
							}

							for _, setting := range updatedSettings {
								if setting.PortID == portID {
									if setting.Enabled {
										return fmt.Errorf("expected port %d to be disabled, but it's still enabled", portID)
									}
									break
								}
							}
						}

						// Step 5: Enable the port
						enabled := true
						update := netgear.POEPortUpdate{
							PortID:  portID,
							Enabled: &enabled,
						}

						if err := client.POE().UpdatePort(ctx, update); err != nil {
							return fmt.Errorf("failed to enable POE on port %d: %w", portID, err)
						}

						// Wait for change to take effect
						time.Sleep(fixtures.GetWaitTimes().ConfigChange)

						// Step 6: Verify GetSettings shows enabled
						finalSettings, err := client.POE().GetSettings(ctx)
						if err != nil {
							return fmt.Errorf("failed to get final POE settings: %w", err)
						}

						for _, setting := range finalSettings {
							if setting.PortID == portID {
								if !setting.Enabled {
									return fmt.Errorf("expected port %d to be enabled, but it's disabled", portID)
								}
								break
							}
						}

						return nil
					}

					result := helper.RunTestWithRestore(
						fmt.Sprintf("poe_enable_disable_port_%d", portID),
						switchConfig.Name,
						[]int{portID},
						testFunc,
					)

					if !result.Passed {
						t.Errorf("Test failed: %v", result.Error)
					}
				})
			}
		})
	}
}

// TestPOEPowerModes implements Test 3.2: POE Power Modes
func TestPOEPowerModes(t *testing.T) {
	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	helper := NewTestHelper(config)
	fixtures := NewTestFixtures()

	for _, switchConfig := range config.Switches {
		// Get compatible modes for this switch model
		compatibleModes := fixtures.GetCompatiblePOEModes(switchConfig.Model)

		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			if switchConfig.ShouldSkipTest("poe_power_modes") {
				t.Skip("Test excluded for this switch")
			}

			for _, mode := range compatibleModes {
				t.Run(fmt.Sprintf("mode_%v", mode), func(t *testing.T) {
					for _, portID := range switchConfig.TestPorts {
						t.Run(fmt.Sprintf("port_%d", portID), func(t *testing.T) {
							testFunc := func(client *netgear.Client, testPorts []int) error {
								ctx := context.Background()

								// Step 1: Get current settings
								settings, err := client.POE().GetSettings(ctx)
								if err != nil {
									return fmt.Errorf("failed to get POE settings: %w", err)
								}

								var currentSetting *netgear.POEPortSettings
								for _, setting := range settings {
									if setting.PortID == portID {
										currentSetting = &setting
										break
									}
								}

								if currentSetting == nil {
									return fmt.Errorf("port %d not found in POE settings", portID)
								}

								// Step 3: Set mode to test value
								update := netgear.POEPortUpdate{
									PortID: portID,
									Mode:   &mode,
								}

								if err := client.POE().UpdatePort(ctx, update); err != nil {
									return fmt.Errorf("failed to set POE mode to %v on port %d: %w", mode, portID, err)
								}

								// Wait for change to take effect
								time.Sleep(fixtures.GetWaitTimes().ConfigChange)

								// Step 4: Verify GetSettings shows new mode
								updatedSettings, err := client.POE().GetSettings(ctx)
								if err != nil {
									return fmt.Errorf("failed to get updated POE settings: %w", err)
								}

								for _, setting := range updatedSettings {
									if setting.PortID == portID {
										if setting.Mode != mode {
											return fmt.Errorf("expected port %d mode to be %v, got %v", portID, mode, setting.Mode)
										}
										break
									}
								}

								return nil
							}

							result := helper.RunTestWithRestore(
								fmt.Sprintf("poe_mode_%v_port_%d", mode, portID),
								switchConfig.Name,
								[]int{portID},
								testFunc,
							)

							if !result.Passed {
								t.Errorf("Test failed: %v", result.Error)
							}
						})
					}
				})
			}
		})
	}
}

// TestPOEPriorityLevels implements Test 3.3: POE Priority Levels
func TestPOEPriorityLevels(t *testing.T) {
	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	helper := NewTestHelper(config)
	fixtures := NewTestFixtures()
	validPriorities := fixtures.ValidPOEPriorities()

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			if switchConfig.ShouldSkipTest("poe_priority_levels") {
				t.Skip("Test excluded for this switch")
			}

			for _, priority := range validPriorities {
				t.Run(fmt.Sprintf("priority_%v", priority), func(t *testing.T) {
					for _, portID := range switchConfig.TestPorts {
						t.Run(fmt.Sprintf("port_%d", portID), func(t *testing.T) {
							testFunc := func(client *netgear.Client, testPorts []int) error {
								ctx := context.Background()

								// Step 1: Get current settings
								settings, err := client.POE().GetSettings(ctx)
								if err != nil {
									return fmt.Errorf("failed to get POE settings: %w", err)
								}

								var currentSetting *netgear.POEPortSettings
								for _, setting := range settings {
									if setting.PortID == portID {
										currentSetting = &setting
										break
									}
								}

								if currentSetting == nil {
									return fmt.Errorf("port %d not found in POE settings", portID)
								}

								// Step 3: Set priority to test value
								update := netgear.POEPortUpdate{
									PortID:   portID,
									Priority: &priority,
								}

								if err := client.POE().UpdatePort(ctx, update); err != nil {
									return fmt.Errorf("failed to set POE priority to %v on port %d: %w", priority, portID, err)
								}

								// Wait for change to take effect
								time.Sleep(fixtures.GetWaitTimes().ConfigChange)

								// Step 4: Verify GetSettings shows new priority
								updatedSettings, err := client.POE().GetSettings(ctx)
								if err != nil {
									return fmt.Errorf("failed to get updated POE settings: %w", err)
								}

								for _, setting := range updatedSettings {
									if setting.PortID == portID {
										if setting.Priority != priority {
											return fmt.Errorf("expected port %d priority to be %v, got %v", portID, priority, setting.Priority)
										}
										break
									}
								}

								return nil
							}

							result := helper.RunTestWithRestore(
								fmt.Sprintf("poe_priority_%v_port_%d", priority, portID),
								switchConfig.Name,
								[]int{portID},
								testFunc,
							)

							if !result.Passed {
								t.Errorf("Test failed: %v", result.Error)
							}
						})
					}
				})
			}
		})
	}
}

// TestPOEPowerLimits implements Test 3.4: POE Power Limits
func TestPOEPowerLimits(t *testing.T) {
	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	helper := NewTestHelper(config)
	fixtures := NewTestFixtures()
	validLimitTypes := fixtures.ValidPOELimitTypes()
	validPowerLimits := fixtures.ValidPowerLimits()
	invalidPowerLimits := fixtures.InvalidPowerLimits()

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			if switchConfig.ShouldSkipTest("poe_power_limits") {
				t.Skip("Test excluded for this switch")
			}

			for _, limitType := range validLimitTypes {
				t.Run(fmt.Sprintf("limit_type_%v", limitType), func(t *testing.T) {
					for _, portID := range switchConfig.TestPorts {
						t.Run(fmt.Sprintf("port_%d", portID), func(t *testing.T) {
							testFunc := func(client *netgear.Client, testPorts []int) error {
								ctx := context.Background()

								// Step 3: Set limit type
								update := netgear.POEPortUpdate{
									PortID:         portID,
									PowerLimitType: &limitType,
								}

								if err := client.POE().UpdatePort(ctx, update); err != nil {
									return fmt.Errorf("failed to set POE limit type to %v on port %d: %w", limitType, portID, err)
								}

								// Wait for change to take effect
								time.Sleep(fixtures.GetWaitTimes().ConfigChange)

								// Step 4: If user type, test various wattage values
								if limitType == netgear.POELimitTypeUser {
									for _, powerLimit := range validPowerLimits {
										// Test valid power limits
										powerUpdate := netgear.POEPortUpdate{
											PortID:         portID,
											PowerLimitType: &limitType,
											PowerLimitW:    &powerLimit,
										}

										if err := client.POE().UpdatePort(ctx, powerUpdate); err != nil {
											return fmt.Errorf("failed to set POE power limit to %.1fW on port %d: %w", powerLimit, portID, err)
										}

										time.Sleep(fixtures.GetWaitTimes().ConfigChange)

										// Verify the setting was applied
										settings, err := client.POE().GetSettings(ctx)
										if err != nil {
											return fmt.Errorf("failed to get POE settings: %w", err)
										}

										for _, setting := range settings {
											if setting.PortID == portID {
												if abs(setting.PowerLimitW - powerLimit) > 0.1 {
													return fmt.Errorf("expected port %d power limit to be %.1fW, got %.1fW", portID, powerLimit, setting.PowerLimitW)
												}
												break
											}
										}
									}

									// Step 6: Test invalid limits
									for _, invalidLimit := range invalidPowerLimits {
										invalidUpdate := netgear.POEPortUpdate{
											PortID:         portID,
											PowerLimitType: &limitType,
											PowerLimitW:    &invalidLimit,
										}

										err := client.POE().UpdatePort(ctx, invalidUpdate)
										if err == nil {
											return fmt.Errorf("expected error when setting invalid power limit %.1fW on port %d, but update succeeded", invalidLimit, portID)
										}
										// Error is expected, continue
									}
								}

								// Step 5: Verify GetSettings shows new limit type
								finalSettings, err := client.POE().GetSettings(ctx)
								if err != nil {
									return fmt.Errorf("failed to get final POE settings: %w", err)
								}

								for _, setting := range finalSettings {
									if setting.PortID == portID {
										if setting.PowerLimitType != limitType {
											return fmt.Errorf("expected port %d limit type to be %v, got %v", portID, limitType, setting.PowerLimitType)
										}
										break
									}
								}

								return nil
							}

							result := helper.RunTestWithRestore(
								fmt.Sprintf("poe_power_limit_%v_port_%d", limitType, portID),
								switchConfig.Name,
								[]int{portID},
								testFunc,
							)

							if !result.Passed {
								t.Errorf("Test failed: %v", result.Error)
							}
						})
					}
				})
			}
		})
	}
}

// TestPOEPowerCycling implements Test 3.5: POE Power Cycling
func TestPOEPowerCycling(t *testing.T) {
	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	helper := NewTestHelper(config)
	fixtures := NewTestFixtures()

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			if switchConfig.ShouldSkipTest("poe_power_cycling") {
				t.Skip("Test excluded for this switch")
			}

			for _, portID := range switchConfig.TestPorts {
				t.Run(fmt.Sprintf("port_%d", portID), func(t *testing.T) {
					testFunc := func(client *netgear.Client, testPorts []int) error {
						ctx := context.Background()

						// Step 1: Get current POE status
						status, err := client.POE().GetPortStatus(ctx, portID)
						if err != nil {
							return fmt.Errorf("failed to get POE status for port %d: %w", portID, err)
						}

						// Step 2: Note power consumption
						originalPowerW := status.PowerW

						// Skip if port is not consuming power (no device connected)
						if originalPowerW <= 0.1 {
							t.Skipf("Port %d has no power consumption (%.2fW), skipping power cycle test", portID, originalPowerW)
							return nil
						}

						// Step 3: Cycle power on port
						if err := client.POE().CyclePower(ctx, portID); err != nil {
							return fmt.Errorf("failed to cycle power on port %d: %w", portID, err)
						}

						// Step 4: Wait for port to come back up
						waitTime := fixtures.GetWaitTimes().POECycleRecovery
						t.Logf("Waiting %v for port %d to recover after power cycle", waitTime, portID)
						time.Sleep(waitTime)

						// Step 5: Verify port is powered again
						recoveredStatus, err := client.POE().GetPortStatus(ctx, portID)
						if err != nil {
							return fmt.Errorf("failed to get POE status after power cycle for port %d: %w", portID, err)
						}

						if recoveredStatus.Status == "disabled" || recoveredStatus.PowerW <= 0.1 {
							return fmt.Errorf("port %d failed to recover after power cycle: status=%s, power=%.2fW", portID, recoveredStatus.Status, recoveredStatus.PowerW)
						}

						// Step 6: Compare power consumption (should be similar)
						powerDifference := abs(originalPowerW - recoveredStatus.PowerW)
						maxAllowedDifference := originalPowerW * 0.2 // Allow 20% variation
						if powerDifference > maxAllowedDifference {
							t.Logf("Warning: Port %d power consumption changed significantly after power cycle: %.2fW -> %.2fW (%.1f%% change)",
								portID, originalPowerW, recoveredStatus.PowerW, (powerDifference/originalPowerW)*100)
						}

						return nil
					}

					result := helper.RunTestWithRestore(
						fmt.Sprintf("poe_power_cycle_port_%d", portID),
						switchConfig.Name,
						[]int{portID},
						testFunc,
					)

					if !result.Passed {
						t.Errorf("Test failed: %v", result.Error)
					}
				})
			}
		})
	}
}

// TestPOEDetectionTypes implements Test 3.6: POE Detection Types
func TestPOEDetectionTypes(t *testing.T) {
	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	helper := NewTestHelper(config)
	fixtures := NewTestFixtures()
	validDetectionTypes := fixtures.ValidDetectionTypes()

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			if switchConfig.ShouldSkipTest("poe_detection_types") {
				t.Skip("Test excluded for this switch")
			}

			for _, detectionType := range validDetectionTypes {
				t.Run(fmt.Sprintf("detection_%s", detectionType), func(t *testing.T) {
					for _, portID := range switchConfig.TestPorts {
						t.Run(fmt.Sprintf("port_%d", portID), func(t *testing.T) {
							testFunc := func(client *netgear.Client, testPorts []int) error {
								ctx := context.Background()

								// Step 3: Set detection type
								update := netgear.POEPortUpdate{
									PortID:        portID,
									DetectionType: &detectionType,
								}

								if err := client.POE().UpdatePort(ctx, update); err != nil {
									return fmt.Errorf("failed to set POE detection type to %s on port %d: %w", detectionType, portID, err)
								}

								// Wait for change to take effect
								time.Sleep(fixtures.GetWaitTimes().ConfigChange)

								// Step 4: Verify GetSettings shows new type
								settings, err := client.POE().GetSettings(ctx)
								if err != nil {
									return fmt.Errorf("failed to get POE settings: %w", err)
								}

								for _, setting := range settings {
									if setting.PortID == portID {
										if setting.DetectionType != detectionType {
											return fmt.Errorf("expected port %d detection type to be %s, got %s", portID, detectionType, setting.DetectionType)
										}
										break
									}
								}

								return nil
							}

							result := helper.RunTestWithRestore(
								fmt.Sprintf("poe_detection_type_%s_port_%d", detectionType, portID),
								switchConfig.Name,
								[]int{portID},
								testFunc,
							)

							if !result.Passed {
								t.Errorf("Test failed: %v", result.Error)
							}
						})
					}
				})
			}
		})
	}
}

