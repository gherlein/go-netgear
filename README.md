# go-netgear

A Go library and CLI tool for controlling Netgear managed switches with Power over Ethernet (PoE) capabilities. This project provides both a programmatic API and command-line interface for managing switch ports, PoE settings, and monitoring power consumption.

## Supported Models

- **GS305EP** / **GS305EPP** - 5-port Gigabit switches with PoE+
- **GS308EP** / **GS308EPP** - 8-port Gigabit switches with PoE+
- **GS316EP** / **GS316EPP** - 16-port Gigabit switches with PoE+

## Features

### Core Functionality
- **PoE Management**: Monitor status, configure settings, and cycle power on PoE ports
- **Port Configuration**: Manage port speed, flow control, rate limiting, and descriptions
- **Switch Discovery**: Automatic model detection and capability discovery
- **Authentication**: Session-based authentication with token caching for performance

### Advanced Features
- **Environment Variable Authentication**: Automatic password resolution from environment variables
- **Multi-Switch Support**: Manage multiple switches with different passwords via `NETGEAR_SWITCHES` configuration
- **Token Persistence**: Cached authentication tokens to reduce login frequency
- **Comprehensive Error Handling**: Detailed error reporting for network and authentication issues

## Installation

```bash
go get github.com/gherlein/go-netgear
```

## Quick Start

### Basic Library Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/gherlein/go-netgear/pkg/netgear"
)

func main() {
    // Create client with automatic environment password detection
    // Tokens are cached in ~/.cache/go-netgear by default
    client, err := netgear.NewClient("192.168.1.10")
    if err != nil {
        log.Fatal(err)
    }

    ctx := context.Background()

    // Login (or use environment variables for automatic authentication)
    err = client.Login(ctx, "your-password")
    if err != nil {
        log.Fatal(err)
    }

    // Get PoE port status
    statuses, err := client.POE().GetStatus(ctx)
    if err != nil {
        log.Fatal(err)
    }

    for _, status := range statuses {
        fmt.Printf("Port %d (%s): %s - %.2fW @ %.1fV\n",
            status.PortID, status.PortName, status.Status,
            status.PowerW, status.VoltageV)
    }
}
```

### Token Cache Configuration

```go
// Use custom cache directory
client, err := netgear.NewClient("192.168.1.10",
    netgear.WithTokenCache("/var/lib/myapp/cache"))

// Use in-memory cache only (no disk persistence)
client, err := netgear.NewClient("192.168.1.10",
    netgear.WithTokenManager(netgear.NewMemoryTokenManager()))
```

### Environment Variable Authentication

```bash
# Single switch
export NETGEAR_PASSWORD_192_168_1_10=mypassword

# Multiple switches in one variable
export NETGEAR_SWITCHES="switch1=pass123;192.168.1.10=mypass;lab-switch=secret"

# Now the library auto-authenticates
go run main.go
```

## Documentation

### API Reference
- **[API Reference](docs/api-reference.md)** - Complete HTTP API documentation with endpoints, parameters, and response formats for all supported switch models

### Authentication Guide
- **[Library Authentication System](docs/lib-auth.md)** - Comprehensive guide to environment variable-based authentication, multi-switch configuration, and programmatic password management

### Key Topics Covered:
- Environment variable configuration patterns
- Multi-switch password management
- Token caching and session handling
- CI/CD and containerized deployments
- Migration from manual to automatic authentication

## Project Structure

```
go-netgear/
├── pkg/netgear/          # Main library package
│   ├── client.go         # Client implementation with auto-auth
│   ├── auth.go           # Token management
│   ├── password.go       # Environment password resolution
│   ├── models.go         # Data structures and model definitions
│   ├── poe.go            # PoE management
│   ├── port.go           # Port configuration
│   └── internal/         # Internal HTTP and parsing utilities
├── internal/             # CLI implementation
│   ├── client/           # CLI client commands
│   ├── models/           # CLI model detection and parsing
│   └── formatter/        # Output formatting (JSON, Markdown)
├── docs/                 # Documentation
└── examples/             # Usage examples
```

## CLI Tools

The library includes command-line tools for direct switch management:

```bash
# Build CLI tools
go build -o build/ ./...

# Use with environment authentication
export NETGEAR_SWITCHES="myswitch=password123"
./build/poe_status myswitch
./build/poe_cycle myswitch 1,3,5
```

## Contributing

This project follows standard Go conventions. See the documentation for API details and implementation patterns.

## License

MIT License