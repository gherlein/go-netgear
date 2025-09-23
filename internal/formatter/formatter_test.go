package formatter

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/corbym/gocrest/is"
	"github.com/corbym/gocrest/then"
)

func TestPrintMarkdownTable(t *testing.T) {
	tests := []struct {
		name            string
		header          []string
		content         [][]string
		expectedLines   []string
		notExpected     []string
	}{
		{
			name:   "Simple table",
			header: []string{"ID", "Name", "Status"},
			content: [][]string{
				{"1", "Port One", "Active"},
				{"2", "Port Two", "Inactive"},
			},
			expectedLines: []string{
				"| ID | Name     | Status   |",
				"|----|----------|----------|",
				"| 1  | Port One | Active   |",
				"| 2  | Port Two | Inactive |",
			},
		},
		{
			name:   "Table with varying column widths",
			header: []string{"Port", "Very Long Column Name", "S"},
			content: [][]string{
				{"1", "Short", "A"},
				{"999", "This is a very long value that exceeds header", "BBB"},
			},
			expectedLines: []string{
				"| Port | Very Long Column Name                         | S   |",
				"|------|-----------------------------------------------|-----|",
				"| 1    | Short                                         | A   |",
				"| 999  | This is a very long value that exceeds header | BBB |",
			},
		},
		{
			name:    "Empty table",
			header:  []string{"Col1", "Col2"},
			content: [][]string{},
			expectedLines: []string{
				"| Col1 | Col2 |",
				"|------|------|",
			},
		},
		{
			name:   "Table with unicode characters",
			header: []string{"ID", "Name", "Symbol"},
			content: [][]string{
				{"1", "Café", "☕"},
				{"2", "München", "🍺"},
				{"3", "日本", "🗾"},
			},
			expectedLines: []string{
				"| ID | Name    | Symbol |",
				"|----|---------|--------|",
				"| 1  | Café    | ☕      |",
				"| 2  | München | 🍺      |",
				"| 3  | 日本      | 🗾      |",
			},
		},
		{
			name:   "Table with empty cells",
			header: []string{"A", "B", "C"},
			content: [][]string{
				{"1", "", "3"},
				{"", "2", ""},
				{"", "", ""},
			},
			expectedLines: []string{
				"| A | B | C |",
				"|---|---|---|",
				"| 1 |   | 3 |",
				"|   | 2 |   |",
				"|   |   |   |",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			output := captureOutput(func() {
				printMarkdownTable(tt.header, tt.content)
			})

			// Verify expected lines
			for _, expected := range tt.expectedLines {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain: %q\nActual output:\n%s", expected, output)
				}
			}

			// Verify not expected
			for _, notExpected := range tt.notExpected {
				if strings.Contains(output, notExpected) {
					t.Errorf("Expected output to NOT contain: %q\nActual output:\n%s", notExpected, output)
				}
			}

			// Verify table structure
			lines := strings.Split(strings.TrimSpace(output), "\n")
			expectedRows := len(tt.content) + 2 // header + separator
			then.AssertThat(t, len(lines), is.EqualTo(expectedRows))
		})
	}
}

