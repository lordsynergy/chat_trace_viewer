package config

import (
	"bufio"
	"os"
	"strings"
)

func loadEnvFiles(paths ...string) {
	original := map[string]struct{}{}
	for _, pair := range os.Environ() {
		if idx := strings.IndexByte(pair, '='); idx > 0 {
			original[pair[:idx]] = struct{}{}
		}
	}

	for _, path := range paths {
		loadEnvFile(path, original)
	}
}

func loadEnvFile(path string, original map[string]struct{}) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if key == "" {
			continue
		}
		if _, exists := original[key]; exists {
			continue
		}
		_ = os.Setenv(key, value)
	}
}
