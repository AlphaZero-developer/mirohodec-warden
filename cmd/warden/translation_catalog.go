package main

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed locales/*.json
var translationAssets embed.FS

// loadTranslations keeps large translation dictionaries out of Go source.
func loadTranslations() map[string]map[string]string {
	locales := make(map[string]map[string]string, 2)
	for _, name := range []string{"en", "ru"} {
		data, err := translationAssets.ReadFile("locales/" + name + ".json")
		if err != nil {
			panic(fmt.Errorf("read locale %s: %w", name, err))
		}
		var entries map[string]string
		if err := json.Unmarshal(data, &entries); err != nil {
			panic(fmt.Errorf("decode locale %s: %w", name, err))
		}
		locales[name] = entries
	}
	return locales
}
