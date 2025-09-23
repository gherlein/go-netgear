package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/corbym/gocrest/is"
	"github.com/corbym/gocrest/then"
)

// Integration tests that test complete workflows

func TestCompleteLoginWorkflow(t *testing.T) {
	models := []NetgearModel{GS305EP, GS316EP}

	for _, model := range models {
		t.Run(string(model), func(t *testing.T) {
			// Setup
			mock := NewMockHTTPServer(model)
			defer mock.Close()

			tokenDir := createTempTokenDir(t)
			defer os.RemoveAll(tokenDir)

			args := createTestGlobalOptions(false, true, MarkdownFormat)
			args.TokenDir = tokenDir

			parsedURL, _ := url.Parse(mock.URL())
			host := parsedURL.Host

			// Execute login
			loginCmd := &LoginCommand{
				Address:  host,
				Password: "test-password",
			}

			err := loginCmd.Run(args)
			then.AssertThat(t, err, is.Nil())

			// Verify token was stored
			tokenFile := tokenFilename(tokenDir, host)
			_, err = os.Stat(tokenFile)
			then.AssertThat(t, err, is.Nil())

			// Verify subsequent authenticated request works
			poeCmd := &PoeStatusCommand{Address: host}
			err = poeCmd.Run(args)
			then.AssertThat(t, err, is.Nil())
		})
	}
}

func TestPOEManagementWorkflow(t *testing.T) {
	// Setup
	mock := NewMockHTTPServer(GS305EP)
	defer mock.Close()

	tokenDir := createTempTokenDir(t)
	defer os.RemoveAll(tokenDir)

	args := createTestGlobalOptions(false, true, MarkdownFormat)
	args.TokenDir = tokenDir

	parsedURL, _ := url.Parse(mock.URL())
	host := parsedURL.Host

	// Setup authentication
	writeTestToken(t, tokenDir, host, mock.sessionToken, GS305EP)

	// 1. Get POE Status
	statusCmd := &PoeStatusCommand{Address: host}
	err := statusCmd.Run(args)
	then.AssertThat(t, err, is.Nil())

	// 2. Get POE Settings
	settingsCmd := &PoeShowSettingsCommand{Address: host}
	err = settingsCmd.Run(args)
	then.AssertThat(t, err, is.Nil())

	// 3. Update POE Settings
	setCmd := &PoeSetConfigCommand{
		Address:  host,
		Ports:    []int{1, 2},
		PortPwr:  "enable",
		PwrMode:  "802.3at",
		PortPrio: "high",
	}
	err = setCmd.Run(args)
	then.AssertThat(t, err, is.Nil())

	// 4. Cycle POE Power
	cycleCmd := &PoeCyclePowerCommand{
		Address: host,
		Ports:   []int{1},
	}
	err = cycleCmd.Run(args)
	then.AssertThat(t, err, is.Nil())

	// Verify all endpoints were called
	requests := mock.GetRequests()
	endpoints := make(map[string]bool)
	for _, req := range requests {
		endpoints[req.URL] = true
	}

	// Check that various endpoints were accessed
	foundStatus := false
	foundSettings := false

	for endpoint := range endpoints {
		if strings.Contains(endpoint, "getPoePortStatus") {
			foundStatus = true
		}
		if strings.Contains(endpoint, "PoEPortConfig") {
			foundSettings = true
		}
	}

	then.AssertThat(t, foundStatus, is.True())
	then.AssertThat(t, foundSettings, is.True())
}

