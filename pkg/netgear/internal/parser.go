package internal

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// ModelDetector contains logic for detecting Netgear switch models
type ModelDetector struct{}

// NewModelDetector creates a new model detector
func NewModelDetector() *ModelDetector {
	return &ModelDetector{}
}

// DetectFromHTML attempts to detect the switch model from HTML content
func (md *ModelDetector) DetectFromHTML(htmlContent string) string {
	// Check for most specific models first to avoid partial matches
	specificModels := []string{"GS316EPP", "GS308EPP", "GS305EPP", "GS316EP", "GS308EP", "GS305EP"}
	for _, model := range specificModels {
		if strings.Contains(htmlContent, model) {
			return model
		}
	}
	
	// If no specific model found but it looks like a redirect page, assume GS30xEPx
	if strings.Contains(htmlContent, "Redirect to Login") || 
	   strings.Contains(htmlContent, "redirect") {
		return "GS30xEPx"
	}
	
	return ""
}

// POEDataParser contains logic for parsing POE-related data
type POEDataParser struct{}

// NewPOEDataParser creates a new POE data parser
func NewPOEDataParser() *POEDataParser {
	return &POEDataParser{}
}

// ParsePOEStatus parses POE status data from HTML/JavaScript response
func (p *POEDataParser) ParsePOEStatus(content string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	
	// Parse GS30x series format (li.poePortStatusListItem or li.poe_port_list_item)
	doc.Find("li.poePortStatusListItem, li.poe_port_list_item").Each(func(i int, s *goquery.Selection) {
		portData := make(map[string]interface{})
		
		// Extract port ID from hidden input
		if id, exists := s.Find("input[type=hidden].port").Attr("value"); exists {
			if portID, err := strconv.Atoi(id); err == nil {
				portData["port_id"] = portID
			}
		}
		
		// Extract port name from poe-port-index span
		if portText := strings.TrimSpace(s.Find("span.poe-port-index span").Text()); portText != "" {
			portData["port_name"] = portText
		}
		
		// Extract POE status from poe-power-mode span
		if status := strings.TrimSpace(s.Find("span.poe-power-mode span").Text()); status != "" {
			portData["status"] = status
		}
		
		// Extract power class from poe-portPwr-width span
		if powerClass := strings.TrimSpace(s.Find("span.poe-portPwr-width span").Text()); powerClass != "" {
			portData["power_class"] = powerClass
		}
		
		// Extract voltage, current, and power from poe_port_status divs
		s.Find("div.poe_port_status div div span").Each(func(j int, span *goquery.Selection) {
			text := strings.TrimSpace(span.Text())
			if text == "" {
				return
			}
			
			// Try to extract numeric values
			if strings.Contains(text, "V") {
				// Voltage
				if val := extractNumericValue(text); val > 0 {
					portData["voltage_v"] = val
				}
			} else if strings.Contains(text, "mA") {
				// Current
				if val := extractNumericValue(text); val > 0 {
					portData["current_ma"] = val
				}
			} else if strings.Contains(text, "W") {
				// Power
				if val := extractNumericValue(text); val > 0 {
					portData["power_w"] = val
				}
			}
		})
		
		// Only add if we found at least a port ID
		if _, hasPortID := portData["port_id"]; hasPortID {
			results = append(results, portData)
		}
	})
	
	// If no GS30x format found, try generic table parsing as fallback
	if len(results) == 0 {
		doc.Find("table").Each(func(i int, table *goquery.Selection) {
			table.Find("tr").Each(func(j int, row *goquery.Selection) {
				if j == 0 {
					return // Skip header row
				}
				
				portData := make(map[string]interface{})
				row.Find("td").Each(func(k int, cell *goquery.Selection) {
					cellText := strings.TrimSpace(cell.Text())
					switch k {
					case 0:
						if portID, err := strconv.Atoi(cellText); err == nil {
							portData["port_id"] = portID
						}
					case 1:
						portData["port_name"] = cellText
					case 2:
						portData["status"] = cellText
					case 3:
						portData["power_class"] = cellText
					case 4:
						if voltage, err := strconv.ParseFloat(cellText, 64); err == nil {
							portData["voltage_v"] = voltage
						}
					case 5:
						if current, err := strconv.ParseFloat(cellText, 64); err == nil {
							portData["current_ma"] = current
						}
					case 6:
						if power, err := strconv.ParseFloat(cellText, 64); err == nil {
							portData["power_w"] = power
						}
					}
				})
				
				if len(portData) > 0 {
					results = append(results, portData)
				}
			})
		})
	}
	
	return results, nil
}

