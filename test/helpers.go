package test

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/gherlein/go-netgear/pkg/netgear"
)

// TestResult represents the result of a test operation
type TestResult struct {
	TestName    string
	SwitchName  string
	Passed      bool
	Error       error
	Duration    time.Duration
	StartTime   time.Time
	EndTime     time.Time
	Details     map[string]interface{}
}

// TestReport aggregates multiple test results
type TestReport struct {
	StartTime    time.Time
	EndTime      time.Time
	TotalTests   int
	PassedTests  int
	FailedTests  int
	SkippedTests int
	Results      []TestResult
}

// AuthenticatedClient represents a pre-authenticated client for a switch
type AuthenticatedClient struct {
	Client     *netgear.Client
	SwitchName string
	Address    string
	LoginTime  time.Time
}

// SharedAuthManager manages authentication tokens across all tests
type SharedAuthManager struct {
	clients map[string]*AuthenticatedClient
	config  *TestConfig
	verbose bool
}

// TestHelper provides utilities for test execution
type TestHelper struct {
	config  *TestConfig
	verbose bool
	authManager *SharedAuthManager
}

// Global shared auth manager - initialized once per test run
var globalAuthManager *SharedAuthManager

// NewSharedAuthManager creates a new shared authentication manager
func NewSharedAuthManager(config *TestConfig) *SharedAuthManager {
	return &SharedAuthManager{
		clients: make(map[string]*AuthenticatedClient),
		config:  config,
		verbose: config.TestOptions.Verbose,
	}
}

// GetSharedAuthManager returns the global auth manager, creating it if needed
func GetSharedAuthManager(config *TestConfig) *SharedAuthManager {
	if globalAuthManager == nil {
		globalAuthManager = NewSharedAuthManager(config)
	}
	return globalAuthManager
}

// NewTestHelper creates a new test helper
func NewTestHelper(config *TestConfig) *TestHelper {
	return &TestHelper{
		config:      config,
		verbose:     config.TestOptions.Verbose,
		authManager: GetSharedAuthManager(config),
	}
}

// AuthenticateAndCacheAll performs authentication to all switches and caches the tokens
func (sam *SharedAuthManager) AuthenticateAndCacheAll() error {
	var authErrors []string

	for _, switchConfig := range sam.config.Switches {
		if sam.verbose {
			log.Printf("Authenticating and caching token for switch %s (%s)", switchConfig.Name, switchConfig.Address)
		}

		// Create client with token cache enabled
		client, err := netgear.NewClient(switchConfig.Address,
			netgear.WithTokenCache(sam.config.TestOptions.CacheDir),
			netgear.WithVerbose(sam.verbose))
		if err != nil {
			authErrors = append(authErrors, fmt.Sprintf("Switch %s: Failed to create client - %v", switchConfig.Name, err))
			continue
		}

		// Perform authentication with retry logic for timing issues
		ctx := context.Background()
		var loginErr error
		authenticated := false

		for attempt := 1; attempt <= 3; attempt++ {
			if attempt > 1 {
				if sam.verbose {
					log.Printf("Retry attempt %d for switch %s", attempt, switchConfig.Name)
				}
				time.Sleep(time.Duration(attempt) * time.Second) // Progressive delay
			}

			loginErr = client.Login(ctx, switchConfig.Password)
			if loginErr == nil && client.IsAuthenticated() {
				authenticated = true
				break
			}
		}

		if !authenticated {
			authErrors = append(authErrors, fmt.Sprintf("Switch %s: Authentication failed after %d attempts - %v", switchConfig.Name, 3, loginErr))
			continue
		}

		// Verify authentication works by attempting a simple read operation
		_, err = client.POE().GetStatus(ctx)
		if err != nil {
			if strings.Contains(err.Error(), "not authenticated") || strings.Contains(err.Error(), "unauthorized") {
				authErrors = append(authErrors, fmt.Sprintf("Switch %s: Authentication verification failed - %v", switchConfig.Name, err))
				continue
			}
			// Other errors (like "not supported") are acceptable for authentication validation
		}

		// Cache the authenticated client
		sam.clients[switchConfig.Name] = &AuthenticatedClient{
			Client:     client,
			SwitchName: switchConfig.Name,
			Address:    switchConfig.Address,
			LoginTime:  time.Now(),
		}

		if sam.verbose {
			log.Printf("✅ Successfully authenticated and cached token for switch %s", switchConfig.Name)
		}
	}

	if len(authErrors) > 0 {
		return fmt.Errorf("❌ Authentication failed for %d switch(es):\n%s", len(authErrors), strings.Join(authErrors, "\n"))
	}

	if sam.verbose {
		log.Printf("✅ All %d switches authenticated and tokens cached successfully", len(sam.config.Switches))
	}
	return nil
}

