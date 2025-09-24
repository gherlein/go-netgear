# Netgear Switch Authentication System

This document describes the authentication mechanism used by Netgear managed switches and how the go-netgear library implements session-based authentication with token caching.

## Overview

Netgear GS-series managed switches use **session-based authentication** through their web interface. Since these switches don't provide a REST API, the library emulates browser-based login using HTTP requests and HTML parsing.

## Authentication Type

**Session-Based Authentication with Challenge-Response**

The authentication system uses a challenge-response mechanism where:
1. The switch provides a random **seed value**
2. The client encrypts the password using this seed
3. The switch validates the encrypted password
4. Upon success, the switch issues a **session token**

This is **not** standard token-based auth (like JWT) but rather **web session cookies** similar to browser login sessions.

## Authentication Flow

### Step 1: Model Detection
Before authentication, the library detects the switch model by requesting the root page and parsing the HTML response for model information.

### Step 2: Seed Value Retrieval
Each login attempt requires a fresh **seed value** (random string) from the switch:

**GS305EP/GS308EP Series (30x models):**
```
GET http://{switch_ip}/login.cgi
```
Response contains: `<input id="rand" value="abc123seed">`

**GS316EP Series (316 models):**
```
GET http://{switch_ip}/wmi/login
```
Response contains: `<input id="rand" value="xyz789seed">`

### Step 3: Password Encryption
The password is encrypted using a **special merge algorithm**:

1. **Character Interleaving**: Alternate characters from password and seed
   ```
   Password: "admin"
   Seed: "abc123"
   Merged: "aabdcm1i2n3"  (a+a, d+b, m+c, i+1, n+2, +3)
   ```

2. **MD5 Hashing**: Apply MD5 hash to the merged string
   ```
   MD5("aabdcm1i2n3") = "5f4dcc3b5aa765d61d8327deb882cf99"
   ```

This encryption mimics the JavaScript login logic used by Netgear's web interface.

### Step 4: Login Request
Submit encrypted credentials to the appropriate endpoint:

**GS305EP/GS308EP Series:**
```
POST http://{switch_ip}/login.cgi
Content-Type: application/x-www-form-urlencoded
Body: password={encrypted_password}
```

**GS316EP Series:**
```
POST http://{switch_ip}/redirect.html
Content-Type: application/x-www-form-urlencoded
Body: LoginPassword={encrypted_password}
```

### Step 5: Session Token Extraction
Upon successful authentication, the switch returns a session token:

**GS305EP/GS308EP Series:**
- Token delivered as HTTP cookie: `Set-Cookie: SID=abc123token`
- Used in subsequent requests as: `Cookie: SID=abc123token`

**GS316EP Series:**
- Token embedded in HTML form: `<input type="hidden" name="Gambit" value="xyz789token">`
- Used in subsequent requests as form parameter: `Gambit=xyz789token`

## Token Storage and Caching

### File-Based Token Caching
The library provides flexible token caching to avoid repeated login attempts.

**Default Storage Locations (following XDG Base Directory Specification):**
1. **Primary**: `$XDG_CACHE_HOME/go-netgear/` (if XDG_CACHE_HOME is set)
2. **Fallback**: `~/.cache/go-netgear/` (standard Linux location)
3. **Last resort**: `/tmp/go-netgear/` (if home directory unavailable)

**Token File Naming:**
```
netgear-token-{hash}.cache
```
Where `{hash}` is an FNV-1a 32-bit hash of the switch's IP address/hostname.

**File Format:**
```
{model}:{token}
```

Examples:
- `~/.cache/go-netgear/netgear-token-a1b2c3d4.cache`
- Content: `GS305EP:abc123sessiontoken`

### Configuring Token Cache Location

The library offers multiple ways to configure token cache location:

#### 1. Use Default Location (Recommended)
```go
// Uses XDG_CACHE_HOME or ~/.cache/go-netgear automatically
client, err := netgear.NewClient("192.168.1.10")
```

#### 2. Custom Cache Directory
```go
// Store tokens in custom directory
client, err := netgear.NewClient("192.168.1.10",
    netgear.WithTokenCache("/opt/myapp/cache"))
```

#### 3. Custom Cache File Path
```go
// Full control over cache file location
client, err := netgear.NewClient("192.168.1.10",
    netgear.WithTokenCacheFile("/var/lib/myapp/tokens"))
```

#### 4. In-Memory Only (No Persistence)
```go
// Use memory-only token storage (no disk caching)
client, err := netgear.NewClient("192.168.1.10",
    netgear.WithTokenManager(netgear.NewMemoryTokenManager()))
```

### Cache Directory Structure
```
~/.cache/go-netgear/
├── netgear-token-a1b2c3d4.cache  # Token for switch 192.168.1.10
├── netgear-token-e5f6g7h8.cache  # Token for switch 192.168.1.11
└── netgear-token-i9j0k1l2.cache  # Token for switch lab-switch.local
```

### Token File Permissions
- **Directory**: `0700` (rwx------) - owner read/write/execute only
- **Token files**: `0600` (rw-------) - owner read/write only
- Ensures tokens are protected from other users on the system

