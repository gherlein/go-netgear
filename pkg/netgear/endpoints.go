package netgear

import "fmt"

// EndpointRegistry manages model-specific endpoint mappings
type EndpointRegistry struct {
	model Model
}

// EndpointType represents different types of operations
type EndpointType string

const (
	EndpointLogin          EndpointType = "login"
	EndpointPOEStatus      EndpointType = "poe_status"
	EndpointPOESettings    EndpointType = "poe_settings"
	EndpointPOEUpdate      EndpointType = "poe_update"
	EndpointPortStatus     EndpointType = "port_status"
	EndpointPortSettings   EndpointType = "port_settings"
	EndpointPortUpdate     EndpointType = "port_update"
	EndpointDashboard      EndpointType = "dashboard"
)

// EndpointInfo contains endpoint URL and whether it's supported
type EndpointInfo struct {
	URL       string
	Supported bool
	Method    string // GET, POST, etc.
}

// NewEndpointRegistry creates a new endpoint registry
func NewEndpointRegistry(model Model) *EndpointRegistry {
	return &EndpointRegistry{model: model}
}

// GetEndpoint returns the endpoint info for a given operation type
func (er *EndpointRegistry) GetEndpoint(endpointType EndpointType) EndpointInfo {
	switch {
	case er.model.IsModel30x():
		return er.getGS30xEndpoint(endpointType)
	case er.model.IsModel316():
		return er.getGS316Endpoint(endpointType)
	default:
		return EndpointInfo{URL: "", Supported: false}
	}
}

// getGS30xEndpoint returns endpoints for GS30x series (GS305EP, GS308EP, GS316EP, etc.)
func (er *EndpointRegistry) getGS30xEndpoint(endpointType EndpointType) EndpointInfo {
	switch endpointType {
	case EndpointLogin:
		return EndpointInfo{URL: "/login.cgi", Supported: true, Method: "POST"}
	case EndpointPOEStatus:
		return EndpointInfo{URL: "/getPoePortStatus.cgi", Supported: true, Method: "GET"}
	case EndpointPOESettings:
		return EndpointInfo{URL: "/PoEPortConfig.cgi", Supported: true, Method: "GET"}
	case EndpointPOEUpdate:
		return EndpointInfo{URL: "/PoEPortConfig.cgi", Supported: true, Method: "POST"}
	case EndpointPortStatus:
		// GS30x series doesn't have a dedicated port status endpoint - use dashboard
		return EndpointInfo{URL: "/dashboard.cgi", Supported: false, Method: "GET"}
	case EndpointPortSettings:
		// GS30x series doesn't have a dedicated port settings endpoint
		return EndpointInfo{URL: "/dashboard.cgi", Supported: false, Method: "GET"}
	case EndpointPortUpdate:
		// GS30x series doesn't have a dedicated port update endpoint - NOT SUPPORTED
		return EndpointInfo{URL: "/PortConfig.cgi", Supported: false, Method: "POST"}
	case EndpointDashboard:
		return EndpointInfo{URL: "/dashboard.cgi", Supported: true, Method: "GET"}
	default:
		return EndpointInfo{URL: "", Supported: false}
	}
}

// getGS316Endpoint returns endpoints for GS316 series
func (er *EndpointRegistry) getGS316Endpoint(endpointType EndpointType) EndpointInfo {
	switch endpointType {
	case EndpointLogin:
		return EndpointInfo{URL: "/login.cgi", Supported: true, Method: "POST"}
	case EndpointPOEStatus:
		return EndpointInfo{URL: "/iss/specific/poePortStatus.html", Supported: true, Method: "GET"}
	case EndpointPOESettings:
		return EndpointInfo{URL: "/iss/specific/poePortConf.html", Supported: true, Method: "GET"}
	case EndpointPOEUpdate:
		return EndpointInfo{URL: "/iss/specific/poePortConf.html", Supported: true, Method: "POST"}
	case EndpointPortStatus:
		return EndpointInfo{URL: "/iss/specific/interface.html", Supported: true, Method: "GET"}
	case EndpointPortSettings:
		return EndpointInfo{URL: "/iss/specific/interface.html", Supported: true, Method: "GET"}
	case EndpointPortUpdate:
		return EndpointInfo{URL: "/iss/specific/interface.html", Supported: true, Method: "POST"}
	case EndpointDashboard:
		return EndpointInfo{URL: "/iss/specific/dashboard.html", Supported: true, Method: "GET"}
	default:
		return EndpointInfo{URL: "", Supported: false}
	}
}

// IsEndpointSupported checks if an endpoint is supported for the current model
func (er *EndpointRegistry) IsEndpointSupported(endpointType EndpointType) bool {
	return er.GetEndpoint(endpointType).Supported
}

// GetSupportedEndpoints returns all supported endpoints for the current model
func (er *EndpointRegistry) GetSupportedEndpoints() map[EndpointType]EndpointInfo {
	allEndpoints := []EndpointType{
		EndpointLogin, EndpointPOEStatus, EndpointPOESettings, EndpointPOEUpdate,
		EndpointPortStatus, EndpointPortSettings, EndpointPortUpdate, EndpointDashboard,
	}

	supported := make(map[EndpointType]EndpointInfo)
	for _, endpoint := range allEndpoints {
		info := er.GetEndpoint(endpoint)
		if info.Supported {
			supported[endpoint] = info
		}
	}
	return supported
}

// ValidateEndpoint checks if an endpoint exists and returns appropriate error
func (er *EndpointRegistry) ValidateEndpoint(endpointType EndpointType) error {
	info := er.GetEndpoint(endpointType)
	if !info.Supported {
		return NewOperationError(
			fmt.Sprintf("%s operation not supported on %s model",
				string(endpointType), string(er.model)), nil)
	}
	return nil
}