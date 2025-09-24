package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"crypto/md5"
	"regexp"
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
	fmt.Println("=== Manual Authentication Test ===")

	switchAddr := "http://tswitch1.herlein.me"
	password := "None1234@"

	// Step 1: Get login page and extract seed
	resp, err := http.Get(switchAddr + "/login.cgi")
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

	// Extract seed value
	re := regexp.MustCompile(`id='rand' value='([0-9]+)'`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		fmt.Printf("Failed to find seed value\n")
		return
	}

	seed := matches[1]
	fmt.Printf("Step 1: Got seed value: %s\n", seed)

	// Step 2: Encrypt password
	merged := specialMerge(password, seed)
	hash := md5.Sum([]byte(merged))
	encrypted := fmt.Sprintf("%x", hash)
	fmt.Printf("Step 2: Password '%s' + Seed '%s' = '%s'\n", password, seed, encrypted)
	fmt.Printf("        Merged string: %s\n", merged)

	// Step 3: Make POST request
	client := &http.Client{}

	data := url.Values{}
	data.Set("password", encrypted)

	req, err := http.NewRequest("POST", switchAddr + "/login.cgi", strings.NewReader(data.Encode()))
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		return
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0")
	req.Header.Set("Accept", "*/*")

	fmt.Printf("Step 3: Sending POST with password=%s\n", encrypted)

	resp2, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to POST: %v\n", err)
		return
	}
	defer resp2.Body.Close()

	// Check response
	fmt.Printf("\n=== Response ===\n")
	fmt.Printf("Status: %s\n", resp2.Status)
	fmt.Printf("Headers:\n")
	for key, values := range resp2.Header {
		if strings.Contains(strings.ToLower(key), "cookie") || strings.Contains(strings.ToLower(key), "location") {
			fmt.Printf("  %s: %v\n", key, values)
		}
	}

	body2, _ := ioutil.ReadAll(resp2.Body)
	bodyStr := string(body2)

	// Check if we got login page back (failure) or something else (success)
	if strings.Contains(bodyStr, "LOG IN") && strings.Contains(bodyStr, "password") {
		fmt.Printf("\n❌ Authentication FAILED - Got login page back\n")
		fmt.Printf("The password '%s' is incorrect for this switch.\n", password)
	} else if strings.Contains(bodyStr, "frame") || strings.Contains(bodyStr, "redirect") {
		fmt.Printf("\n✅ Authentication might have succeeded - Got different page\n")
		fmt.Printf("Response preview: %.200s...\n", bodyStr)
	} else {
		fmt.Printf("\nUnknown response:\n")
		fmt.Printf("Response preview: %.500s...\n", bodyStr)
	}
}