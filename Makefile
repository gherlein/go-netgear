# Makefile for go-netgear project

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint

# Build parameters
BINARY_NAME=go-netgear
BINARY_UNIX=$(BINARY_NAME)_unix

# Test parameters
TEST_TIMEOUT=10m
TEST_VERBOSE=-v
TEST_PACKAGE=./test

.PHONY: all build clean test run-tests test-verbose test-short lint fmt vet mod-tidy help

# Default target
all: test build

# Build the project
build:
	$(GOBUILD) -o $(BINARY_NAME) -v ./cmd/go-netgear-cli

# Clean build artifacts
clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_UNIX)

# Run all tests with comprehensive output
run-tests:
	@echo "=== Running Comprehensive Test Suite ==="
	@echo "This will run all implemented tests for the go-netgear library."
	@echo "Tests will skip automatically if no hardware configuration is available."
	@echo ""
	@mkdir -p /tmp/go-netgear-test-results
	@$(GOTEST) $(TEST_VERBOSE) -timeout $(TEST_TIMEOUT) $(TEST_PACKAGE) 2>&1 | tee /tmp/go-netgear-test-results/output.log | \
		sed -E 's/(PASS:)/\o033[32m\1\o033[0m/g; s/(FAIL:)/\o033[31m\1\o033[0m/g; s/(--- PASS)/\o033[32m\1\o033[0m/g; s/(--- FAIL)/\o033[31m\1\o033[0m/g; s/(--- SKIP)/\o033[33m\1\o033[0m/g; s/(SKIP:)/\o033[33m\1\o033[0m/g'
	@echo ""
	@echo "=== Test Results Summary ==="
	@$(MAKE) --no-print-directory _generate-test-summary
	@echo ""
	@echo "=== Test Suite Complete ==="
	@echo "Note: Tests that require actual Netgear switches will skip if hardware is not configured."
	@echo "To run tests with real hardware, configure test/test_config.json with valid switch details."
	@echo ""
	@echo "Test results cached in: /tmp/go-netgear-test-results/output.log"

# Run tests with standard output
test:
	$(GOTEST) -timeout $(TEST_TIMEOUT) $(TEST_PACKAGE)

# Run tests with verbose output
test-verbose:
	$(GOTEST) $(TEST_VERBOSE) -timeout $(TEST_TIMEOUT) $(TEST_PACKAGE)

# Run only fast tests (no network timeouts)
test-short:
	$(GOTEST) $(TEST_VERBOSE) -timeout 30s -short $(TEST_PACKAGE)

# Run tests for specific phases
test-config:
	$(GOTEST) $(TEST_VERBOSE) -run "TestLoadTestConfig|TestConfigValidation" $(TEST_PACKAGE)

test-auth:
	$(GOTEST) $(TEST_VERBOSE) -run "Test.*Authentication|TestInvalidCredentials" $(TEST_PACKAGE)

test-poe:
	$(GOTEST) $(TEST_VERBOSE) -run "TestPOE.*" $(TEST_PACKAGE)

test-port:
	$(GOTEST) $(TEST_VERBOSE) -run "TestPort.*" $(TEST_PACKAGE)

test-readonly:
	$(GOTEST) $(TEST_VERBOSE) -run "TestPOEStatusReading|TestPortStatusReading|TestModelDetection" $(TEST_PACKAGE)

test-error:
	$(GOTEST) $(TEST_VERBOSE) -run "TestInvalid.*|TestNetwork.*|TestConcurrent.*" $(TEST_PACKAGE)

test-fixtures:
	$(GOTEST) $(TEST_VERBOSE) -run "Test.*Fixtures|Test.*Helper" $(TEST_PACKAGE)

# Run tests without network dependencies
test-offline:
	$(GOTEST) $(TEST_VERBOSE) -run "TestLoadTestConfig|TestConfigValidation|TestNewTestHelper|TestNewTestFixtures|TestValid.*|TestGet.*|TestCreate.*|TestContains.*|TestCompare.*|TestGenerate.*|TestAuthenticationTimeout|TestNetworkTimeout" $(TEST_PACKAGE)