### Token Retrieval Process
1. Check if token file exists for the target host
2. Read file contents and parse `model:token` format
3. Validate that the model is supported
4. Use cached token for authenticated requests
5. If token is invalid/expired, perform fresh login

## Session Management

### Session Lifecycle
- **Creation**: Login generates new session token
- **Usage**: Token sent with each authenticated request
- **Expiration**: Sessions expire after period of inactivity (switch-determined)
- **Renewal**: No refresh mechanism - must re-login when expired

### Session Validation
The library detects expired sessions by checking response content:
- Expired sessions return login redirect pages
- Library automatically detects this and returns authentication error
- User must call `Login()` again to establish new session

## Security Considerations

### Password Security
- **Plain text passwords** are never stored on disk
- Only **encrypted passwords** (MD5 hash) used in network requests
- **Interactive password input** supports hidden terminal input
- **Environment variable support** for automated deployments

### Token Security
- **Session tokens** are stored in local filesystem
- **File permissions** set to 0644 (readable by user/group)
- **No encryption** of stored tokens (filesystem security assumed)
- **Tokens are host-specific** and cannot be used across switches

### Network Security
- **HTTP only** - switches don't support HTTPS
- **No certificate validation** (not applicable)
- **Session tokens** sent in clear text over network
- **Recommendation**: Use on trusted networks only

## Implementation Examples

### Manual Authentication
```go
// Create client
client, err := netgear.NewClient("192.168.1.10")
if err != nil {
    log.Fatal(err)
}

// Login with password
ctx := context.Background()
err = client.Login(ctx, "admin_password")
if err != nil {
    log.Fatal(err)
}

// Token is now cached automatically
```

### Environment-Based Authentication
```go
// Set environment variable
os.Setenv("NETGEAR_PASSWORD_192_168_1_10", "admin_password")

// Client auto-authenticates during NewClient()
client, err := netgear.NewClient("192.168.1.10")
if err != nil {
    log.Fatal(err)
}

// Already authenticated if environment password was found
if !client.IsAuthenticated() {
    // Fallback to manual login
    err = client.LoginAuto(ctx)
}
```

### Token Management
```go
// Check authentication status
if client.IsAuthenticated() {
    fmt.Println("Session active")
}

// Get current token info (for debugging)
model := client.GetModel()      // "GS305EP"
address := client.GetAddress()  // "192.168.1.10"

// Force logout (clears cached token)
err := client.Logout(ctx)
```

## Error Handling

### Common Authentication Errors

**Invalid Credentials:**
```
ErrInvalidCredentials: "authentication failed - check password"
```

**Session Expired:**
```
ErrNotAuthenticated: "session expired - please login again"
```

**Network Errors:**
```
ErrNetworkError: "failed to connect to switch"
```

**Model Detection Failed:**
```
ErrModelNotDetected: "unable to determine switch model"
```

### Recovery Strategies
1. **Invalid credentials**: Verify password and retry
2. **Expired session**: Call `Login()` with password
3. **Network issues**: Check connectivity and retry
4. **Model detection**: Verify switch is supported model

## Multi-Switch Support

### Parallel Sessions
The token caching system supports managing multiple switches simultaneously:

```go
// Each switch gets its own cached token
client1, _ := netgear.NewClient("192.168.1.10")
client2, _ := netgear.NewClient("192.168.1.11")

// Different tokens cached:
// ~/.config/ntgrrc/token-a1b2c3d4  (for .10)
// ~/.config/ntgrrc/token-e5f6g7h8  (for .11)
```

### Environment Configuration
```bash
export NETGEAR_SWITCHES="switch1=pass123;192.168.1.10=admin;lab-switch=secret"
```

Each switch maintains its own authentication session and cached token.

## Debugging Authentication Issues

### Enable Verbose Logging
```go
client, err := netgear.NewClient("192.168.1.10",
    netgear.WithVerbose(true))
```

### Cache Management

#### Programmatic Cache Management
```go
// Get cache directory location
if ftm, ok := client.tokenMgr.(*netgear.FileTokenManager); ok {
    cacheDir := ftm.GetCacheDir()
    fmt.Printf("Token cache directory: %s\n", cacheDir)
}

// Clear all cached tokens
if ftm, ok := client.tokenMgr.(*netgear.FileTokenManager); ok {
    err := ftm.ClearAllTokens()
    if err != nil {
        log.Printf("Failed to clear tokens: %v", err)
    }
}

// Clear token for specific switch
err := client.Logout(ctx)  // Removes token for this client's switch
```

#### Manual Token Inspection
```bash
# Default cache location
ls ~/.cache/go-netgear/
cat ~/.cache/go-netgear/netgear-token-*.cache

# Clear all cached tokens
rm ~/.cache/go-netgear/netgear-token-*.cache

# If using XDG_CACHE_HOME
ls $XDG_CACHE_HOME/go-netgear/
```

### Debug Report
The library includes a debug report feature that tests all endpoints:
```bash
./ntgrrc debug-report --address 192.168.1.10
```

This provides detailed information about authentication status and available endpoints for troubleshooting.