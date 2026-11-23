package main

import (
	"errors"
	"strconv"
	"strings"
	"time"
)

func parseOptionalUntil(args []string, permanentByDefault bool) (int64, string, error) {
	if len(args) == 0 {
		if permanentByDefault {
			return 0, "", nil
		}
		return 0, "", errors.New("duration is required")
	}

	if d, permanent, rest, ok, err := parseDurationArgPrefix(args); ok || err != nil {
		if err != nil {
			return 0, "", err
		}
		if permanent || d == 0 {
			return 0, strings.Join(rest, " "), nil
		}
		return time.Now().UTC().Add(d).Unix(), strings.Join(rest, " "), nil
	}

	if permanentByDefault {
		return 0, strings.Join(args, " "), nil
	}
	return 0, "", errors.New("duration is required")
}

func parseRequiredUntil(args []string) (int64, string, error) {
	if len(args) == 0 {
		return 0, "", errors.New("duration is required")
	}
	d, permanent, rest, ok, err := parseDurationArgPrefix(args)
	if err != nil {
		return 0, "", err
	}
	if !ok {
		return 0, "", errors.New("duration is required")
	}
	if permanent || d == 0 {
		return 0, strings.Join(rest, " "), nil
	}
	if d < 30*time.Second {
		return 0, "", errors.New("Telegram treats restrictions under 30 seconds as forever; use 30s or more")
	}
	return time.Now().UTC().Add(d).Unix(), strings.Join(rest, " "), nil
}

func parseDurationArgPrefix(args []string) (time.Duration, bool, []string, bool, error) {
	if len(args) == 0 {
		return 0, false, args, false, nil
	}
	if d, permanent, ok, err := parseDurationToken(args[0]); ok || err != nil {
		return d, permanent, args[1:], ok, err
	}
	if len(args) >= 2 {
		if d, permanent, ok, err := parseDurationToken(args[0] + args[1]); ok || err != nil {
			return d, permanent, args[2:], ok, err
		}
	}
	return 0, false, args, false, nil
}

func durationArgPrefixLen(args []string) int {
	if len(args) == 0 {
		return 0
	}
	if _, _, ok, err := parseDurationToken(args[0]); ok && err == nil {
		return 1
	}
	if len(args) >= 2 {
		if _, _, ok, err := parseDurationToken(args[0] + args[1]); ok && err == nil {
			return 2
		}
	}
	return 0
}

func parseDurationToken(value string) (time.Duration, bool, bool, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "forever", "permanent", "perm", "inf", "infinite", "navsegda", "навсегда", "перманент", "перма":
		return 0, true, true, nil
	}
	d, err := parseDuration(value)
	if err != nil {
		if looksLikeDuration(value) {
			return 0, false, true, err
		}
		return 0, false, false, nil
	}
	return d, false, true, nil
}

func parseDuration(value string) (time.Duration, error) {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" {
		return 0, errors.New("empty duration")
	}
	value = normalizeLocalizedDuration(value)

	if strings.HasSuffix(value, "d") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(value, "d"), 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(n * float64(24*time.Hour)), nil
	}
	if strings.HasSuffix(value, "w") {
		n, err := strconv.ParseFloat(strings.TrimSuffix(value, "w"), 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(n * float64(7*24*time.Hour)), nil
	}
	return time.ParseDuration(value)
}

func normalizeLocalizedDuration(value string) string {
	replacements := []struct {
		suffix string
		unit   string
	}{
		{suffix: "секунд", unit: "s"},
		{suffix: "секунды", unit: "s"},
		{suffix: "секунда", unit: "s"},
		{suffix: "сек", unit: "s"},
		{suffix: "с", unit: "s"},
		{suffix: "минут", unit: "m"},
		{suffix: "минуты", unit: "m"},
		{suffix: "минута", unit: "m"},
		{suffix: "мин", unit: "m"},
		{suffix: "м", unit: "m"},
		{suffix: "часов", unit: "h"},
		{suffix: "часа", unit: "h"},
		{suffix: "час", unit: "h"},
		{suffix: "ч", unit: "h"},
		{suffix: "дней", unit: "d"},
		{suffix: "дня", unit: "d"},
		{suffix: "день", unit: "d"},
		{suffix: "дн", unit: "d"},
		{suffix: "д", unit: "d"},
		{suffix: "недель", unit: "w"},
		{suffix: "недели", unit: "w"},
		{suffix: "неделя", unit: "w"},
		{suffix: "нед", unit: "w"},
		{suffix: "н", unit: "w"},
	}
	for _, replacement := range replacements {
		if strings.HasSuffix(value, replacement.suffix) {
			return strings.TrimSpace(strings.TrimSuffix(value, replacement.suffix)) + replacement.unit
		}
	}
	return value
}

func looksLikeDuration(value string) bool {
	if value == "" {
		return false
	}
	normalized := normalizeLocalizedDuration(strings.ToLower(strings.TrimSpace(value)))
	last := normalized[len(normalized)-1]
	return last == 's' || last == 'm' || last == 'h' || last == 'd' || last == 'w'
}

func formatDurationShort(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	d = d.Round(time.Second)
	if d >= time.Minute {
		return d.Truncate(time.Second).String()
	}
	return d.String()
}

func rehabTreatmentDuration(rec ModerationRecord, endedAt time.Time) time.Duration {
	startedUnix := rec.StartedAtUnix
	if startedUnix == 0 {
		startedUnix = rec.SinceUnix
	}
	if startedUnix == 0 {
		return 0
	}
	d := endedAt.UTC().Sub(time.Unix(startedUnix, 0).UTC())
	if d < 0 {
		return 0
	}
	return d
}