// ParsePOESettings parses POE settings data from HTML/JavaScript response
func (p *POEDataParser) ParsePOESettings(content string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	// For GS30x series (like GS308EPP), the POE settings are in div.poe-port-box elements
	// First, try to find port circles to get port numbers
	portNumbers := make([]int, 0)
	doc.Find("li.port_circle span.port_circle_num").Each(func(i int, s *goquery.Selection) {
		portText := strings.TrimSpace(s.Text())
		if portID, err := strconv.Atoi(portText); err == nil {
			portNumbers = append(portNumbers, portID)
		}
	})

	// Look for security hash token
	var securityHash string
	doc.Find("input[name='hash'], input[id='hash']").Each(func(i int, s *goquery.Selection) {
		if value, exists := s.Attr("value"); exists && value != "" {
			securityHash = value
		}
	})

	// Store the security hash in the results if found
	securityHashData := map[string]interface{}{
		"security_hash": securityHash,
	}
	if securityHash != "" {
		results = append(results, securityHashData)
	}

	// If we found port numbers, create default settings for each port
	if len(portNumbers) > 0 {
		for _, portID := range portNumbers {
			portData := map[string]interface{}{
				"port_id":              portID,
				"port_name":            fmt.Sprintf("Port %d", portID),
				"enabled":              true,  // Default assumption - POE is typically enabled
				"mode":                 "auto", // Default mode
				"priority":             "low",  // Default priority
				"power_limit_type":     "class", // Default limit type
				"power_limit_w":        30.0,   // Default 30W limit for POE+
				"detection_type":       "ieee", // Default IEEE detection
				"longer_detection_time": false, // Default no longer detection
			}

			// Try to extract actual settings for this port from various input elements
			// Look for port-specific inputs that might contain real settings
			portSelector := fmt.Sprintf("[data-port='%d'], [name*='port%d'], [id*='port%d'], [id*='Port%d']",
				portID, portID, portID, portID)

			doc.Find(portSelector).Each(func(j int, input *goquery.Selection) {
				inputType, _ := input.Attr("type")
				inputName, _ := input.Attr("name")
				inputId, _ := input.Attr("id")
				isChecked := input.Is(":checked") || input.AttrOr("checked", "") != ""

				// Debug output commented out for now
				// inputValue, _ := input.Attr("value")
				// fmt.Printf("DEBUG: Port %d input: type='%s' name='%s' value='%s' id='%s' checked=%v\n",
				//	portID, inputType, inputName, inputValue, inputId, isChecked)

				// Extract settings based on input type and name patterns
				if inputType == "checkbox" {
					if strings.Contains(strings.ToLower(inputName), "enable") ||
					   strings.Contains(strings.ToLower(inputId), "enable") {
						portData["enabled"] = isChecked
					}
				}
			})

			results = append(results, portData)
		}
	}

	// If no port-specific parsing worked, fall back to the original method for any forms/tables
	if len(results) == 0 {
		doc.Find("form, table").Each(func(i int, element *goquery.Selection) {
			settingsData := make(map[string]interface{})

			element.Find("input, select").Each(func(j int, input *goquery.Selection) {
				name, _ := input.Attr("name")
				value, _ := input.Attr("value")

				if name != "" {
					settingsData[name] = value
				}
			})

			if len(settingsData) > 0 {
				results = append(results, settingsData)
			}
		})
	}

	// fmt.Printf("DEBUG: ParsePOESettings returning %d results\n", len(results))
	return results, nil
}

