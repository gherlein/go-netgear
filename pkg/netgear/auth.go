package netgear

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// TokenManager handles token persistence
type TokenManager interface {
	// GetToken retrieves a stored token
	GetToken(ctx context.Context, address string) (token string, model Model, err error)

	// StoreToken saves a token
	StoreToken(ctx context.Context, address string, token string, model Model) error

	// DeleteToken removes a stored token
	DeleteToken(ctx context.Context, address string) error
}

// MemoryTokenManager stores tokens in memory
type MemoryTokenManager struct {
	tokens map[string]tokenData
	mu     sync.RWMutex
}

type tokenData struct {
	token string
	model Model
}

// NewMemoryTokenManager creates a new in-memory token manager
func NewMemoryTokenManager() *MemoryTokenManager {
	return &MemoryTokenManager{
		tokens: make(map[string]tokenData),
	}
}

// GetToken retrieves a stored token
func (m *MemoryTokenManager) GetToken(ctx context.Context, address string) (string, Model, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.tokens[address]
	if !exists {
		return "", "", NewAuthError("token not found", nil)
	}

	return data.token, data.model, nil
}

// StoreToken saves a token
func (m *MemoryTokenManager) StoreToken(ctx context.Context, address string, token string, model Model) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tokens[address] = tokenData{
		token: token,
		model: model,
	}

	return nil
}

// DeleteToken removes a stored token
func (m *MemoryTokenManager) DeleteToken(ctx context.Context, address string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.tokens, address)
	return nil
}

// FileTokenManager stores tokens in files (current behavior)
type FileTokenManager struct {
	cacheDir string
}

// NewFileTokenManager creates a new file-based token manager
// If cacheDir is empty, it defaults to XDG_CACHE_HOME or ~/.cache/go-netgear
func NewFileTokenManager(cacheDir string) *FileTokenManager {
	if cacheDir == "" {
		cacheDir = getDefaultCacheDir()
	}
	return &FileTokenManager{cacheDir: cacheDir}
}

// getDefaultCacheDir returns the appropriate cache directory following XDG Base Directory Specification
func getDefaultCacheDir() string {
	// Try XDG_CACHE_HOME first
	if xdgCache := os.Getenv("XDG_CACHE_HOME"); xdgCache != "" {
		return filepath.Join(xdgCache, "go-netgear")
	}

	// Fall back to ~/.cache/go-netgear
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".cache", "go-netgear")
	}

	// Last resort: use temp directory
	return filepath.Join(os.TempDir(), "go-netgear")
}

// GetToken retrieves a stored token from file
func (m *FileTokenManager) GetToken(ctx context.Context, address string) (string, Model, error) {
	tokenFile := m.getTokenFilename(address)

	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return "", "", NewAuthError("failed to read token file", err)
	}

	content := string(data)
	if content == "" {
		return "", "", NewAuthError("token file is empty, please upgrade your token file", nil)
	}

	if !strings.Contains(content, ":") {
		return "", "", NewAuthError("malformed token file", nil)
	}

	parts := strings.SplitN(content, ":", 2)
	if len(parts) != 2 {
		return "", "", NewAuthError("malformed token file", nil)
	}

	modelStr := strings.TrimSpace(parts[0])
	token := strings.TrimSpace(parts[1])

	model := Model(modelStr)
	if !model.IsSupported() {
		return "", "", NewModelError(fmt.Sprintf("unknown model '%s' in token file", modelStr), nil)
	}

	return token, model, nil
}

// StoreToken saves a token to file
func (m *FileTokenManager) StoreToken(ctx context.Context, address string, token string, model Model) error {
	// Ensure cache directory exists
	if err := os.MkdirAll(m.cacheDir, 0700); err != nil {
		return NewAuthError("failed to create token cache directory", err)
	}

	tokenFile := m.getTokenFilename(address)
	content := fmt.Sprintf("%s:%s", string(model), token)

	// Write token with secure permissions (readable by owner only)
	err := os.WriteFile(tokenFile, []byte(content), 0600)
	if err != nil {
		return NewAuthError("failed to write token file", err)
	}

	return nil
}

// DeleteToken removes a stored token file
func (m *FileTokenManager) DeleteToken(ctx context.Context, address string) error {
	tokenFile := m.getTokenFilename(address)

	err := os.Remove(tokenFile)
	if err != nil && !os.IsNotExist(err) {
		return NewAuthError("failed to delete token file", err)
	}

	return nil
}

// getTokenFilename generates the filename for a token based on the address
func (m *FileTokenManager) getTokenFilename(address string) string {
	// Use FNV hash to create a consistent filename from the address
	h := fnv.New32a()
	h.Write([]byte(address))
	hash := h.Sum32()

	return filepath.Join(m.cacheDir, fmt.Sprintf("netgear-token-%x.cache", hash))
}

// GetCacheDir returns the cache directory being used
func (m *FileTokenManager) GetCacheDir() string {
	return m.cacheDir
}

// ClearAllTokens removes all cached tokens in the cache directory
func (m *FileTokenManager) ClearAllTokens() error {
	// Find all token files
	pattern := filepath.Join(m.cacheDir, "netgear-token-*.cache")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return NewAuthError("failed to list token files", err)
	}

	// Remove each token file
	var lastErr error
	for _, file := range files {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			lastErr = err
		}
	}

	if lastErr != nil {
		return NewAuthError("failed to remove some token files", lastErr)
	}

	return nil
}

// AuthenticationType represents the type of authentication used
type AuthenticationType string

const (
	AuthTypeSession AuthenticationType = "session" // Cookie-based (30x series)
	AuthTypeGambit  AuthenticationType = "gambit"  // URL parameter-based (316 series)
)

// GetAuthenticationType returns the authentication type for a model
func GetAuthenticationType(model Model) AuthenticationType {
	if model.IsModel316() {
		return AuthTypeGambit
	}
	return AuthTypeSession
}