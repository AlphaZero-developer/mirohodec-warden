package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestMafiaAIBotsChooseNightTarget(t *testing.T) {
	store := newMafiaTestStore(t)
	targetID := int64(9502)
	ai := &scriptedAIService{replies: []string{fmt.Sprintf(`{"target_id":%d,"reason":"quiet target"}`, targetID)}}
	service := NewMafiaServiceWithAI(store, ai, AIConfig{Enabled: true})
	now := time.Now().UTC()
	settings := defaultMafiaRoomSettings()
	settings.EnableAIBots = true
	room := MafiaRoom{
		ID:          "AI-BOT-NIGHT",
		OwnerID:     9500,
		GroupChatID: -1009500,
		Status:      MafiaStatusNight,
		Settings:    settings,
		DayNumber:   1,
		Players: []MafiaPlayer{
			{TelegramID: mafiaTestBotBaseID - 20, DisplayName: "Bot Mafia", Role: MafiaRoleMafia, Alive: true, IsTestBot: true},
			{TelegramID: 9501, DisplayName: "One", Role: MafiaRoleCivilian, Alive: true},
			{TelegramID: targetID, DisplayName: "Target", Role: MafiaRoleCivilian, Alive: true},
		},
		NightActions: MafiaNightActions{MafiaVotes: map[string]int64{}, DetectiveChecks: map[string]int64{}, DetectiveResults: map[string]bool{}},
		CreatedAt:    now.Unix(),
		UpdatedAt:    now.Unix(),
	}
	if err := store.SaveMafiaRoom(room); err != nil {
		t.Fatalf("SaveMafiaRoom: %v", err)
	}

	_, result, err := service.ResolveNight(room.ID, now)
	if err != nil {
		t.Fatalf("ResolveNight: %v", err)
	}
	if result.KilledID != targetID {
		t.Fatalf("KilledID = %d, want %d", result.KilledID, targetID)
	}
	if ai.calls == 0 || !strings.Contains(ai.lastUserMessage, `"valid_targets"`) {
		t.Fatalf("ai bot was not used: calls=%d payload=%q", ai.calls, ai.lastUserMessage)
	}
}

func TestMafiaAIBotsVoteAndSpeak(t *testing.T) {
	store := newMafiaTestStore(t)
	targetID := int64(9600)
	ai := &scriptedAIService{replies: []string{fmt.Sprintf(`{"target_id":%d,"vote":"target","say":"Я бы посмотрел на Owner."}`, targetID)}}
	service := NewMafiaServiceWithAI(store, ai, AIConfig{Enabled: true})
	now := time.Now().UTC()
	settings := defaultMafiaRoomSettings()
	settings.EnableAIBots = true
	room := MafiaRoom{
		ID:          "AI-BOT-VOTE",
		OwnerID:     targetID,
		GroupChatID: -1009600,
		Status:      MafiaStatusDayDiscussion,
		Settings:    settings,
		DayNumber:   2,
		Players: []MafiaPlayer{
			{TelegramID: targetID, DisplayName: "Owner", Role: MafiaRoleCivilian, Alive: true},
			{TelegramID: mafiaTestBotBaseID - 21, DisplayName: "Bot Voter", Role: MafiaRoleCivilian, Alive: true, IsTestBot: true},
			{TelegramID: 9602, DisplayName: "Two", Role: MafiaRoleCivilian, Alive: true},
		},
		CreatedAt: now.Unix(),
		UpdatedAt: now.Unix(),
	}
	if err := store.SaveMafiaRoom(room); err != nil {
		t.Fatalf("SaveMafiaRoom: %v", err)
	}

	updated, err := service.StartVoting(room.ID, now)
	if err != nil {
		t.Fatalf("StartVoting: %v", err)
	}
	botID := mafiaTestBotBaseID - 21
	if got := updated.Votes[idKey(botID)]; got != idKey(targetID) {
		t.Fatalf("bot vote = %q, want %q", got, idKey(targetID))
	}
	if text := mafiaAIBotMessagesText(updated, MafiaStatusVoting); !strings.Contains(text, "Bot Voter") || !strings.Contains(text, "Owner") {
		t.Fatalf("ai bot message was not rendered: %q", text)
	}
}

