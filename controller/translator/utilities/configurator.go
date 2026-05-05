package utilities

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"unicode"
)

func isLikelyRawToken(s string) bool {
	if len(s) != 20 {
		return false
	}
	// Check if it's alphanumeric
	tokenPattern := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	return tokenPattern.MatchString(s)
}

func normalizeLookupKey(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(unicode.ToLower(r))
		}
	}
	return b.String()
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

	var config map[string]string
	if err := json.Unmarshal(fileData, &config); err != nil {
		return "", fmt.Errorf("error parsing JSON: %w", err)
	}

	if isLikelyRawToken(key) {
		return key, nil
	}

	lookup := make(map[string]string, len(config))
	for name, value := range config {
		lookup[normalizeLookupKey(name)] = value
	}

	commandPrefix := config["Command"]
	candidates := []string{key}
	if strings.HasPrefix(strings.ToLower(key), "say") {
		candidates = append(candidates, key[3:])
	}
	if commandPrefix != "" && strings.HasPrefix(strings.ToLower(key), strings.ToLower(commandPrefix)) {
		candidates = append(candidates, key[len(commandPrefix):])
	}

	for _, candidate := range candidates {
		if value, exists := lookup[normalizeLookupKey(candidate)]; exists {
			return value, nil
		}
	}

	return "", fmt.Errorf("key %q not found in config", key)
}
