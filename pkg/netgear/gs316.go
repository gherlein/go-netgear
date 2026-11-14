package netgear

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// GS316 series PoE support. The request formats mirror what the switch's own
// web UI (function.js) sends, and the parsers were written against pages
// captured from a GS316EPP; see testdata/gs316epp.

const (
	gs316PoEPortCount  = 15
	gs316PoEStatusPath = "/iss/specific/poePortStatus.html"
	gs316PoEConfigPath = "/iss/specific/poePortConf.html"
	gs316LogoutPath    = "/iss/specific/logout.html"

	gs316NotSet         = "NOTSET"
	gs316ReplySuccess   = "SUCCESS"
	gs316ReplyScheduled = "ONSCHE"
	gs316AdminEnabled   = "Enable"

	// replyExcerptLength bounds how much of an unexpected reply goes into an
	// error, since a refusal can be a whole HTML page.
	replyExcerptLength = 120
)

func (m *POEManager) getStatusGS316(ctx context.Context) ([]POEPortStatus, error) {
	// Without GetData=TRUE the firmware returns its unfilled template, with
	// placeholders where the readings belong.
	query := url.Values{"GetData": {"TRUE"}}
	response, err := m.client.makeAuthenticatedRequest(ctx, "GET", gs316PoEStatusPath, query)
	if err != nil {
		return nil, NewOperationError("failed to get POE status", err)
	}
	statuses, err := parseGS316POEStatus(response)
	if err != nil {
		return nil, NewParsingError("failed to parse POE status", err)
	}
	return statuses, nil
}

func (m *POEManager) getSettingsGS316(ctx context.Context) ([]POEPortSettings, error) {
	response, err := m.client.makeAuthenticatedRequest(ctx, "GET", gs316PoEConfigPath, nil)
	if err != nil {
		return nil, NewOperationError("failed to get POE settings", err)
	}
	settings, err := parseGS316POESettings(response)
	if err != nil {
		return nil, NewParsingError("failed to parse POE settings", err)
	}
	return settings, nil
}

// updatePortsGS316 validates every update before sending any, so a bad entry
// cannot leave the batch half applied.
func (m *POEManager) updatePortsGS316(ctx context.Context, updates []POEPortUpdate) error {
	for _, update := range updates {
		if err := validateGS316Port(update.PortID); err != nil {
			return err
		}
		if update.Mode != nil || update.Priority != nil || update.PowerLimitType != nil ||
			update.PowerLimitW != nil || update.DetectionType != nil {
			return NewOperationError(fmt.Sprintf("port %d: only Enabled can be changed on GS316 switches", update.PortID), nil)
		}
		if update.Enabled == nil {
			return NewOperationError(fmt.Sprintf("port %d: update changes nothing", update.PortID), nil)
		}
	}

	for _, update := range updates {
		adminState := "0"
		if *update.Enabled {
			adminState = "1"
		}
		// Same fields in the same order as the UI's submitPoEports.
		form := []formField{
			{"TYPE", "submitPoe"},
			{"PORT_NO", strconv.Itoa(update.PortID)},
			{"POWER_LIMIT_VALUE", gs316NotSet},
			{"PRIORITY", gs316NotSet},
			{"POWER_MODE", gs316NotSet},
			{"POWER_LIMIT_TYPE", gs316NotSet},
			{"DETECTION", gs316NotSet},
			{"ADMIN_STATE", adminState},
			{"DISCONNECT_TYPE", gs316NotSet},
		}
		response, err := m.client.postOrderedGambitForm(ctx, gs316PoEConfigPath, form)
		if err != nil {
			return NewOperationError(fmt.Sprintf("failed to update port %d", update.PortID), err)
		}
		if err := checkGS316Reply(response); err != nil {
			return NewOperationError(fmt.Sprintf("update failed for port %d", update.PortID), err)
		}
	}
	return nil
}

func (m *POEManager) cyclePowerGS316(ctx context.Context, portIDs []int) error {
	selected := make(map[int]bool, len(portIDs))
	for _, portID := range portIDs {
		if err := validateGS316Port(portID); err != nil {
			return err
		}
		selected[portID] = true
	}

	// One character per PoE port, in port order, "1" marking a port to cycle.
	var ports strings.Builder
	for portID := 1; portID <= gs316PoEPortCount; portID++ {
		if selected[portID] {
			ports.WriteByte('1')
		} else {
			ports.WriteByte('0')
		}
	}

	form := []formField{{"TYPE", "resetPoe"}, {"PoePort", ports.String()}}
	response, err := m.client.postOrderedGambitForm(ctx, gs316PoEConfigPath, form)
	if err != nil {
		return NewOperationError(fmt.Sprintf("failed to cycle power for ports %v", portIDs), err)
	}
	if err := checkGS316Reply(response); err != nil {
		return NewOperationError(fmt.Sprintf("power cycle failed for ports %v", portIDs), err)
	}
	return nil
}

