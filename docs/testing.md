## Testing

## Initial Prompt

Write a plan to develop a set of tests for every single feature/API call.  These must be run by establishing a known state before each call.  The
user must log in, have the session cached, and then check a configuration, then change it, then check to ensure it changed, then restore it to how it was initially found.  We must test every unique setting individually.

## Approach

### Testing Philosophy

1. **State Management**: Every test must capture the initial state, modify it, verify the change, and restore the original state
2. **Isolation**: Each setting is tested independently to identify specific failures
3. **Session Persistence**: Tests verify that cached sessions work correctly across operations
4. **Error Recovery**: Tests must handle and report errors gracefully without leaving the switch in an inconsistent state
5. **Model Compatibility**: Tests must handle differences between switch models (GS30x vs GS316 series)

### Test Environment Requirements

1. **Real Hardware**: Tests require actual Netgear switches (cannot be mocked due to HTML scraping nature)
2. **Network Access**: Direct network connectivity to switches under test
3. **Credentials**: Valid admin passwords for test switches
4. **Test Isolation**: Dedicated test switches or scheduled maintenance windows (tests modify live configurations)

### Test Structure Pattern

Each test follows this pattern:

```
1. Setup Phase
   - Create client with fresh cache directory
   - Perform authentication
   - Verify session is cached

2. State Capture Phase
   - Read current configuration
   - Store original values
   - Validate data structure

3. Modification Phase
   - Apply test changes
   - Verify changes were accepted

4. Verification Phase
   - Read configuration again
   - Confirm changes are reflected
   - Test edge cases and boundaries

5. Restoration Phase
   - Restore original values
   - Verify restoration succeeded

6. Cleanup Phase
   - Clear test cache
   - Report results
```

### Coverage Scope

#### Authentication & Session Management
- Token caching in different directories
- Session expiration handling
- Multi-switch authentication
- Environment variable authentication
- Invalid credential handling

#### POE Management
- Enable/disable per port
- Power modes (802.3af, 802.3at, legacy, pre-802.3at)
- Priority levels (low, high, critical)
- Power limits (none, class, user-defined)
- Power cycling
- Detection types

#### Port Configuration
- Speed settings (auto, 10M/100M half/full, disable)
- Flow control (on/off)
- Port naming/description
- Rate limiting (ingress/egress)
- Port enable/disable

#### Read-Only Operations
- POE status monitoring
- Port status monitoring
- Model detection
- Error message parsing

## Specific Plan

### Phase 1: Test Infrastructure Setup

#### 1.1 Test Configuration File
Create `test_config.json`:
```json
{
  "switches": [
    {
      "name": "test-gs305ep",
      "address": "192.168.1.10",
      "model": "GS305EP",
      "password": "${TEST_SWITCH_PASSWORD_1}",
      "test_ports": [1, 2, 3],
      "skip_tests": []
    },
    {
      "name": "test-gs316ep",
      "address": "192.168.1.11",
      "model": "GS316EP",
      "password": "${TEST_SWITCH_PASSWORD_2}",
      "test_ports": [1, 2, 3, 4],
      "skip_tests": ["rate_limiting"]
    }
  ],
  "test_options": {
    "cache_dir": "/tmp/netgear-test-cache",
    "verbose": true,
    "parallel": false,
    "restore_on_failure": true
  }
}
```

#### 1.2 Test Helper Package
Create `pkg/netgear/testing/helpers.go`:
- State capture/restore utilities
- Comparison functions for settings
- Test report generation
- Error recovery procedures

#### 1.3 Test Fixtures
Create standard test data for:
- Port names
- Power limits
- Rate limits
- Invalid values for negative testing

### Phase 2: Authentication & Session Tests

#### Test 2.1: Basic Authentication
```
Test: test_basic_authentication
Steps:
1. Create client with custom cache directory
2. Login with valid credentials
3. Verify IsAuthenticated() returns true
4. Check token file exists in cache directory
5. Logout
6. Verify token file is removed
```

#### Test 2.2: Token Cache Persistence
```
Test: test_token_cache_persistence
Steps:
1. Create client and login
2. Create second client with same cache directory
3. Verify second client uses cached token (no login required)
4. Perform operation with second client
5. Verify operation succeeds
```

#### Test 2.3: Session Expiration
```
Test: test_session_expiration
Steps:
1. Create client and login
2. Manually corrupt token in cache
3. Attempt operation
4. Verify error indicates authentication required
5. Re-login
6. Verify operation now succeeds
```

