package main

import (
	"fmt"
	"strings"
	"time"
)

func (a *App) formatTreatmentDuration(d time.Duration) string {
	minutes := int((d + time.Minute/2) / time.Minute)
	if minutes < 1 {
		minutes = 1
	}
	hours := minutes / 60
	minutes = minutes % 60
	if hours == 0 {
		return fmt.Sprintf("%d %s", minutes, a.durationUnit("minute", minutes))
	}
	if minutes == 0 {
		return fmt.Sprintf("%d %s", hours, a.durationUnit("hour", hours))
	}
	return fmt.Sprintf("%d %s %d %s", hours, a.durationUnit("hour", hours), minutes, a.durationUnit("minute", minutes))
}

func (a *App) durationUnit(kind string, n int) string {
	locale := strings.ToLower(strings.TrimSpace(a.config().I18n.Locale))
	if strings.HasPrefix(locale, "ru") {
		switch kind {
		case "hour":
			return russianPlural(n, a.t("duration_hour_one"), a.t("duration_hour_few"), a.t("duration_hour_many"))
		default:
			return russianPlural(n, a.t("duration_minute_one"), a.t("duration_minute_few"), a.t("duration_minute_many"))
		}
	}
	if n == 1 {
		return a.t("duration_" + kind + "_one")
	}
	return a.t("duration_" + kind + "_many")
}

func isActive(untilUnix int64, now time.Time) bool {
	return untilUnix == 0 || untilUnix > now.Unix()
}

func formatUntil(untilUnix int64) string {
	if untilUnix == 0 {
		return "forever"
	}
	return formatUnix(untilUnix)
}

func formatUnix(ts int64) string {
	return time.Unix(ts, 0).Format(time.RFC3339)
}

func formatMaybeUnix(ts int64) string {
	if ts == 0 {
		return "never"
	}
	return formatUnix(ts)
}

func formatRelativeDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	d = d.Round(time.Minute)
	if d >= 24*time.Hour {
		days := int(d / (24 * time.Hour))
		return fmt.Sprintf("через %d %s", days, russianPlural(days, "день", "дня", "дней"))
	}
	if d >= time.Hour {
		hours := int(d / time.Hour)
		return fmt.Sprintf("через %d %s", hours, russianPlural(hours, "час", "часа", "часов"))
	}
	minutes := int(d / time.Minute)
	if minutes <= 0 {
		return "меньше чем через минуту"
	}
	return fmt.Sprintf("через %d %s", minutes, russianPlural(minutes, "минуту", "минуты", "минут"))
}

func releaseLine(untilUnix int64, now time.Time) string {
	if untilUnix == 0 {
		return "Освобождение: вручную"
	}
	until := time.Unix(untilUnix, 0).UTC()
	return fmt.Sprintf("Освобождение: %s (%s)", formatRelativeDuration(until.Sub(now)), formatUnix(untilUnix))
}

func russianPlural(n int, one, few, many string) string {
	n = n % 100
	if n >= 11 && n <= 14 {
		return many
	}
	switch n % 10 {
	case 1:
		return one
	case 2, 3, 4:
		return few
	default:
		return many
	}
}
