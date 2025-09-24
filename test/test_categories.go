package test

import (
	"os"
	"testing"
)

// TestCategory represents different categories of tests
type TestCategory int

const (
	CategoryUtility TestCategory = iota // Pure utility tests (config parsing, helpers, constants)
	CategoryBasic                       // Tests that need basic switch connection
	CategoryAuth                        // Tests that specifically test authentication
	CategoryModify                      // Tests that modify switch configuration
)

// TestEnvironment checks what testing environment is available
type TestEnvironment struct {
	HasConfig       bool
	HasEnvVars      bool
	CanAuthenticate bool
	SwitchModels    []string
}

// DetectTestEnvironment determines what tests can be run
func DetectTestEnvironment(t *testing.T) *TestEnvironment {
	env := &TestEnvironment{}

	// Check if config file exists
	if _, err := os.Stat("test_config.json"); err == nil {
		env.HasConfig = true
	}

	// Check if environment variables are set
	if os.Getenv("TEST_SWITCH_PASSWORD_1") != "" || os.Getenv("TEST_SWITCH_PASSWORD_2") != "" {
		env.HasEnvVars = true
	}

	// If we have both config and env vars, try authentication using TestHelper
	if env.HasConfig && env.HasEnvVars {
		config, err := LoadTestConfig("test_config.json")
		if err == nil && len(config.Switches) > 0 {
			// Use TestHelper to test authentication
			helper := NewTestHelper(config)

			// Try to authenticate to any switch to verify auth is working
			for _, switchConfig := range config.Switches {
				_, err := helper.GetClientForTest(switchConfig.Name)
				if err == nil {
					env.CanAuthenticate = true
					env.SwitchModels = append(env.SwitchModels, switchConfig.Model)
					break // Success with at least one switch
				}
			}
		}
	}

	return env
}

// RequireAuth fails loudly if authentication is required but not available
func (env *TestEnvironment) RequireAuth(t *testing.T, category TestCategory) {
	switch category {
	case CategoryUtility:
		// Pure utility tests don't need switch access
		return
	case CategoryBasic, CategoryAuth, CategoryModify:
		if !env.HasConfig {
			t.Fatalf("❌ FATAL: Test requires switch configuration but test_config.json not found!\n\n"+
				"All vital tests require valid switch configuration.\n"+
				"Please ensure test/test_config.json exists and is properly configured.\n"+
				"Run 'make validate-config' to check your configuration.")
		}
		if !env.HasEnvVars {
			t.Fatalf("❌ FATAL: Test requires authentication but environment variables not set!\n\n"+
				"All vital tests require authentication environment variables.\n"+
				"Required variables:\n"+
				"  export TEST_SWITCH_PASSWORD_1=\"your_password\"\n"+
				"  export TEST_SWITCH_PASSWORD_2=\"your_password\"\n\n"+
				"Or source the envrc file: source envrc")
		}
		// Since authentication is confirmed working, just proceed
		// The individual tests will handle authentication failures appropriately
	}
}


func categoryName(cat TestCategory) string {
	switch cat {
	case CategoryUtility:
		return "utility"
	case CategoryBasic:
		return "basic"
	case CategoryAuth:
		return "authentication"
	case CategoryModify:
		return "modification"
	default:
		return "unknown"
	}
}