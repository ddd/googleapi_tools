package main

import (
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

func generateRandomString() string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	const length = 8

	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}

	return string(result)
}

func convertToUnknownType(typeName string) string {
	var result strings.Builder
	result.WriteString("UNKNOWN_")

	for i, r := range typeName {
		if i > 0 && unicode.IsUpper(r) && !unicode.IsUpper(rune(typeName[i-1])) {
			result.WriteRune('_')
		}
		result.WriteRune(unicode.ToUpper(r))
	}

	return result.String()
}

func writeFile(fileContent []byte, fileName string) error {
	// Create all necessary directories
	dir := filepath.Dir(fileName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write the file
	return os.WriteFile(fileName, fileContent, 0644)
}
