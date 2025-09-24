package test

import (
	"testing"
	"time"

	"github.com/gherlein/go-netgear/pkg/netgear"
)

func TestNewTestHelper(t *testing.T) {
	config := &TestConfig{
		TestOptions: TestOptions{
			Verbose: false,
			CacheDir: "/tmp/test",
		},
	}

	helper := NewTestHelper(config)
	if helper == nil {
		t.Fatal("Expected non-nil helper")
	}
	if helper.verbose {
		t.Error("Expected verbose to be false")
	}
	if helper.config != config {
		t.Error("Expected config to be set correctly")
	}
}

func TestComparePOEState(t *testing.T) {
	helper := &TestHelper{verbose: false}

	state1 := POEState{
		Enabled:        true,
		Mode:           netgear.POEMode8023af,
		Priority:       netgear.POEPriorityHigh,
		PowerLimitType: netgear.POELimitTypeUser,
		PowerLimitW:    15.0,
		DetectionType:  "IEEE 802",
	}

	state2 := POEState{
		Enabled:        false,
		Mode:           netgear.POEMode8023at,
		Priority:       netgear.POEPriorityLow,
		PowerLimitType: netgear.POELimitTypeClass,
		PowerLimitW:    15.1, // Within tolerance
		DetectionType:  "Legacy",
	}

	differences := helper.ComparePOEState(state1, state2)

	// Should find differences for all fields except PowerLimitW (within tolerance)
	expectedDiffs := 5 // Enabled, Mode, Priority, PowerLimitType, DetectionType
	if len(differences) != expectedDiffs {
		t.Errorf("Expected %d differences, got %d: %v", expectedDiffs, len(differences), differences)
	}

	// Test identical states
	differences = helper.ComparePOEState(state1, state1)
	if len(differences) != 0 {
		t.Errorf("Expected no differences for identical states, got %d: %v", len(differences), differences)
	}
}

func TestComparePortState(t *testing.T) {
	helper := &TestHelper{verbose: false}

	state1 := PortState{
		Name:         "Port1",
		Speed:        netgear.PortSpeedAuto,
		FlowControl:  true,
		IngressLimit: "unlimited",
		EgressLimit:  "100Mbps",
	}

	state2 := PortState{
		Name:         "Port2",
		Speed:        netgear.PortSpeed100MFull,
		FlowControl:  false,
		IngressLimit: "10Mbps",
		EgressLimit:  "50Mbps",
	}

	differences := helper.ComparePortState(state1, state2)

	// Should find differences for all fields
	expectedDiffs := 5
	if len(differences) != expectedDiffs {
		t.Errorf("Expected %d differences, got %d: %v", expectedDiffs, len(differences), differences)
	}

	// Test identical states
	differences = helper.ComparePortState(state1, state1)
	if len(differences) != 0 {
		t.Errorf("Expected no differences for identical states, got %d: %v", len(differences), differences)
	}
}

func TestGenerateTestReport(t *testing.T) {
	helper := &TestHelper{verbose: false}

	now := time.Now()
	results := []TestResult{
		{TestName: "test1", Passed: true, StartTime: now, EndTime: now.Add(time.Second)},
		{TestName: "test2", Passed: false, StartTime: now.Add(time.Second), EndTime: now.Add(2*time.Second)},
		{TestName: "test3", Passed: true, StartTime: now.Add(2*time.Second), EndTime: now.Add(3*time.Second)},
	}

	report := helper.GenerateTestReport(results)

	if report.TotalTests != 3 {
		t.Errorf("Expected 3 total tests, got %d", report.TotalTests)
	}
	if report.PassedTests != 2 {
		t.Errorf("Expected 2 passed tests, got %d", report.PassedTests)
	}
	if report.FailedTests != 1 {
		t.Errorf("Expected 1 failed test, got %d", report.FailedTests)
	}
	if !report.StartTime.Equal(now) {
		t.Errorf("Expected start time %v, got %v", now, report.StartTime)
	}
}

func TestContainsPOETest(t *testing.T) {
	tests := []struct {
		testName string
		expected bool
	}{
		{"test_poe_enable", true},
		{"test_power_cycle", true},
		{"test_POE_mode", true},
		{"test_port_speed", false},
		{"test_flow_control", false},
	}

	for _, tt := range tests {
		result := containsPOETest(tt.testName)
		if result != tt.expected {
			t.Errorf("containsPOETest(%s) = %v, want %v", tt.testName, result, tt.expected)
		}
	}
}

func TestContainsPortTest(t *testing.T) {
	tests := []struct {
		testName string
		expected bool
	}{
		{"test_port_speed", true},
		{"test_flow_control", true},
		{"test_rate_limit", true},
		{"test_poe_enable", false},
		{"test_authentication", false},
	}

	for _, tt := range tests {
		result := containsPortTest(tt.testName)
		if result != tt.expected {
			t.Errorf("containsPortTest(%s) = %v, want %v", tt.testName, result, tt.expected)
		}
	}
}