package netgear

import (
	"context"
	"fmt"
	"net/url"
	"strconv"

	"github.com/gherlein/go-netgear/pkg/netgear/internal"
)

// POEManager handles POE-related operations
type POEManager struct {
	client *Client
	parser *internal.POEDataParser
}

// newPOEManager creates a new POE manager (internal constructor)
func newPOEManager(client *Client) *POEManager {
	return &POEManager{
		client: client,
		parser: internal.NewPOEDataParser(),
	}
}

// GetStatus retrieves POE status for all ports
func (m *POEManager) GetStatus(ctx context.Context) ([]POEPortStatus, error) {
	if !m.client.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}

	// Determine the appropriate endpoint based on model
	var endpoint string
	if m.client.model.IsModel30x() {
		endpoint = "/getPoePortStatus.cgi"
	} else if m.client.model.IsModel316() {
		endpoint = "/iss/specific/poePortStatus.html"
	} else {
		return nil, NewOperationError("POE status not supported for this model", nil)
	}

	// Make authenticated request
	response, err := m.client.makeAuthenticatedRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, NewOperationError("failed to get POE status", err)
	}

	// Parse the response
	rawData, err := m.parser.ParsePOEStatus(response)
	if err != nil {
		return nil, NewParsingError("failed to parse POE status", err)
	}

	// Convert to strongly typed structures
	var statuses []POEPortStatus
	for _, raw := range rawData {
		status := POEPortStatus{}
		
		if portID, ok := raw["port_id"].(int); ok {
			status.PortID = portID
		}
		if portName, ok := raw["port_name"].(string); ok {
			status.PortName = portName
		}
		if statusStr, ok := raw["status"].(string); ok {
			status.Status = statusStr
		}
		if powerClass, ok := raw["power_class"].(string); ok {
			status.PowerClass = powerClass
		}
		if voltage, ok := raw["voltage_v"].(float64); ok {
			status.VoltageV = voltage
		}
		if current, ok := raw["current_ma"].(float64); ok {
			status.CurrentMA = current
		}
		if power, ok := raw["power_w"].(float64); ok {
			status.PowerW = power
		}
		if temp, ok := raw["temperature_c"].(float64); ok {
			status.TemperatureC = temp
		}
		if errorStatus, ok := raw["error_status"].(string); ok {
			status.ErrorStatus = errorStatus
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}

// GetSettings retrieves POE settings for all ports
func (m *POEManager) GetSettings(ctx context.Context) ([]POEPortSettings, error) {
	if !m.client.IsAuthenticated() {
		return nil, ErrNotAuthenticated
	}

	// Determine the appropriate endpoint based on model
	var endpoint string
	if m.client.model.IsModel30x() {
		endpoint = "/PoEPortConfig.cgi"
	} else if m.client.model.IsModel316() {
		endpoint = "/iss/specific/poePortConf.html"
	} else {
		return nil, NewOperationError("POE settings not supported for this model", nil)
	}

	// Make authenticated request
	response, err := m.client.makeAuthenticatedRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, NewOperationError("failed to get POE settings", err)
	}

	// Parse the response
	rawData, err := m.parser.ParsePOESettings(response)
	if err != nil {
		return nil, NewParsingError("failed to parse POE settings", err)
	}

	// Convert to strongly typed structures
	var settings []POEPortSettings
	for _, raw := range rawData {
		setting := POEPortSettings{}
		
		if portID, ok := raw["port_id"].(int); ok {
			setting.PortID = portID
		}
		if portName, ok := raw["port_name"].(string); ok {
			setting.PortName = portName
		}
		if enabled, ok := raw["enabled"].(bool); ok {
			setting.Enabled = enabled
		}
		if mode, ok := raw["mode"].(string); ok {
			setting.Mode = POEMode(mode)
		}
		if priority, ok := raw["priority"].(string); ok {
			setting.Priority = POEPriority(priority)
		}
		if limitType, ok := raw["power_limit_type"].(string); ok {
			setting.PowerLimitType = POELimitType(limitType)
		}
		if limitW, ok := raw["power_limit_w"].(float64); ok {
			setting.PowerLimitW = limitW
		}
		if detectionType, ok := raw["detection_type"].(string); ok {
			setting.DetectionType = detectionType
		}
		if longerDetection, ok := raw["longer_detection_time"].(bool); ok {
			setting.LongerDetectionTime = longerDetection
		}

		settings = append(settings, setting)
	}

	return settings, nil
}