func TestPortManagementWorkflow(t *testing.T) {
	// Setup
	mock := NewMockHTTPServer(GS308EPP)
	defer mock.Close()

	tokenDir := createTempTokenDir(t)
	defer os.RemoveAll(tokenDir)

	args := createTestGlobalOptions(false, true, JsonFormat) // Test JSON output
	args.TokenDir = tokenDir

	parsedURL, _ := url.Parse(mock.URL())
	host := parsedURL.Host

	// Setup authentication
	writeTestToken(t, tokenDir, host, mock.sessionToken, GS308EPP)

	// 1. Get Port Settings
	output := captureOutput(func() {
		settingsCmd := &PortSettingsCommand{Address: host}
		err := settingsCmd.Run(args)
		then.AssertThat(t, err, is.Nil())
	})

	// Verify JSON output
	if !strings.Contains(output, `"port_settings":`) {
		t.Errorf("Expected JSON output to contain port_settings, got: %s", output)
	}

	// 2. Update Port Settings
	portName := "Test Port"
	setCmd := &PortSetCommand{
		Address:     host,
		Ports:       []int{1, 2},
		Name:        &portName,
		Speed:       "100M full",
		FlowControl: "On",
	}
	err := setCmd.Run(args)
	then.AssertThat(t, err, is.Nil())
}

func TestErrorHandlingWorkflow(t *testing.T) {
	tests := []struct {
		name           string
		setupFunc      func() (*MockHTTPServer, string, string)
		commandFunc    func(host string) error
		expectedError  string
	}{
		{
			name: "Unauthenticated POE request",
			setupFunc: func() (*MockHTTPServer, string, string) {
				mock := NewMockHTTPServer(GS305EP)
				parsedURL, _ := url.Parse(mock.URL())
				tokenDir := createTempTokenDir(t)
				// Don't set up token
				return mock, parsedURL.Host, tokenDir
			},
			commandFunc: func(host string) error {
				cmd := &PoeStatusCommand{Address: host}
				args := createTestGlobalOptions(false, true, MarkdownFormat)
				return cmd.Run(args)
			},
			expectedError: "login",
		},
		{
			name: "Invalid model detection",
			setupFunc: func() (*MockHTTPServer, string, string) {
				// Create a mock that returns unrecognized model
				mock := &MockHTTPServer{
					model:        GS305EP,
					sessionToken: "test-session-token",
					gambitToken:  "test-gambit-token",
				}
				
				mock.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/" {
						w.Write([]byte(`<html><title>Unknown Switch</title></html>`))
					}
				}))
				
				parsedURL, _ := url.Parse(mock.URL())
				tokenDir := createTempTokenDir(t)
				return mock, parsedURL.Host, tokenDir
			},
			commandFunc: func(host string) error {
				cmd := &LoginCommand{
					Address:  host,
					Password: "test",
				}
				args := createTestGlobalOptions(false, true, MarkdownFormat)
				return cmd.Run(args)
			},
			expectedError: "auto-detect",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock, host, tokenDir := tt.setupFunc()
			defer mock.Close()
			defer os.RemoveAll(tokenDir)

			err := tt.commandFunc(host)
			then.AssertThat(t, err, is.Not(is.Nil()))
			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error to contain %q, got: %v", tt.expectedError, err)
			}
		})
	}
}

func TestMultiModelCompatibility(t *testing.T) {
	models := []struct {
		model        NetgearModel
		authType     string
		statusPath   string
		settingsPath string
	}{
		{
			model:        GS305EP,
			authType:     "cookie",
			statusPath:   "/getPoePortStatus.cgi",
			settingsPath: "/PoEPortConfig.cgi",
		},
		{
			model:        GS316EP,
			authType:     "gambit",
			statusPath:   "/iss/specific/poePortStatus.html",
			settingsPath: "/iss/specific/poePortConf.html",
		},
	}

	for _, tt := range models {
		t.Run(string(tt.model), func(t *testing.T) {
			// Setup
			mock := NewMockHTTPServer(tt.model)
			defer mock.Close()

			tokenDir := createTempTokenDir(t)
			defer os.RemoveAll(tokenDir)

			args := createTestGlobalOptions(false, true, MarkdownFormat)
			args.TokenDir = tokenDir

			parsedURL, _ := url.Parse(mock.URL())
			host := parsedURL.Host

			// Login
			loginCmd := &LoginCommand{
				Address:  host,
				Password: "test-password",
			}
			err := loginCmd.Run(args)
			then.AssertThat(t, err, is.Nil())

			// Get POE Status
			statusCmd := &PoeStatusCommand{Address: host}
			err = statusCmd.Run(args)
			then.AssertThat(t, err, is.Nil())

			// Verify correct endpoints were used
			requests := mock.GetRequests()
			foundStatus := false
			foundAuth := false

			for _, req := range requests {
				if strings.Contains(req.URL, tt.statusPath) {
					foundStatus = true
					
					// Check authentication method
					if tt.authType == "cookie" {
						then.AssertThat(t, req.Header.Get("Cookie"), is.Not(is.EqualTo("")))
					} else if tt.authType == "gambit" {
						if !strings.Contains(req.URL, "Gambit=") {
							t.Errorf("Expected URL to contain Gambit parameter, got: %s", req.URL)
						}
					}
					foundAuth = true
				}
			}

			then.AssertThat(t, foundStatus, is.True())
			then.AssertThat(t, foundAuth, is.True())
		})
	}
}

