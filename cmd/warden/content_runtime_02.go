package main

import (
	"strconv"
	"strings"
)

func normalizeContentChanceText(value string) string {
	return strings.ReplaceAll(strings.TrimSpace(value), ",", ".")
}

func parseContentChance(value string, fallback float64) float64 {
	original := normalizeContentChanceText(value)
	if original == "" {
		return fallback
	}
	value = strings.TrimSpace(strings.TrimSuffix(original, "%"))
	f, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	if strings.HasSuffix(original, "%") || f > 1 {
		f /= 100
	}
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

func contentRuntimeList(value string) []string {
	value = strings.TrimSpace(value)
	switch strings.ToLower(value) {
	case "", "-", "none", "off", "нет":
		return nil
	}
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ';' || r == ',' || r == '\n' || r == '\r'
	})
	out := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, field := range fields {
		field = strings.TrimSpace(field)
		if field == "" {
			continue
		}
		key := strings.ToLower(field)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, field)
	}
	return out
}