func validateGS316Port(portID int) error {
	if portID < 1 || portID > gs316PoEPortCount {
		return NewOperationError(fmt.Sprintf("port %d is outside the PoE ports 1..%d", portID, gs316PoEPortCount), nil)
	}
	return nil
}

func checkGS316Reply(response string) error {
	reply := strings.TrimSpace(response)
	switch reply {
	case gs316ReplySuccess:
		return nil
	case gs316ReplyScheduled:
		return fmt.Errorf("the PoE schedule is enabled on this port; disable it before changing port power")
	}
	if len(reply) > replyExcerptLength {
		reply = reply[:replyExcerptLength] + "..."
	}
	return fmt.Errorf("switch replied %q", reply)
}

func parseGS316POEStatus(content string) ([]POEPortStatus, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}
	var statuses []POEPortStatus
	var parseErr error
	doc.Find("div.port-wrap").EachWithBreak(func(_ int, port *goquery.Selection) bool {
		status, err := parseGS316PortStatus(port)
		if err != nil {
			parseErr = err
			return false
		}
		statuses = append(statuses, status)
		return true
	})
	if parseErr != nil {
		return nil, parseErr
	}
	if len(statuses) == 0 {
		return nil, fmt.Errorf("no PoE ports found in status page")
	}
	return statuses, nil
}

func parseGS316PortStatus(port *goquery.Selection) (POEPortStatus, error) {
	portID, portName, err := parseGS316PortNumber(port)
	if err != nil {
		return POEPortStatus{}, err
	}
	status := POEPortStatus{
		PortID:      portID,
		PortName:    portName,
		Status:      fieldText(port, "span.Status-text"),
		PowerClass:  fieldText(port, "span.Class-text"),
		ErrorStatus: fieldText(port, "p.Fault-Status-text"),
	}
	numbers := []struct {
		selector string
		target   *float64
	}{
		{"p.OutputVoltage-text", &status.VoltageV},
		{"p.OutputCurrent-text", &status.CurrentMA},
		{"p.OutputPower-text", &status.PowerW},
		{"p.Temperature-text", &status.TemperatureC},
	}
	for _, number := range numbers {
		value, err := parseFloatField(port, number.selector)
		if err != nil {
			return POEPortStatus{}, fmt.Errorf("port %d: %w", portID, err)
		}
		*number.target = value
	}
	return status, nil
}

func parseGS316POESettings(content string) ([]POEPortSettings, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("parse HTML: %w", err)
	}
	var settings []POEPortSettings
	var parseErr error
	doc.Find("div#POE_SETTING div.port-wrap").EachWithBreak(func(_ int, port *goquery.Selection) bool {
		setting, err := parseGS316PortSettings(port)
		if err != nil {
			parseErr = err
			return false
		}
		settings = append(settings, setting)
		return true
	})
	if parseErr != nil {
		return nil, parseErr
	}
	if len(settings) == 0 {
		return nil, fmt.Errorf("no PoE ports found in settings page")
	}
	return settings, nil
}

func parseGS316PortSettings(port *goquery.Selection) (POEPortSettings, error) {
	portID, portName, err := parseGS316PortNumber(port)
	if err != nil {
		return POEPortSettings{}, err
	}
	setting := POEPortSettings{
		PortID:              portID,
		PortName:            portName,
		Enabled:             fieldText(port, "span.admin-state") == gs316AdminEnabled,
		Mode:                POEMode(fieldText(port, "span.Power-Mode-text")),
		Priority:            POEPriority(strings.ToLower(fieldText(port, "p.port-priority"))),
		PowerLimitType:      POELimitType(strings.ToLower(fieldText(port, "p.Power-Limit-Type-text"))),
		DetectionType:       fieldText(port, "p.Detection-Type-text"),
		LongerDetectionTime: fieldText(port, "p.Longer-Detection-text") == gs316AdminEnabled,
	}
	// The firmware leaves the limit empty for limit types without a value.
	if fieldText(port, "p.Power-Limit-text") != "" {
		limit, err := parseFloatField(port, "p.Power-Limit-text")
		if err != nil {
			return POEPortSettings{}, fmt.Errorf("port %d: %w", portID, err)
		}
		setting.PowerLimitW = limit
	}
	return setting, nil
}

// parseGS316PortNumber reads "N" or "N - name"; the firmware separates a
// configured port name with a non-breaking space.
func parseGS316PortNumber(port *goquery.Selection) (int, string, error) {
	text := strings.ReplaceAll(fieldText(port, "span.port-number"), " ", " ")
	number, name, _ := strings.Cut(text, " - ")
	portID, err := strconv.Atoi(strings.TrimSpace(number))
	if err != nil {
		return 0, "", fmt.Errorf("port number %q: %w", text, err)
	}
	return portID, strings.TrimSpace(name), nil
}

func fieldText(port *goquery.Selection, selector string) string {
	return strings.TrimSpace(port.Find(selector).First().Text())
}

func parseFloatField(port *goquery.Selection, selector string) (float64, error) {
	text := fieldText(port, selector)
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, fmt.Errorf("%s value %q is not a number", selector, text)
	}
	return value, nil
}
