package test

import (
	"testing"
	"time"

	"github.com/gherlein/go-netgear/pkg/netgear"
)

func TestNewTestFixtures(t *testing.T) {
	fixtures := NewTestFixtures()
	if fixtures == nil {
		t.Fatal("Expected non-nil fixtures")
	}
}

func TestValidPOEModes(t *testing.T) {
	fixtures := NewTestFixtures()
	modes := fixtures.ValidPOEModes()

	expectedModes := []netgear.POEMode{
		netgear.POEMode8023af,
		netgear.POEMode8023at,
		netgear.POEModeLegacy,
		netgear.POEModePre8023at,
	}

	if len(modes) != len(expectedModes) {
		t.Errorf("Expected %d modes, got %d", len(expectedModes), len(modes))
	}

	for i, expected := range expectedModes {
		if i >= len(modes) || modes[i] != expected {
			t.Errorf("Expected mode %v at index %d, got %v", expected, i, modes[i])
		}
	}
}

func TestValidPOEPriorities(t *testing.T) {
	fixtures := NewTestFixtures()
	priorities := fixtures.ValidPOEPriorities()

	expectedPriorities := []netgear.POEPriority{
		netgear.POEPriorityLow,
		netgear.POEPriorityHigh,
		netgear.POEPriorityCritical,
	}

	if len(priorities) != len(expectedPriorities) {
		t.Errorf("Expected %d priorities, got %d", len(expectedPriorities), len(priorities))
	}

	for i, expected := range expectedPriorities {
		if i >= len(priorities) || priorities[i] != expected {
			t.Errorf("Expected priority %v at index %d, got %v", expected, i, priorities[i])
		}
	}
}

func TestValidPortSpeeds(t *testing.T) {
	fixtures := NewTestFixtures()
	speeds := fixtures.ValidPortSpeeds()

	expectedSpeeds := []netgear.PortSpeed{
		netgear.PortSpeedAuto,
		netgear.PortSpeed10MHalf,
		netgear.PortSpeed10MFull,
		netgear.PortSpeed100MHalf,
		netgear.PortSpeed100MFull,
		netgear.PortSpeedDisable,
	}

	if len(speeds) != len(expectedSpeeds) {
		t.Errorf("Expected %d speeds, got %d", len(expectedSpeeds), len(speeds))
	}
}

func TestGetMaxPowerLimit(t *testing.T) {
	fixtures := NewTestFixtures()

	tests := []struct {
		model    string
		expected float64
	}{
		{"GS305EP", 63.0},
		{"GS305EPP", 120.0},
		{"GS308EP", 83.0},
		{"GS308EPP", 123.0},
		{"GS316EP", 180.0},
		{"GS316EPP", 231.0},
		{"UNKNOWN", 30.0}, // Default
	}

	for _, tt := range tests {
		result := fixtures.GetMaxPowerLimit(tt.model)
		if result != tt.expected {
			t.Errorf("GetMaxPowerLimit(%s) = %v, want %v", tt.model, result, tt.expected)
		}
	}
}

func TestGetPortCount(t *testing.T) {
	fixtures := NewTestFixtures()

	tests := []struct {
		model    string
		expected int
	}{
		{"GS305EP", 5},
		{"GS305EPP", 5},
		{"GS308EP", 8},
		{"GS308EPP", 8},
		{"GS316EP", 16},
		{"GS316EPP", 16},
		{"UNKNOWN", 8}, // Default
	}

	for _, tt := range tests {
		result := fixtures.GetPortCount(tt.model)
		if result != tt.expected {
			t.Errorf("GetPortCount(%s) = %v, want %v", tt.model, result, tt.expected)
		}
	}
}

func TestGetValidPortNumbers(t *testing.T) {
	fixtures := NewTestFixtures()

	// Test GS305EP (5 ports)
	ports := fixtures.GetValidPortNumbers("GS305EP")
	expected := []int{1, 2, 3, 4, 5}

	if len(ports) != len(expected) {
		t.Errorf("Expected %d ports, got %d", len(expected), len(ports))
	}

	for i, expectedPort := range expected {
		if i >= len(ports) || ports[i] != expectedPort {
			t.Errorf("Expected port %d at index %d, got %d", expectedPort, i, ports[i])
		}
	}
}

