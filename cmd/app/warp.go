package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type CFCreds struct {
	SecretKey string
	PublicKey string
	Reserved  []int
	V4        string
	V6        string
	Endpoint  string
}

// Parses the Cloudflare generator output. Tailored specifically for the output of
// github.com/badafans/warp-reg.
func parseCFCreds(output string) (CFCreds, error) {
	var result CFCreds

	patterns := map[string]*regexp.Regexp{
		"private_key": regexp.MustCompile(`(?m)^private_key:\s*(\S+)`),
		"public_key":  regexp.MustCompile(`(?m)^public_key:\s*(\S+)`),
		"reserved":    regexp.MustCompile(`(?m)^reserved:\s*\[([0-9,\s]+)\]`),
		"v4":          regexp.MustCompile(`(?m)^v4:\s*(\S+)`),
		"v6":          regexp.MustCompile(`(?m)^v6:\s*(\S+)`),
		"endpoint":    regexp.MustCompile(`(?m)^endpoint:\s*(\S+)`),
	}

	for key, pattern := range patterns {
		matches := pattern.FindStringSubmatch(output)
		if len(matches) < 2 {
			return result, errors.New("missing required field: " + key)
		}
		switch key {
		case "private_key":
			result.SecretKey = matches[1]
		case "public_key":
			result.PublicKey = matches[1]
		case "reserved":
			values := strings.Split(matches[1], ",")
			for _, v := range values {
				var num int
				fmt.Sscanf(strings.TrimSpace(v), "%d", &num)
				result.Reserved = append(result.Reserved, num)
			}
		case "v4":
			result.V4 = matches[1]
		case "v6":
			result.V6 = matches[1]
		case "endpoint":
			result.Endpoint = matches[1]
		}
	}

	return result, nil
}

func splitTwoChars(s string) (string, string, error) {
	if len([]rune(s)) != 2 { // Ensure exactly two characters
		return "", "", errors.New("input must contain exactly two characters")
	}

	runes := []rune(s) // Convert string to slice of runes to handle Unicode correctly
	return string(runes[0]), string(runes[1]), nil
}

// updateJSONValue finds a key in JSON, ensures it's unique, and replaces its value.
// It does not parse JSON into any kind of meaningful structure. It simply extracts the
// raw text from JSON and dumbly looks for a specific pattern. This pattern consists of
// a key and a pair of symbols that enclose the value (symbolPair). For example, if the
// value that is supposed to be replaced is a string, the symbolPair that needs to be
// passed to updateJSONValue should be `""` (using backticks to allow for passing of the
// quotes symbols). If the value is an array, the symbolPair should be "[]". If the
// value is an int or a bool, or any other type that does not have any characters that
// identify the beginning and the end of the value, this function is not applicable.
// The symbolPair can only contain two characters (representing the startValueSymbol and
// the endValueSymbol). If there is any other amount of characters passed as a
// symbolPair, the function will error.
func updateJSONValue(filePath, key, symbolPair, newValue string) error {
	// Load JSON file as raw text
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	jsonStr := string(data)

	// Extract the characters enclosing the value
	startValueSymbol, endValueSymbol, err := splitTwoChars(symbolPair)
	if err != nil {
		return fmt.Errorf("invalid symbol pair %s: %w", symbolPair, err)
	}

	// Construct regex pattern to match `"key": <somevalue>`, where the < and >
	// represent the symbolPair.
	// The pattern is bizarre, but basically it consists of:
	//   - a key in quotes;
	//   - any amount of whitespace and/or newlines followed by a colon;
	//   - any amount of whitespace and/or newlines after a colon;
	//   - the actual value that can consist of any amount of any characters,
	//     including whitespace and/or newlines (hence the [\S\s] and not just a dot
	//     because a dot does not include newlines). This actual value is enclosed by
	//     the startValueSymbol and the endValueSymbol.
	pattern := fmt.Sprintf(`"%s"\s*:\s*\%s([\S\s]*?)\%s`, regexp.QuoteMeta(key), startValueSymbol, endValueSymbol)
	re := regexp.MustCompile(pattern)

	// Find all matches
	matches := re.FindAllStringSubmatch(jsonStr, -1)

	// Ensure the key appears exactly once
	if len(matches) == 0 {
		return fmt.Errorf("key %s with the value symbol pair %s not found in JSON", key, symbolPair)
	} else if len(matches) > 1 {
		return fmt.Errorf("key %s appears in JSON multiple times", key)
	}

	// Perform replacement (keeping key intact, changing only the value)
	updatedJSON := re.ReplaceAllString(jsonStr, fmt.Sprintf(`"%s": %s%s%s`, key, startValueSymbol, newValue, endValueSymbol))

	// Write the modified string back to the file
	err = os.WriteFile(filePath, []byte(updatedJSON), 0644)
	if err != nil {
		return fmt.Errorf("error writing file: %w", err)
	}

	fmt.Println("Successfully updated JSON")
	return nil
}
func updateWarp(xrayConfigFilePath string, warpConfig Warp) error {
	cfCredGenerator, err := downloadFile(warpConfig.cfCredGenURL, filepath.Dir(xrayConfigFilePath), "")
	if err != nil {
		return err
	}

	cfCredsRaw, err := executeCommand(cfCredGenerator)
	if err != nil {
		return err
	}

	cfCreds, err := parseCFCreds(cfCredsRaw)
	if err != nil {
		return err
	}

	err = updateJSONValue(xrayConfigFilePath, "secretKey", `""`, cfCreds.SecretKey)
	// TODO: Update the rest of the JSON values in a similar fashion
	if err != nil {
		return err
	}

	return nil
}
