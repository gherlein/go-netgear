package models

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/corbym/gocrest/is"
	"github.com/corbym/gocrest/then"
)

func TestIsModel30x(t *testing.T) {
	tests := []struct {
		model    types.NetgearModel
		expected bool
	}{
		{GS305EP, true},
		{GS305EPP, true},
		{GS308EP, true},
		{GS308EPP, true},
		{GS30xEPx, true},
		{GS316EP, false},
		{GS316EPP, false},
		{types.NetgearModel("Unknown"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.model), func(t *testing.T) {
			result := common.IsModel30x(tt.model)
			then.AssertThat(t, result, is.EqualTo(tt.expected))
		})
	}
}

func TestIsModel316(t *testing.T) {
	tests := []struct {
		model    types.NetgearModel
		expected bool
	}{
		{GS305EP, false},
		{GS305EPP, false},
		{GS308EP, false},
		{GS308EPP, false},
		{GS30xEPx, false},
		{GS316EP, true},
		{GS316EPP, true},
		{types.NetgearModel("Unknown"), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.model), func(t *testing.T) {
			result := common.IsModel316(tt.model)
			then.AssertThat(t, result, is.EqualTo(tt.expected))
		})
	}
}

func TestIsSupportedModelComprehensive(t *testing.T) {
	tests := []struct {
		modelName string
		expected  bool
	}{
		{"GS305EP", true},
		{"GS305EPP", true},
		{"GS308EP", true},
		{"GS308EPP", true},
		{"GS316EP", true},
		{"GS316EPP", true},
		{"GS30xEPx", true},
		{"UnknownModel", false},
		{"", false},
		{"GS305", false}, // Different series
	}

	for _, tt := range tests {
		t.Run(tt.modelName, func(t *testing.T) {
			result := isSupportedModel(tt.modelName)
			then.AssertThat(t, result, is.EqualTo(tt.expected))
		})
	}
}

func TestDetectNetgearModelFromResponseComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected types.NetgearModel
	}{
		{
			name:     "GS316EPP detection",
			body:     `<html><title>GS316EPP Switch</title></html>`,
			expected: GS316EPP,
		},
		{
			name:     "GS316EP detection",
			body:     `<html><title>GS316EP Management</title></html>`,
			expected: GS316EP,
		},
		{
			name:     "GS30x series detection",
			body:     `<html><title>Redirect to Login</title></html>`,
			expected: GS30xEPx,
		},
		{
			name:     "Case insensitive title tag",
			body:     `<html><TITLE>GS316EP</TITLE></html>`,
			expected: GS316EP,
		},
		{
			name:     "No model detected",
			body:     `<html><title>Unknown Switch</title></html>`,
			expected: "",
		},
		{
			name:     "Empty response",
			body:     "",
			expected: "",
		},
		{
			name:     "GS316EPP appears before GS316EP in content",
			body:     `<html><title>GS316EPP appears before GS316EP</title></html>`,
			expected: GS316EPP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectNetgearModelFromResponse(tt.body)
			then.AssertThat(t, result, is.EqualTo(tt.expected))
		})
	}
}

