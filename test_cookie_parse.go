package main

import (
	"fmt"
	"net/http"
	"strings"
)

// Simulate the library's extractSessionToken method
func extractSessionToken(resp *http.Response) string {
	cookie := resp.Header.Get("Set-Cookie")
	sessionIdPrefixes := []string{
		"SID=", // GS305EPx, GS308EPx
	}

	for _, prefix := range sessionIdPrefixes {
		if strings.HasPrefix(cookie, prefix) {
			sidVal := cookie[len(prefix):]
			// Split on semicolon to get just the token value
			if idx := strings.Index(sidVal, ";"); idx != -1 {
				return sidVal[:idx]
			}
			return sidVal
		}
	}

	return ""
}

func main() {
	fmt.Println("=== Cookie Parsing Test ===")

	// Create a mock response with the actual Set-Cookie header we received
	resp := &http.Response{
		Header: make(http.Header),
	}

	// Test the exact header format we received
	setCookieValue := "SID=lFSsZv`lnhqK_AxKkoJin]qfjWS_CrCU\\ppuVHqIOuPOtGjGmnGYBsxjdMGyEJKRrZMPlSLGFiTcDuyH;SameSite=Lax;PATH=/;HttpOnly"
	resp.Header.Set("Set-Cookie", setCookieValue)

	fmt.Printf("Set-Cookie header: %s\n", setCookieValue)

	// Extract using the library's method
	token := extractSessionToken(resp)

	if token == "" {
		fmt.Println("❌ Token extraction failed!")
	} else {
		fmt.Printf("✅ Extracted token: %s\n", token)
		fmt.Printf("Token length: %d\n", len(token))

		// Show what the final cookie header would be
		cookieHeader := fmt.Sprintf("SID=%s", token)
		fmt.Printf("Final cookie header: %s\n", cookieHeader)
	}

	// Test if there are any issues with special characters
	fmt.Printf("\nSpecial character analysis:\n")
	if strings.Contains(token, "`") {
		fmt.Printf("- Contains backtick: ✓\n")
	}
	if strings.Contains(token, "]") {
		fmt.Printf("- Contains bracket: ✓\n")
	}
	if strings.Contains(token, "\\") {
		fmt.Printf("- Contains backslash: ✓\n")
	}
}