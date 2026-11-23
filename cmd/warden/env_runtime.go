package main

import (
	"bufio"
	"errors"
	"os"
	"strings"
)

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		name, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, exists := os.LookupEnv(name); exists {
			_ = os.Setenv(name, cleanEnvValue(os.Getenv(name)))
			continue
		}
		value = cleanEnvValue(value)
		if err := os.Setenv(name, value); err != nil {
			return err
		}
	}
	return scanner.Err()
}

func cleanEnvValue(value string) string {
	value = strings.TrimSpace(value)
	return strings.Trim(value, "\"'")
}
