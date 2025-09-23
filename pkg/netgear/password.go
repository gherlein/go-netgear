package netgear

import (
	"os"
	"strings"
)

// PasswordManager interface for password resolution
type PasswordManager interface {
	GetPassword(address string) (string, bool)
	GetSwitchConfig(address string) (*SwitchConfig, bool)
}

// SwitchConfig represents a switch configuration from environment variables
type SwitchConfig struct {
	Host     string
	Password string
	Model    string // optional
}

// EnvironmentPasswordManager handles password resolution from environment variables
type EnvironmentPasswordManager struct {
	verbose bool
}

// NewEnvironmentPasswordManager creates a new environment-based password manager
func NewEnvironmentPasswordManager() *EnvironmentPasswordManager {
	return &EnvironmentPasswordManager{
		verbose: false,
	}
}

// NewEnvironmentPasswordManagerWithVerbose creates a new environment-based password manager with verbose logging
func NewEnvironmentPasswordManagerWithVerbose(verbose bool) *EnvironmentPasswordManager {
	return &EnvironmentPasswordManager{
		verbose: verbose,
	}
}

// GetPassword retrieves password from environment variables (backwards compatibility)
func (e *EnvironmentPasswordManager) GetPassword(address string) (string, bool) {
	config, found := e.GetSwitchConfig(address)
	if found {
		return config.Password, true
	}
	return "", false
}

// GetSwitchConfig retrieves full switch configuration including optional model
func (e *EnvironmentPasswordManager) GetSwitchConfig(address string) (*SwitchConfig, bool) {
	// Priority 1: Host-specific environment variable (highest priority)
	normalizedHost := e.normalizeHost(address)
	envVar := "NETGEAR_PASSWORD_" + normalizedHost
	if password := os.Getenv(envVar); password != "" {
		if e.verbose {
			println("Found host-specific password for", address, "via", envVar)
		}
		
		// Check for model specification
		modelVar := "NETGEAR_MODEL_" + normalizedHost
		model := os.Getenv(modelVar)
		
		return &SwitchConfig{
			Host:     address,
			Password: password,
			Model:    model,
		}, true
	}

	// Priority 2: Multi-switch configuration variable
	if config, found := e.parseMultiSwitchConfig(address); found {
		if e.verbose {
			println("Found switch config for", address, "in NETGEAR_SWITCHES")
		}
		return config, true
	}

	// No password found
	if e.verbose {
		println("No password found for", address)
	}
	return nil, false
}

// parseMultiSwitchConfig parses NETGEAR_SWITCHES environment variable for a specific host
func (e *EnvironmentPasswordManager) parseMultiSwitchConfig(targetHost string) (*SwitchConfig, bool) {
	switchesVar := os.Getenv("NETGEAR_SWITCHES")
	if switchesVar == "" {
		return nil, false
	}

	// Parse format: host1=password1[,model1];host2=password2[,model2];...
	entries := strings.Split(switchesVar, ";")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}

		// Split host=password[,model]
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			continue
		}

		host := strings.TrimSpace(parts[0])
		passwordAndModel := strings.TrimSpace(parts[1])

		// Check if this entry matches our target host
		if host != targetHost {
			continue
		}

		// Parse password[,model]
		config := &SwitchConfig{
			Host: host,
		}

		if strings.Contains(passwordAndModel, ",") {
			// Has model specification
			modelParts := strings.SplitN(passwordAndModel, ",", 2)
			config.Password = strings.TrimSpace(modelParts[0])
			config.Model = strings.TrimSpace(modelParts[1])
		} else {
			// Password only
			config.Password = passwordAndModel
		}

		return config, true
	}

	return nil, false
}

// normalizeHost converts host to environment variable format
func (e *EnvironmentPasswordManager) normalizeHost(host string) string {
	// Replace dots and colons with underscores, convert to uppercase
	normalized := strings.ReplaceAll(host, ".", "_")
	normalized = strings.ReplaceAll(normalized, ":", "_")
	return strings.ToUpper(normalized)
}

// SetVerbose enables or disables verbose logging
func (e *EnvironmentPasswordManager) SetVerbose(verbose bool) {
	e.verbose = verbose
}