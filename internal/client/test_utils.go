package client

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"github.com/gherlein/go-netgear/internal/common"
	"github.com/gherlein/go-netgear/internal/formatter"
	"github.com/gherlein/go-netgear/internal/types"
)

// MockHTTPServer creates a test HTTP server that mimics switch behavior
type MockHTTPServer struct {
	server       *httptest.Server
	model        types.NetgearModel
	sessionToken string
	gambitToken  string
	requests     []RequestLog
}

type RequestLog struct {
	Method string
	URL    string
	Body   string
	Header http.Header
}

// NewMockHTTPServer creates a new mock server for testing
func NewMockHTTPServer(model types.NetgearModel) *MockHTTPServer {
	mock := &MockHTTPServer{
		model:        model,
		sessionToken: "test-session-token",
		gambitToken:  "test-gambit-token",
	}
	
	mock.server = httptest.NewServer(http.HandlerFunc(mock.handler))
	return mock
}

func (m *MockHTTPServer) handler(w http.ResponseWriter, r *http.Request) {
	// Log request
	body := ""
	if r.Body != nil {
		bodyBytes := make([]byte, 1024)
		n, _ := r.Body.Read(bodyBytes)
		body = string(bodyBytes[:n])
	}
	
	m.requests = append(m.requests, RequestLog{
		Method: r.Method,
		URL:    r.URL.String(),
		Body:   body,
		Header: r.Header,
	})
	
	// Route to appropriate handler
	switch {
	case r.URL.Path == "/" && r.Method == "GET":
		m.handleRoot(w, r)
	case r.URL.Path == "/login.cgi" && r.Method == "GET":
		m.handleLoginPage(w, r)
	case r.URL.Path == "/login.cgi" && r.Method == "POST":
		m.handleLogin(w, r)
	case r.URL.Path == "/wmi/login" && r.Method == "GET":
		m.handleLoginPage(w, r)
	case r.URL.Path == "/redirect.html" && r.Method == "POST":
		m.handleLogin316(w, r)
	case r.URL.Path == "/getPoePortStatus.cgi":
		m.handlePOEStatus(w, r)
	case r.URL.Path == "/PoEPortConfig.cgi" && r.Method == "GET":
		m.handlePOESettings(w, r)
	case r.URL.Path == "/PoEPortConfig.cgi" && r.Method == "POST":
		m.handlePOEUpdate(w, r)
	case strings.Contains(r.URL.Path, "/poePortStatus.html"):
		m.handlePOEStatus316(w, r)
	case strings.Contains(r.URL.Path, "/poePortConf.html"):
		m.handlePOESettings316(w, r)
	case r.URL.Path == "/dashboard.cgi":
		m.handlePortSettings(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (m *MockHTTPServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	var content string
	switch m.model {
	case types.GS305EP, types.GS305EPP, types.GS308EP, types.GS308EPP:
		content = loadTestFile(string(m.model), "_root.html")
		if content == "" {
			content = `<html><title>Redirect to Login</title></html>`
		}
	case types.GS316EP, types.GS316EPP:
		content = loadTestFile(string(m.model), "_root.html")
		if content == "" {
			content = `<html><title>types.GS316EP</title></html>`
		}
	}
	w.Write([]byte(content))
}

func (m *MockHTTPServer) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	var fileName string
	if common.IsModel30x(m.model) {
		fileName = "login.cgi.html"
	} else {
		fileName = "login.html"
	}
	
	content := loadTestFile(string(m.model), fileName)
	if content == "" {
		// Fallback content with seed value
		content = `<html><input id="rand" value="1234567890"/></html>`
	}
	w.Write([]byte(content))
}

func (m *MockHTTPServer) handleLogin(w http.ResponseWriter, r *http.Request) {
	// Check if password is provided
	if !strings.Contains(r.URL.String(), "password=") && r.Method == "POST" {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	
	// Set session cookie
	http.SetCookie(w, &http.Cookie{
		Name:  "SID",
		Value: m.sessionToken,
		Path:  "/",
	})
	
	w.Write([]byte(`<html>Login successful</html>`))
}

func (m *MockHTTPServer) handleLogin316(w http.ResponseWriter, r *http.Request) {
	// Return redirect page with gambit token
	content := fmt.Sprintf(`<html>
	<form>
		<input type="hidden" name="Gambit" value="%s"/>
	</form>
	</html>`, m.gambitToken)
	w.Write([]byte(content))
}

func (m *MockHTTPServer) handlePOEStatus(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	if !m.isAuthenticated(r) {
		w.Write([]byte(`<html><a href="/login.cgi">Login</a></html>`))
		return
	}
	
	content := loadTestFile(string(m.model), "getPoePortStatus.cgi.html")
	if content == "" {
		// Fallback content
		content = `<html><ul>
		<li class="poePortStatusListItem">
			<input type="hidden" class="port" value="1"/>
			<span class="poe-port-index"><span>1</span></span>
			<span class="poe-power-mode"><span>Searching</span></span>
			<span class="poe-portPwr-width"><span>ml003@0@</span></span>
			<div class="poe_port_status"><div><div>
				<span>Voltage:</span><span>0</span>
				<span>Current:</span><span>0</span>
				<span>Power:</span><span>0.00</span>
				<span>Temperature:</span><span>25</span>
				<span>Error:</span><span>No Error</span>
			</div></div></div>
		</li>
		</ul></html>`
	}
	w.Write([]byte(content))
}

func (m *MockHTTPServer) handlePOEStatus316(w http.ResponseWriter, r *http.Request) {
	if !m.isAuthenticated316(r) {
		w.Write([]byte(`<html><a href="/redirect.html">Login</a></html>`))
		return
	}
	
	content := loadTestFile(string(m.model), "poePortStatus_GetData_true.html")
	if content == "" {
		content = `<html><div class="port-wrap">
			<span class="port-number">1</span>
			<span class="Status-text">Searching</span>
			<span class="Class-text">0</span>
			<p class="OutputVoltage-text">0</p>
			<p class="OutputCurrent-text">0</p>
			<p class="OutputPower-text">0.00</p>
			<p class="Temperature-text">25</p>
			<p class="Fault-Status-text">No Error</p>
		</div></html>`
	}
	w.Write([]byte(content))
}

func (m *MockHTTPServer) handlePOESettings(w http.ResponseWriter, r *http.Request) {
	if !m.isAuthenticated(r) {
		w.Write([]byte(`<html><a href="/login.cgi">Login</a></html>`))
		return
	}
	
	content := loadTestFile(string(m.model), "PoEPortConfig.cgi.html")
	w.Write([]byte(content))
}

func (m *MockHTTPServer) handlePOESettings316(w http.ResponseWriter, r *http.Request) {
	if !m.isAuthenticated316(r) {
		w.Write([]byte(`<html><a href="/redirect.html">Login</a></html>`))
		return
	}
	
	content := loadTestFile(string(m.model), "poePortConf.html")
	w.Write([]byte(content))
}

func (m *MockHTTPServer) handlePOEUpdate(w http.ResponseWriter, r *http.Request) {
	if !m.isAuthenticated(r) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	
	// Return success page
	w.Write([]byte(`<html>Configuration updated</html>`))
}

func (m *MockHTTPServer) handlePortSettings(w http.ResponseWriter, r *http.Request) {
	if !m.isAuthenticated(r) {
		w.Write([]byte(`<html><a href="/login.cgi">Login</a></html>`))
		return
	}
	
	content := loadTestFile(string(m.model), "dashboard.cgi.html")
	if content == "" {
		// Fallback content
		content = `<html><table id="portSetTable">
		<tr>
			<td>1</td>
			<td><input type="text" value="Port 1"/></td>
			<td><select><option selected>Auto</option></select></td>
			<td><select><option selected>No Limit</option></select></td>
			<td><select><option selected>No Limit</option></select></td>
			<td><input type="checkbox"/></td>
			<td>Connected</td>
			<td>1000M Full</td>
		</tr>
		</table></html>`
	}
	w.Write([]byte(content))
}

func (m *MockHTTPServer) isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("SID")
	return err == nil && cookie.Value == m.sessionToken
}

func (m *MockHTTPServer) isAuthenticated316(r *http.Request) bool {
	return strings.Contains(r.URL.String(), "Gambit="+m.gambitToken)
}

func (m *MockHTTPServer) Close() {
	m.server.Close()
}

func (m *MockHTTPServer) URL() string {
	return m.server.URL
}

func (m *MockHTTPServer) GetRequests() []RequestLog {
	return m.requests
}

// Test helpers

func createTestGlobalOptions(verbose, quiet bool, format formatter.OutputFormat) *types.GlobalOptions {
	return &types.GlobalOptions{
		Verbose:      verbose,
		Quiet:        quiet,
		OutputFormat: format,
		TokenDir:     "",
	}
}

func createTempTokenDir(t *testing.T) string {
	dir, err := os.MkdirTemp("", "ntgrrc-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	return dir
}

func writeTestToken(t *testing.T, dir, host, token string, model types.NetgearModel) {
	tokenPath := tokenFilename(dir, host)
	
	// Ensure the directory exists
	tokenDir := filepath.Dir(tokenPath)
	err := os.MkdirAll(tokenDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create token directory: %v", err)
	}
	
	content := fmt.Sprintf("%s:%s", model, token)
	err = os.WriteFile(tokenPath, []byte(content), 0600)
	if err != nil {
		t.Fatalf("Failed to write test token: %v", err)
	}
}

// loadTestFile loads a test data file for a given model
func loadTestFile(model string, fileName string) string {
	fullFileName := filepath.Join("test-data", model, fileName)
	bytes, err := os.ReadFile(fullFileName)
	if err != nil {
		panic(err)
	}
	return string(bytes)
}

// loadTestFileIfNotExists loads a test file or returns empty string if it doesn't exist
func loadTestFileIfNotExists(model string, fileName string) string {
	fullFileName := filepath.Join("test-data", model, fileName)
	bytes, err := os.ReadFile(fullFileName)
	if err != nil {
		return ""
	}
	return string(bytes)
}