package client

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/corbym/gocrest/is"
	"github.com/corbym/gocrest/then"
)

func TestGetTokenFilename(t *testing.T) {
	tests := []struct {
		name     string
		tokenDir string
		host     string
		expected string
	}{
		{
			name:     "Simple host",
			tokenDir: "/tmp/tokens",
			host:     "192.168.1.1",
			expected: "/tmp/tokens/.config/ntgrrc/token-3229584271",
		},
		{
			name:     "Host with port",
			tokenDir: "/var/tokens",
			host:     "switch.local:8080",
			expected: "/var/tokens/.config/ntgrrc/token-1651705421",
		},
		{
			name:     "Empty token dir uses temp",
			tokenDir: "",
			host:     "test-host",
			expected: filepath.Join(os.TempDir(), ".config/ntgrrc/token-1672363210"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tokenFilename(tt.tokenDir, tt.host)
			then.AssertThat(t, result, is.EqualTo(tt.expected))
		})
	}
}

func TestStoreAndReadToken(t *testing.T) {
	tests := []struct {
		name        string
		host        string
		token       string
		model       NetgearModel
		expectError bool
	}{
		{
			name:  "Store and read GS305EP token",
			host:  "192.168.1.1",
			token: "test-token-123",
			model: GS305EP,
		},
		{
			name:  "Store and read GS316EP token",
			host:  "switch.local",
			token: "gambit-token-456",
			model: GS316EP,
		},
		{
			name:  "Empty token",
			host:  "test-host",
			token: "",
			model: GS308EPP,
		},
		{
			name:  "Token with special characters",
			host:  "test-host",
			token: "token!@#$%^&*()_+-={}[]|\\:\";<>?,./",
			model: GS316EPP,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tokenDir := createTempTokenDir(t)
			defer os.RemoveAll(tokenDir)

			args := &GlobalOptions{
				TokenDir: tokenDir,
			}

			// Store token
			err := storeToken(args, tt.host, tt.token)
			then.AssertThat(t, err, is.Nil())

			// Verify file exists
			tokenFile := tokenFilename(tokenDir, tt.host)
			_, err = os.Stat(tokenFile)
			then.AssertThat(t, err, is.Nil())

			// Verify file permissions (should be 0600)
			info, _ := os.Stat(tokenFile)
			then.AssertThat(t, info.Mode().Perm(), is.EqualTo(os.FileMode(0600)))

			// Update args with model for reading
			args.model = tt.model

			// Read token back
			_, _, err = readTokenAndModel2GlobalOptions(args, tt.host)
			if tt.expectError {
				then.AssertThat(t, err, is.Not(is.Nil()))
			} else {
				then.AssertThat(t, err, is.Nil())
				then.AssertThat(t, args.token, is.EqualTo(tt.token))
				then.AssertThat(t, args.model, is.EqualTo(tt.model))
			}
		})
	}
}