# Validate test configuration file
validate-config:
	$(GOCMD) run ./cmd/go-netgear-cli --validate-config

# Validate custom config file
validate-config-custom:
	@if [ -z "$(CONFIG)" ]; then \
		echo "Usage: make validate-config-custom CONFIG=/path/to/config.json"; \
		exit 1; \
	fi
	$(GOCMD) run ./cmd/go-netgear-cli --validate-config --config $(CONFIG)

# Test the enhanced run-tests with a subset (for demonstration)
run-tests-demo:
	@echo "=== Running Test Suite Demo (Fixtures Only) ==="
	@echo "This demonstrates the colorized output and summary features."
	@echo ""
	@mkdir -p /tmp/go-netgear-test-results
	@$(GOTEST) $(TEST_VERBOSE) -timeout 30s -run "Test.*Fixtures|Test.*Helper|TestValidPOE|TestValidPort|TestNew.*" $(TEST_PACKAGE) 2>&1 | tee /tmp/go-netgear-test-results/output.log | \
		sed -E 's/(PASS:)/\o033[32m\1\o033[0m/g; s/(FAIL:)/\o033[31m\1\o033[0m/g; s/(--- PASS)/\o033[32m\1\o033[0m/g; s/(--- FAIL)/\o033[31m\1\o033[0m/g; s/(--- SKIP)/\o033[33m\1\o033[0m/g; s/(SKIP:)/\o033[33m\1\o033[0m/g'
	@echo ""
	@echo "=== Test Results Summary ==="
	@$(MAKE) --no-print-directory _generate-test-summary
	@echo ""
	@echo "=== Demo Complete ==="

# Show cached test results summary
show-test-results:
	@echo "=== Cached Test Results ==="
	@$(MAKE) --no-print-directory _generate-test-summary

# Code quality checks
lint:
	@if command -v $(GOLINT) >/dev/null 2>&1; then \
		$(GOLINT) run ./...; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Format code
fmt:
	$(GOFMT) -s -w .

# Run go vet
vet:
	$(GOCMD) vet ./...

# Tidy go modules
mod-tidy:
	$(GOMOD) tidy

# Download dependencies
mod-download:
	$(GOMOD) download

# Run all quality checks
check: fmt vet lint test

# Cross compilation
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BINARY_UNIX) -v