func TestDetectNetgearModel(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   string
		responseCode   int
		serverError    bool
		verbose        bool
		expectedModel  types.NetgearModel
		expectError    bool
	}{
		{
			name:          "Successful GS316EP detection",
			responseBody:  `<html><title>GS316EP</title></html>`,
			responseCode:  200,
			expectedModel: GS316EP,
			expectError:   false,
		},
		{
			name:          "Successful GS30x detection",
			responseBody:  `<html><title>Redirect to Login</title></html>`,
			responseCode:  200,
			expectedModel: GS30xEPx,
			expectError:   false,
		},
		{
			name:          "Non-200 response but successful detection",
			responseBody:  `<html><title>GS316EP</title></html>`,
			responseCode:  302,
			expectedModel: GS316EP,
			expectError:   false,
			verbose:       true,
		},
		{
			name:         "Server error",
			serverError:  true,
			expectError:  true,
		},
		{
			name:         "No model detected",
			responseBody: `<html><title>Unknown</title></html>`,
			responseCode: 200,
			expectError:  true,
		},
		{
			name:         "Empty response",
			responseBody: "",
			responseCode: 200,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup mock server
			var server *httptest.Server
			if tt.serverError {
				// Create a server that immediately closes connections
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					// Close the connection to simulate network error
					hj, _ := w.(http.Hijacker)
					conn, _, _ := hj.Hijack()
					conn.Close()
				}))
			} else {
				server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.responseCode)
					w.Write([]byte(tt.responseBody))
				}))
			}
			defer server.Close()

			args := &GlobalOptions{
				Verbose: tt.verbose,
			}

			// Extract host from server URL
			serverURL := server.URL
			host := serverURL[7:] // Remove "http://"

			// Execute
			model, err := DetectNetgearModel(args, host)

			// Verify
			if tt.expectError {
				then.AssertThat(t, err, is.Not(is.Nil()))
			} else {
				then.AssertThat(t, err, is.Nil())
				then.AssertThat(t, model, is.EqualTo(tt.expectedModel))
			}
		})
	}
}

func TestDetectNetgearModel_EdgeCases(t *testing.T) {
	t.Run("Invalid URL", func(t *testing.T) {
		args := &GlobalOptions{Verbose: false}
		_, err := DetectNetgearModel(args, "invalid-host-name-!@#$%")
		then.AssertThat(t, err, is.Not(is.Nil()))
	})

	t.Run("Timeout simulation", func(t *testing.T) {
		// Create a server that delays response beyond reasonable timeout
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Sleep for a very long time to simulate timeout
			time.Sleep(10 * time.Second)
			w.Write([]byte("Too late"))
		}))
		defer server.Close()

		args := &GlobalOptions{Verbose: false}
		host := server.URL[7:] // Remove "http://"
		
		// This should eventually fail with a timeout or similar error
		_, err := DetectNetgearModel(args, host)
		then.AssertThat(t, err, is.Not(is.Nil()))
	})
}

func TestDetectNetgearModel_VerboseOutput(t *testing.T) {
	// Test that verbose mode doesn't break functionality
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`<html><title>GS316EP</title></html>`))
	}))
	defer server.Close()

	args := &GlobalOptions{Verbose: true}
	host := server.URL[7:]

	model, err := DetectNetgearModel(args, host)
	
	// Should still detect model despite non-200 status
	then.AssertThat(t, err, is.Nil())
	then.AssertThat(t, model, is.EqualTo(GS316EP))
}

// Custom error for testing
type testError struct {
	message string
}

func (e *testError) Error() string {
	return e.message
}

func TestDetectNetgearModel_ErrorPropagation(t *testing.T) {
	// Test that errors from HTTP client are properly propagated
	tests := []struct {
		name        string
		setupServer func() *httptest.Server
		expectError string
	}{
		{
			name: "Connection refused",
			setupServer: func() *httptest.Server {
				// Create and immediately close server
				s := httptest.NewServer(nil)
				s.Close()
				return s
			},
			expectError: "connect",
		},
		{
			name: "Invalid response body",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Length", "100")
					w.WriteHeader(200)
					// Don't write the promised 100 bytes
					w.Write([]byte("short"))
				}))
			},
			expectError: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := tt.setupServer()
			if server.URL != "" {
				defer server.Close()
			}

			args := &GlobalOptions{Verbose: false}
			host := server.URL[7:] // Remove "http://"

			_, err := DetectNetgearModel(args, host)
			then.AssertThat(t, err, is.Not(is.Nil()))
			
			if tt.expectError != "" {
				if !strings.Contains(err.Error(), tt.expectError) {
					t.Errorf("Expected error to contain %q, got: %v", tt.expectError, err)
				}
			}
		})
	}
}