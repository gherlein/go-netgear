package main

import (
	"fmt"
	"os"
	"context"
	"net/http"
	"time"
	"github.com/gherlein/go-netgear/test"
	"github.com/gherlein/go-netgear/pkg/netgear"
)

func main() {
	fmt.Println("=== Simple Authentication Test ===")

	// Check environment variables
	fmt.Printf("TEST_SWITCH_PASSWORD_1: '%s'\n", os.Getenv("TEST_SWITCH_PASSWORD_1"))

	// Load config
	config, err := test.LoadTestConfig("test/test_config.json")
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		return
	}

	switchConfig := config.Switches[0]
	fmt.Printf("Switch: %s at %s with password '%s'\n", switchConfig.Name, switchConfig.Address, switchConfig.Password)

	// Test basic HTTP connectivity
	fmt.Printf("\n=== Testing Basic Connectivity ===\n")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("http://" + switchConfig.Address + "/")
	if err != nil {
		fmt.Printf("HTTP connectivity failed: %v\n", err)
		return
	}
	fmt.Printf("HTTP connectivity OK: %d %s\n", resp.StatusCode, resp.Status)
	resp.Body.Close()

	// Test with direct password (not environment lookup)
	fmt.Printf("\n=== Testing Direct Password Authentication ===\n")
	netClient, err := netgear.NewClient(switchConfig.Address,
		netgear.WithVerbose(true),
		netgear.WithEnvironmentAuth(false)) // Disable environment auth
	if err != nil {
		fmt.Printf("Failed to create netgear client: %v\n", err)
		return
	}

	ctx := context.Background()
	err = netClient.Login(ctx, "None1234@") // Use password directly
	if err != nil {
		fmt.Printf("Direct password login failed: %v\n", err)
	} else {
		fmt.Printf("Direct password login succeeded!\n")
		fmt.Printf("IsAuthenticated: %v\n", netClient.IsAuthenticated())
		return
	}

	// Test with different passwords
	fmt.Printf("\n=== Testing Common Password Variations ===\n")
	passwords := []string{
		"None1234@",
		"password",
		"admin",
		"",
	}

	for _, pwd := range passwords {
		fmt.Printf("Trying password: '%s'\n", pwd)
		client2, _ := netgear.NewClient(switchConfig.Address,
			netgear.WithVerbose(false),
			netgear.WithEnvironmentAuth(false))

		err = client2.Login(ctx, pwd)
		if err == nil {
			fmt.Printf("SUCCESS with password: '%s'\n", pwd)
			return
		} else {
			fmt.Printf("Failed: %v\n", err)
		}
	}

	fmt.Printf("\nAll password attempts failed.\n")
}