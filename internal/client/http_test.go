package client

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/corbym/gocrest/is"
	"github.com/corbym/gocrest/then"
)

func TestRequestPage(t *testing.T) {
	tests := []struct {
		name         string
		model        NetgearModel
		setupToken   bool
		expectAuth   bool
		expectError  bool
	}{
		{
			name:         "Authenticated request for GS305EP",
			model:        GS305EP,
			setupToken:   true,
			expectAuth:   true,
			expectError:  false,
		},
		{
			name:         "Unauthenticated request for GS305EP",
			model:        GS305EP,
			setupToken:   false,
			expectAuth:   false,
			expectError:  true,
		},
		{
			name:         "Authenticated request for GS316EP",
			model:        GS316EP,
			setupToken:   true,
			expectAuth:   true,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mock := NewMockHTTPServer(tt.model)
			defer mock.Close()

			tokenDir := createTempTokenDir(t)
			defer os.RemoveAll(tokenDir)

			args := createTestGlobalOptions(false, true, MarkdownFormat)
			args.TokenDir = tokenDir

			parsedURL, _ := url.Parse(mock.URL())
			host := parsedURL.Host

			if tt.setupToken {
				if IsModel30x(tt.model) {
					writeTestToken(t, tokenDir, host, mock.sessionToken, tt.model)
				} else {
					writeTestToken(t, tokenDir, host, mock.gambitToken, tt.model)
				}
			}

			// Execute
			response, err := requestPage(args, host, mock.URL()+"/getPoePortStatus.cgi")

			// Verify
			if tt.expectError {
				then.AssertThat(t, err, is.Not(is.Nil()))
			} else {
				then.AssertThat(t, err, is.Nil())
				then.AssertThat(t, response, is.Not(is.EqualTo("")))

				// Check if authentication worked
				if tt.expectAuth {
					then.AssertThat(t, checkIsLoginRequired(response), is.False())
				}
			}

			// Verify request was made with correct headers
			requests := mock.GetRequests()
			if len(requests) > 0 && tt.setupToken {
				lastReq := requests[len(requests)-1]
				if IsModel30x(tt.model) {
					then.AssertThat(t, lastReq.Header.Get("Cookie"), is.EqualTo("SID="+mock.sessionToken))
				}
			}
		})
	}
}

func TestPostPage(t *testing.T) {
	tests := []struct {
		name        string
		model       NetgearModel
		requestBody string
		expectAuth  bool
	}{
		{
			name:        "POST request for GS305EP",
			model:       GS305EP,
			requestBody: "test=data",
			expectAuth:  true,
		},
		{
			name:        "POST request for GS316EP",
			model:       GS316EP,
			requestBody: "test=data",
			expectAuth:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mock := NewMockHTTPServer(tt.model)
			defer mock.Close()

			tokenDir := createTempTokenDir(t)
			defer os.RemoveAll(tokenDir)

			args := createTestGlobalOptions(false, true, MarkdownFormat)
			args.TokenDir = tokenDir

			parsedURL, _ := url.Parse(mock.URL())
			host := parsedURL.Host

			// Setup token
			if IsModel30x(tt.model) {
				writeTestToken(t, tokenDir, host, mock.sessionToken, tt.model)
			} else {
				writeTestToken(t, tokenDir, host, mock.gambitToken, tt.model)
			}

			// Execute
			response, err := postPage(args, host, mock.URL()+"/PoEPortConfig.cgi", tt.requestBody)

			// Verify
			then.AssertThat(t, err, is.Nil())
			then.AssertThat(t, response, is.Not(is.EqualTo("")))

			// Verify request details
			requests := mock.GetRequests()
			then.AssertThat(t, len(requests), is.GreaterThan(0))
			
			lastReq := requests[len(requests)-1]
			then.AssertThat(t, lastReq.Method, is.EqualTo("POST"))
			then.AssertThat(t, lastReq.Body, is.EqualTo(tt.requestBody))
		})
	}
}

func TestDoHttpRequestAndReadResponse_ModelSpecific(t *testing.T) {
	tests := []struct {
		name          string
		model         NetgearModel
		urlPath       string
		expectGambit  bool
	}{
		{
			name:         "GS305EP uses cookie auth",
			model:        GS305EP,
			urlPath:      "/test",
			expectGambit: false,
		},
		{
			name:         "GS316EP uses Gambit parameter",
			model:        GS316EP,
			urlPath:      "/test?param=value",
			expectGambit: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			mock := NewMockHTTPServer(tt.model)
			defer mock.Close()

			tokenDir := createTempTokenDir(t)
			defer os.RemoveAll(tokenDir)

			args := createTestGlobalOptions(false, true, MarkdownFormat)
			args.TokenDir = tokenDir

			parsedURL, _ := url.Parse(mock.URL())
			host := parsedURL.Host

			// Setup token
			token := "test-token"
			writeTestToken(t, tokenDir, host, token, tt.model)

			// Execute
			fullURL := mock.URL() + tt.urlPath
			_, err := doHttpRequestAndReadResponse(args, "GET", host, fullURL, "")

			// Verify
			then.AssertThat(t, err, is.Nil())

			// Check request details
			requests := mock.GetRequests()
			then.AssertThat(t, len(requests), is.GreaterThan(0))
			
			lastReq := requests[len(requests)-1]
			
			if tt.expectGambit {
				if !strings.Contains(lastReq.URL, "Gambit="+token) {
					t.Errorf("Expected URL to contain Gambit token, got: %s", lastReq.URL)
				}
			} else {
				then.AssertThat(t, lastReq.Header.Get("Cookie"), is.EqualTo("SID="+token))
			}
		})
	}
}

func TestDoHttpRequestAndReadResponse_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setupToken  bool
		invalidHost bool
		expectError bool
	}{
		{
			name:        "Missing token",
			setupToken:  false,
			invalidHost: false,
			expectError: true,
		},
		{
			name:        "Invalid host",
			setupToken:  true,
			invalidHost: true,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenDir := createTempTokenDir(t)
			defer os.RemoveAll(tokenDir)

			args := createTestGlobalOptions(false, true, MarkdownFormat)
			args.TokenDir = tokenDir

			host := "test-host"
			if tt.invalidHost {
				host = "invalid-host-that-does-not-exist"
			}

			if tt.setupToken && !tt.invalidHost {
				writeTestToken(t, tokenDir, host, "test-token", GS305EP)
			}

			// Execute
			_, err := doHttpRequestAndReadResponse(args, "GET", host, 
				fmt.Sprintf("http://%s/test", host), "")

			// Verify
			if tt.expectError {
				then.AssertThat(t, err, is.Not(is.Nil()))
			} else {
				then.AssertThat(t, err, is.Nil())
			}
		})
	}
}

func TestDoUnauthenticatedHttpRequestAndReadResponse(t *testing.T) {
	// Setup
	mock := NewMockHTTPServer(GS305EP)
	defer mock.Close()

	args := createTestGlobalOptions(true, false, MarkdownFormat) // verbose=true

	// Execute
	response, err := doUnauthenticatedHttpRequestAndReadResponse(args, "GET", 
		mock.URL()+"/", "")

	// Verify
	then.AssertThat(t, err, is.Nil())
	then.AssertThat(t, response, is.Not(is.EqualTo("")))
	
	// Check that no authentication headers were sent
	requests := mock.GetRequests()
	then.AssertThat(t, len(requests), is.EqualTo(1))
	then.AssertThat(t, requests[0].Header.Get("Cookie"), is.EqualTo(""))
}