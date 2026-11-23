package main

import (
	"fmt"
	"strings"
	"time"
)

func mafiaPlayerSuspicionAliases(player MafiaPlayer) []string {
	var aliases []string
	add := func(value string) {
		value = mafiaNormalizeSuspicionText(value)
		value = strings.Trim(value, "@.,:;!?()[]{}\"'")
		if len([]rune(value)) >= 3 {
			aliases = append(aliases, value)
		}
	}
	add(player.DisplayName)
	add(player.Username)
	if player.Username != "" {
		add("@" + player.Username)
	}
	for _, part := range strings.Fields(player.DisplayName) {
		add(part)
	}
	return aliases
}

func mafiaAIBotMemoryExists(records []MafiaAIBotMemory, memory MafiaAIBotMemory) bool {
	if memory.MessageID != 0 {
		for _, record := range records {
			if record.MessageID == memory.MessageID && record.SpeakerID == memory.SpeakerID {
				return true
			}
		}
	}
	return false
}

func pruneMafiaAIBotMemory(records []MafiaAIBotMemory, currentDay int) []MafiaAIBotMemory {
	out := records[:0]
	for _, record := range records {
		if record.DayNumber >= currentDay-1 {
			out = append(out, record)
		}
	}
	if len(out) > mafiaAIBotMemoryLimit {
		out = out[len(out)-mafiaAIBotMemoryLimit:]
	}
	return out
}

func appendMafiaAIBotMessage(room *MafiaRoom, player MafiaPlayer, text string) {
	if room == nil {
		return
	}
	text = trimRunes(strings.TrimSpace(text), 260)
	if text == "" {
		return
	}
	room.AIBotMessages = append(room.AIBotMessages, MafiaAIBotMessage{
		DayNumber:  room.DayNumber,
		Phase:      room.Status,
		PlayerID:   player.TelegramID,
		PlayerName: player.DisplayName,
		Text:       text,
		CreatedAt:  time.Now().UTC().Unix(),
	})
	if len(room.AIBotMessages) > 50 {
		room.AIBotMessages = room.AIBotMessages[len(room.AIBotMessages)-50:]
	}
}

func mafiaAIBotMessagesText(room MafiaRoom, phase MafiaRoomStatus) string {
	lines := make([]string, 0, len(room.AIBotMessages))
	for _, msg := range room.AIBotMessages {
		if msg.DayNumber != room.DayNumber || msg.Phase != phase || strings.TrimSpace(msg.Text) == "" {
			continue
		}
		name := strings.TrimSpace(msg.PlayerName)
		if name == "" {
			name = mafiaPlayerPublicName(room, msg.PlayerID)
		}
		lines = append(lines, fmt.Sprintf("🤖 %s: %s", name, msg.Text))
	}
	return strings.Join(lines, "\n")
}

func mafiaAIBotMessageLine(msg MafiaAIBotMessage) string {
	name := strings.TrimSpace(msg.PlayerName)
	if name == "" {
		name = "AI-бот"
	}
	return fmt.Sprintf("🤖 %s: %s", name, strings.TrimSpace(msg.Text))
}
