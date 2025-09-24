package test

import (
	"fmt"
	"time"

	"github.com/gherlein/go-netgear/pkg/netgear"
)

// TestFixtures provides standard test data for various test scenarios
type TestFixtures struct{}

// NewTestFixtures creates a new test fixtures instance
func NewTestFixtures() *TestFixtures {
	return &TestFixtures{}
}

// ValidPOEModes returns all valid POE modes for testing
func (f *TestFixtures) ValidPOEModes() []netgear.POEMode {
	return []netgear.POEMode{
		netgear.POEMode8023af,
		netgear.POEMode8023at,
		netgear.POEModeLegacy,
		netgear.POEModePre8023at,
	}
}

// ValidPOEPriorities returns all valid POE priorities for testing
func (f *TestFixtures) ValidPOEPriorities() []netgear.POEPriority {
	return []netgear.POEPriority{
		netgear.POEPriorityLow,
		netgear.POEPriorityHigh,
		netgear.POEPriorityCritical,
	}
}

// ValidPOELimitTypes returns all valid POE limit types for testing
func (f *TestFixtures) ValidPOELimitTypes() []netgear.POELimitType {
	return []netgear.POELimitType{
		netgear.POELimitTypeNone,
		netgear.POELimitTypeClass,
		netgear.POELimitTypeUser,
	}
}

// ValidPowerLimits returns valid power limit values for testing (in watts)
func (f *TestFixtures) ValidPowerLimits() []float64 {
	return []float64{5.0, 15.0, 30.0, 60.0, 90.0}
}

// InvalidPowerLimits returns invalid power limit values for negative testing
func (f *TestFixtures) InvalidPowerLimits() []float64 {
	return []float64{0.0, -5.0, 1000.0, -1.0}
}

// ValidPortSpeeds returns all valid port speeds for testing
func (f *TestFixtures) ValidPortSpeeds() []netgear.PortSpeed {
	return []netgear.PortSpeed{
		netgear.PortSpeedAuto,
		netgear.PortSpeed10MHalf,
		netgear.PortSpeed10MFull,
		netgear.PortSpeed100MHalf,
		netgear.PortSpeed100MFull,
		netgear.PortSpeedDisable,
	}
}

// ValidRateLimits returns valid rate limit values for testing
func (f *TestFixtures) ValidRateLimits() []string {
	return []string{
		"64kbps",
		"128kbps",
		"256kbps",
		"512kbps",
		"1Mbps",
		"10Mbps",
		"100Mbps",
		"unlimited",
	}
}

// InvalidRateLimits returns invalid rate limit values for negative testing
func (f *TestFixtures) InvalidRateLimits() []string {
	return []string{
		"-10Mbps",
		"0kbps",
		"999999Mbps",
		"invalid",
		"",
	}
}

// ValidPortNames returns valid port names for testing
func (f *TestFixtures) ValidPortNames() []string {
	timestamp := time.Now().Format("20060102_150405")
	return []string{
		fmt.Sprintf("TEST_PORT_%s", timestamp),
		fmt.Sprintf("Port_%s", timestamp),
		"Test Port",
		"SERVER_01",
		"AP-WiFi-01",
		"Camera_Lobby",
	}
}

// InvalidPortNames returns invalid port names for negative testing
func (f *TestFixtures) InvalidPortNames() []string {
	// Create a very long name that might exceed switch limits
	longName := ""
	for i := 0; i < 100; i++ {
		longName += "A"
	}

	return []string{
		longName,
		"", // Empty name
		"Port\x00WithNull", // Null character
		"Port\nWithNewline", // Newline character
	}
}

// ValidDetectionTypes returns valid POE detection types
func (f *TestFixtures) ValidDetectionTypes() []string {
	return []string{
		"IEEE 802",
		"Legacy",
		"4pt 802.3af + Legacy",
	}
}

// InvalidDetectionTypes returns invalid detection types for negative testing
func (f *TestFixtures) InvalidDetectionTypes() []string {
	return []string{
		"Invalid Detection",
		"",
		"Unknown Type",
	}
}

// FlowControlValues returns flow control test values
func (f *TestFixtures) FlowControlValues() []bool {
	return []bool{true, false}
}

// POEEnabledValues returns POE enabled test values
func (f *TestFixtures) POEEnabledValues() []bool {
	return []bool{true, false}
}

// TestPortNameWithTimestamp generates a unique test port name with timestamp
func (f *TestFixtures) TestPortNameWithTimestamp(prefix string) string {
	timestamp := time.Now().Format("20060102_150405_000")
	return fmt.Sprintf("%s_%s", prefix, timestamp)
}

