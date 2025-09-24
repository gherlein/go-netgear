package test

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// TestConfig represents the overall test configuration
type TestConfig struct {
	Switches    []SwitchConfig `json:"switches"`
	TestOptions TestOptions    `json:"test_options"`
}

// SwitchConfig represents configuration for a single test switch
type SwitchConfig struct {
	Name      string   `json:"name"`
	Address   string   `json:"address"`
	Model     string   `json:"model"`
	Password  string   `json:"password"`
	TestPorts []int    `json:"test_ports"`
	SkipTests []string `json:"skip_tests"`
}

// TestOptions represents global test execution options
type TestOptions struct {
	CacheDir         string `json:"cache_dir"`
	Verbose          bool   `json:"verbose"`
	Parallel         bool   `json:"parallel"`
	RestoreOnFailure bool   `json:"restore_on_failure"`
}

// LoadTestConfig loads test configuration from file and resolves environment variables
func LoadTestConfig(filename string) (*TestConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Replace environment variables in the format ${VAR_NAME}
	configStr := string(data)
	configStr = expandEnvVariables(configStr)

	var config TestConfig
	if err := json.Unmarshal([]byte(configStr), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	// Set defaults if not specified
	config.setDefaults()

	return &config, nil
}

// expandEnvVariables replaces ${VAR_NAME} with environment variable values
func expandEnvVariables(s string) string {
	// Find all ${...} patterns
	for {
		start := strings.Index(s, "${")
		if start == -1 {
			break
		}

		end := strings.Index(s[start:], "}")
		if end == -1 {
			break
		}
		end += start

		varName := s[start+2 : end]
		varValue := os.Getenv(varName)

		// Replace ${VAR_NAME} with actual value
		s = s[:start] + varValue + s[end+1:]
	}

	return s
}

// Validate checks if the configuration is valid
func (c *TestConfig) Validate() error {
	if len(c.Switches) == 0 {
		return fmt.Errorf("no switches configured")
	}

	for i, sw := range c.Switches {
		if sw.Name == "" {
			return fmt.Errorf("switch %d: name is required", i)
		}
		if sw.Address == "" {
			return fmt.Errorf("switch %s: address is required", sw.Name)
		}
		if sw.Model == "" {
			return fmt.Errorf("switch %s: model is required", sw.Name)
		}
		if sw.Password == "" {
			return fmt.Errorf("switch %s: password is required", sw.Name)
		}
		if len(sw.TestPorts) == 0 {
			return fmt.Errorf("switch %s: at least one test port is required", sw.Name)
		}

		// Validate model is supported
		validModels := []string{"GS305EP", "GS305EPP", "GS308EP", "GS308EPP", "GS308EEP", "GS316EP", "GS316EPP"}
		validModel := false
		for _, m := range validModels {
			if sw.Model == m {
				validModel = true
				break
			}
		}
		if !validModel {
			return fmt.Errorf("switch %s: unsupported model %s", sw.Name, sw.Model)
		}

		// Validate port numbers are reasonable
		for _, port := range sw.TestPorts {
			if port < 1 || port > 16 {
				return fmt.Errorf("switch %s: invalid port number %d (must be 1-16)", sw.Name, port)
			}
		}
	}

	return nil
}

// setDefaults sets default values for optional configuration fields
func (c *TestConfig) setDefaults() {
	if c.TestOptions.CacheDir == "" {
		c.TestOptions.CacheDir = "/tmp/netgear-test-cache"
	}
}

// GetSwitchByName returns a switch configuration by name
func (c *TestConfig) GetSwitchByName(name string) (*SwitchConfig, error) {
	for _, sw := range c.Switches {
		if sw.Name == name {
			return &sw, nil
		}
	}
	return nil, fmt.Errorf("switch with name %s not found", name)
}

// ShouldSkipTest checks if a test should be skipped for a switch
func (s *SwitchConfig) ShouldSkipTest(testName string) bool {
	for _, skip := range s.SkipTests {
		if skip == testName || strings.HasPrefix(testName, skip) {
			return true
		}
	}
	return false
}

// IsModel30x returns true if the switch is a 30x series model
func (s *SwitchConfig) IsModel30x() bool {
	return strings.HasPrefix(s.Model, "GS30")
}

// IsModel316 returns true if the switch is a 316 series model
func (s *SwitchConfig) IsModel316() bool {
	return strings.HasPrefix(s.Model, "GS316")
}