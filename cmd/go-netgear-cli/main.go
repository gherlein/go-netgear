package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"

	"github.com/gherlein/go-netgear/test"
)

const (
	ExitSuccess = 0
	ExitError   = 1
)

func main() {
	var (
		validateConfig = flag.Bool("validate-config", false, "Validate the test configuration file and exit")
		configPath     = flag.String("config", "test/test_config.json", "Path to test configuration file")
		help           = flag.Bool("help", false, "Show help information")
		h              = flag.Bool("h", false, "Show help information (short)")
	)

	flag.Parse()

	if *help || *h {
		printHelp()
		os.Exit(ExitSuccess)
	}

	if *validateConfig {
		validateTestConfig(*configPath)
		return
	}

	// Default behavior - show help if no flags provided
	printHelp()
}

func validateTestConfig(configPath string) {
	fmt.Printf("Validating test configuration file: %s\n\n", configPath)

	// Check if file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		absPath, _ := filepath.Abs(configPath)
		fmt.Printf("❌ FAIL: Configuration file not found: %s\n", configPath)
		fmt.Printf("   Expected file at: %s\n", absPath)
		os.Exit(ExitError)
	}

	// Attempt to load and validate the configuration
	config, err := test.LoadTestConfig(configPath)
	if err != nil {
		fmt.Printf("❌ FAIL: Configuration validation failed\n")
		fmt.Printf("   Error: %v\n\n", err)

		// Provide helpful suggestions based on error type
		printConfigErrorHelp(err)
		os.Exit(ExitError)
	}

	// If we get here, basic loading succeeded
	fmt.Printf("✅ Configuration file loaded successfully\n")

	// Perform detailed validation
	if len(config.Switches) == 0 {
		fmt.Printf("❌ FAIL: No switches configured\n")
		fmt.Printf("   The configuration file must contain at least one switch in the 'switches' array\n")
		os.Exit(ExitError)
	}

	fmt.Printf("   Found %d switch(es) configured\n\n", len(config.Switches))

	// Validate each switch configuration
	allValid := true
	for i, switchConfig := range config.Switches {
		fmt.Printf("Switch %d: %s\n", i+1, switchConfig.Name)

		valid := validateSwitchConfig(switchConfig, i+1)
		if !valid {
			allValid = false
		}
		fmt.Println()
	}

	// Validate test options
	fmt.Printf("Test Options:\n")
	fmt.Printf("   Cache Directory: %s\n", config.TestOptions.CacheDir)
	fmt.Printf("   Verbose: %t\n", config.TestOptions.Verbose)
	fmt.Printf("   Parallel: %t\n", config.TestOptions.Parallel)
	fmt.Printf("   Restore on Failure: %t\n", config.TestOptions.RestoreOnFailure)

	// Check if cache directory can be created
	if config.TestOptions.CacheDir != "" {
		if err := os.MkdirAll(config.TestOptions.CacheDir, 0755); err != nil {
			fmt.Printf("   ⚠️  Warning: Cannot create cache directory: %v\n", err)
		} else {
			fmt.Printf("   ✅ Cache directory accessible\n")
		}
	}

	fmt.Println()

	if allValid {
		fmt.Printf("🎉 PASS: Configuration file is valid and ready for testing\n")

		// Check environment variables by reading raw JSON file
		fmt.Printf("\nEnvironment Variables:\n")
		checkEnvironmentVariables(configPath)

		fmt.Printf("\nTo run tests with this configuration:\n")
		fmt.Printf("   make run-tests\n")
		fmt.Printf("   go test -v ./test\n")
		os.Exit(ExitSuccess)
	} else {
		fmt.Printf("❌ FAIL: Configuration file has validation errors\n")
		os.Exit(ExitError)
	}
}

func validateSwitchConfig(switchConfig test.SwitchConfig, index int) bool {
	valid := true

	// Check required fields
	if switchConfig.Name == "" {
		fmt.Printf("   ❌ Name: Missing (required)\n")
		valid = false
	} else {
		fmt.Printf("   ✅ Name: %s\n", switchConfig.Name)
	}

	if switchConfig.Address == "" {
		fmt.Printf("   ❌ Address: Missing (required)\n")
		valid = false
	} else {
		fmt.Printf("   ✅ Address: %s\n", switchConfig.Address)
	}

	if switchConfig.Model == "" {
		fmt.Printf("   ❌ Model: Missing (required)\n")
		valid = false
	} else {
		// Validate model is supported
		validModels := []string{"GS305EP", "GS305EPP", "GS308EP", "GS308EPP", "GS308EEP", "GS316EP", "GS316EPP"}
		modelValid := false
		for _, validModel := range validModels {
			if switchConfig.Model == validModel {
				modelValid = true
				break
			}
		}

		if modelValid {
			fmt.Printf("   ✅ Model: %s (supported)\n", switchConfig.Model)
		} else {
			fmt.Printf("   ❌ Model: %s (unsupported - valid: %v)\n", switchConfig.Model, validModels)
			valid = false
		}
	}

	if switchConfig.Password == "" {
		fmt.Printf("   ❌ Password: Missing (required)\n")
		valid = false
	} else if len(switchConfig.Password) > 3 && switchConfig.Password[0] == '$' && switchConfig.Password[1] == '{' {
		fmt.Printf("   ✅ Password: Environment variable (%s)\n", switchConfig.Password)
	} else if len(switchConfig.Password) >= 6 {
		fmt.Printf("   ✅ Password: Configured (hidden)\n")
	} else {
		fmt.Printf("   ⚠️  Password: Very short (may be invalid)\n")
	}

	// Validate test ports
	if len(switchConfig.TestPorts) == 0 {
		fmt.Printf("   ⚠️  Test Ports: None configured (tests will skip)\n")
	} else {
		fmt.Printf("   ✅ Test Ports: %v\n", switchConfig.TestPorts)

		// Check port numbers are reasonable
		for _, port := range switchConfig.TestPorts {
			if port < 1 || port > 48 { // Most switches have 1-48 ports max
				fmt.Printf("   ⚠️  Port %d: Unusual port number (may be invalid)\n", port)
			}
		}
	}

	// Show skip tests if any
	if len(switchConfig.SkipTests) > 0 {
		fmt.Printf("   ℹ️  Skipped Tests: %v\n", switchConfig.SkipTests)
	}

	return valid
}