// GetAuthenticatedClient returns a cached authenticated client for a switch
func (sam *SharedAuthManager) GetAuthenticatedClient(switchName string) (*netgear.Client, error) {
	authClient, exists := sam.clients[switchName]
	if !exists {
		return nil, fmt.Errorf("no authenticated client found for switch %s", switchName)
	}

	// Verify client is still authenticated
	if !authClient.Client.IsAuthenticated() {
		return nil, fmt.Errorf("cached client for switch %s is no longer authenticated", switchName)
	}

	return authClient.Client, nil
}

// ValidateAllSwitchAuthentication authenticates to all switches and caches tokens for reuse
// This should be called once at the beginning of the test suite
func (h *TestHelper) ValidateAllSwitchAuthentication() error {
	return h.authManager.AuthenticateAndCacheAll()
}

// GetAuthenticatedClient returns a cached authenticated client for a switch
func (h *TestHelper) GetAuthenticatedClient(switchName string) (*netgear.Client, error) {
	return h.authManager.GetAuthenticatedClient(switchName)
}

// GetClientForTest returns an authenticated client for use in tests
// This method should be used by all tests instead of creating new clients
func (h *TestHelper) GetClientForTest(switchName string) (*netgear.Client, error) {
	// First try to get cached authenticated client
	client, err := h.GetAuthenticatedClient(switchName)
	if err == nil {
		return client, nil
	}

	// If cached client not available or invalid, fall back to creating new one
	// This should only happen if authentication validation wasn't run first
	if h.verbose {
		log.Printf("Warning: No cached authentication for %s, creating new client", switchName)
	}

	client, err = h.CreateClient(switchName)
	if err != nil {
		return nil, err
	}

	if err := h.AuthenticateClient(client, switchName); err != nil {
		return nil, err
	}

	return client, nil
}

// ClearAuthenticationCache clears all cached authentication tokens
// This is useful for test cleanup or when authentication needs to be refreshed
func (h *TestHelper) ClearAuthenticationCache() {
	if h.authManager != nil {
		for switchName, authClient := range h.authManager.clients {
			if authClient.Client != nil {
				ctx := context.Background()
				authClient.Client.Logout(ctx) // Clean logout
			}
			if h.verbose {
				log.Printf("Cleared cached authentication for switch %s", switchName)
			}
		}
		// Clear the cache
		h.authManager.clients = make(map[string]*AuthenticatedClient)
	}
}

// CreateClient creates a Netgear client for a switch with test-specific configuration
func (h *TestHelper) CreateClient(switchName string) (*netgear.Client, error) {
	switchConfig, err := h.config.GetSwitchByName(switchName)
	if err != nil {
		return nil, err
	}

	// Create client with test cache directory
	client, err := netgear.NewClient(switchConfig.Address,
		netgear.WithTokenCache(h.config.TestOptions.CacheDir),
		netgear.WithVerbose(h.verbose))

	if err != nil {
		return nil, fmt.Errorf("failed to create client for switch %s: %w", switchName, err)
	}

	return client, nil
}

