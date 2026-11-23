package main

import (
	"fmt"
	"strings"
	"time"
)

const (
	rehabWardQuiet = "quiet"
	rehabWardRowdy = "rowdy"
	rehabWardVIP   = "vip"

	quietWardMinMessagesBetweenReplies = 2
	quietWardMaxMessagesBetweenReplies = 3
	rowdyToQuietGoodMessages           = 4
	vipReviewMinMessages               = 3
	vipReviewChancePercent             = 3
)

type rehabWardInfo struct {
	ID                string
	Title             string
	Notice            string
	EscapeChanceDelta int
	BribeChanceDelta  int
	BribeCostDelta    int64
}

type rehabWardProgress struct {
	JailSince         int64
	Ward              string
	Messages          int
	GoodMessages      int
	QuietSinceReply   int
	QuietNextReplyAt  int
	VIPReviewAttempts int
}

func rehabWard(rec ModerationRecord) rehabWardInfo {
	switch normalizeRehabWard(rec.RehabWard) {
	case rehabWardRowdy:
		return rehabWardInfo{
			ID:                rehabWardRowdy,
			Title:             "буйная палата",
			Notice:            "режим шумный: побег чуть сложнее, санитар берёт дороже",
			EscapeChanceDelta: -1,
			BribeChanceDelta:  -3,
			BribeCostDelta:    1,
		}
	case rehabWardVIP:
		return rehabWardInfo{
			ID:                rehabWardVIP,
			Title:             "VIP-палата",
			Notice:            "лечение с видом на здравый смысл: подкуп дешевле и чуть вероятнее",
			EscapeChanceDelta: 2,
			BribeChanceDelta:  5,
			BribeCostDelta:    -1,
		}
	default:
		return rehabWardInfo{
			ID:     rehabWardQuiet,
			Title:  "тихая палата",
			Notice: "обычный режим: меньше шума, больше осмысления",
		}
	}
}

func normalizeRehabWard(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.TrimPrefix(value, "палата=")
	value = strings.TrimPrefix(value, "ward=")
	value = strings.TrimPrefix(value, "mode=")
	value = strings.Trim(value, " .,;:!?")
	switch value {
	case "quiet", "silent", "тихая", "тихо", "обычная", "обычный":
		return rehabWardQuiet
	case "rowdy", "hard", "буйная", "буйно", "жесткая", "жёсткая", "строгая":
		return rehabWardRowdy
	case "vip", "вип", "vip-палата", "вип-палата":
		return rehabWardVIP
	default:
		return ""
	}
}

func extractRehabWardOverride(args []string) (string, []string) {
	if len(args) == 0 {
		return "", args
	}
	ward := normalizeRehabWard(args[0])
	if ward == "" {
		return "", args
	}
	out := append([]string(nil), args[1:]...)
	return ward, out
}

func parseJailOptions(args []string, permanentByDefault bool) (int64, string, string, error) {
	ward, rest := extractRehabWardOverride(args)
	if d, permanent, tail, ok, err := parseDurationArgPrefix(rest); ok || err != nil {
		if err != nil {
			return 0, "", "", err
		}
		if parsedWard, parsedTail := extractRehabWardOverride(tail); parsedWard != "" {
			ward = parsedWard
			tail = parsedTail
		}
		if permanent || d == 0 {
			return 0, ward, strings.Join(tail, " "), nil
		}
		return time.Now().UTC().Add(d).Unix(), ward, strings.Join(tail, " "), nil
	}
	if !permanentByDefault {
		return 0, "", "", fmt.Errorf("duration is required")
	}
	return 0, ward, strings.Join(rest, " "), nil
}

func (a *App) chooseRehabWard(chatID int64, target targetUser, reason, override string, now time.Time) string {
	if ward := normalizeRehabWard(override); ward != "" {
		return ward
	}
	if a == nil || a.store == nil {
		return rehabWardQuiet
	}
	reasonLower := strings.ToLower(reason)
	if strings.Contains(reasonLower, "спам") ||
		strings.Contains(reasonLower, "spam") ||
		strings.Contains(reasonLower, "оск") ||
		strings.Contains(reasonLower, "мат") ||
		strings.Contains(reasonLower, "буй") ||
		strings.Contains(reasonLower, "флуд") ||
		a.store.ActiveWarningCount(chatID, target.ID, now) >= 2 ||
		a.store.Karma(chatID, target.ID) <= -3 {
		return rehabWardRowdy
	}
	member, ok := a.store.Activity(chatID, target.ID)
	if a.store.Karma(chatID, target.ID) >= 10 ||
		(ok && (member.Messages >= 500 || messagesTodayAt(member, now) >= 50)) {
		return rehabWardVIP
	}
	return rehabWardQuiet
}

func rehabWardLine(rec ModerationRecord) string {
	info := rehabWard(rec)
	return fmt.Sprintf("Палата: %s - %s.", info.Title, info.Notice)
}

func adjustRehabEscapeChanceForWard(chance int, rec ModerationRecord) int {
	chance += rehabWard(rec).EscapeChanceDelta
	if chance < 0 {
		return 0
	}
	if chance > 100 {
		return 100
	}
	return chance
}

func rehabWardProgressKey(chatID, userID int64) string {
	return fmt.Sprintf("%d:%d", chatID, userID)
}