func TestCreateTestPOEUpdate(t *testing.T) {
	fixtures := NewTestFixtures()

	enabled := true
	mode := netgear.POEMode8023af
	priority := netgear.POEPriorityHigh

	update := fixtures.CreateTestPOEUpdate(1, enabled, mode, priority)

	if update.PortID != 1 {
		t.Errorf("Expected PortID 1, got %d", update.PortID)
	}
	if update.Enabled == nil || *update.Enabled != enabled {
		t.Errorf("Expected Enabled %v, got %v", enabled, update.Enabled)
	}
	if update.Mode == nil || *update.Mode != mode {
		t.Errorf("Expected Mode %v, got %v", mode, update.Mode)
	}
	if update.Priority == nil || *update.Priority != priority {
		t.Errorf("Expected Priority %v, got %v", priority, update.Priority)
	}
}

func TestCreateTestPortUpdate(t *testing.T) {
	fixtures := NewTestFixtures()

	name := "TestPort"
	speed := netgear.PortSpeedAuto
	flowControl := true

	update := fixtures.CreateTestPortUpdate(1, name, speed, flowControl)

	if update.PortID != 1 {
		t.Errorf("Expected PortID 1, got %d", update.PortID)
	}
	if update.Name == nil || *update.Name != name {
		t.Errorf("Expected Name %v, got %v", name, update.Name)
	}
	if update.Speed == nil || *update.Speed != speed {
		t.Errorf("Expected Speed %v, got %v", speed, update.Speed)
	}
	if update.FlowControl == nil || *update.FlowControl != flowControl {
		t.Errorf("Expected FlowControl %v, got %v", flowControl, update.FlowControl)
	}
}

func TestGetTestScenarios(t *testing.T) {
	fixtures := NewTestFixtures()
	scenarios := fixtures.GetTestScenarios()

	expectedScenarios := []string{
		"basic_poe_enable",
		"poe_mode_cycle",
		"poe_priority_test",
		"port_speed_test",
		"flow_control_test",
	}

	for _, expectedName := range expectedScenarios {
		scenario, exists := scenarios[expectedName]
		if !exists {
			t.Errorf("Expected scenario %s not found", expectedName)
			continue
		}

		if scenario.Name != expectedName {
			t.Errorf("Expected scenario name %s, got %s", expectedName, scenario.Name)
		}

		if scenario.Description == "" {
			t.Errorf("Scenario %s has empty description", expectedName)
		}

		if scenario.TestType == "" {
			t.Errorf("Scenario %s has empty test type", expectedName)
		}
	}
}

func TestGetWaitTimes(t *testing.T) {
	fixtures := NewTestFixtures()
	waitTimes := fixtures.GetWaitTimes()

	if waitTimes.POECycleRecovery <= 0 {
		t.Error("POECycleRecovery should be positive")
	}
	if waitTimes.PortSpeedChange <= 0 {
		t.Error("PortSpeedChange should be positive")
	}
	if waitTimes.ConfigChange <= 0 {
		t.Error("ConfigChange should be positive")
	}
	if waitTimes.AuthTimeout <= 0 {
		t.Error("AuthTimeout should be positive")
	}

	// Check reasonable values
	if waitTimes.POECycleRecovery != 30*time.Second {
		t.Errorf("Expected POECycleRecovery 30s, got %v", waitTimes.POECycleRecovery)
	}
}

func TestValidRateLimits(t *testing.T) {
	fixtures := NewTestFixtures()
	rateLimits := fixtures.ValidRateLimits()

	expectedLimits := []string{
		"64kbps", "128kbps", "256kbps", "512kbps",
		"1Mbps", "10Mbps", "100Mbps", "unlimited",
	}

	if len(rateLimits) != len(expectedLimits) {
		t.Errorf("Expected %d rate limits, got %d", len(expectedLimits), len(rateLimits))
	}

	for _, expected := range expectedLimits {
		found := false
		for _, actual := range rateLimits {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected rate limit %s not found", expected)
		}
	}
}

func TestInvalidRateLimits(t *testing.T) {
	fixtures := NewTestFixtures()
	invalidLimits := fixtures.InvalidRateLimits()

	// Should have some invalid limits for negative testing
	if len(invalidLimits) == 0 {
		t.Error("Expected some invalid rate limits for negative testing")
	}

	// Check for specific invalid cases
	expectedInvalids := []string{"-10Mbps", "0kbps", "999999Mbps", "invalid", ""}
	for _, expected := range expectedInvalids {
		found := false
		for _, actual := range invalidLimits {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected invalid rate limit %s not found", expected)
		}
	}
}