// AuthenticateClient performs authentication for a test switch with retry logic
func (h *TestHelper) AuthenticateClient(client *netgear.Client, switchName string) error {
	switchConfig, err := h.config.GetSwitchByName(switchName)
	if err != nil {
		return err
	}

	// Perform authentication with retry logic for timing issues
	ctx := context.Background()
	var loginErr error
	authenticated := false

	for attempt := 1; attempt <= 3; attempt++ {
		if attempt > 1 {
			if h.verbose {
				log.Printf("Retry attempt %d for switch %s", attempt, switchName)
			}
			time.Sleep(time.Duration(attempt) * time.Second) // Progressive delay
		}

		loginErr = client.Login(ctx, switchConfig.Password)
		if loginErr == nil && client.IsAuthenticated() {
			authenticated = true
			break
		}
	}

	if !authenticated {
		return fmt.Errorf("authentication failed for switch %s after 3 attempts: %w", switchName, loginErr)
	}

	if h.verbose {
		log.Printf("Successfully authenticated to switch %s (%s)", switchName, client.GetAddress())
	}

	return nil
}

// POEState represents the complete POE state for a port
type POEState struct {
	PortID         int
	Enabled        bool
	Mode           netgear.POEMode
	Priority       netgear.POEPriority
	PowerLimitType netgear.POELimitType
	PowerLimitW    float64
	DetectionType  string
}

// PortState represents the complete port configuration state
type PortState struct {
	PortID       int
	Name         string
	Speed        netgear.PortSpeed
	FlowControl  bool
	IngressLimit string
	EgressLimit  string
}

// CapturePOEState captures the current POE configuration for specified ports
func (h *TestHelper) CapturePOEState(client *netgear.Client, ports []int) (map[int]POEState, error) {
	ctx := context.Background()
	states := make(map[int]POEState)

	// Get all POE settings
	settings, err := client.POE().GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get POE settings: %w", err)
	}

	// Extract state for requested ports
	for _, setting := range settings {
		for _, portID := range ports {
			if setting.PortID == portID {
				states[portID] = POEState{
					PortID:         setting.PortID,
					Enabled:        setting.Enabled,
					Mode:           setting.Mode,
					Priority:       setting.Priority,
					PowerLimitType: setting.PowerLimitType,
					PowerLimitW:    setting.PowerLimitW,
					DetectionType:  setting.DetectionType,
				}
				break
			}
		}
	}

	return states, nil
}

// RestorePOEState restores POE configuration to previous state
func (h *TestHelper) RestorePOEState(client *netgear.Client, states map[int]POEState) error {
	ctx := context.Background()

	for portID, state := range states {
		update := netgear.POEPortUpdate{
			PortID:         portID,
			Enabled:        &state.Enabled,
			Mode:           &state.Mode,
			Priority:       &state.Priority,
			PowerLimitType: &state.PowerLimitType,
			PowerLimitW:    &state.PowerLimitW,
			DetectionType:  &state.DetectionType,
		}

		if err := client.POE().UpdatePort(ctx, update); err != nil {
			return fmt.Errorf("failed to restore POE state for port %d: %w", portID, err)
		}

		if h.verbose {
			log.Printf("Restored POE state for port %d", portID)
		}
	}

	return nil
}

// CapturePortState captures the current port configuration for specified ports
func (h *TestHelper) CapturePortState(client *netgear.Client, ports []int) (map[int]PortState, error) {
	ctx := context.Background()
	states := make(map[int]PortState)

	// Get all port settings
	settings, err := client.Ports().GetSettings(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get port settings: %w", err)
	}

	// Extract state for requested ports
	for _, setting := range settings {
		for _, portID := range ports {
			if setting.PortID == portID {
				states[portID] = PortState{
					PortID:       setting.PortID,
					Name:         setting.PortName,
					Speed:        setting.Speed,
					FlowControl:  setting.FlowControl,
					IngressLimit: setting.IngressLimit,
					EgressLimit:  setting.EgressLimit,
				}
				break
			}
		}
	}

	return states, nil
}

// RestorePortState restores port configuration to previous state
func (h *TestHelper) RestorePortState(client *netgear.Client, states map[int]PortState) error {
	ctx := context.Background()

	for portID, state := range states {
		update := netgear.PortUpdate{
			PortID:       portID,
			Name:         &state.Name,
			Speed:        &state.Speed,
			FlowControl:  &state.FlowControl,
			IngressLimit: &state.IngressLimit,
			EgressLimit:  &state.EgressLimit,
		}

		if err := client.Ports().UpdatePort(ctx, update); err != nil {
			return fmt.Errorf("failed to restore port state for port %d: %w", portID, err)
		}

		if h.verbose {
			log.Printf("Restored port state for port %d", portID)
		}
	}

	return nil
}

