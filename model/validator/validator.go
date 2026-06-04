package validator

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var (
	ValidCustomWords map[string]bool
)

func init() {
	ValidCustomWords = make(map[string]bool)
	loadWords("controller/translator/words.txt")
	loadWords("controller/translator/allowed_words.txt")
	loadWords("controller/translator/scramy_words.txt")
	loadWords("controller/translator/scramy_allowed_words.txt")
}

func loadWords(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("Warning: Could not open word file %s: %v\n", filePath, err)
		// Try falling back to parent directory if running from subdirectory (e.g. tests)
		file, err = os.Open("../" + filePath)
		if err != nil {
			fmt.Printf("Warning: Could not open word file ../%s: %v\n", filePath, err)
			return
		}
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		word := strings.TrimSpace(scanner.Text())
		if word != "" {
			ValidCustomWords[strings.ToUpper(word)] = true
		}
	}
}

func IsValidWord(word string) bool {
	return ValidCustomWords[strings.ToUpper(word)]
}