func TestReadTokenAndModel2GlobalOptions_ErrorCases(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(tokenDir, host string)
		host          string
		expectError   bool
		errorContains string
	}{
		{
			name: "Missing token file",
			setupFunc: func(tokenDir, host string) {
				// Don't create any file
			},
			host:          "missing-host",
			expectError:   true,
			errorContains: "no such file",
		},
		{
			name: "Malformed token file - no newline",
			setupFunc: func(tokenDir, host string) {
				tokenFile := tokenFilename(tokenDir, host)
				os.WriteFile(tokenFile, []byte("tokenonly"), 0600)
			},
			host:          "malformed-host",
			expectError:   true,
			errorContains: "malformed",
		},
		{
			name: "Empty token file",
			setupFunc: func(tokenDir, host string) {
				tokenFile := tokenFilename(tokenDir, host)
				os.WriteFile(tokenFile, []byte(""), 0600)
			},
			host:          "empty-host",
			expectError:   true,
			errorContains: "upgrade",
		},
		{
			name: "Token file with only colon",
			setupFunc: func(tokenDir, host string) {
				tokenFile := tokenFilename(tokenDir, host)
				os.WriteFile(tokenFile, []byte(":"), 0600)
			},
			host:          "colon-host",
			expectError:   true,
			errorContains: "unknown model",
		},
		{
			name: "Unsupported model in token file",
			setupFunc: func(tokenDir, host string) {
				tokenFile := tokenFilename(tokenDir, host)
				os.WriteFile(tokenFile, []byte("UnsupportedModel:token123"), 0600)
			},
			host:          "unsupported-host",
			expectError:   true,
			errorContains: "unknown model",
		},
		{
			name: "Token file with extra data",
			setupFunc: func(tokenDir, host string) {
				tokenFile := tokenFilename(tokenDir, host)
				os.WriteFile(tokenFile, []byte("GS305EP:token123:extra:data"), 0600)
			},
			host:        "extra-data-host",
			expectError: false, // Should still work, ignoring extra data
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			tokenDir := createTempTokenDir(t)
			defer os.RemoveAll(tokenDir)

			args := &GlobalOptions{
				TokenDir: tokenDir,
			}

			tt.setupFunc(tokenDir, tt.host)

			// Execute
			_, _, err := readTokenAndModel2GlobalOptions(args, tt.host)

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

func TestTokenDirectoryCreation(t *testing.T) {
	t.Run("Creates directory if not exists", func(t *testing.T) {
		// Use a nested directory that doesn't exist
		baseDir := createTempTokenDir(t)
		defer os.RemoveAll(baseDir)
		
		tokenDir := filepath.Join(baseDir, "nested", "dirs", "tokens")
		
		args := &GlobalOptions{
			TokenDir: tokenDir,
		}

		// Store token should create the directory
		err := storeToken(args, "test-host", "test-token")
		then.AssertThat(t, err, is.Nil())

		// Verify directory was created
		info, err := os.Stat(tokenDir)
		then.AssertThat(t, err, is.Nil())
		then.AssertThat(t, info.IsDir(), is.True())
	})

	t.Run("Handles existing directory", func(t *testing.T) {
		tokenDir := createTempTokenDir(t)
		defer os.RemoveAll(tokenDir)

		args := &GlobalOptions{
			TokenDir: tokenDir,
		}

		// Store multiple tokens
		err := storeToken(args, "host1", "token1")
		then.AssertThat(t, err, is.Nil())

		err = storeToken(args, "host2", "token2")
		then.AssertThat(t, err, is.Nil())

		// Verify both tokens exist
		_, err = os.Stat(tokenFilename(tokenDir, "host1"))
		then.AssertThat(t, err, is.Nil())

		_, err = os.Stat(tokenFilename(tokenDir, "host2"))
		then.AssertThat(t, err, is.Nil())
	})
}

func TestTokenOverwrite(t *testing.T) {
	tokenDir := createTempTokenDir(t)
	defer os.RemoveAll(tokenDir)

	args := &GlobalOptions{
		TokenDir: tokenDir,
		model:    GS305EP,
	}

	host := "test-host"

	// Store initial token
	err := storeToken(args, host, "initial-token")
	then.AssertThat(t, err, is.Nil())

	// Read and verify initial token
	_, _, err = readTokenAndModel2GlobalOptions(args, host)
	then.AssertThat(t, err, is.Nil())
	then.AssertThat(t, args.token, is.EqualTo("initial-token"))

	// Overwrite with new token
	args.model = GS316EP // Change model too
	err = storeToken(args, host, "new-token")
	then.AssertThat(t, err, is.Nil())

	// Read and verify new token
	_, _, err = readTokenAndModel2GlobalOptions(args, host)
	then.AssertThat(t, err, is.Nil())
	then.AssertThat(t, args.token, is.EqualTo("new-token"))
	then.AssertThat(t, args.model, is.EqualTo(GS316EP))
}

func TestTokenFilePermissions(t *testing.T) {
	if os.Getenv("CI") == "true" {
		t.Skip("Skipping permission test in CI environment")
	}

	tokenDir := createTempTokenDir(t)
	defer os.RemoveAll(tokenDir)

	// Create a read-only directory
	readOnlyDir := filepath.Join(tokenDir, "readonly")
	os.Mkdir(readOnlyDir, 0755)
	os.Chmod(readOnlyDir, 0555) // Make it read-only

	args := &GlobalOptions{
		TokenDir: readOnlyDir,
	}

	// Try to store token in read-only directory
	err := storeToken(args, "test-host", "test-token")
	then.AssertThat(t, err, is.Not(is.Nil()))
	if !strings.Contains(err.Error(), "permission") {
		t.Errorf("Expected permission error, got: %v", err)
	}
}

func TestConcurrentTokenAccess(t *testing.T) {
	tokenDir := createTempTokenDir(t)
	defer os.RemoveAll(tokenDir)

	args := &GlobalOptions{
		TokenDir: tokenDir,
	}

	// Test concurrent writes to different hosts
	hosts := []string{"host1", "host2", "host3", "host4", "host5"}
	done := make(chan bool, len(hosts))

	for i, host := range hosts {
		go func(h string, token string) {
			err := storeToken(args, h, token)
			then.AssertThat(t, err, is.Nil())
			done <- true
		}(host, fmt.Sprintf("token-%d", i))
	}

	// Wait for all goroutines
	for range hosts {
		<-done
	}

	// Verify all tokens were stored correctly
	for i, host := range hosts {
		localArgs := &GlobalOptions{
			TokenDir: tokenDir,
			model:    GS305EP,
		}
		_, _, err := readTokenAndModel2GlobalOptions(localArgs, host)
		then.AssertThat(t, err, is.Nil())
		then.AssertThat(t, localArgs.token, is.EqualTo(fmt.Sprintf("token-%d", i)))
	}
}