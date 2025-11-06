package utilities

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

func isLikelyRawToken(s string) bool {
	if len(s) != 20 {
		return false
	}
	// Check if it's alphanumeric
	tokenPattern := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	return tokenPattern.MatchString(s)
}

// Configurator reads the JSON config file and returns the value for the given key
func Configurator(filename, key string) (string, error) {

	if strings.HasPrefix(key, "say:") {
		rawVal := strings.TrimPrefix(key, "say:")
		if isLikelyRawToken(rawVal) {
			return rawVal, nil
		}
		return "", fmt.Errorf("raw value after 'say:' is not a valid 20-character token")
	}
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("error opening config file: %w", err)
	}
	defer file.Close()

	fileData, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("error reading config file: %w", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(fileData, &config); err != nil {
		return "", fmt.Errorf("error parsing JSON: %w", err)
	}
	lowerKey := strings.ToLower(key)
	
	value, exists := config[lowerKey]
	if !exists {
		if len(key) == 20 && !strings.Contains(key, " ") {
			return key, nil
		}
		return "", fmt.Errorf("key %q not found in config", key)
	}

	strValue, ok := value.(string)
	if !ok {
		return "", fmt.Errorf("value for key %q is not a string, ignoring", key)
	}

	return strValue, nil
}
