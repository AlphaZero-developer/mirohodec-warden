package main

import (
	"fmt"
	"strings"
	"time"
)

func memberRewardsFromRecord(rec MemberRecord) MemberRewards {
	return MemberRewards{
		Achievements:        append([]string(nil), rec.Achievements...),
		ShopTitle:           rec.ShopTitle,
		ShopTitleUntilUnix:  rec.ShopTitleUntilUnix,
		LayaTone:            rec.LayaTone,
		LayaCharmUntilUnix:  rec.LayaCharmUntilUnix,
		QuoteBoostUntilUnix: rec.QuoteBoostUntilUnix,
	}
}

func applyMemberRewards(rec MemberRecord, rewards MemberRewards) MemberRecord {
	rec.Achievements = normalizeAchievements(rewards.Achievements)
	rec.ShopTitle = strings.TrimSpace(rewards.ShopTitle)
	rec.ShopTitleUntilUnix = rewards.ShopTitleUntilUnix
	rec.LayaTone = normalizeLayaTone(rewards.LayaTone)
	rec.LayaCharmUntilUnix = rewards.LayaCharmUntilUnix
	rec.QuoteBoostUntilUnix = rewards.QuoteBoostUntilUnix
	return rec
}

func normalizeLayaTone(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "soft", "мягче", "мягко", "warm":
		return "soft"
	case "hard", "жёстче", "жестче", "строго", "sharp":
		return "hard"
	case "neutral", "нейтрально", "обычно", "normal", "":
		return "neutral"
	default:
		return "neutral"
	}
}

func layaToneTitle(value string) string {
	switch normalizeLayaTone(value) {
	case "soft":
		return "мягче"
	case "hard":
		return "жёстче"
	default:
		return "нейтрально"
	}
}

func normalizeAchievements(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = normalizeAchievement(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func normalizeAchievement(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "_")
	value = strings.Trim(value, ".,;:!?")
	return value
}

func memberHasAchievement(rec MemberRecord, achievement string) bool {
	achievement = normalizeAchievement(achievement)
	if achievement == "" {
		return false
	}
	for _, old := range rec.Achievements {
		if normalizeAchievement(old) == achievement {
			return true
		}
	}
	return false
}

func (s *Store) UnlockAchievement(chatID, userID int64, achievement string, now time.Time) (bool, error) {
	achievement = normalizeAchievement(achievement)
	if achievement == "" || userID == 0 {
		return false, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ch := s.ensureChatLocked(chatID)
	key := idKey(userID)
	member := ch.Members[key]
	if member.ID == 0 {
		member.ID = userID
		member.DisplayName = fmt.Sprintf("user %d", userID)
		member.FirstSeen = now.Unix()
	}
	if memberHasAchievement(member, achievement) {
		return false, nil
	}
	member.Achievements = append(normalizeAchievements(member.Achievements), achievement)
	ch.Members[key] = member
	if err := s.saveLocked(); err != nil {
		return false, err
	}
	return true, nil
}

func (s *Store) SetMemberRewards(chatID, userID int64, rewards MemberRewards) error {
	if userID == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	ch := s.ensureChatLocked(chatID)
	key := idKey(userID)
	member := ch.Members[key]
	if member.ID == 0 {
		member.ID = userID
		member.DisplayName = fmt.Sprintf("user %d", userID)
		now := time.Now().UTC().Unix()
		member.FirstSeen = now
		member.LastSeen = now
	}
	member = applyMemberRewards(member, rewards)
	ch.Members[key] = member
	return s.saveLocked()
}

func achievementTitle(achievement string) string {
	switch normalizeAchievement(achievement) {
	case "activity_100":
		return "Сто сообщений"
	case "activity_500":
		return "Житель чата"
	case "activity_1000":
		return "Голос в стенах"
	case "quote_first":
		return "Охотник за цитатами"
	case "helper":
		return "Помогатор"
	case "top_activity":
		return "Верхушка активности"
	case "top_karma":
		return "Репутационный капитал"
	case "shop_first":
		return "Первый чек"
	case "laya_tuned":
		return "Настроил Лайю"
	case "weekly_karma":
		return "Выходной буст"
	case "weekly_quote_hunter":
		return "Недельная охота"
	default:
		return strings.ReplaceAll(strings.TrimSpace(achievement), "_", " ")
	}
}

func achievementList(rec MemberRecord) []string {
	values := normalizeAchievements(rec.Achievements)
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, achievementTitle(value))
	}
	return out
}
