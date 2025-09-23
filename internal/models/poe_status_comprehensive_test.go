package models

import (
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/corbym/gocrest/is"
	"github.com/corbym/gocrest/then"
)

func TestPoeStatusCommand_Run(t *testing.T) {
	tests := []struct {
		name         string
		model        types.NetgearModel
		authenticated bool
		expectError  bool
	}{
		{
			name:         "Successful status retrieval GS305EP",
			model:        GS305EP,
			authenticated: true,
			expectError:  false,
		},
		{
			name:         "Successful status retrieval GS316EP",
			model:        GS316EP,
			authenticated: true,
			expectError:  false,
		},
		{
			name:         "Unauthenticated request",
			model:        GS305EP,
			authenticated: false,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mock := NewMockHTTPServer(tt.model)
			defer mock.Close()

			tokenDir := createTempTokenDir(t)
			defer os.RemoveAll(tokenDir)

			args := createTestGlobalOptions(false, true, formatter.MarkdownFormat)
			args.TokenDir = tokenDir

			parsedURL, _ := url.Parse(mock.URL())
			host := parsedURL.Host

			if tt.authenticated {
				if common.IsModel30x(tt.model) {
					writeTestToken(t, tokenDir, portSet.Address, mock.sessionToken, tt.model)
				} else {
					writeTestToken(t, tokenDir, portSet.Address, mock.gambitToken, tt.model)
				}
			}

			cmd := &PoeStatusCommand{
				Address: host,
			}

			// Execute
			err := cmd.Run(args)

			// Verify
			if tt.expectError {
				then.AssertThat(t, err, is.Not(is.Nil()))
			} else {
				then.AssertThat(t, err, is.Nil())
			}
		})
	}
}

func TestFindPortStatusInGs30xEPxHtml(t *testing.T) {
	html := `
	<html>
	<ul>
		<li class="poePortStatusListItem">
			<input type="hidden" class="port" value="1"/>
			<span class="poe-port-index"><span>1 - Camera Port</span></span>
			<span class="poe-power-mode"><span>Delivering Power</span></span>
			<span class="poe-portPwr-width"><span>ml003@3@</span></span>
			<div class="poe_port_status">
				<div><div>
					<span>Voltage:</span><span>48</span>
					<span>Current:</span><span>150</span>
					<span>Power:</span><span>7.20</span>
					<span>Temperature:</span><span>35</span>
					<span>Error:</span><span>No Error</span>
				</div></div>
			</div>
		</li>
		<li class="poePortStatusListItem">
			<input type="hidden" class="port" value="2"/>
			<span class="poe-port-index"><span>2</span></span>
			<span class="poe-power-mode"><span>Searching</span></span>
			<span class="poe-portPwr-width"><span>ml003@0@</span></span>
			<div class="poe_port_status">
				<div><div>
					<span>Voltage:</span><span>0</span>
					<span>Current:</span><span>0</span>
					<span>Power:</span><span>0.00</span>
					<span>Temperature:</span><span>30</span>
					<span>Error:</span><span>Power Denied</span>
				</div></div>
			</div>
		</li>
	</ul>
	</html>`

	statuses, err := findPortStatusInGs30xEPxHtml(strings.NewReader(html))

	then.AssertThat(t, err, is.Nil())
	then.AssertThat(t, len(statuses), is.EqualTo(2))

	// Verify first port
	port1 := statuses[0]
	then.AssertThat(t, port1.PortIndex, is.EqualTo(int8(1)))
	then.AssertThat(t, port1.PortName, is.EqualTo("Camera Port"))
	then.AssertThat(t, port1.PoePortStatus, is.EqualTo("Delivering Power"))
	then.AssertThat(t, port1.PoePowerClass, is.EqualTo("3"))
	then.AssertThat(t, port1.VoltageInVolt, is.EqualTo(int32(48)))
	then.AssertThat(t, port1.CurrentInMilliAmps, is.EqualTo(int32(150)))
	then.AssertThat(t, port1.PowerInWatt, is.EqualTo(float32(7.20)))
	then.AssertThat(t, port1.TemperatureInCelsius, is.EqualTo(int32(35)))
	then.AssertThat(t, port1.ErrorStatus, is.EqualTo("No Error"))

	// Verify second port
	port2 := statuses[1]
	then.AssertThat(t, port2.PortIndex, is.EqualTo(int8(2)))
	then.AssertThat(t, port2.PortName, is.EqualTo(""))
	then.AssertThat(t, port2.PoePortStatus, is.EqualTo("Searching"))
	then.AssertThat(t, port2.PoePowerClass, is.EqualTo("0"))
	then.AssertThat(t, port2.ErrorStatus, is.EqualTo("Power Denied"))
}

