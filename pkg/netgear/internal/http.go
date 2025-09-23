package internal

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPClient wraps the standard HTTP client with netgear-specific functionality
type HTTPClient struct {
	client  *http.Client
	baseURL string
	verbose bool
}

// NewHTTPClient creates a new HTTP client for netgear switch communication
func NewHTTPClient(address string, timeout time.Duration, verbose bool) *HTTPClient {
	// Ensure address has protocol
	if !strings.HasPrefix(address, "http://") && !strings.HasPrefix(address, "https://") {
		address = "http://" + address
	}

	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Don't follow redirects, we want to handle them ourselves
				return http.ErrUseLastResponse
			},
		},
		baseURL: address,
		verbose: verbose,
	}
}

// Get performs a GET request
func (h *HTTPClient) Get(ctx context.Context, path string, headers map[string]string) (*http.Response, error) {
	return h.request(ctx, "GET", path, nil, headers)
}

// Post performs a POST request
func (h *HTTPClient) Post(ctx context.Context, path string, data url.Values, headers map[string]string) (*http.Response, error) {
	var body io.Reader
	if data != nil {
		body = strings.NewReader(data.Encode())
		if headers == nil {
			headers = make(map[string]string)
		}
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	}
	
	return h.request(ctx, "POST", path, body, headers)
}

// request is the internal method for making HTTP requests
func (h *HTTPClient) request(ctx context.Context, method, path string, body io.Reader, headers map[string]string) (*http.Response, error) {
	fullURL := h.baseURL + path
	
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Set default User-Agent if not provided
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "ntgrrc-library/1.0")
	}

	if h.verbose {
		fmt.Printf("Making %s request to %s\n", method, fullURL)
		if body != nil {
			fmt.Printf("Request body: %v\n", body)
		}
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if h.verbose {
		fmt.Printf("Response status: %s\n", resp.Status)
	}

	return resp, nil
}

// ReadBody reads and returns the response body as a string
func (h *HTTPClient) ReadBody(resp *http.Response) (string, error) {
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	bodyStr := string(body)
	if h.verbose && len(bodyStr) > 0 {
		// Only show first 500 characters to avoid flooding logs
		preview := bodyStr
		if len(preview) > 500 {
			preview = preview[:500] + "..."
		}
		fmt.Printf("Response body preview: %s\n", preview)
	}

	return bodyStr, nil
}

// EncryptPassword encrypts a password using MD5 (for legacy netgear compatibility)
func EncryptPassword(password string) string {
	hash := md5.Sum([]byte(password))
	return fmt.Sprintf("%x", hash)
}

// EncryptPasswordWithSeed encrypts password using seed value and special merge algorithm
func EncryptPasswordWithSeed(password, seedValue string) string {
	mergedStr := specialMerge(password, seedValue)
	hash := md5.Sum([]byte(mergedStr))
	return fmt.Sprintf("%x", hash)
}

// specialMerge implements the special interleaving algorithm from Netgear's login.js
func specialMerge(password, seedValue string) string {
	var result strings.Builder
	maxLen := len(password)
	if len(seedValue) > maxLen {
		maxLen = len(seedValue)
	}
	
	for i := 0; i < maxLen; i++ {
		if i < len(password) {
			result.WriteByte(password[i])
		}
		if i < len(seedValue) {
			result.WriteByte(seedValue[i])
		}
	}
	
	return result.String()
}

// GetRedirectLocation extracts the redirect location from a response
func GetRedirectLocation(resp *http.Response) string {
	return resp.Header.Get("Location")
}

// IsRedirect checks if the response is a redirect
func IsRedirect(resp *http.Response) bool {
	return resp.StatusCode >= 300 && resp.StatusCode < 400
}

// SetVerbose enables or disables verbose logging
func (h *HTTPClient) SetVerbose(verbose bool) {
	h.verbose = verbose
}

// GetBaseURL returns the base URL
func (h *HTTPClient) GetBaseURL() string {
	return h.baseURL
}