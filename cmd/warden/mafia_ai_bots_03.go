package main

import (
	"strconv"
	"strings"
)

func mafiaAIBotValidTargetID(targetID int64, candidates []MafiaPlayer) int64 {
	if targetID == 0 {
		return 0
	}
	for _, candidate := range candidates {
		if candidate.TelegramID == targetID {
			return targetID
		}
	}
	return 0
}

func mafiaAIBotVoteTargetID(vote string, candidates []MafiaPlayer) int64 {
	vote = strings.TrimSpace(vote)
	if vote == "" {
		return 0
	}
	targetID, err := strconv.ParseInt(vote, 10, 64)
	if err != nil {
		return 0
	}
	return mafiaAIBotValidTargetID(targetID, candidates)
}

func mafiaAIBotPlayerPayload(player MafiaPlayer, includeRole bool) map[string]interface{} {
	out := map[string]interface{}{
		"id":        player.TelegramID,
		"name":      player.DisplayName,
		"backstory": player.Backstory,
		"is_bot":    player.IsTestBot,
	}
	if includeRole {
		out["role"] = player.Role
	}
	return out
}

func mafiaAIBotPlayersPayload(players []MafiaPlayer, selfID int64) []map[string]interface{} {
	out := make([]map[string]interface{}, 0, len(players))
	for _, player := range players {
		out = append(out, map[string]interface{}{
			"id":      player.TelegramID,
			"name":    player.DisplayName,
			"is_self": player.TelegramID == selfID,
		})
	}
	return out
}

func mafiaAIBotMafiaAlliesPayload(room MafiaRoom, player MafiaPlayer) []map[string]interface{} {
	if !isMafiaTeam(player.Role) {
		return nil
	}
	allies := make([]MafiaPlayer, 0, len(room.Players))
	for _, candidate := range room.Players {
		if candidate.TelegramID == player.TelegramID || !candidate.Alive || !isMafiaTeam(candidate.Role) {
			continue
		}
		allies = append(allies, candidate)
	}
	return mafiaAIBotPlayersPayload(allies, player.TelegramID)
}

func mafiaAliveAIBotPlayers(room MafiaRoom) []MafiaPlayer {
	out := make([]MafiaPlayer, 0, len(room.Players))
	for _, player := range room.Players {
		if player.IsTestBot && player.Alive {
			out = append(out, player)
		}
	}
	return out
}

func mafiaAIBotMemoryPayload(room MafiaRoom, limit int) []map[string]interface{} {
	if limit <= 0 {
		limit = 12
	}
	items := make([]MafiaAIBotMemory, 0, len(room.AIBotMemory))
	for _, memory := range room.AIBotMemory {
		if memory.DayNumber == room.DayNumber || memory.DayNumber == room.DayNumber-1 {
			items = append(items, memory)
		}
	}
	if len(items) > limit {
		items = items[len(items)-limit:]
	}
	out := make([]map[string]interface{}, 0, len(items))
	for _, memory := range items {
		out = append(out, map[string]interface{}{
			"day":          memory.DayNumber,
			"phase":        memory.Phase,
			"speaker_id":   memory.SpeakerID,
			"speaker_name": memory.SpeakerName,
			"text":         memory.Text,
			"target_ids":   memory.TargetIDs,
			"target_names": memory.TargetNames,
		})
	}
	return out
}

type mafiaAIBotKillPressure struct {
	Player   MafiaPlayer
	Score    int
	Reasons  []string
	Messages []string
}

func mafiaAIBotKillRecommendationsPayload(room MafiaRoom, candidates []MafiaPlayer, bot MafiaPlayer) []map[string]interface{} {
	pressures := mafiaAIBotKillPressures(room, candidates, bot)
	if len(pressures) > 6 {
		pressures = pressures[:6]
	}
	out := make([]map[string]interface{}, 0, len(pressures))
	for _, pressure := range pressures {
		out = append(out, map[string]interface{}{
			"id":                        pressure.Player.TelegramID,
			"name":                      pressure.Player.DisplayName,
			"pressure_score":            pressure.Score,
			"reasons":                   pressure.Reasons,
			"recent_suspicion_messages": pressure.Messages,
		})
	}
	return out
}

func mafiaAIBotPreferredKillTarget(room MafiaRoom, candidates []MafiaPlayer, bot MafiaPlayer) (int64, string) {
	pressures := mafiaAIBotKillPressures(room, candidates, bot)
	if len(pressures) == 0 || pressures[0].Score <= 0 {
		return 0, ""
	}
	reason := strings.Join(pressures[0].Reasons, "; ")
	if reason == "" {
		reason = "слишком активно собирает версии против мафии"
	}
	return pressures[0].Player.TelegramID, reason
}