// ComparePOEState compares two POE states and returns differences
func (h *TestHelper) ComparePOEState(expected, actual POEState) []string {
	var differences []string

	if expected.Enabled != actual.Enabled {
		differences = append(differences, fmt.Sprintf("Enabled: expected %v, got %v", expected.Enabled, actual.Enabled))
	}
	if expected.Mode != actual.Mode {
		differences = append(differences, fmt.Sprintf("Mode: expected %v, got %v", expected.Mode, actual.Mode))
	}
	if expected.Priority != actual.Priority {
		differences = append(differences, fmt.Sprintf("Priority: expected %v, got %v", expected.Priority, actual.Priority))
	}
	if expected.PowerLimitType != actual.PowerLimitType {
		differences = append(differences, fmt.Sprintf("PowerLimitType: expected %v, got %v", expected.PowerLimitType, actual.PowerLimitType))
	}
	// Use tolerance for floating point comparison
	if abs(expected.PowerLimitW - actual.PowerLimitW) > 0.1 {
		differences = append(differences, fmt.Sprintf("PowerLimitW: expected %.1f, got %.1f", expected.PowerLimitW, actual.PowerLimitW))
	}
	if expected.DetectionType != actual.DetectionType {
		differences = append(differences, fmt.Sprintf("DetectionType: expected %v, got %v", expected.DetectionType, actual.DetectionType))
	}

	return differences
}

// ComparePortState compares two port states and returns differences
func (h *TestHelper) ComparePortState(expected, actual PortState) []string {
	var differences []string

	if expected.Name != actual.Name {
		differences = append(differences, fmt.Sprintf("Name: expected %v, got %v", expected.Name, actual.Name))
	}
	if expected.Speed != actual.Speed {
		differences = append(differences, fmt.Sprintf("Speed: expected %v, got %v", expected.Speed, actual.Speed))
	}
	if expected.FlowControl != actual.FlowControl {
		differences = append(differences, fmt.Sprintf("FlowControl: expected %v, got %v", expected.FlowControl, actual.FlowControl))
	}
	if expected.IngressLimit != actual.IngressLimit {
		differences = append(differences, fmt.Sprintf("IngressLimit: expected %v, got %v", expected.IngressLimit, actual.IngressLimit))
	}
	if expected.EgressLimit != actual.EgressLimit {
		differences = append(differences, fmt.Sprintf("EgressLimit: expected %v, got %v", expected.EgressLimit, actual.EgressLimit))
	}

	return differences
}

