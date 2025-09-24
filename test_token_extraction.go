package main

import (
	"fmt"
	"strings"
)

func extractSessionToken(setCookieHeader string) string {
	sessionIdPrefixes := []string{
		"SID=", // GS305EPx, GS308EPx
	}

	for _, prefix := range sessionIdPrefixes {
		if strings.HasPrefix(setCookieHeader, prefix) {
			sidVal := setCookieHeader[len(prefix):]
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
	// Test with the actual Set-Cookie header we received
	setCookie := "SID=lFSsZv`lnhqK_AxKkoJin]qfjWS_CrCU\\ppuVHqIOuPOtGjGmnGYBsxjdMGyEJKRrZMPlSLGFiTcDuyH;SameSite=Lax;PATH=/;HttpOnly"

	fmt.Println("Set-Cookie header:")
	fmt.Println(setCookie)

	extracted := extractSessionToken(setCookie)
	fmt.Printf("\nExtracted token: '%s'\n", extracted)
	fmt.Printf("Length: %d\n", len(extracted))

	// Test if empty
	if extracted == "" {
		fmt.Println("❌ Token extraction failed!")
	} else {
		fmt.Println("✅ Token extracted successfully!")

		// Create the cookie header that would be sent
		cookieHeader := fmt.Sprintf("SID=%s", extracted)
		fmt.Printf("Cookie header to send: %s\n", cookieHeader)
	}
}