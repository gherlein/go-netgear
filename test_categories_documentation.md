# Test Categories Documentation

## 🎯 **Test Categorization Philosophy**

For a **switch management library**, the primary value is in **actually managing switches**. Therefore:

### **CategoryUtility (No Authentication Required)**
**ONLY** tests that validate pure Go code, constants, and configuration parsing:
- `TestLoadTestConfig` - Tests JSON parsing and env var resolution
- `TestConfigValidation` - Tests validation logic on data structures
- `TestNewTestHelper` - Tests helper object creation
- `TestNewTestFixtures` - Tests fixture object creation
- `TestValidPOEModes` - Tests POE mode constants validation
- `TestValidPOEPriorities` - Tests POE priority constants validation
- `TestValidPortSpeeds` - Tests port speed constants validation
- `TestValidRateLimits` - Tests rate limit validation logic

**These test infrastructure code, NOT the netgear library functionality.**

### **CategoryBasic/CategoryAuth/CategoryModify (Authentication REQUIRED)**
**ALL** tests that interact with actual switches:
- Reading switch status/configuration
- Testing authentication mechanisms
- Modifying switch configuration
- Error handling with real switch responses
- Model detection from real switches

## 🚨 **Key Principle**
**If a test doesn't communicate with an actual switch, it's probably not testing the core value of this library.**

The utility tests are infrastructure - valuable but not the main purpose. The real tests validate that we can successfully manage real network switches.