func printConfigErrorHelp(err error) {
	errStr := err.Error()

	fmt.Printf("Common configuration issues:\n\n")

	if contains(errStr, "invalid character") || contains(errStr, "json") {
		fmt.Printf("JSON Syntax Error:\n")
		fmt.Printf("   • Check for missing commas between fields\n")
		fmt.Printf("   • Ensure all strings are quoted with double quotes\n")
		fmt.Printf("   • Verify all brackets {} and [] are properly closed\n")
		fmt.Printf("   • Use a JSON validator to check syntax\n\n")
	}

	if contains(errStr, "password is required") {
		fmt.Printf("Missing Password:\n")
		fmt.Printf("   • Each switch must have a 'password' field\n")
		fmt.Printf("   • Use environment variables: \"${TEST_SWITCH_PASSWORD_1}\"\n")
		fmt.Printf("   • Or provide direct password: \"your_password_here\"\n\n")
	}

	if contains(errStr, "name is required") {
		fmt.Printf("Missing Switch Name:\n")
		fmt.Printf("   • Each switch must have a unique 'name' field\n")
		fmt.Printf("   • Example: \"name\": \"my-switch-1\"\n\n")
	}

	if contains(errStr, "address is required") {
		fmt.Printf("Missing Address:\n")
		fmt.Printf("   • Each switch must have an 'address' field\n")
		fmt.Printf("   • Use IP address: \"192.168.1.10\"\n")
		fmt.Printf("   • Or hostname: \"switch.example.com\"\n\n")
	}

	if contains(errStr, "model is required") {
		fmt.Printf("Missing Model:\n")
		fmt.Printf("   • Each switch must specify a 'model'\n")
		fmt.Printf("   • Supported: GS305EP, GS305EPP, GS308EP, GS308EPP, GS316EP, GS316EPP\n\n")
	}

	fmt.Printf("Example valid configuration:\n")
	fmt.Printf(`{
  "switches": [
    {
      "name": "test-switch-1",
      "address": "192.168.1.10",
      "model": "GS308EP",
      "password": "${TEST_SWITCH_PASSWORD_1}",
      "test_ports": [1, 2, 3],
      "skip_tests": []
    }
  ],
  "test_options": {
    "cache_dir": "/tmp/netgear-test-cache",
    "verbose": true,
    "parallel": false,
    "restore_on_failure": true
  }
}
`)
}

func printHelp() {
	fmt.Printf("go-netgear - Netgear Switch Management Library\n\n")
	fmt.Printf("Usage:\n")
	fmt.Printf("  go run main.go [options]\n\n")
	fmt.Printf("Options:\n")
	fmt.Printf("  --validate-config        Validate test configuration file and exit\n")
	fmt.Printf("  --config <path>          Path to test configuration file (default: test/test_config.json)\n")
	fmt.Printf("  --help, -h               Show this help information\n\n")
	fmt.Printf("Examples:\n")
	fmt.Printf("  go run main.go --validate-config\n")
	fmt.Printf("  go run main.go --validate-config --config /path/to/config.json\n\n")
	fmt.Printf("For running tests:\n")
	fmt.Printf("  make run-tests           Run comprehensive test suite\n")
	fmt.Printf("  make test-offline        Run tests without network dependencies\n")
	fmt.Printf("  make help                Show all available make targets\n")
}

func checkEnvironmentVariables(configPath string) {
	// Read raw JSON file to find environment variable references
	rawData, err := ioutil.ReadFile(configPath)
	if err != nil {
		fmt.Printf("   Error reading config file: %v\n", err)
		return
	}

	// Find all environment variable references using regex
	envVarRegex := regexp.MustCompile(`\$\{([^}]+)\}`)
	matches := envVarRegex.FindAllStringSubmatch(string(rawData), -1)

	if len(matches) == 0 {
		fmt.Printf("   No environment variables configured\n")
		return
	}

	// Check each environment variable
	envVarsFound := make(map[string]bool)
	for _, match := range matches {
		if len(match) > 1 {
			envVar := match[1]
			if _, seen := envVarsFound[envVar]; !seen {
				envVarsFound[envVar] = true
				if os.Getenv(envVar) != "" {
					fmt.Printf("   %s: ✅ Set\n", envVar)
				} else {
					fmt.Printf("   %s: ❌ Not set (required)\n", envVar)
				}
			}
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr ||
		   (len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}