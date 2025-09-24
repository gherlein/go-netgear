package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"crypto/md5"
	"regexp"
	"time"
)

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

func main() {
	fmt.Println("=== Detailed Authentication Check ===")

	switchAddr := "http://tswitch1.herlein.me"
	password := "None1234@"

	// Create client with cookie jar to handle sessions
	jar, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: jar,
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			fmt.Printf("Redirect: %s\n", req.URL)
			return nil
		},
	}

	// Step 1: Get initial page (might set initial cookies)
	fmt.Println("\n1. Getting initial page...")
	resp, err := client.Get(switchAddr + "/")
	if err != nil {
		fmt.Printf("Failed to get initial page: %v\n", err)
		return
	}
	resp.Body.Close()

	// Step 2: Get login page and extract seed
	fmt.Println("\n2. Getting login page...")
	resp, err = client.Get(switchAddr + "/login.cgi")
	if err != nil {
		fmt.Printf("Failed to get login page: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read body: %v\n", err)
		return
	}

	// Show any cookies we got
	cookies := jar.Cookies(&url.URL{Scheme: "http", Host: "tswitch1.herlein.me"})
	if len(cookies) > 0 {
		fmt.Println("Cookies after login page:")
		for _, cookie := range cookies {
			fmt.Printf("  %s = %s\n", cookie.Name, cookie.Value)
		}
	}

	// Extract seed value
	re := regexp.MustCompile(`id='rand' value='([0-9]+)'`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		fmt.Printf("Failed to find seed value\n")
		return
	}

	seed := matches[1]
	fmt.Printf("Seed value: %s\n", seed)

	// Step 3: Encrypt password
	merged := specialMerge(password, seed)
	hash := md5.Sum([]byte(merged))
	encrypted := fmt.Sprintf("%x", hash)
	fmt.Printf("\n3. Password encryption:\n")
	fmt.Printf("   Password: %s\n", password)
	fmt.Printf("   Seed: %s\n", seed)
	fmt.Printf("   Merged: %s\n", merged)
	fmt.Printf("   MD5: %s\n", encrypted)

	// Step 4: Make POST request with various header combinations
	fmt.Println("\n4. Trying POST with different headers...")

	// Try 1: Basic POST
	data := url.Values{}
	data.Set("password", encrypted)

	req, _ := http.NewRequest("POST", switchAddr + "/login.cgi", strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Referer", switchAddr + "/login.cgi")
	req.Header.Set("Origin", switchAddr)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	fmt.Println("Request headers:")
	for key, values := range req.Header {
		fmt.Printf("  %s: %s\n", key, values[0])
	}

	resp2, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to POST: %v\n", err)
		return
	}
	defer resp2.Body.Close()

	// Check response
	fmt.Printf("\n=== Response ===\n")
	fmt.Printf("Status: %s\n", resp2.Status)
	fmt.Printf("Response headers:\n")
	for key, values := range resp2.Header {
		fmt.Printf("  %s: %v\n", key, values)
	}

	// Check cookies after login
	cookies = jar.Cookies(&url.URL{Scheme: "http", Host: "tswitch1.herlein.me"})
	if len(cookies) > 0 {
		fmt.Println("\nCookies after POST:")
		for _, cookie := range cookies {
			fmt.Printf("  %s = %s\n", cookie.Name, cookie.Value)
		}
	}

	body2, _ := ioutil.ReadAll(resp2.Body)
	bodyStr := string(body2)

	// Analyze response
	fmt.Println("\n=== Response Analysis ===")
	if strings.Contains(bodyStr, "LOG IN") && strings.Contains(bodyStr, "password") {
		fmt.Printf("❌ Got login page back - authentication failed\n")

		// Check if there's an error message
		if strings.Contains(bodyStr, "error") || strings.Contains(bodyStr, "incorrect") || strings.Contains(bodyStr, "invalid") {
			errorRe := regexp.MustCompile(`(error|incorrect|invalid)[^<]*`)
			if errorMatches := errorRe.FindStringSubmatch(strings.ToLower(bodyStr)); len(errorMatches) > 0 {
				fmt.Printf("Error message found: %s\n", errorMatches[0])
			}
		}
	} else if strings.Contains(bodyStr, "frame") || strings.Contains(bodyStr, "redirect") || strings.Contains(bodyStr, "home") {
		fmt.Printf("✅ Got different page - might be success\n")
	} else {
		fmt.Printf("? Unknown response type\n")
	}

	fmt.Printf("\nFirst 500 chars of response:\n%.500s\n", bodyStr)

	// Try to access a protected page to verify auth
	fmt.Println("\n5. Testing if we can access protected pages...")
	resp3, err := client.Get(switchAddr + "/getPoePortStatus.cgi")
	if err != nil {
		fmt.Printf("Failed to get POE status: %v\n", err)
		return
	}
	defer resp3.Body.Close()

	body3, _ := ioutil.ReadAll(resp3.Body)
	if strings.Contains(string(body3), "login") || strings.Contains(string(body3), "Login") {
		fmt.Printf("❌ Got redirected to login - not authenticated\n")
	} else if len(body3) > 100 {
		fmt.Printf("✅ Got data from protected page - authenticated!\n")
		fmt.Printf("Response length: %d bytes\n", len(body3))
	} else {
		fmt.Printf("? Got short response: %d bytes\n", len(body3))
	}
}