package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ParseJSON unmarshals JSON bytes into a target struct with optional strict mode
func ParseJSON[T any](jsonBytes []byte, target *T, strict bool) error {
	if target == nil {
		return fmt.Errorf("target must be a non-nil pointer")
	}

	dec := json.NewDecoder(bytes.NewReader(jsonBytes))

	if strict {
		dec.DisallowUnknownFields()
	}

	err := dec.Decode(target)
	if err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}
	return nil
}

// TODO: ParseJSONFile shall use ParseJSON above, and the testing shall be split accordingly
// ParseJSONFile reads a JSON file and decodes it into the given target.
// target must be a non-nil pointer to a struct/map/slice that matches the JSON structure.
// If strict is true, unknown fields in the JSON file will result in an error.
// Returns an error if file reading or JSON parsing fails.
func ParseJSONFile[T any](jsonFilePath string, target *T, strict bool) error {
	if target == nil {
		return fmt.Errorf("target must be a non-nil pointer")
	}

	if !FileExists(jsonFilePath) {
		return fmt.Errorf("file %q does not exist", filepath.Base(jsonFilePath))
	}

	file, err := os.Open(jsonFilePath)
	if err != nil {
		return fmt.Errorf("failed to open JSON file %q: %w", jsonFilePath, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)

	if strict {
		decoder.DisallowUnknownFields()
	}

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("failed to decode JSON from %q: %w", jsonFilePath, err)
	}

	return nil
}

// WriteStructToJSONFile writes the given data structure to a JSON file at the specified
// file path. The JSON output is formatted with indentation for readability.
func WriteStructToJSONFile(data any, filePath string) error {
	// Create or truncate the file
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("error creating file: %v", err)
	}
	defer file.Close()

	// Create JSON encoder that writes directly to the file
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "    ") // For pretty-printed JSON

	// Encode the data to JSON and write to file
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("error encoding JSON: %v", err)
	}

	return nil
}
