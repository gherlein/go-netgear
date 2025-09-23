# How-To Guide: Common Switch Operations

This guide demonstrates the sequence of API calls needed to perform common operations with Netgear managed switches using the go-netgear library.

## Prerequisites

```go
import (
    "context"
    "fmt"
    "log"

    "github.com/gherlein/go-netgear/pkg/netgear"
)
```

## 1. Login to a Switch

### Basic Authentication
```go
func loginToSwitch(address, password string) (*netgear.Client, error) {
    // Create client
    client, err := netgear.NewClient(address)
    if err != nil {
        return nil, fmt.Errorf("failed to create client: %w", err)
    }

    // Login with password
    ctx := context.Background()
    err = client.Login(ctx, password)
    if err != nil {
        return nil, fmt.Errorf("login failed: %w", err)
    }

    fmt.Printf("Successfully logged into switch %s (Model: %s)\n",
        client.GetAddress(), client.GetModel())

    return client, nil
}
```

### Environment Variable Authentication (Recommended)
```go
func loginWithEnvironment(address string) (*netgear.Client, error) {
    // Set environment variable first:
    // export NETGEAR_PASSWORD_192_168_1_10=mypassword
    // or
    // export NETGEAR_SWITCHES="switch1=pass123;192.168.1.10=mypass"

    // Client auto-authenticates if environment password is found
    client, err := netgear.NewClient(address)
    if err != nil {
        return nil, fmt.Errorf("failed to create client: %w", err)
    }

    // Check if already authenticated
    if !client.IsAuthenticated() {
        // Trigger environment variable lookup
        ctx := context.Background()
        err = client.LoginAuto(ctx)
        if err != nil {
            return nil, fmt.Errorf("auto-login failed: %w", err)
        }
    }

    fmt.Printf("Successfully authenticated to switch %s (Model: %s)\n",
        client.GetAddress(), client.GetModel())

    return client, nil
}
```

## 2. Get Status of All Ports

```go
func getAllPortStatus(client *netgear.Client) error {
    ctx := context.Background()

    // Get port settings and status
    ports, err := client.Ports().GetSettings(ctx)
    if err != nil {
        return fmt.Errorf("failed to get port status: %w", err)
    }

    fmt.Println("Port Status:")
    fmt.Println("Port | Name          | Speed        | Status     | Ingress   | Egress    | Flow Control")
    fmt.Println("----|---------------|--------------|------------|-----------|-----------|-------------")

    for _, port := range ports {
        fmt.Printf("%4d | %-13s | %-12s | %-10s | %-9s | %-9s | %s\n",
            port.PortID,
            port.PortName,
            string(port.Speed),
            string(port.Status),
            port.IngressLimit,
            port.EgressLimit,
            map[bool]string{true: "Enabled", false: "Disabled"}[port.FlowControl])
    }

    return nil
}
```

## 3. Get PoE State of All Ports

```go
func getAllPoEStatus(client *netgear.Client) error {
    ctx := context.Background()

    // Get PoE port status
    poeStatus, err := client.POE().GetStatus(ctx)
    if err != nil {
        return fmt.Errorf("failed to get PoE status: %w", err)
    }

    fmt.Println("\nPoE Status:")
    fmt.Println("Port | Name          | Status   | Power Class | Voltage | Current | Power   | Temp")
    fmt.Println("----|---------------|----------|-------------|---------|---------|---------|-----")

    for _, status := range poeStatus {
        fmt.Printf("%4d | %-13s | %-8s | %-11s | %6.1fV | %6.0fmA | %6.1fW | %4.0f°C\n",
            status.PortID,
            status.PortName,
            status.Status,
            status.PowerClass,
            status.VoltageV,
            status.CurrentMA,
            status.PowerW,
            status.TemperatureC)
    }

    return nil
}
```

## 4. Disable PoE Power of a Port

```go
func disablePoEPower(client *netgear.Client, portID int) error {
    ctx := context.Background()

    // Create update to disable PoE power
    enabled := false
    update := netgear.POEPortUpdate{
        PortID:  portID,
        Enabled: &enabled,
    }

    err := client.POE().UpdatePort(ctx, update)
    if err != nil {
        return fmt.Errorf("failed to disable PoE power on port %d: %w", portID, err)
    }

    fmt.Printf("Successfully disabled PoE power on port %d\n", portID)
    return nil
}
```

## 5. Enable PoE Power of a Port

```go
func enablePoEPower(client *netgear.Client, portID int) error {
    ctx := context.Background()

    // Create update to enable PoE power
    enabled := true
    update := netgear.POEPortUpdate{
        PortID:  portID,
        Enabled: &enabled,
    }

    err := client.POE().UpdatePort(ctx, update)
    if err != nil {
        return fmt.Errorf("failed to enable PoE power on port %d: %w", portID, err)
    }

    fmt.Printf("Successfully enabled PoE power on port %d\n", portID)
    return nil
}
```

