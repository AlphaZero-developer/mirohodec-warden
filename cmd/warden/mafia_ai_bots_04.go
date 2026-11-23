package main

import (
	"fmt"
	"sort"
	"strings"
)

func mafiaAIBotKillPressures(room MafiaRoom, candidates []MafiaPlayer, bot MafiaPlayer) []mafiaAIBotKillPressure {
	if len(candidates) == 0 {
		return nil
	}
	mafiaIDs := map[int64]bool{}
	for _, player := range room.Players {
		if player.Alive && isMafiaTeam(player.Role) {
			mafiaIDs[player.TelegramID] = true
		}
	}
	byID := make(map[int64]*mafiaAIBotKillPressure, len(candidates))
	for _, candidate := range candidates {
		byID[candidate.TelegramID] = &mafiaAIBotKillPressure{Player: candidate}
	}
	for _, memory := range room.AIBotMemory {
		pressure := byID[memory.SpeakerID]
		if pressure == nil || memory.SpeakerID == bot.TelegramID {
			continue
		}
		if memory.DayNumber < room.DayNumber-1 {
			continue
		}
		score := 0
		var reasons []string
		if mafiaHasStrongSuspicionWord(memory.Text) {
			score += 2
			reasons = append(reasons, "сильно подозревал мафию")
		} else if mafiaHasSuspicionWord(memory.Text) {
			score++
			reasons = append(reasons, "обсуждал подозрения")
		}
		if mafiaMemoryTargetsAny(memory.TargetIDs, mafiaIDs) {
			score += 4
			reasons = append(reasons, "давил на кого-то из мафии")
		}
		if memory.DayNumber == room.DayNumber {
			score++
		}
		if score == 0 {
			continue
		}
		pressure.Score += score
		pressure.Reasons = appendUniqueStrings(pressure.Reasons, reasons...)
		if len(pressure.Messages) < 3 {
			pressure.Messages = append(pressure.Messages, memory.Text)
		}
	}
	out := make([]mafiaAIBotKillPressure, 0, len(byID))
	for _, pressure := range byID {
		if pressure.Score > 0 {
			out = append(out, *pressure)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Player.TelegramID < out[j].Player.TelegramID
	})
	return out
}

func mafiaMemoryTargetsAny(targetIDs []int64, needles map[int64]bool) bool {
	for _, targetID := range targetIDs {
		if needles[targetID] {
			return true
		}
	}
	return false
}

func appendUniqueStrings(values []string, add ...string) []string {
	seen := make(map[string]bool, len(values)+len(add))
	for _, value := range values {
		seen[value] = true
	}
	for _, value := range add {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		values = append(values, value)
	}
	return values
}

func mafiaAIBotKillRecommendationText(room MafiaRoom, targetID int64, reason string) string {
	name := mafiaPlayerPublicName(room, targetID)
	if strings.TrimSpace(name) == "" {
		name = fmt.Sprintf("%d", targetID)
	}
	reason = trimRunes(strings.TrimSpace(reason), 220)
	if reason == "" {
		reason = "слишком активно собирает версии против мафии"
	}
	return fmt.Sprintf("Я бы убрал %s этой ночью: %s.", name, reason)
}

func mafiaAIBotKillHintKey(room MafiaRoom) string {
	if normalizeMafiaRoomID(room.ID) == "" || room.Status != MafiaStatusNight {
		return ""
	}
	phaseKey := room.PhaseStartedAt
	if phaseKey == 0 {
		phaseKey = int64(room.DayNumber)
	}
	return fmt.Sprintf("%s:%s:%d:%d", normalizeMafiaRoomID(room.ID), room.Status, room.DayNumber, phaseKey)
}

func mafiaLooksLikeSuspicionText(text string) bool {
	normalized := mafiaNormalizeSuspicionText(text)
	if len([]rune(normalized)) < 4 || commandName(text) != "" {
		return false
	}
	return mafiaHasSuspicionWord(normalized)
}

func mafiaHasSuspicionWord(text string) bool {
	text = mafiaNormalizeSuspicionText(text)
	for _, needle := range []string{
		"маф", "подозр", "странн", "молчит", "тихий", "тихая", "врет", "врёт", "не верю",
		"голос", "казн", "провер", "алиби", "черн", "красн", "мирн", "палится", "давит",
	} {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func mafiaHasStrongSuspicionWord(text string) bool {
	text = mafiaNormalizeSuspicionText(text)
	for _, needle := range []string{"маф", "подозр", "казн", "голос", "провер", "черн", "палится"} {
		if strings.Contains(text, needle) {
			return true
		}
	}
	return false
}

func mafiaNormalizeSuspicionText(text string) string {
	text = strings.ToLower(strings.TrimSpace(text))
	text = strings.ReplaceAll(text, "ё", "е")
	return strings.Join(strings.Fields(text), " ")
}

func mafiaSuspicionTargets(room MafiaRoom, text string, speakerID int64) ([]int64, []string) {
	normalized := mafiaNormalizeSuspicionText(text)
	seen := map[int64]bool{}
	var ids []int64
	var names []string
	for _, player := range room.Players {
		if !player.Alive || player.TelegramID == speakerID {
			continue
		}
		if !seen[player.TelegramID] && mafiaTextMentionsPlayer(normalized, player) {
			seen[player.TelegramID] = true
			ids = append(ids, player.TelegramID)
			names = append(names, player.DisplayName)
		}
	}
	if len(ids) > 4 {
		ids = ids[:4]
		names = names[:4]
	}
	return ids, names
}

func mafiaTextMentionsPlayer(normalizedText string, player MafiaPlayer) bool {
	for _, alias := range mafiaPlayerSuspicionAliases(player) {
		if alias != "" && strings.Contains(normalizedText, alias) {
			return true
		}
	}
	return false
}