#### Test 2.4: Environment Authentication
```
Test: test_environment_authentication
Steps:
1. Set NETGEAR_PASSWORD_<host> environment variable
2. Create client
3. Verify auto-authentication occurs
4. Perform operation without explicit login
5. Verify operation succeeds
```

### Phase 3: POE Configuration Tests

#### Test 3.1: POE Enable/Disable
```
Test: test_poe_enable_disable_port_{n}
For each test port:
1. Get current POE settings
2. Store original enabled state
3. If enabled, disable it
4. Verify GetSettings shows disabled
5. Enable the port
6. Verify GetSettings shows enabled
7. Restore original state
8. Verify restoration
```

#### Test 3.2: POE Power Modes
```
Test: test_poe_mode_{mode}_port_{n}
For each mode [802.3af, 802.3at, legacy, pre-802.3at]:
For each test port:
1. Get current settings
2. Store original mode
3. Set mode to test value
4. Verify GetSettings shows new mode
5. Test incompatible mode transitions
6. Restore original mode
7. Verify restoration
```

#### Test 3.3: POE Priority Levels
```
Test: test_poe_priority_{level}_port_{n}
For each priority [low, high, critical]:
For each test port:
1. Get current settings
2. Store original priority
3. Set priority to test value
4. Verify GetSettings shows new priority
5. Test priority conflicts (all ports critical)
6. Restore original priority
7. Verify restoration
```

#### Test 3.4: POE Power Limits
```
Test: test_poe_power_limit_{type}_port_{n}
For each limit type [none, class, user]:
For each test port:
1. Get current settings
2. Store original limit settings
3. Set limit type
4. If user type, test various wattage values (5W, 15W, 30W)
5. Verify GetSettings shows new limits
6. Test invalid limits (0W, 100W)
7. Restore original settings
8. Verify restoration
```

#### Test 3.5: POE Power Cycling
```
Test: test_poe_power_cycle_port_{n}
For each test port:
1. Get current POE status
2. Note power consumption
3. Cycle power on port
4. Wait for port to come back up (30 seconds)
5. Verify port is powered again
6. Compare power consumption (should be similar)
```

#### Test 3.6: POE Detection Types
```
Test: test_poe_detection_type_{type}_port_{n}
For each detection type [IEEE 802, legacy, 4pt 802.3af + Legacy]:
For each test port:
1. Get current settings
2. Store original detection type
3. Set detection type
4. Verify GetSettings shows new type
5. Test with longer detection time enabled/disabled
6. Restore original settings
7. Verify restoration
```

### Phase 4: Port Configuration Tests

#### Test 4.1: Port Speed Settings
```
Test: test_port_speed_{speed}_port_{n}
For each speed [auto, 10M half, 10M full, 100M half, 100M full, disable]:
For each test port:
1. Get current port settings
2. Store original speed
3. Set speed to test value
4. Verify GetSettings shows new speed
5. If not "disable", verify port remains operational
6. Restore original speed
7. Verify restoration
```

#### Test 4.2: Port Flow Control
```
Test: test_port_flow_control_port_{n}
For each test port:
1. Get current settings
2. Store original flow control state
3. If enabled, disable it
4. Verify GetSettings shows disabled
5. Enable flow control
6. Verify GetSettings shows enabled
7. Restore original state
8. Verify restoration
```

#### Test 4.3: Port Naming
```
Test: test_port_naming_port_{n}
For each test port:
1. Get current settings
2. Store original port name
3. Set name to "TEST_PORT_{n}_{timestamp}"
4. Verify GetSettings shows new name
5. Test special characters in name
6. Test maximum name length
7. Restore original name
8. Verify restoration
```

#### Test 4.4: Port Rate Limiting
```
Test: test_port_rate_limits_port_{n}
For each test port:
1. Get current settings
2. Store original rate limits
3. Test ingress limits [64kbps, 1Mbps, 10Mbps, 100Mbps, unlimited]
4. Test egress limits [64kbps, 1Mbps, 10Mbps, 100Mbps, unlimited]
5. Verify GetSettings shows each limit
6. Test invalid limits
7. Restore original limits
8. Verify restoration
```

#### Test 4.5: Port Enable/Disable (Network)
```
Test: test_port_network_enable_disable_port_{n}
For each test port:
1. Get current port settings
2. Store original state
3. Set speed to "disable"
4. Verify port shows as disabled
5. Verify no network connectivity through port
6. Set speed to "auto"
7. Verify port is enabled
8. Verify network connectivity restored
9. Restore original speed setting
10. Verify restoration
```

### Phase 5: Read-Only Operation Tests

