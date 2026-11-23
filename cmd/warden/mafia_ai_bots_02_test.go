package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestMafiaAIBotKillRecommendationTargetsStrongSuspector(t *testing.T) {
	store := newMafiaTestStore(t)
	ownerID := int64(9900)
	botID := mafiaTestBotBaseID - 41
	ai := &scriptedAIService{replies: []string{fmt.Sprintf(`{"target_id":%d,"reason":"он сильно давил на мафию"}`, ownerID)}}
	service := NewMafiaServiceWithAI(store, ai, AIConfig{Enabled: true})
	now := time.Now().UTC()
	settings := defaultMafiaRoomSettings()
	settings.EnableAIBots = true
	room := MafiaRoom{
		ID:             "AI-KILL-HINT",
		OwnerID:        ownerID,
		GroupChatID:    -1009900,
		Status:         MafiaStatusNight,
		Settings:       settings,
		DayNumber:      2,
		PhaseStartedAt: now.Unix(),
		Players: []MafiaPlayer{
			{TelegramID: ownerID, DisplayName: "Owner", Role: MafiaRoleCivilian, Alive: true},
			{TelegramID: 9901, DisplayName: "Quiet", Role: MafiaRoleCivilian, Alive: true},
			{TelegramID: botID, DisplayName: "Bot Mafia", Role: MafiaRoleMafia, MafiaAlias: "Shade", Alive: true, IsTestBot: true},
		},
		NightActions: MafiaNightActions{MafiaVotes: map[string]int64{}, DetectiveChecks: map[string]int64{}, DetectiveResults: map[string]bool{}},
		AIBotMemory: []MafiaAIBotMemory{{
			DayNumber:   2,
			Phase:       MafiaStatusDayDiscussion,
			SpeakerID:   ownerID,
			SpeakerName: "Owner",
			Text:        "Bot Mafia точно мафия, завтра голосуем его",
			TargetIDs:   []int64{botID},
			TargetNames: []string{"Bot Mafia"},
			CreatedAt:   now.Add(-time.Minute).Unix(),
		}},
		CreatedAt: now.Add(-10 * time.Minute).Unix(),
		UpdatedAt: now.Add(-time.Minute).Unix(),
	}
	if err := store.SaveMafiaRoom(room); err != nil {
		t.Fatalf("SaveMafiaRoom: %v", err)
	}

	updated, messages, recipients, err := service.MaybeAddAIBotKillRecommendations(room.ID, now)
	if err != nil {
		t.Fatalf("MaybeAddAIBotKillRecommendations: %v", err)
	}
	if got := updated.NightActions.MafiaVotes[idKey(botID)]; got != ownerID {
		t.Fatalf("bot kill vote = %d, want %d", got, ownerID)
	}
	if len(messages) != 1 || !strings.Contains(messages[0].Text, "Owner") || !strings.Contains(messages[0].Text, "давил") {
		t.Fatalf("unexpected shadow recommendation: %#v", messages)
	}
	if len(recipients) != 1 || recipients[0] != botID {
		t.Fatalf("unexpected recipients: %#v", recipients)
	}
	if !strings.Contains(ai.lastUserMessage, `"kill_recommendations"`) || !strings.Contains(ai.lastUserMessage, "Owner") {
		t.Fatalf("ai payload did not include kill recommendations: %q", ai.lastUserMessage)
	}

	updated, messages, _, err = service.MaybeAddAIBotKillRecommendations(room.ID, now.Add(10*time.Second))
	if err != nil {
		t.Fatalf("MaybeAddAIBotKillRecommendations duplicate: %v", err)
	}
	if len(messages) != 0 || len(updated.MafiaPrivateChat) != 1 {
		t.Fatalf("duplicate recommendation should not be added, messages=%#v chat=%#v", messages, updated.MafiaPrivateChat)
	}
}