// WaitForPortRecovery waits for a port to recover after a change (like power cycling)
func (h *TestHelper) WaitForPortRecovery(client *netgear.Client, portID int, maxWaitTime time.Duration) error {
	ctx := context.Background()
	start := time.Now()

	for time.Since(start) < maxWaitTime {
		status, err := client.POE().GetPortStatus(ctx, portID)
		if err == nil && status.Status != "disabled" && status.PowerW > 0 {
			if h.verbose {
				log.Printf("Port %d recovered after %v", portID, time.Since(start))
			}
			return nil
		}

		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("port %d did not recover within %v", portID, maxWaitTime)
}

// RunTestWithRestore runs a test function with automatic state restoration
func (h *TestHelper) RunTestWithRestore(testName, switchName string, testPorts []int, testFunc func(*netgear.Client, []int) error) TestResult {
	startTime := time.Now()
	result := TestResult{
		TestName:   testName,
		SwitchName: switchName,
		StartTime:  startTime,
		Details:    make(map[string]interface{}),
	}

	// Get authenticated client from cache
	client, err := h.GetClientForTest(switchName)
	if err != nil {
		result.Error = fmt.Errorf("failed to get authenticated client: %w", err)
		result.EndTime = time.Now()
		result.Duration = result.EndTime.Sub(result.StartTime)
		return result
	}

	// Capture initial states
	var poeStates map[int]POEState
	var portStates map[int]PortState

	if containsPOETest(testName) {
		poeStates, err = h.CapturePOEState(client, testPorts)
		if err != nil {
			result.Error = fmt.Errorf("failed to capture POE state: %w", err)
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result
		}
		result.Details["initial_poe_states"] = poeStates
	}

	if containsPortTest(testName) {
		portStates, err = h.CapturePortState(client, testPorts)
		if err != nil {
			result.Error = fmt.Errorf("failed to capture port state: %w", err)
			result.EndTime = time.Now()
			result.Duration = result.EndTime.Sub(result.StartTime)
			return result
		}
		result.Details["initial_port_states"] = portStates
	}

	// Run the actual test
	testErr := testFunc(client, testPorts)

	// Restore states regardless of test result
	var restoreErrors []error

	if poeStates != nil {
		if err := h.RestorePOEState(client, poeStates); err != nil {
			restoreErrors = append(restoreErrors, fmt.Errorf("POE restore: %w", err))
		}
	}

	if portStates != nil {
		if err := h.RestorePortState(client, portStates); err != nil {
			restoreErrors = append(restoreErrors, fmt.Errorf("Port restore: %w", err))
		}
	}

	// Set final result
	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	if testErr != nil {
		result.Error = testErr
	} else if len(restoreErrors) > 0 {
		result.Error = fmt.Errorf("test passed but restoration failed: %v", restoreErrors)
	} else {
		result.Passed = true
	}

	result.Details["restore_errors"] = restoreErrors

	return result
}

// GenerateTestReport creates a formatted test report
func (h *TestHelper) GenerateTestReport(results []TestResult) *TestReport {
	report := &TestReport{
		Results:    results,
		TotalTests: len(results),
	}

	if len(results) > 0 {
		report.StartTime = results[0].StartTime
		report.EndTime = results[len(results)-1].EndTime

		for _, result := range results {
			if result.Passed {
				report.PassedTests++
			} else {
				report.FailedTests++
			}
		}
	}

	return report
}

// PrintTestReport prints a formatted test report
func (h *TestHelper) PrintTestReport(report *TestReport) {
	fmt.Printf("\n=== Test Suite Report ===\n")
	fmt.Printf("Start Time: %s\n", report.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("End Time: %s\n", report.EndTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("Duration: %v\n", report.EndTime.Sub(report.StartTime))
	fmt.Printf("Total Tests: %d\n", report.TotalTests)
	fmt.Printf("Passed: %d\n", report.PassedTests)
	fmt.Printf("Failed: %d\n", report.FailedTests)
	fmt.Printf("Success Rate: %.1f%%\n", float64(report.PassedTests)/float64(report.TotalTests)*100)

	if report.FailedTests > 0 {
		fmt.Printf("\n=== Failed Tests ===\n")
		for _, result := range report.Results {
			if !result.Passed {
				fmt.Printf("- %s (%s): %v\n", result.TestName, result.SwitchName, result.Error)
			}
		}
	}

	if h.verbose {
		fmt.Printf("\n=== Detailed Results ===\n")
		for _, result := range report.Results {
			status := "PASS"
			if !result.Passed {
				status = "FAIL"
			}
			fmt.Printf("%s: %s (%s) - %v\n", status, result.TestName, result.SwitchName, result.Duration)
		}
	}
}

// Helper functions

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func containsPOETest(testName string) bool {
	poeKeywords := []string{"poe", "power", "cycle"}
	testNameLower := strings.ToLower(testName)
	for _, keyword := range poeKeywords {
		if strings.Contains(testNameLower, keyword) {
			return true
		}
	}
	return false
}

func containsPortTest(testName string) bool {
	portKeywords := []string{"port", "speed", "flow", "rate", "limit"}
	testNameLower := strings.ToLower(testName)
	for _, keyword := range portKeywords {
		if strings.Contains(testNameLower, keyword) {
			return true
		}
	}
	return false
}