#### Test 5.1: POE Status Reading
```
Test: test_poe_status_reading
Steps:
1. Get POE status for all ports
2. Verify data structure completeness
3. Verify voltage/current/power calculations are consistent
4. Verify temperature readings are reasonable
5. Compare with GetSettings enabled state
```

#### Test 5.2: Port Status Reading
```
Test: test_port_status_reading
Steps:
1. Get port settings for all ports
2. Verify all expected ports are present
3. Verify link status matches physical connections
4. Verify speed negotiations are valid
5. Cross-reference with POE port data
```

#### Test 5.3: Model Detection
```
Test: test_model_detection
Steps:
1. Create client without model specification
2. Verify model is auto-detected correctly
3. Compare with known model from config
4. Verify model-specific endpoints are used
```

### Phase 6: Error Handling Tests

#### Test 6.1: Invalid Credentials
```
Test: test_invalid_credentials
Steps:
1. Attempt login with wrong password
2. Verify appropriate error is returned
3. Verify no token is cached
4. Verify IsAuthenticated() returns false
```

#### Test 6.2: Network Disconnection
```
Test: test_network_disconnection
Steps:
1. Login successfully
2. Simulate network disconnection (firewall rule)
3. Attempt operation
4. Verify timeout error is returned
5. Restore network
6. Verify operations resume
```

#### Test 6.3: Invalid Port Numbers
```
Test: test_invalid_port_numbers
Steps:
1. Attempt operations on port 0
2. Attempt operations on port 999
3. Verify appropriate errors are returned
4. Verify switch state is not corrupted
```

#### Test 6.4: Invalid Configuration Values
```
Test: test_invalid_configuration_values
Steps:
1. Attempt to set invalid POE mode
2. Attempt to set power limit of 1000W
3. Attempt to set negative rate limits
4. Verify all return appropriate errors
5. Verify current settings remain unchanged
```

### Phase 7: Multi-Switch Tests

#### Test 7.1: Parallel Switch Management
```
Test: test_parallel_switch_management
Steps:
1. Create clients for multiple switches
2. Login to all switches
3. Perform operations on each in parallel
4. Verify no token conflicts
5. Verify each maintains separate session
```

#### Test 7.2: Model-Specific Features
```
Test: test_model_specific_features
Steps:
1. Test features unique to GS316 series
2. Test features unique to GS30x series
3. Verify appropriate errors for unsupported features
4. Verify common features work on all models
```

### Phase 8: Performance Tests

#### Test 8.1: Rapid Configuration Changes
```
Test: test_rapid_configuration_changes
Steps:
1. Perform 100 configuration changes in succession
2. Measure average response time
3. Verify no session drops
4. Verify all changes are applied
5. Restore original configuration
```

#### Test 8.2: Cache Performance
```
Test: test_cache_performance
Steps:
1. Login to 10 switches
2. Measure time for initial authentication
3. Clear program state (keep cache)
4. Re-create clients for same switches
5. Measure time using cached tokens
6. Verify cached access is significantly faster
```

### Test Execution Framework

#### Test Runner Script
Create `test/run_tests.sh`:
```bash
#!/bin/bash
# Load test configuration
# Set up test environment
# Run tests in sequence or parallel
# Generate test report
# Clean up test artifacts
```

#### Test Report Format
```
Test Suite: Netgear Switch API Tests
Date: 2024-01-10 14:30:00
Duration: 45 minutes

Summary:
- Total Tests: 156
- Passed: 150
- Failed: 3
- Skipped: 3

Failed Tests:
1. test_poe_mode_legacy_port_3:
   Error: Switch reported "invalid mode transition"

2. test_port_rate_limits_port_5:
   Error: Egress limit not applied correctly

Detailed Results:
[Full test output for each test]
```

### Implementation Priority

1. **Critical Path** (Week 1):
   - Authentication tests
   - Basic POE enable/disable
   - Basic port enable/disable

2. **Core Features** (Week 2):
   - All POE configuration tests
   - Port speed and flow control

3. **Extended Features** (Week 3):
   - Rate limiting
   - Port naming
   - Detection types

4. **Robustness** (Week 4):
   - Error handling tests
   - Multi-switch tests
   - Performance tests

### Safety Considerations

1. **Production Switches**: Never run on production switches
2. **Test Ports**: Only modify designated test ports
3. **Restoration**: Always attempt to restore original state
4. **Failure Handling**: Stop test suite if restoration fails
5. **Logging**: Maintain detailed logs of all operations
6. **Rollback Plan**: Document manual restoration procedures
