package test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/gherlein/go-netgear/pkg/netgear"
)

// TestPOEStatusReading implements Test 5.1: POE Status Reading
func TestPOEStatusReading(t *testing.T) {
	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	helper := NewTestHelper(config)

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			if switchConfig.ShouldSkipTest("poe_status_reading") {
				t.Skip("Test excluded for this switch")
			}

			client, err := helper.GetClientForTest(switchConfig.Name)
			if err != nil {
				t.Fatalf("Failed to get authenticated client: %v", err)
			}

			ctx := context.Background()

			// Step 1: Get POE status for all ports
			status, err := client.POE().GetStatus(ctx)
			if err != nil {
				t.Fatalf("Failed to get POE status: %v", err)
			}

			// Step 2: Verify data structure completeness
			if len(status) == 0 {
				t.Error("Expected POE status data, got empty array")
			}

			for _, portStatus := range status {
				// Check that required fields are populated
				if portStatus.PortID <= 0 {
					t.Errorf("Invalid PortID: %d", portStatus.PortID)
				}

				if portStatus.PortName == "" {
					t.Errorf("Empty PortName for port %d", portStatus.PortID)
				}

				if portStatus.Status == "" {
					t.Errorf("Empty Status for port %d", portStatus.PortID)
				}

				// Step 3: Verify voltage/current/power calculations are consistent
				// Power = Voltage * Current (approximately, considering efficiency)
				if portStatus.PowerW > 0 && portStatus.VoltageV > 0 && portStatus.CurrentMA > 0 {
					calculatedPowerW := (portStatus.VoltageV * portStatus.CurrentMA) / 1000.0
					powerDifference := abs(calculatedPowerW - portStatus.PowerW)
					maxAllowedDifference := calculatedPowerW * 0.15 // Allow 15% variation for efficiency

					if powerDifference > maxAllowedDifference {
						t.Logf("Warning: Power calculation inconsistency for port %d: calculated=%.2fW, reported=%.2fW",
							portStatus.PortID, calculatedPowerW, portStatus.PowerW)
					}
				}

				// Step 4: Verify temperature readings are reasonable (if available)
				if portStatus.TemperatureC != 0 {
					if portStatus.TemperatureC < -40 || portStatus.TemperatureC > 85 {
						t.Errorf("Unreasonable temperature for port %d: %.1f°C", portStatus.PortID, portStatus.TemperatureC)
					}
				}
			}

			// Step 5: Compare with GetSettings enabled state
			settings, err := client.POE().GetSettings(ctx)
			if err != nil {
				t.Fatalf("Failed to get POE settings: %v", err)
			}

			// Create maps for easy lookup
			statusMap := make(map[int]netgear.POEPortStatus)
			settingsMap := make(map[int]netgear.POEPortSettings)

			for _, s := range status {
				statusMap[s.PortID] = s
			}

			for _, s := range settings {
				settingsMap[s.PortID] = s
			}

			// Compare enabled state consistency
			for portID := range statusMap {
				statusData := statusMap[portID]
				settingsData, hasSettings := settingsMap[portID]

				if !hasSettings {
					t.Errorf("Port %d found in status but not in settings", portID)
					continue
				}

				// If POE is disabled in settings, status should reflect this
				if !settingsData.Enabled {
					if statusData.Status != "disabled" && statusData.PowerW > 0.1 {
						t.Errorf("Port %d is disabled in settings but shows active in status (power=%.2fW)",
							portID, statusData.PowerW)
					}
				}
			}
		})
	}
}

