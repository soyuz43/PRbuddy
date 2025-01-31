package utils

import (
	"encoding/json"
	"fmt"
)

// MarshalJSON converts the given data to a pretty-printed JSON string.
func MarshalJSON(data interface{}) (string, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("JSON marshaling failed: %w", err)
	}
	return string(jsonBytes), nil
}