## 6. Disable Network Throughput of a Port

```go
func disablePortThroughput(client *netgear.Client, portID int) error {
    ctx := context.Background()

    // Set port speed to "disable" to block all network traffic
    speed := netgear.PortSpeedDisable
    update := netgear.PortUpdate{
        PortID: portID,
        Speed:  &speed,
    }

    err := client.Ports().UpdatePort(ctx, update)
    if err != nil {
        return fmt.Errorf("failed to disable port %d throughput: %w", portID, err)
    }

    fmt.Printf("Successfully disabled network throughput on port %d\n", portID)
    return nil
}
```

## 7. Enable Network Throughput of a Port

```go
func enablePortThroughput(client *netgear.Client, portID int) error {
    ctx := context.Background()

    // Set port speed to "auto" to enable network traffic with auto-negotiation
    speed := netgear.PortSpeedAuto
    update := netgear.PortUpdate{
        PortID: portID,
        Speed:  &speed,
    }

    err := client.Ports().UpdatePort(ctx, update)
    if err != nil {
        return fmt.Errorf("failed to enable port %d throughput: %w", portID, err)
    }

    fmt.Printf("Successfully enabled network throughput on port %d (auto-negotiation)\n", portID)
    return nil
}
```

## Complete Example: Full Workflow

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/gherlein/go-netgear/pkg/netgear"
)

func main() {
    switchAddress := "192.168.1.10"

    // Step 1: Login to switch
    client, err := loginWithEnvironment(switchAddress)
    if err != nil {
        log.Fatalf("Login failed: %v", err)
    }
    defer client.Logout(context.Background())

    // Step 2: Get status of all ports
    err = getAllPortStatus(client)
    if err != nil {
        log.Printf("Failed to get port status: %v", err)
    }

    // Step 3: Get PoE state of all ports
    err = getAllPoEStatus(client)
    if err != nil {
        log.Printf("Failed to get PoE status: %v", err)
    }

    // Example operations on port 1
    portID := 1

    // Step 4: Disable PoE power on port 1
    err = disablePoEPower(client, portID)
    if err != nil {
        log.Printf("Failed to disable PoE: %v", err)
    }

    // Step 5: Enable PoE power on port 1
    err = enablePoEPower(client, portID)
    if err != nil {
        log.Printf("Failed to enable PoE: %v", err)
    }

    // Step 6: Disable network throughput on port 1
    err = disablePortThroughput(client, portID)
    if err != nil {
        log.Printf("Failed to disable throughput: %v", err)
    }

    // Step 7: Enable network throughput on port 1
    err = enablePortThroughput(client, portID)
    if err != nil {
        log.Printf("Failed to enable throughput: %v", err)
    }

    fmt.Println("All operations completed successfully!")
}

// Helper functions (loginToSwitch, loginWithEnvironment, etc.) go here...
```

## Environment Setup for Authentication

Before running the code, set up environment variables for automatic authentication:

```bash
# Single switch
export NETGEAR_PASSWORD_192_168_1_10=your_switch_password

# Multiple switches
export NETGEAR_SWITCHES="switch1=pass123;192.168.1.10=mypass;lab-switch=secret"

# Run your program
go run main.go
```

## Error Handling Best Practices

1. **Always check for authentication errors** and provide clear error messages
2. **Handle network timeouts** gracefully - switches may be slow to respond
3. **Verify port numbers** are valid for your switch model before operations
4. **Check switch model compatibility** for specific features
5. **Use context with timeouts** for all operations:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

err := client.POE().UpdatePort(ctx, update)
```

## Port Speed Options

Available port speed settings:
- `netgear.PortSpeedAuto` - Auto-negotiation (recommended for enabling)
- `netgear.PortSpeed10MHalf` - 10 Mbps half-duplex
- `netgear.PortSpeed10MFull` - 10 Mbps full-duplex
- `netgear.PortSpeed100MHalf` - 100 Mbps half-duplex
- `netgear.PortSpeed100MFull` - 100 Mbps full-duplex
- `netgear.PortSpeedDisable` - Disable port (blocks all traffic)

## PoE Configuration Options

When updating PoE settings, you can also configure:
- **Power Mode**: `netgear.POEMode8023af`, `netgear.POEMode8023at`, etc.
- **Priority**: `netgear.POEPriorityLow`, `netgear.POEPriorityHigh`, `netgear.POEPriorityCritical`
- **Power Limit**: Custom wattage limits
- **Detection Type**: How the switch detects PoE devices

See the API documentation for complete details on all available configuration options.