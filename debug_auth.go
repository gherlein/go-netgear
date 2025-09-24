package main

import (
	"fmt"
	"os"
	"context"
	"strings"
	"github.com/gherlein/go-netgear/test"
	"github.com/gherlein/go-netgear/pkg/netgear"
	"github.com/gherlein/go-netgear/pkg/netgear/internal"
)

func main() {
	fmt.Println("=== Authentication Debug ===")

	// Check environment variables
	fmt.Printf("TEST_SWITCH_PASSWORD_1: '%s'\n", os.Getenv("TEST_SWITCH_PASSWORD_1"))
	fmt.Printf("TEST_SWITCH_PASSWORD_2: '%s'\n", os.Getenv("TEST_SWITCH_PASSWORD_2"))

	// Load config
	config, err := test.LoadTestConfig("test/test_config.json")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	fmt.Printf("Loaded %d switches\n", len(config.Switches))
	for i, sw := range config.Switches {
		fmt.Printf("Switch %d: %s at %s with password '%s'\n", i+1, sw.Name, sw.Address, sw.Password)
	}

	// Try manual authentication step-by-step
	fmt.Println("\n=== Step-by-Step Authentication ===")
	switchConfig := config.Switches[0]

	client, err := netgear.NewClient(switchConfig.Address, netgear.WithVerbose(true))
	if err != nil {
		fmt.Printf("Failed to create client: %v\n", err)
		return
	}

	ctx := context.Background()

	// Step 1: Test seed extraction
	fmt.Printf("\n1. Getting seed value from /login.cgi...\n")
	httpClient := internal.NewHTTPClient(switchConfig.Address, 0, true)
	resp, err := httpClient.Get(ctx, "/login.cgi", nil)
	if err != nil {
		fmt.Printf("Failed to get login page: %v\n", err)
		return
	}

	body, err := httpClient.ReadBody(resp)
	if err != nil {
		fmt.Printf("Failed to read login page body: %v\n", err)
		return
	}

	seedValue := internal.ExtractSeedValue(body)
	if seedValue == "" {
		fmt.Printf("No seed value found! Login page content preview:\n")
		fmt.Printf("%.500s...\n", body)
		return
	}
	fmt.Printf("Found seed value: '%s'\n", seedValue)

	// Step 2: Test password encryption
	fmt.Printf("\n2. Encrypting password...\n")
	encryptedPassword := internal.EncryptPasswordWithSeed(switchConfig.Password, seedValue)
	fmt.Printf("Original password: '%s'\n", switchConfig.Password)
	fmt.Printf("Encrypted password: '%s'\n", encryptedPassword)

	// Step 3: Try the actual login
	fmt.Printf("\n3. Attempting login...\n")
	err = client.Login(ctx, switchConfig.Password)
	if err != nil {
		fmt.Printf("Login failed: %v\n", err)

		// Check the final error message
		if strings.Contains(err.Error(), "No password found") {
			fmt.Printf("\nDEBUG: The error suggests password lookup failed\n")
			fmt.Printf("This could mean the PasswordManager is not finding the password\n")
		}
		return
	}

	fmt.Printf("Login succeeded! IsAuthenticated: %v\n", client.IsAuthenticated())

	// Try a simple API call
	status, err := client.POE().GetStatus(ctx)
	if err != nil {
		fmt.Printf("POE GetStatus failed: %v\n", err)
	} else {
		fmt.Printf("POE GetStatus succeeded! Got %d ports\n", len(status))
	}
}