func TestMafiaAIBotSuspicionMemoryStoresOnlyRelevantMessages(t *testing.T) {
	store := newMafiaTestStore(t)
	service := NewMafiaServiceWithAI(store, nil, AIConfig{Enabled: true})
	now := time.Now().UTC()
	settings := defaultMafiaRoomSettings()
	settings.EnableAIBots = true
	room := MafiaRoom{
		ID:          "AI-MEMORY",
		OwnerID:     9700,
		GroupChatID: -1009700,
		Status:      MafiaStatusDayDiscussion,
		Settings:    settings,
		DayNumber:   1,
		Players: []MafiaPlayer{
			{TelegramID: 9700, DisplayName: "Owner", Role: MafiaRoleCivilian, Alive: true},
			{TelegramID: 9701, DisplayName: "Mira Quiet", Username: "mira", Role: MafiaRoleCivilian, Alive: true},
			{TelegramID: mafiaTestBotBaseID - 30, DisplayName: "Bot Memory", Role: MafiaRoleCivilian, Alive: true, IsTestBot: true},
		},
		CreatedAt: now.Unix(),
		UpdatedAt: now.Unix(),
	}
	if err := store.SaveMafiaRoom(room); err != nil {
		t.Fatalf("SaveMafiaRoom: %v", err)
	}

	updated, remembered, err := service.RecordAIBotSuspicion(room.ID, 9700, 10, "Mira Quiet подозрительно молчит, похоже мафия", now)
	if err != nil {
		t.Fatalf("RecordAIBotSuspicion: %v", err)
	}
	if !remembered || len(updated.AIBotMemory) != 1 {
		t.Fatalf("expected one memory record, remembered=%v memory=%#v", remembered, updated.AIBotMemory)
	}
	if len(updated.AIBotMemory[0].TargetIDs) != 1 || updated.AIBotMemory[0].TargetIDs[0] != 9701 {
		t.Fatalf("unexpected targets: %#v", updated.AIBotMemory[0])
	}

	updated, remembered, err = service.RecordAIBotSuspicion(room.ID, 9700, 11, "я просто пью чай", now)
	if err != nil {
		t.Fatalf("RecordAIBotSuspicion unrelated: %v", err)
	}
	if remembered || len(updated.AIBotMemory) != 1 {
		t.Fatalf("unrelated message should be ignored, remembered=%v memory=%#v", remembered, updated.AIBotMemory)
	}
}

func TestMafiaAIBotDiscussionUsesSuspicionMemory(t *testing.T) {
	store := newMafiaTestStore(t)
	ai := &scriptedAIService{replies: []string{`{"say":"Mira слишком тихая, я бы не отпускал эту версию."}`}}
	service := NewMafiaServiceWithAI(store, ai, AIConfig{Enabled: true})
	now := time.Now().UTC()
	settings := defaultMafiaRoomSettings()
	settings.EnableAIBots = true
	room := MafiaRoom{
		ID:          "AI-DISCUSS",
		OwnerID:     9800,
		GroupChatID: -1009800,
		Status:      MafiaStatusDayDiscussion,
		Settings:    settings,
		DayNumber:   1,
		Players: []MafiaPlayer{
			{TelegramID: 9800, DisplayName: "Owner", Role: MafiaRoleCivilian, Alive: true},
			{TelegramID: 9801, DisplayName: "Mira", Role: MafiaRoleCivilian, Alive: true},
			{TelegramID: mafiaTestBotBaseID - 31, DisplayName: "Bot Talker", Role: MafiaRoleCivilian, Alive: true, IsTestBot: true},
		},
		CreatedAt: now.Unix(),
		UpdatedAt: now.Unix(),
	}
	if err := store.SaveMafiaRoom(room); err != nil {
		t.Fatalf("SaveMafiaRoom: %v", err)
	}
	if _, remembered, err := service.RecordAIBotSuspicion(room.ID, 9800, 20, "Mira подозрительно тихая", now); err != nil || !remembered {
		t.Fatalf("RecordAIBotSuspicion remembered=%v err=%v", remembered, err)
	}

	updated, msg, err := service.MaybeAddAIBotDiscussionMessage(room.ID, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("MaybeAddAIBotDiscussionMessage: %v", err)
	}
	if msg == nil || !strings.Contains(msg.Text, "Mira") {
		t.Fatalf("expected ai bot message about memory, msg=%#v", msg)
	}
	if len(updated.AIBotMessages) != 1 || updated.AIBotLastSpokeUnix == 0 {
		t.Fatalf("expected stored ai bot message, room=%#v", updated)
	}
	if !strings.Contains(ai.lastUserMessage, `"suspicion_memory"`) || !strings.Contains(ai.lastUserMessage, "Mira") {
		t.Fatalf("ai payload did not include memory: %q", ai.lastUserMessage)
	}
}