// PortDataParser contains logic for parsing port-related data
type PortDataParser struct{}

// NewPortDataParser creates a new port data parser
func NewPortDataParser() *PortDataParser {
	return &PortDataParser{}
}

// ParsePortSettings parses port settings from HTML content
func (p *PortDataParser) ParsePortSettings(content string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	
	// Parse port settings from tables or forms
	doc.Find("table").Each(func(i int, table *goquery.Selection) {
		table.Find("tr").Each(func(j int, row *goquery.Selection) {
			if j == 0 {
				return // Skip header
			}
			
			portData := make(map[string]interface{})
			row.Find("td").Each(func(k int, cell *goquery.Selection) {
				cellText := strings.TrimSpace(cell.Text())
				switch k {
				case 0:
					if portID, err := strconv.Atoi(cellText); err == nil {
						portData["port_id"] = portID
					}
				case 1:
					portData["port_name"] = cellText
				case 2:
					portData["speed"] = cellText
				case 3:
					portData["ingress_limit"] = cellText
				case 4:
					portData["egress_limit"] = cellText
				case 5:
					portData["flow_control"] = strings.ToLower(cellText) == "on"
				case 6:
					portData["status"] = cellText
				case 7:
					portData["link_speed"] = cellText
				}
			})
			
			if len(portData) > 0 {
				results = append(results, portData)
			}
		})
	})
	
	return results, nil
}

// ExtractSessionToken extracts session token from response content
func ExtractSessionToken(content string) string {
	// Look for SID cookie or session token in various formats
	// Updated patterns to match the actual token format which can contain special characters
	patterns := []string{
		`SID=([^;]+)`,  // Match everything up to semicolon for SID cookie
		`sessionid=([^;]+)`,  // Match everything up to semicolon for sessionid
		`token["\s]*[:=]["\s]*"([^"]+)"`,  // Match quoted token values
		`token["\s]*[:=]["\s]*([a-fA-F0-9]+)`,  // Match hex token values (fallback)
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			return matches[1]
		}
	}

	return ""
}

// ExtractGambitToken extracts Gambit token from response
func ExtractGambitToken(content string) string {
	// Look for Gambit token in JavaScript or HTML
	patterns := []string{
		`Gambit["\s]*[:=]["\s]*([a-fA-F0-9]+)`,
		`gambit["\s]*[:=]["\s]*([a-fA-F0-9]+)`,
		`rand["\s]*[:=]["\s]*([0-9]+)`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			return matches[1]
		}
	}
	
	return ""
}

// ExtractErrorMessage extracts error messages from response content
func ExtractErrorMessage(content string) string {
	// Look for common error patterns
	patterns := []string{
		`error["\s]*[:=]["\s]*"([^"]+)"`,
		`<div[^>]*error[^>]*>([^<]+)</div>`,
		`alert\s*\(\s*"([^"]+)"\s*\)`,
	}
	
	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(content)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}
	}
	
	return ""
}

// ExtractSeedValue extracts the random seed value from login page HTML
func ExtractSeedValue(content string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return ""
	}
	
	// Look for input element with id="rand"
	randVal, exists := doc.Find("#rand").First().Attr("value")
	if exists {
		return randVal
	}
	
	return ""
}

// extractNumericValue extracts a numeric value from a string that may contain units
func extractNumericValue(text string) float64 {
	// Remove common units and non-numeric characters
	cleaned := strings.ReplaceAll(text, "V", "")
	cleaned = strings.ReplaceAll(cleaned, "mA", "")
	cleaned = strings.ReplaceAll(cleaned, "W", "")
	cleaned = strings.ReplaceAll(cleaned, "A", "")
	cleaned = strings.TrimSpace(cleaned)
	
	if val, err := strconv.ParseFloat(cleaned, 64); err == nil {
		return val
	}
	
	return 0
}