# Internal target to generate test summary from cached results
_generate-test-summary:
	@if [ -f /tmp/go-netgear-test-results/output.log ]; then \
		echo "Processing test results..."; \
		TOTAL_TESTS=$$(grep -c "^=== RUN" /tmp/go-netgear-test-results/output.log 2>/dev/null | head -1); \
		PASSED_TESTS=$$(grep -c "^--- PASS:" /tmp/go-netgear-test-results/output.log 2>/dev/null | head -1); \
		FAILED_TESTS=$$(grep -c "^--- FAIL:" /tmp/go-netgear-test-results/output.log 2>/dev/null | head -1); \
		SKIPPED_TESTS=$$(grep -c "^--- SKIP:" /tmp/go-netgear-test-results/output.log 2>/dev/null | head -1); \
		BENCHMARK_TESTS=$$(grep -c "^=== RUN.*Benchmark" /tmp/go-netgear-test-results/output.log 2>/dev/null | head -1); \
		TOTAL_TESTS=$${TOTAL_TESTS:-0}; \
		PASSED_TESTS=$${PASSED_TESTS:-0}; \
		FAILED_TESTS=$${FAILED_TESTS:-0}; \
		SKIPPED_TESTS=$${SKIPPED_TESTS:-0}; \
		BENCHMARK_TESTS=$${BENCHMARK_TESTS:-0}; \
		echo ""; \
		echo "📊 Test Statistics:"; \
		echo "   Total Tests Run: $$TOTAL_TESTS"; \
		printf "   \033[32m✅ Passed: %s\033[0m\n" "$$PASSED_TESTS"; \
		printf "   \033[31m❌ Failed: %s\033[0m\n" "$$FAILED_TESTS"; \
		printf "   \033[33m⏭️  Skipped: %s\033[0m\n" "$$SKIPPED_TESTS"; \
		if [ "$$BENCHMARK_TESTS" -gt 0 ]; then \
			echo "   🏃 Benchmarks: $$BENCHMARK_TESTS"; \
		fi; \
		echo ""; \
		if [ "$$FAILED_TESTS" -gt 0 ]; then \
			printf "\033[31m💥 Failed Tests:\033[0m\n"; \
			grep "^--- FAIL:" /tmp/go-netgear-test-results/output.log | sed 's/^--- FAIL: /   • /' || true; \
			echo ""; \
		fi; \
		if [ "$$FAILED_TESTS" -eq 0 ] && [ "$$PASSED_TESTS" -gt 0 ]; then \
			printf "\033[32m🎉 All tests passed successfully!\033[0m\n"; \
		elif [ "$$PASSED_TESTS" -eq 0 ] && [ "$$SKIPPED_TESTS" -gt 0 ]; then \
			printf "\033[33m⚠️  All tests were skipped (likely due to missing configuration)\033[0m\n"; \
		fi; \
		DURATION=$$(grep "^PASS\|^FAIL" /tmp/go-netgear-test-results/output.log | tail -1 | grep -o '[0-9]*\.[0-9]*s' || echo "N/A"); \
		if [ "$$DURATION" != "N/A" ]; then \
			echo "⏱️  Total Duration: $$DURATION"; \
		fi; \
	else \
		echo "❌ Test results not found in cache"; \
	fi

# Development setup
dev-setup:
	$(GOMOD) download
	@echo "Development environment setup complete"
	@echo "Run 'make run-tests' to execute the test suite"
	@echo "Run 'make help' to see all available targets"

# Display help
help:
	@echo "Available make targets:"
	@echo ""
	@echo "Building:"
	@echo "  build          - Build the project binaries"
	@echo "  build-linux    - Cross-compile for Linux"
	@echo "  clean          - Clean build artifacts"
	@echo ""
	@echo "Testing:"
	@echo "  run-tests      - Run comprehensive test suite with colorized output and summary"
	@echo "  run-tests-demo - Demo colorized output and summary (fixtures only)"
	@echo "  test           - Run all tests with standard output"
	@echo "  test-verbose   - Run all tests with verbose output"
	@echo "  test-short     - Run fast tests only (no network timeouts)"
	@echo "  test-offline   - Run tests that don't require network connectivity"
	@echo ""
	@echo "Phase-specific tests:"
	@echo "  test-config    - Run configuration tests"
	@echo "  test-auth      - Run authentication tests"
	@echo "  test-poe       - Run POE configuration tests"
	@echo "  test-port      - Run port configuration tests"
	@echo "  test-readonly  - Run read-only operation tests"
	@echo "  test-error     - Run error handling tests"
	@echo "  test-fixtures  - Run fixture and helper tests"
	@echo ""
	@echo "Configuration Validation:"
	@echo "  validate-config        - Validate test configuration file"
	@echo "  validate-config-custom - Validate custom config file (CONFIG=/path/to/file)"
	@echo ""
	@echo "Code Quality:"
	@echo "  fmt            - Format code with gofmt"
	@echo "  vet            - Run go vet"
	@echo "  lint           - Run golangci-lint (if installed)"
	@echo "  check          - Run all quality checks and tests"
	@echo ""
	@echo "Dependencies:"
	@echo "  mod-tidy       - Tidy go modules"
	@echo "  mod-download   - Download dependencies"
	@echo "  dev-setup      - Set up development environment"
	@echo ""
	@echo "Configuration:"
	@echo "  To run tests with real hardware, configure test/test_config.json"
	@echo "  Set environment variables TEST_SWITCH_PASSWORD_1, TEST_SWITCH_PASSWORD_2, etc."