func TestConcurrentCommands(t *testing.T) {
	// Setup
	mock := NewMockHTTPServer(GS305EP)
	defer mock.Close()

	tokenDir := createTempTokenDir(t)
	defer os.RemoveAll(tokenDir)

	parsedURL, _ := url.Parse(mock.URL())
	host := parsedURL.Host

	// Setup authentication
	writeTestToken(t, tokenDir, host, mock.sessionToken, GS305EP)

	// Run multiple commands concurrently
	done := make(chan error, 3)

	go func() {
		args := createTestGlobalOptions(false, true, MarkdownFormat)
		args.TokenDir = tokenDir
		cmd := &PoeStatusCommand{Address: host}
		done <- cmd.Run(args)
	}()

	go func() {
		args := createTestGlobalOptions(false, true, JsonFormat)
		args.TokenDir = tokenDir
		cmd := &PoeShowSettingsCommand{Address: host}
		done <- cmd.Run(args)
	}()

	go func() {
		args := createTestGlobalOptions(false, true, MarkdownFormat)
		args.TokenDir = tokenDir
		cmd := &PortSettingsCommand{Address: host}
		done <- cmd.Run(args)
	}()

	// Wait for all commands to complete
	for i := 0; i < 3; i++ {
		err := <-done
		then.AssertThat(t, err, is.Nil())
	}

	// Verify multiple requests were made
	requests := mock.GetRequests()
	then.AssertThat(t, len(requests), is.GreaterThanOrEqualTo(3))
}

func TestOutputFormatConsistency(t *testing.T) {
	// Setup
	mock := NewMockHTTPServer(GS305EP)
	defer mock.Close()

	tokenDir := createTempTokenDir(t)
	defer os.RemoveAll(tokenDir)

	parsedURL, _ := url.Parse(mock.URL())
	host := parsedURL.Host

	// Setup authentication
	writeTestToken(t, tokenDir, host, mock.sessionToken, GS305EP)

	// Test both output formats
	formats := []OutputFormat{MarkdownFormat, JsonFormat}

	for _, format := range formats {
		t.Run(string(format), func(t *testing.T) {
			output := captureOutput(func() {
				args := createTestGlobalOptions(false, true, format)
				args.TokenDir = tokenDir
				cmd := &PoeStatusCommand{Address: host}
				err := cmd.Run(args)
				then.AssertThat(t, err, is.Nil())
			})

			if format == MarkdownFormat {
				// Check for markdown table structure
				if !strings.Contains(output, "|") {
					t.Errorf("Expected markdown table format, got: %s", output)
				}
				if !strings.Contains(output, "Port ID") {
					t.Errorf("Expected Port ID header, got: %s", output)
				}
				if !strings.Contains(output, "---") {
					t.Errorf("Expected table separator, got: %s", output)
				}
			} else {
				// Check for valid JSON
				if !strings.Contains(output, "{") {
					t.Errorf("Expected JSON format, got: %s", output)
				}
				if !strings.Contains(output, `"poe_status":`) {
					t.Errorf("Expected poe_status key, got: %s", output)
				}
				
				// Verify it's valid JSON
				var result map[string]interface{}
				err := json.Unmarshal([]byte(output), &result)
				then.AssertThat(t, err, is.Nil())
			}
		})
	}
}