// GetCompatiblePOEModes returns POE modes compatible with a specific switch model
func (f *TestFixtures) GetCompatiblePOEModes(model string) []netgear.POEMode {
	allModes := f.ValidPOEModes()

	// For now, assume all models support all modes
	// This can be refined based on actual switch capabilities
	switch model {
	case "GS305EP", "GS305EPP":
		// 5-port switches - might have different capabilities
		return allModes
	case "GS308EP", "GS308EPP":
		// 8-port switches
		return allModes
	case "GS316EP", "GS316EPP":
		// 16-port switches
		return allModes
	default:
		return allModes
	}
}

// GetMaxPowerLimit returns the maximum power limit for a specific switch model
func (f *TestFixtures) GetMaxPowerLimit(model string) float64 {
	switch model {
	case "GS305EP":
		return 63.0 // 5-port with 63W budget
	case "GS305EPP":
		return 120.0 // 5-port with higher budget
	case "GS308EP":
		return 83.0 // 8-port with 83W budget
	case "GS308EPP":
		return 123.0 // 8-port with higher budget
	case "GS316EP":
		return 180.0 // 16-port
	case "GS316EPP":
		return 231.0 // 16-port with higher budget
	default:
		return 30.0 // Conservative default
	}
}

// GetPortCount returns the number of ports for a specific switch model
func (f *TestFixtures) GetPortCount(model string) int {
	switch model {
	case "GS305EP", "GS305EPP":
		return 5
	case "GS308EP", "GS308EPP":
		return 8
	case "GS316EP", "GS316EPP":
		return 16
	default:
		return 8 // Default assumption
	}
}

// GetValidPortNumbers returns valid port numbers for a specific switch model
func (f *TestFixtures) GetValidPortNumbers(model string) []int {
	portCount := f.GetPortCount(model)
	ports := make([]int, portCount)
	for i := 0; i < portCount; i++ {
		ports[i] = i + 1 // Ports are 1-indexed
	}
	return ports
}

// GetInvalidPortNumbers returns invalid port numbers for negative testing
func (f *TestFixtures) GetInvalidPortNumbers() []int {
	return []int{0, -1, 17, 100, 999}
}

// CreateTestPOEUpdate creates a POE port update for testing
func (f *TestFixtures) CreateTestPOEUpdate(portID int, enabled bool, mode netgear.POEMode, priority netgear.POEPriority) netgear.POEPortUpdate {
	return netgear.POEPortUpdate{
		PortID:   portID,
		Enabled:  &enabled,
		Mode:     &mode,
		Priority: &priority,
	}
}

// CreateTestPortUpdate creates a port update for testing
func (f *TestFixtures) CreateTestPortUpdate(portID int, name string, speed netgear.PortSpeed, flowControl bool) netgear.PortUpdate {
	return netgear.PortUpdate{
		PortID:      portID,
		Name:        &name,
		Speed:       &speed,
		FlowControl: &flowControl,
	}
}

// GetTestScenarios returns predefined test scenarios for comprehensive testing
func (f *TestFixtures) GetTestScenarios() map[string]TestScenario {
	return map[string]TestScenario{
		"basic_poe_enable": {
			Name:        "basic_poe_enable",
			Description: "Test basic POE enable/disable functionality",
			TestType:    "poe",
			Parameters: map[string]interface{}{
				"enabled": []bool{true, false},
			},
		},
		"poe_mode_cycle": {
			Name:        "poe_mode_cycle",
			Description: "Test cycling through all POE modes",
			TestType:    "poe",
			Parameters: map[string]interface{}{
				"modes": f.ValidPOEModes(),
			},
		},
		"poe_priority_test": {
			Name:        "poe_priority_test",
			Description: "Test POE priority settings",
			TestType:    "poe",
			Parameters: map[string]interface{}{
				"priorities": f.ValidPOEPriorities(),
			},
		},
		"port_speed_test": {
			Name:        "port_speed_test",
			Description: "Test port speed configurations",
			TestType:    "port",
			Parameters: map[string]interface{}{
				"speeds": f.ValidPortSpeeds(),
			},
		},
		"flow_control_test": {
			Name:        "flow_control_test",
			Description: "Test flow control enable/disable",
			TestType:    "port",
			Parameters: map[string]interface{}{
				"flow_control": f.FlowControlValues(),
			},
		},
	}
}

// TestScenario represents a predefined test scenario
type TestScenario struct {
	Name        string
	Description string
	TestType    string // "poe", "port", "auth", etc.
	Parameters  map[string]interface{}
}

// WaitTimes provides standard wait times for different operations
type WaitTimes struct {
	POECycleRecovery  time.Duration
	PortSpeedChange   time.Duration
	ConfigChange      time.Duration
	AuthTimeout       time.Duration
}

// GetWaitTimes returns standard wait times for test operations
func (f *TestFixtures) GetWaitTimes() WaitTimes {
	return WaitTimes{
		POECycleRecovery: 30 * time.Second,
		PortSpeedChange:  5 * time.Second,
		ConfigChange:     2 * time.Second,
		AuthTimeout:      10 * time.Second,
	}
}