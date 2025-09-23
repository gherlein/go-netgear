package netgear

// Model represents a Netgear switch model
type Model string

const (
	ModelGS305EP  Model = "GS305EP"
	ModelGS305EPP Model = "GS305EPP"
	ModelGS308EP  Model = "GS308EP"
	ModelGS308EPP Model = "GS308EPP"
	ModelGS316EP  Model = "GS316EP"
	ModelGS316EPP Model = "GS316EPP"
	ModelGS30xEPx Model = "GS30xEPx"
)

// IsModel30x returns true if the model is part of the 30x series
func (m Model) IsModel30x() bool {
	switch m {
	case ModelGS305EP, ModelGS305EPP, ModelGS308EP, ModelGS308EPP, ModelGS30xEPx:
		return true
	default:
		return false
	}
}

// IsModel316 returns true if the model is part of the 316 series
func (m Model) IsModel316() bool {
	switch m {
	case ModelGS316EP, ModelGS316EPP:
		return true
	default:
		return false
	}
}

// IsSupported returns true if the model is supported
func (m Model) IsSupported() bool {
	switch m {
	case ModelGS305EP, ModelGS305EPP, ModelGS308EP, ModelGS308EPP, 
		 ModelGS316EP, ModelGS316EPP, ModelGS30xEPx:
		return true
	default:
		return false
	}
}

// POEPortStatus represents the status of a POE port
type POEPortStatus struct {
	PortID       int     `json:"port_id"`
	PortName     string  `json:"port_name"`
	Status       string  `json:"status"`
	PowerClass   string  `json:"power_class"`
	VoltageV     float64 `json:"voltage_v"`
	CurrentMA    float64 `json:"current_ma"`
	PowerW       float64 `json:"power_w"`
	TemperatureC float64 `json:"temperature_c"`
	ErrorStatus  string  `json:"error_status"`
}

// POEPortSettings represents POE port configuration
type POEPortSettings struct {
	PortID              int          `json:"port_id"`
	PortName            string       `json:"port_name"`
	Enabled             bool         `json:"enabled"`
	Mode                POEMode      `json:"mode"`
	Priority            POEPriority  `json:"priority"`
	PowerLimitType      POELimitType `json:"power_limit_type"`
	PowerLimitW         float64      `json:"power_limit_w"`
	DetectionType       string       `json:"detection_type"`
	LongerDetectionTime bool         `json:"longer_detection_time"`
}

// PortSettings represents switch port configuration
type PortSettings struct {
	PortID       int        `json:"port_id"`
	PortName     string     `json:"port_name"`
	Speed        PortSpeed  `json:"speed"`
	IngressLimit string     `json:"ingress_limit"`
	EgressLimit  string     `json:"egress_limit"`
	FlowControl  bool       `json:"flow_control"`
	Status       PortStatus `json:"status"`
	LinkSpeed    string     `json:"link_speed"`
}

// POEMode represents POE power mode
type POEMode string

const (
	POEMode8023af    POEMode = "802.3af"
	POEMode8023at    POEMode = "802.3at"
	POEModeLegacy    POEMode = "legacy"
	POEModePre8023at POEMode = "pre-802.3at"
)

// POEPriority represents POE port priority
type POEPriority string

const (
	POEPriorityLow      POEPriority = "low"
	POEPriorityHigh     POEPriority = "high"
	POEPriorityCritical POEPriority = "critical"
)

// POELimitType represents POE power limit type
type POELimitType string

const (
	POELimitTypeNone  POELimitType = "none"
	POELimitTypeClass POELimitType = "class"
	POELimitTypeUser  POELimitType = "user"
)

// PortSpeed represents port speed configuration
type PortSpeed string

const (
	PortSpeedAuto     PortSpeed = "auto"
	PortSpeed10MHalf  PortSpeed = "10M half"
	PortSpeed10MFull  PortSpeed = "10M full"
	PortSpeed100MHalf PortSpeed = "100M half"
	PortSpeed100MFull PortSpeed = "100M full"
	PortSpeedDisable  PortSpeed = "disable"
)

// PortStatus represents port status
type PortStatus string

const (
	PortStatusAvailable PortStatus = "available"
	PortStatusConnected PortStatus = "connected"
	PortStatusDisabled  PortStatus = "disabled"
)

// POEPortUpdate represents changes to apply to a POE port
type POEPortUpdate struct {
	PortID         int           `json:"port_id"`
	Enabled        *bool         `json:"enabled,omitempty"`
	Mode           *POEMode      `json:"mode,omitempty"`
	Priority       *POEPriority  `json:"priority,omitempty"`
	PowerLimitType *POELimitType `json:"power_limit_type,omitempty"`
	PowerLimitW    *float64      `json:"power_limit_w,omitempty"`
	DetectionType  *string       `json:"detection_type,omitempty"`
}

// PortUpdate represents changes to apply to a port
type PortUpdate struct {
	PortID       int        `json:"port_id"`
	Name         *string    `json:"name,omitempty"`
	Speed        *PortSpeed `json:"speed,omitempty"`
	IngressLimit *string    `json:"ingress_limit,omitempty"`
	EgressLimit  *string    `json:"egress_limit,omitempty"`
	FlowControl  *bool      `json:"flow_control,omitempty"`
}