// TestPortStatusReading implements Test 5.2: Port Status Reading
func TestPortStatusReading(t *testing.T) {
	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	helper := NewTestHelper(config)
	fixtures := NewTestFixtures()

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			if switchConfig.ShouldSkipTest("port_status_reading") {
				t.Skip("Test excluded for this switch")
			}

			client, err := helper.GetClientForTest(switchConfig.Name)
			if err != nil {
				t.Fatalf("Failed to get authenticated client: %v", err)
			}

			ctx := context.Background()

			// Step 1: Get port settings for all ports
			settings, err := client.Ports().GetSettings(ctx)
			if err != nil {
				t.Fatalf("Failed to get port settings: %v", err)
			}

			// Step 2: Verify all expected ports are present
			expectedPortCount := fixtures.GetPortCount(switchConfig.Model)
			if len(settings) != expectedPortCount {
				t.Errorf("Expected %d ports for model %s, got %d", expectedPortCount, switchConfig.Model, len(settings))
			}

			expectedPorts := fixtures.GetValidPortNumbers(switchConfig.Model)
			foundPorts := make(map[int]bool)

			for _, setting := range settings {
				foundPorts[setting.PortID] = true

				// Check that required fields are populated
				if setting.PortID <= 0 {
					t.Errorf("Invalid PortID: %d", setting.PortID)
				}

				if setting.PortName == "" {
					t.Errorf("Empty PortName for port %d", setting.PortID)
				}

				// Step 4: Verify speed negotiations are valid
				validSpeeds := fixtures.ValidPortSpeeds()
				speedValid := false
				for _, validSpeed := range validSpeeds {
					if setting.Speed == validSpeed {
						speedValid = true
						break
					}
				}

				if !speedValid {
					t.Errorf("Invalid speed setting for port %d: %v", setting.PortID, setting.Speed)
				}

				// Verify status is one of the valid values
				validStatuses := []netgear.PortStatus{
					netgear.PortStatusAvailable,
					netgear.PortStatusConnected,
					netgear.PortStatusDisabled,
				}
				statusValid := false
				for _, validStatus := range validStatuses {
					if setting.Status == validStatus {
						statusValid = true
						break
					}
				}

				if !statusValid {
					t.Errorf("Invalid status for port %d: %v", setting.PortID, setting.Status)
				}

				// Verify rate limits are valid format
				if setting.IngressLimit != "" {
					if !isValidRateLimit(setting.IngressLimit) {
						t.Errorf("Invalid ingress limit format for port %d: %s", setting.PortID, setting.IngressLimit)
					}
				}

				if setting.EgressLimit != "" {
					if !isValidRateLimit(setting.EgressLimit) {
						t.Errorf("Invalid egress limit format for port %d: %s", setting.PortID, setting.EgressLimit)
					}
				}
			}

			for _, expectedPort := range expectedPorts {
				if !foundPorts[expectedPort] {
					t.Errorf("Expected port %d not found in port settings", expectedPort)
				}
			}

			// Step 5: Cross-reference with POE port data
			poeStatus, err := client.POE().GetStatus(ctx)
			if err != nil {
				t.Fatalf("Failed to get POE status: %v", err)
			}

			poeStatusMap := make(map[int]netgear.POEPortStatus)
			for _, status := range poeStatus {
				poeStatusMap[status.PortID] = status
			}

			for _, setting := range settings {
				if poeData, hasPOE := poeStatusMap[setting.PortID]; hasPOE {
					// Check that port names match
					if setting.PortName != poeData.PortName {
						t.Errorf("Port name mismatch for port %d: port=%s, poe=%s",
							setting.PortID, setting.PortName, poeData.PortName)
					}
				}
			}
		})
	}
}

// TestModelDetection implements Test 5.3: Model Detection
func TestModelDetection(t *testing.T) {
	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config file: %v", err)
	}

	for _, switchConfig := range config.Switches {
		t.Run(fmt.Sprintf("switch_%s", switchConfig.Name), func(t *testing.T) {
			if switchConfig.ShouldSkipTest("model_detection") {
				t.Skip("Test excluded for this switch")
			}

			// Step 1: Create client without model specification (model detection will happen automatically)
			client, err := netgear.NewClient(switchConfig.Address,
				netgear.WithVerbose(config.TestOptions.Verbose))
			if err != nil {
				t.Skipf("Skipping test - failed to create client: %v", err)
			}

			ctx := context.Background()
			if err := client.Login(ctx, switchConfig.Password); err != nil {
				t.Fatalf("Failed to login: %v", err)
			}

			// Step 2: Verify model is auto-detected correctly
			detectedModel := client.GetModel()
			if string(detectedModel) == "" {
				t.Error("Model was not detected")
			}

			// Step 3: Compare with known model from config
			expectedModel := netgear.Model(switchConfig.Model)
			if detectedModel != expectedModel {
				t.Errorf("Model detection mismatch: expected %s, got %s", expectedModel, detectedModel)
			}

			// Step 4: Verify model-specific endpoints are used
			// Try to get POE status to verify the correct endpoints are being used
			_, err = client.POE().GetStatus(ctx)
			if err != nil {
				// Check if this is a "not supported" error for unsupported models
				if strings.Contains(err.Error(), "not supported") {
					if expectedModel.IsSupported() {
						t.Errorf("POE operations should be supported for model %s but got error: %v", expectedModel, err)
					}
				} else {
					t.Errorf("Failed to get POE status (may indicate wrong endpoints): %v", err)
				}
			} else {
				// POE operations succeeded, verify this is expected
				if !expectedModel.IsSupported() {
					t.Errorf("POE operations succeeded for unsupported model %s", expectedModel)
				}
			}

			// Test port operations as well
			_, err = client.Ports().GetSettings(ctx)
			if err != nil {
				t.Errorf("Failed to get port settings: %v", err)
			}

			// Cleanup
			client.Logout(ctx)
		})
	}
}

// Helper function to validate rate limit format
func isValidRateLimit(limit string) bool {
	if limit == "unlimited" {
		return true
	}

	// Check for patterns like "1Mbps", "512kbps", etc.
	validSuffixes := []string{"kbps", "Mbps", "Gbps"}
	for _, suffix := range validSuffixes {
		if strings.HasSuffix(limit, suffix) {
			return true
		}
	}

	return false
}