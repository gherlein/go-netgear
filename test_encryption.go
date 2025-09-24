package main

import (
	"crypto/md5"
	"fmt"
	"strings"
)

// specialMerge implements the Netgear interleaving algorithm
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
	password := "None1234@"
	seed := "1696951167"

	// Test the merge algorithm
	merged := specialMerge(password, seed)
	fmt.Printf("Password: %s\n", password)
	fmt.Printf("Seed: %s\n", seed)
	fmt.Printf("Merged: %s\n", merged)

	// Generate MD5
	hash := md5.Sum([]byte(merged))
	encrypted := fmt.Sprintf("%x", hash)
	fmt.Printf("MD5 Hash: %s\n", encrypted)

	// Test with different seeds to see pattern
	fmt.Println("\n=== Testing with different seeds ===")
	seeds := []string{"1696951167", "1234567890", "0000000000"}
	for _, s := range seeds {
		merged := specialMerge(password, s)
		hash := md5.Sum([]byte(merged))
		fmt.Printf("Seed: %s -> Hash: %x\n", s, hash)
	}
}