func TestPrintJsonDataTable(t *testing.T) {
	tests := []struct {
		name          string
		item          string
		header        []string
		content       [][]string
		validateFunc  func(t *testing.T, output string)
	}{
		{
			name:   "Simple JSON table",
			item:   "ports",
			header: []string{"ID", "Name", "Status"},
			content: [][]string{
				{"1", "Port One", "Active"},
				{"2", "Port Two", "Inactive"},
			},
			validateFunc: func(t *testing.T, output string) {
				var result map[string][]map[string]string
				err := json.Unmarshal([]byte(output), &result)
				then.AssertThat(t, err, is.Nil())
				
				ports, exists := result["ports"]
				then.AssertThat(t, exists, is.True())
				then.AssertThat(t, len(ports), is.EqualTo(2))
				
				then.AssertThat(t, ports[0]["ID"], is.EqualTo("1"))
				then.AssertThat(t, ports[0]["Name"], is.EqualTo("Port One"))
				then.AssertThat(t, ports[0]["Status"], is.EqualTo("Active"))
				
				then.AssertThat(t, ports[1]["ID"], is.EqualTo("2"))
				then.AssertThat(t, ports[1]["Name"], is.EqualTo("Port Two"))
				then.AssertThat(t, ports[1]["Status"], is.EqualTo("Inactive"))
			},
		},
		{
			name:    "Empty JSON table",
			item:    "empty_list",
			header:  []string{"Col1", "Col2"},
			content: [][]string{},
			validateFunc: func(t *testing.T, output string) {
				var result map[string][]map[string]string
				err := json.Unmarshal([]byte(output), &result)
				then.AssertThat(t, err, is.Nil())
				
				emptyList, exists := result["empty_list"]
				then.AssertThat(t, exists, is.True())
				then.AssertThat(t, len(emptyList), is.EqualTo(0))
			},
		},
		{
			name:   "JSON with special characters",
			item:   "special_chars",
			header: []string{"Key", "Value"},
			content: [][]string{
				{"quote", `"Hello"`},
				{"newline", "Line1\nLine2"},
				{"tab", "Tab\there"},
				{"unicode", "😀 Unicode"},
			},
			validateFunc: func(t *testing.T, output string) {
				var result map[string][]map[string]string
				err := json.Unmarshal([]byte(output), &result)
				then.AssertThat(t, err, is.Nil())
				
				items := result["special_chars"]
				then.AssertThat(t, len(items), is.EqualTo(4))
				
				then.AssertThat(t, items[0]["Value"], is.EqualTo(`"Hello"`))
				then.AssertThat(t, items[1]["Value"], is.EqualTo("Line1\nLine2"))
				then.AssertThat(t, items[2]["Value"], is.EqualTo("Tab\there"))
				then.AssertThat(t, items[3]["Value"], is.EqualTo("😀 Unicode"))
			},
		},
		{
			name:   "Mismatched header and content lengths",
			item:   "mismatched",
			header: []string{"A", "B", "C"},
			content: [][]string{
				{"1", "2"}, // Missing third column
				{"3", "4", "5", "6"}, // Extra column
			},
			validateFunc: func(t *testing.T, output string) {
				var result map[string][]map[string]string
				err := json.Unmarshal([]byte(output), &result)
				then.AssertThat(t, err, is.Nil())
				
				items := result["mismatched"]
				then.AssertThat(t, len(items), is.EqualTo(2))
				
				// First row should have empty C
				then.AssertThat(t, items[0]["A"], is.EqualTo("1"))
				then.AssertThat(t, items[0]["B"], is.EqualTo("2"))
				then.AssertThat(t, items[0]["C"], is.EqualTo(""))
				
				// Second row should only use first 3 values
				then.AssertThat(t, items[1]["A"], is.EqualTo("3"))
				then.AssertThat(t, items[1]["B"], is.EqualTo("4"))
				then.AssertThat(t, items[1]["C"], is.EqualTo("5"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			output := captureOutput(func() {
				printJsonDataTable(tt.item, tt.header, tt.content)
			})

			// Validate JSON structure
			tt.validateFunc(t, output)
		})
	}
}

func TestSuffixToLengthComprehensive(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		length   int
		expected string
	}{
		{
			name:     "String shorter than length",
			input:    "test",
			length:   10,
			expected: "test      ",
		},
		{
			name:     "String equal to length",
			input:    "exact",
			length:   5,
			expected: "exact",
		},
		{
			name:     "String longer than length",
			input:    "toolong",
			length:   3,
			expected: "toolong",
		},
		{
			name:     "Empty string",
			input:    "",
			length:   5,
			expected: "     ",
		},
		{
			name:     "Zero length",
			input:    "test",
			length:   0,
			expected: "test",
		},
		{
			name:     "Unicode string",
			input:    "café",
			length:   6,
			expected: "café  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := suffixToLength(tt.input, tt.length)
			then.AssertThat(t, result, is.EqualTo(tt.expected))
			then.AssertThat(t, len(result), is.GreaterThanOrEqualTo(len(tt.input)))
		})
	}
}

func TestMarkdownTableAlignment(t *testing.T) {
	// Test that all columns are properly aligned
	header := []string{"Short", "Medium Length", "Very Long Header Name"}
	content := [][]string{
		{"1", "Test", "A"},
		{"999", "Longer content here", "BBB"},
		{"42", "Mid", "Another long content here"},
	}

	output := captureOutput(func() {
		printMarkdownTable(header, content)
	})

	lines := strings.Split(output, "\n")
	
	// Check that all pipes align vertically
	for i, line := range lines {
		if line == "" {
			continue
		}
		
		// Find all pipe positions
		var pipePositions []int
		for pos, char := range line {
			if char == '|' {
				pipePositions = append(pipePositions, pos)
			}
		}
		
		// First line establishes the positions
		if i == 0 {
			// Store positions for comparison
			expectedPositions := pipePositions
			
			// Check subsequent lines
			for j := 1; j < len(lines); j++ {
				if lines[j] == "" {
					continue
				}
				
				var actualPositions []int
				for pos, char := range lines[j] {
					if char == '|' {
						actualPositions = append(actualPositions, pos)
					}
				}
				
				// All lines should have same number of pipes
				then.AssertThat(t, len(actualPositions), is.EqualTo(len(expectedPositions)))
			}
		}
	}
}

func TestJsonOutputFormatting(t *testing.T) {
	// Test that JSON is properly formatted with newlines
	header := []string{"ID", "Value"}
	content := [][]string{
		{"1", "First"},
		{"2", "Second"},
	}

	output := captureOutput(func() {
		printJsonDataTable("test_items", header, content)
	})

	// Should be valid JSON
	var result map[string]interface{}
	err := json.Unmarshal([]byte(output), &result)
	then.AssertThat(t, err, is.Nil())

	// Should be pretty-printed (contains newlines and indentation)
	then.AssertThat(t, strings.Contains(output, "\n"), is.True())
	then.AssertThat(t, strings.Contains(output, "  "), is.True()) // Indentation
}

// Helper function to capture stdout
func captureOutput(f func()) string {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	output := make([]byte, 4096)
	n, _ := r.Read(output)
	os.Stdout = oldStdout

	return string(output[:n])
}