// UpdatePort updates settings for specific ports
func (m *POEManager) UpdatePort(ctx context.Context, updates ...POEPortUpdate) error {
	if !m.client.IsAuthenticated() {
		return ErrNotAuthenticated
	}

	if len(updates) == 0 {
		return NewOperationError("no updates provided", nil)
	}

	// Determine the appropriate endpoint based on model
	var endpoint string
	if m.client.model.IsModel30x() {
		endpoint = "/PoEPortConfig.cgi"
	} else if m.client.model.IsModel316() {
		endpoint = "/iss/specific/poePortConf.html"
	} else {
		return NewOperationError("POE updates not supported for this model", nil)
	}

	// Prepare form data for each update
	for _, update := range updates {
		data := url.Values{}
		
		// Add port identification
		data.Set("port", strconv.Itoa(update.PortID))
		
		// Add updates based on what's provided
		if update.Enabled != nil {
			if *update.Enabled {
				data.Set("enabled", "1")
			} else {
				data.Set("enabled", "0")
			}
		}
		
		if update.Mode != nil {
			data.Set("mode", string(*update.Mode))
		}
		
		if update.Priority != nil {
			data.Set("priority", string(*update.Priority))
		}
		
		if update.PowerLimitType != nil {
			data.Set("power_limit_type", string(*update.PowerLimitType))
		}
		
		if update.PowerLimitW != nil {
			data.Set("power_limit_w", fmt.Sprintf("%.2f", *update.PowerLimitW))
		}
		
		if update.DetectionType != nil {
			data.Set("detection_type", *update.DetectionType)
		}

		// Make the update request
		response, err := m.client.makeAuthenticatedRequest(ctx, "POST", endpoint, data)
		if err != nil {
			return NewOperationError(fmt.Sprintf("failed to update port %d", update.PortID), err)
		}

		// Check for errors in response
		if errorMsg := internal.ExtractErrorMessage(response); errorMsg != "" {
			return NewOperationError(fmt.Sprintf("update failed for port %d: %s", update.PortID, errorMsg), nil)
		}
	}

	return nil
}

// CyclePower performs a power cycle on specified ports
func (m *POEManager) CyclePower(ctx context.Context, portIDs ...int) error {
	if !m.client.IsAuthenticated() {
		return ErrNotAuthenticated
	}

	if len(portIDs) == 0 {
		return NewOperationError("no ports specified for power cycle", nil)
	}

	// Determine the appropriate endpoint based on model
	var endpoint string
	if m.client.model.IsModel30x() {
		endpoint = "/PoEPortConfig.cgi"
	} else if m.client.model.IsModel316() {
		endpoint = "/iss/specific/poePortConf.html"
	} else {
		return NewOperationError("POE power cycle not supported for this model", nil)
	}

	// Cycle power for each port
	for _, portID := range portIDs {
		data := url.Values{}
		data.Set("port", strconv.Itoa(portID))
		data.Set("action", "cycle")
		
		response, err := m.client.makeAuthenticatedRequest(ctx, "POST", endpoint, data)
		if err != nil {
			return NewOperationError(fmt.Sprintf("failed to cycle power for port %d", portID), err)
		}

		// Check for errors in response
		if errorMsg := internal.ExtractErrorMessage(response); errorMsg != "" {
			return NewOperationError(fmt.Sprintf("power cycle failed for port %d: %s", portID, errorMsg), nil)
		}

		if m.client.verbose {
			fmt.Printf("Successfully cycled power for port %d\n", portID)
		}
	}

	return nil
}

// EnablePort enables POE on the specified port
func (m *POEManager) EnablePort(ctx context.Context, portID int) error {
	enabled := true
	return m.UpdatePort(ctx, POEPortUpdate{
		PortID:  portID,
		Enabled: &enabled,
	})
}

// DisablePort disables POE on the specified port
func (m *POEManager) DisablePort(ctx context.Context, portID int) error {
	enabled := false
	return m.UpdatePort(ctx, POEPortUpdate{
		PortID:  portID,
		Enabled: &enabled,
	})
}

// SetPortMode sets the POE mode for a specific port
func (m *POEManager) SetPortMode(ctx context.Context, portID int, mode POEMode) error {
	return m.UpdatePort(ctx, POEPortUpdate{
		PortID: portID,
		Mode:   &mode,
	})
}

// SetPortPriority sets the POE priority for a specific port
func (m *POEManager) SetPortPriority(ctx context.Context, portID int, priority POEPriority) error {
	return m.UpdatePort(ctx, POEPortUpdate{
		PortID:   portID,
		Priority: &priority,
	})
}

// SetPortPowerLimit sets the power limit for a specific port
func (m *POEManager) SetPortPowerLimit(ctx context.Context, portID int, limitType POELimitType, limitW float64) error {
	return m.UpdatePort(ctx, POEPortUpdate{
		PortID:         portID,
		PowerLimitType: &limitType,
		PowerLimitW:    &limitW,
	})
}

// GetPortStatus gets the POE status for a specific port
func (m *POEManager) GetPortStatus(ctx context.Context, portID int) (*POEPortStatus, error) {
	statuses, err := m.GetStatus(ctx)
	if err != nil {
		return nil, err
	}

	for _, status := range statuses {
		if status.PortID == portID {
			return &status, nil
		}
	}

	return nil, NewOperationError(fmt.Sprintf("port %d not found", portID), nil)
}

// GetPortSettings gets the POE settings for a specific port
func (m *POEManager) GetPortSettings(ctx context.Context, portID int) (*POEPortSettings, error) {
	settings, err := m.GetSettings(ctx)
	if err != nil {
		return nil, err
	}

	for _, setting := range settings {
		if setting.PortID == portID {
			return &setting, nil
		}
	}

	return nil, NewOperationError(fmt.Sprintf("port %d not found", portID), nil)
}