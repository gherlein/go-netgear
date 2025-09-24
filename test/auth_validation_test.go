package test

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

// TestEnvironmentVariableResolution verifies that environment variables are properly resolved
func TestEnvironmentVariableResolution(t *testing.T) {
	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("❌ FATAL: Cannot load test configuration - %v", err)
	}

	t.Log("🔍 Verifying environment variable resolution...")

	// Read raw config file to find environment variable references
	rawData, err := os.ReadFile("test_config.json")
	if err != nil {
		t.Fatalf("Failed to read raw config file: %v", err)
	}

	// Find all environment variable references using regex
	envVarRegex := regexp.MustCompile(`\$\{([^}]+)\}`)
	matches := envVarRegex.FindAllStringSubmatch(string(rawData), -1)

	expectedEnvVars := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			expectedEnvVars[match[1]] = true
		}
	}

	if len(expectedEnvVars) == 0 {
		t.Log("ℹ️  No environment variables found in configuration")
		return
	}

	t.Logf("📋 Found %d environment variable(s) in configuration", len(expectedEnvVars))

	// Verify each environment variable is set and passwords are resolved
	var missingVars []string
	var resolvedPasswords []string

	for envVar := range expectedEnvVars {
		value := os.Getenv(envVar)
		if value == "" {
			missingVars = append(missingVars, envVar)
			t.Errorf("❌ Environment variable %s is not set", envVar)
		} else {
			t.Logf("✅ Environment variable %s is set", envVar)
		}
	}

	// Check that passwords in config are actually resolved (not containing ${...})
	for _, switchConfig := range config.Switches {
		if strings.Contains(switchConfig.Password, "${") {
			t.Errorf("❌ Switch %s password not resolved: %s", switchConfig.Name, switchConfig.Password)
		} else {
			resolvedPasswords = append(resolvedPasswords, switchConfig.Name)
			t.Logf("✅ Switch %s password resolved successfully", switchConfig.Name)
		}
	}

	if len(missingVars) > 0 {
		t.Fatalf("❌ FATAL: Missing environment variables: %v\n\n"+
			"Expected environment variables based on envrc template:\n"+
			"  export TEST_SWITCH_PASSWORD_1=\"password1\"\n"+
			"  export TEST_SWITCH_PASSWORD_2=\"password2\"\n\n"+
			"Please set these environment variables or source the envrc file.", missingVars)
	}

	t.Logf("✅ Environment variable resolution successful - %d password(s) resolved", len(resolvedPasswords))
}

// TestAuthenticationValidation runs after env vars are verified to validate authentication to all switches
// This test will fail (and halt the test suite) if any switch cannot be authenticated
func TestAuthenticationValidation(t *testing.T) {
	config, err := LoadTestConfig("test_config.json")
	if err != nil {
		t.Fatalf("❌ FATAL: Cannot load test configuration - %v\n\nAll tests require valid configuration to authenticate to switches.\nRun 'make validate-config' to check your configuration.", err)
	}

	if len(config.Switches) == 0 {
		t.Fatalf("❌ FATAL: No switches configured in test configuration.\n\nAll vital tests require at least one configured switch.\nPlease configure switches in test/test_config.json.")
	}

	helper := NewTestHelper(config)

	t.Logf("🔐 Validating authentication to %d configured switch(es)...", len(config.Switches))

	// Show which switches we're testing
	for _, sw := range config.Switches {
		t.Logf("  • %s (%s) - Model: %s", sw.Name, sw.Address, sw.Model)
	}

	// Test authentication to all switches
	if err := helper.ValidateAllSwitchAuthentication(); err != nil {
		t.Fatalf("❌ FATAL: Authentication validation failed:\n%v\n\n"+
			"All vital tests require working authentication to configured switches.\n"+
			"Please check:\n"+
			"  • Network connectivity to switch(es)\n"+
			"  • Switch password(s) and environment variables (run 'env | grep TEST_SWITCH')\n"+
			"  • Switch configuration in test/test_config.json\n"+
			"  • Run 'make validate-config' to verify configuration\n"+
			"  • Verify switches are accessible via web browser", err)
	}

	t.Logf("✅ Authentication validation successful - all %d switch(es) are accessible", len(config.Switches))
}