func TestFindPortStatusInGs316EPxHtml(t *testing.T) {
	html := `
	<html>
	<div class="port-wrap">
		<span class="port-number">1 - Access Point</span>
		<span class="Status-text">Delivering Power</span>
		<span class="Class-text">ml003@4@</span>
		<p class="OutputVoltage-text">53</p>
		<p class="OutputCurrent-text">280</p>
		<p class="OutputPower-text">14.84</p>
		<p class="Temperature-text">42</p>
		<p class="Fault-Status-text">No Error</p>
	</div>
	<div class="port-wrap">
		<span class="port-number">16</span>
		<span class="Status-text">Disabled</span>
		<span class="Class-text"></span>
		<p class="OutputVoltage-text">0</p>
		<p class="OutputCurrent-text">0</p>
		<p class="OutputPower-text">0.00</p>
		<p class="Temperature-text">30</p>
		<p class="Fault-Status-text">Port Disabled</p>
	</div>
	</html>`

	statuses, err := findPortStatusInGs316EPxHtml(strings.NewReader(html))

	then.AssertThat(t, err, is.Nil())
	then.AssertThat(t, len(statuses), is.EqualTo(2))

	// Verify first port
	port1 := statuses[0]
	then.AssertThat(t, port1.PortIndex, is.EqualTo(int8(1)))
	then.AssertThat(t, port1.PortName, is.EqualTo("Access Point"))
	then.AssertThat(t, port1.PoePortStatus, is.EqualTo("Delivering Power"))
	then.AssertThat(t, port1.PoePowerClass, is.EqualTo("4"))
	then.AssertThat(t, port1.VoltageInVolt, is.EqualTo(int32(53)))
	then.AssertThat(t, port1.CurrentInMilliAmps, is.EqualTo(int32(280)))
	then.AssertThat(t, port1.PowerInWatt, is.EqualTo(float32(14.84)))
	then.AssertThat(t, port1.TemperatureInCelsius, is.EqualTo(int32(42)))
	then.AssertThat(t, port1.ErrorStatus, is.EqualTo("No Error"))

	// Verify last port
	port16 := statuses[1]
	then.AssertThat(t, port16.PortIndex, is.EqualTo(int8(16)))
	then.AssertThat(t, port16.PortName, is.EqualTo(""))
	then.AssertThat(t, port16.PoePortStatus, is.EqualTo("Disabled"))
}

func TestGetPowerClassFromI18nString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ml003@0@", "0"},
		{"ml003@1@", "1"},
		{"ml003@2@", "2"},
		{"ml003@3@", "3"},
		{"ml003@4@", "4"},
		{"", ""},
		{"no-at-signs", ""},
		{"@only-one", ""},
		{"@multiple@at@signs@", "multiple"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := getPowerClassFromI18nString(tt.input)
			then.AssertThat(t, result, is.EqualTo(tt.expected))
		})
	}
}

func TestParsePortIdAndName(t *testing.T) {
	tests := []struct {
		input        string
		expectedId   int8
		expectedName string
	}{
		{"1 - Camera", 1, "Camera"},
		{"2 - Access Point", 2, "Access Point"},
		{"10 - ", 10, ""},
		{"5", 5, ""},
		{"16 - Multi - Word - Name", 16, "Multi - Word - Name"},
		{"1\u00a0-\u00a0Camera", 1, "Camera"}, // Non-breaking spaces
		{"", 0, ""},
		{"invalid", 0, ""},
		{"99 -   Spaces   ", 99, "Spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			id, name := parsePortIdAndName(tt.input)
			then.AssertThat(t, id, is.EqualTo(tt.expectedId))
			then.AssertThat(t, name, is.EqualTo(tt.expectedName))
		})
	}
}

func TestPrettyPrintPoePortStatus_Markdown(t *testing.T) {
	statuses := []PoePortStatus{
		{
			PortIndex:            1,
			PortName:             "Camera",
			PoePortStatus:        "Delivering Power",
			PoePowerClass:        "3",
			VoltageInVolt:        48,
			CurrentInMilliAmps:   150,
			PowerInWatt:          7.20,
			TemperatureInCelsius: 35,
			ErrorStatus:          "No Error",
		},
		{
			PortIndex:            2,
			PortName:             "",
			PoePortStatus:        "Searching",
			PoePowerClass:        "",
			VoltageInVolt:        0,
			CurrentInMilliAmps:   0,
			PowerInWatt:          0.00,
			TemperatureInCelsius: 30,
			ErrorStatus:          "Power Denied",
		},
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	prettyPrintPoePortStatus(formatter.MarkdownFormat, statuses)

	w.Close()
	output := make([]byte, 1024)
	n, _ := r.Read(output)
	os.Stdout = oldStdout

	outputStr := string(output[:n])

	// Verify markdown table format
	expectedContent := []string{
		"| Port ID | Port Name | Status",
		"|---------|",
		"| 1       | Camera    | Delivering Power",
		"| 2       |           | Searching",
		"7.20",
		"Power Denied",
	}
	
	for _, expected := range expectedContent {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Expected output to contain %q, got: %s", expected, outputStr)
		}
	}
}

