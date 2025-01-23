// internal/utils/json_utils.go

package utils

import (
	"encoding/json"
	"fmt"
)

// MarshalJSON marshals the given data into a pretty-printed JSON string.
// It wraps any errors encountered during the marshaling process.
func MarshalJSON(data interface{}) (string, error) {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("JSON marshaling failed: %w", err)
	}
	return string(jsonBytes), nil
}
