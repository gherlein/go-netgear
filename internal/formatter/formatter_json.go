package formatter

import (
	"encoding/json"
	"fmt"
)

func PrintJsonDataTable(item string, header []string, content [][]string) {
	// Create slice of maps for proper JSON structure
	var items []map[string]string
	
	for _, row := range content {
		rowData := make(map[string]string)
		// Handle cases where row length doesn't match header length
		for i, headerName := range header {
			if i < len(row) {
				rowData[headerName] = row[i]
			} else {
				rowData[headerName] = ""
			}
		}
		items = append(items, rowData)
	}
	
	// Create the final structure
	result := map[string][]map[string]string{
		item: items,
	}
	
	// Use proper JSON marshaling with indentation to handle escaping
	jsonData, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}
	
	fmt.Println(string(jsonData))
}
