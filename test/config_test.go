package test

import (
	"os"
	"testing"
)

func TestLoadTestConfig(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_SWITCH_PASSWORD_1", "test_password_1")
	os.Setenv("TEST_SWITCH_PASSWORD_2", "test_password_2")

	// Load configuration
	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify configuration loaded correctly
	if len(config.Switches) != 1 {
		t.Errorf("Expected 1 switch, got %d", len(config.Switches))
	}

	// Check first switch
	sw1 := config.Switches[0]
	if sw1.Name != "tswitch1" {
		t.Errorf("Expected switch name 'tswitch1', got %s", sw1.Name)
	}
	if sw1.Password != "test_password_1" {
		t.Errorf("Environment variable substitution failed for password")
	}
	if len(sw1.TestPorts) != 8 {
		t.Errorf("Expected 8 test ports, got %d", len(sw1.TestPorts))
	}
	if !sw1.ShouldSkipTest("rate_limiting") {
		t.Error("Expected rate_limiting to be skipped")
	}

	// Check test options
	if config.TestOptions.CacheDir != "/tmp/netgear-test-cache" {
		t.Errorf("Expected cache dir '/tmp/netgear-test-cache', got %s", config.TestOptions.CacheDir)
	}
	if config.TestOptions.Verbose {
		t.Error("Expected verbose to be false")
	}
	if !config.TestOptions.RestoreOnFailure {
		t.Error("Expected restore_on_failure to be true")
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    TestConfig
		shouldErr bool
		errMsg    string
	}{
		{
			name:      "empty config",
			config:    TestConfig{},
			shouldErr: true,
			errMsg:    "no switches configured",
		},
		{
			name: "missing name",
			config: TestConfig{
				Switches: []SwitchConfig{
					{Address: "192.168.1.10", Model: "GS305EP", Password: "pass", TestPorts: []int{1}},
				},
			},
			shouldErr: true,
			errMsg:    "name is required",
		},
		{
			name: "invalid model",
			config: TestConfig{
				Switches: []SwitchConfig{
					{Name: "test", Address: "192.168.1.10", Model: "INVALID", Password: "pass", TestPorts: []int{1}},
				},
			},
			shouldErr: true,
			errMsg:    "unsupported model",
		},
		{
			name: "invalid port number",
			config: TestConfig{
				Switches: []SwitchConfig{
					{Name: "test", Address: "192.168.1.10", Model: "GS305EP", Password: "pass", TestPorts: []int{0}},
				},
			},
			shouldErr: true,
			errMsg:    "invalid port number",
		},
		{
			name: "valid config",
			config: TestConfig{
				Switches: []SwitchConfig{
					{Name: "test", Address: "192.168.1.10", Model: "GS305EP", Password: "pass", TestPorts: []int{1, 2, 3}},
				},
			},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.shouldErr {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			}
		})
	}
}

func TestGetSwitchByName(t *testing.T) {
	config := &TestConfig{
		Switches: []SwitchConfig{
			{Name: "switch1", Address: "192.168.1.10"},
			{Name: "switch2", Address: "192.168.1.11"},
		},
	}

	// Test existing switch
	sw, err := config.GetSwitchByName("switch1")
	if err != nil {
		t.Errorf("Failed to get existing switch: %v", err)
	}
	if sw.Address != "192.168.1.10" {
		t.Errorf("Got wrong switch, expected address 192.168.1.10, got %s", sw.Address)
	}

	// Test non-existing switch
	_, err = config.GetSwitchByName("nonexistent")
	if err == nil {
		t.Error("Expected error for non-existent switch")
	}
}

func TestShouldSkipTest(t *testing.T) {
	sw := &SwitchConfig{
		SkipTests: []string{"rate_limiting", "poe_cycle"},
	}

	tests := []struct {
		testName   string
		shouldSkip bool
	}{
		{"rate_limiting", true},
		{"rate_limiting_test", true}, // Prefix match
		{"poe_cycle", true},
		{"poe_enable", false},
		{"port_speed", false},
	}

	for _, tt := range tests {
		result := sw.ShouldSkipTest(tt.testName)
		if result != tt.shouldSkip {
			t.Errorf("ShouldSkipTest(%s) = %v, want %v", tt.testName, result, tt.shouldSkip)
		}
	}
}

func TestModelHelpers(t *testing.T) {
	tests := []struct {
		model     string
		is30x     bool
		is316     bool
	}{
		{"GS305EP", true, false},
		{"GS305EPP", true, false},
		{"GS308EP", true, false},
		{"GS308EPP", true, false},
		{"GS316EP", false, true},
		{"GS316EPP", false, true},
	}

	for _, tt := range tests {
		sw := &SwitchConfig{Model: tt.model}
		if sw.IsModel30x() != tt.is30x {
			t.Errorf("Model %s: IsModel30x() = %v, want %v", tt.model, sw.IsModel30x(), tt.is30x)
		}
		if sw.IsModel316() != tt.is316 {
			t.Errorf("Model %s: IsModel316() = %v, want %v", tt.model, sw.IsModel316(), tt.is316)
		}
	}
}