func TestPrettyPrintPoePortStatus_JSON(t *testing.T) {
	statuses := []PoePortStatus{
		{
			PortIndex:            1,
			PortName:             "Test Port",
			PoePortStatus:        "Active",
			PoePowerClass:        "4",
			VoltageInVolt:        53,
			CurrentInMilliAmps:   300,
			PowerInWatt:          15.90,
			TemperatureInCelsius: 40,
			ErrorStatus:          "No Error",
		},
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	prettyPrintPoePortStatus(formatter.JsonFormat, statuses)

	w.Close()
	output := make([]byte, 1024)
	n, _ := r.Read(output)
	os.Stdout = oldStdout

	outputStr := string(output[:n])

	// Verify JSON format
	expectedJSONContent := []string{
		`"poe_status":`,
		`"Port ID": "1"`,
		`"Port Name": "Test Port"`,
		`"Status": "Active"`,
		`"PortPwr (W)": "15.90"`,
	}
	
	for _, expected := range expectedJSONContent {
		if !strings.Contains(outputStr, expected) {
			t.Errorf("Expected JSON to contain %q, got: %s", expected, outputStr)
		}
	}
}

func TestRequestPoeStatus_ErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		setupMock     func() *MockHTTPServer
		expectError   bool
		errorContains string
	}{
		{
			name: "Login required",
			setupMock: func() *MockHTTPServer {
				mock := NewMockHTTPServer(GS305EP)
				// Don't set up authentication
				return mock
			},
			expectError:   true,
			errorContains: "please, (re-)login first",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mock := tt.setupMock()
			defer mock.Close()

			tokenDir := createTempTokenDir(t)
			defer os.RemoveAll(tokenDir)

			args := createTestGlobalOptions(false, true, formatter.MarkdownFormat)
			args.TokenDir = tokenDir

			parsedURL, _ := url.Parse(mock.URL())
			host := parsedURL.Host

			// Execute
			_, err := requestPoeStatus(args, host)

			// Verify
			if tt.expectError {
				then.AssertThat(t, err, is.Not(is.Nil()))
				if tt.errorContains != "" {
					if !strings.Contains(err.Error(), tt.errorContains) {
						t.Errorf("Expected error to contain %q, got: %v", tt.errorContains, err)
					}
				}
			} else {
				then.AssertThat(t, err, is.Nil())
			}
		})
	}
}

func TestCheckIsLoginRequired(t *testing.T) {
	tests := []struct {
		name     string
		response string
		expected bool
	}{
		{
			name:     "Empty response",
			response: "",
			expected: true,
		},
		{
			name:     "Short response",
			response: "short",
			expected: true,
		},
		{
			name:     "Contains login.cgi",
			response: `<html><a href="/login.cgi">Login</a></html>`,
			expected: true,
		},
		{
			name:     "Contains wmi/login",
			response: `<html><a href="/wmi/login">Login</a></html>`,
			expected: true,
		},
		{
			name:     "Contains redirect.html",
			response: `<html><script>location.href="/redirect.html"</script></html>`,
			expected: true,
		},
		{
			name:     "Valid response",
			response: `<html><body>Valid POE status data here with lots of content</body></html>`,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := common.CheckIsLoginRequired(tt.response)
			then.AssertThat(t, result, is.EqualTo(tt.expected))
		})
	}
}

func TestRequestPoePortStatusPage_ModelSpecific(t *testing.T) {
	tests := []struct {
		name         string
		model        types.NetgearModel
		expectedPath string
	}{
		{
			name:         "GS305EP status page",
			model:        GS305EP,
			expectedPath: "/getPoePortStatus.cgi",
		},
		{
			name:         "GS316EP status page",
			model:        GS316EP,
			expectedPath: "/iss/specific/poePortStatus.html?GetData=TRUE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mock := NewMockHTTPServer(tt.model)
			defer mock.Close()

			tokenDir := createTempTokenDir(t)
			defer os.RemoveAll(tokenDir)

			args := createTestGlobalOptions(false, true, formatter.MarkdownFormat)
			args.TokenDir = tokenDir

			parsedURL, _ := url.Parse(mock.URL())
			host := parsedURL.Host

			// Setup authentication
			if common.IsModel30x(tt.model) {
				writeTestToken(t, tokenDir, portSet.Address, mock.sessionToken, tt.model)
			} else {
				writeTestToken(t, tokenDir, portSet.Address, mock.gambitToken, tt.model)
			}

			// Execute
			_, err := requestPoePortStatusPage(args, host)
			then.AssertThat(t, err, is.Nil())

			// Verify correct endpoint was called
			requests := mock.GetRequests()
			then.AssertThat(t, len(requests), is.GreaterThan(0))
			
			lastReq := requests[len(requests)-1]
			if !strings.Contains(lastReq.URL, tt.expectedPath) {
				t.Errorf("Expected URL to contain %q, got: %s", tt.expectedPath, lastReq.URL)
			}
		})
	}
}