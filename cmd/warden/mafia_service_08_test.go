package main

import (
	"testing"
	"time"

	tele "gopkg.in/telebot.v3"
)

func TestMafiaDeadOwnerKeepsHostControls(t *testing.T) {
	room := MafiaRoom{
		ID:        "OWNER-DEAD",
		OwnerID:   9400,
		Status:    MafiaStatusNight,
		Settings:  defaultMafiaRoomSettings(),
		DayNumber: 1,
		Players: []MafiaPlayer{
			{TelegramID: 9400, DisplayName: "Owner", Role: MafiaRoleCivilian, Alive: false},
			{TelegramID: 9401, DisplayName: "Mafia", Role: MafiaRoleMafia, Alive: true},
			{TelegramID: 9402, DisplayName: "Civilian", Role: MafiaRoleCivilian, Alive: true},
			{TelegramID: 9403, DisplayName: "Doctor", Role: MafiaRoleDoctor, Alive: true},
		},
	}
	markup := mafiaRoomMarkup(room, room.OwnerID)
	if mafiaMarkupContains(markup, "Завершить ночь") || !mafiaMarkupContains(markup, "Завершить партию") {
		t.Fatalf("dead owner markup must keep only emergency owner controls")
	}
	reply := mafiaRoomReplyKeyboard(room, room.OwnerID)
	if mafiaReplyMarkupContains(reply, "Завершить ночь") || mafiaReplyMarkupContains(reply, "Скип фазы") || !mafiaReplyMarkupContains(reply, "Завершить партию") {
		t.Fatalf("dead owner reply keyboard must keep only emergency owner controls")
	}
}

func TestMafiaOwnerCanEndActiveGameWithoutStats(t *testing.T) {
	store := newMafiaTestStore(t)
	service := NewMafiaService(store)
	now := time.Now().UTC()
	owner := &tele.User{ID: 9500, FirstName: "Owner"}
	if _, err := service.RegisterUser(owner, now); err != nil {
		t.Fatalf("RegisterUser: %v", err)
	}
	room := MafiaRoom{
		ID:          "OWNER-END",
		OwnerID:     owner.ID,
		GroupChatID: -1009500,
		Status:      MafiaStatusNight,
		Settings:    defaultMafiaRoomSettings(),
		DayNumber:   1,
		Players: []MafiaPlayer{
			{TelegramID: owner.ID, DisplayName: "Owner", Role: MafiaRoleCivilian, Alive: false},
			{TelegramID: 9501, DisplayName: "Mafia", Role: MafiaRoleMafia, Alive: true},
			{TelegramID: 9502, DisplayName: "Civilian", Role: MafiaRoleCivilian, Alive: true},
			{TelegramID: 9503, DisplayName: "Doctor", Role: MafiaRoleDoctor, Alive: true},
		},
		CreatedAt: now.Unix(),
		UpdatedAt: now.Unix(),
	}
	if err := store.SaveMafiaRoom(room); err != nil {
		t.Fatalf("SaveMafiaRoom: %v", err)
	}
	updated, err := service.EndGameByOwner(room.ID, owner.ID, now)
	if err != nil {
		t.Fatalf("EndGameByOwner: %v", err)
	}
	if updated.Status != MafiaStatusCancelled || updated.CancelledBy != owner.ID {
		t.Fatalf("unexpected ended room: %#v", updated)
	}
	user, ok := store.MafiaUser(owner.ID)
	if !ok {
		t.Fatalf("owner user not found")
	}
	if user.Stats.Games != 0 || user.Stats.Wins != 0 || user.Stats.CancelledGamesCaused != 0 {
		t.Fatalf("manual end must not update stats: %#v", user.Stats)
	}
}

func TestMafiaRankedEloUsesTeamExpectedResultOnce(t *testing.T) {
	store := newMafiaTestStore(t)
	service, room := startMafiaTestRoom(t, store, 6)
	var civilian MafiaPlayer
	var mafia MafiaPlayer
	for _, player := range room.Players {
		if isMafiaTeam(player.Role) && mafia.TelegramID == 0 {
			mafia = player
		}
		if !isMafiaTeam(player.Role) && civilian.TelegramID == 0 {
			civilian = player
		}
	}
	if civilian.TelegramID == 0 || mafia.TelegramID == 0 {
		t.Fatalf("expected both teams in room: %#v", room.Players)
	}

	service.finishMafiaRoom(&room, "civilian")
	civilianRating, ok := store.MafiaPlayerRating(civilian.TelegramID)
	if !ok {
		t.Fatalf("civilian rating missing")
	}
	mafiaRating, ok := store.MafiaPlayerRating(mafia.TelegramID)
	if !ok {
		t.Fatalf("mafia rating missing")
	}
	if civilianRating.OverallElo != 1020 || civilianRating.CivilianElo != 1020 {
		t.Fatalf("civilian rating = %#v, want overall/civilian 1020", civilianRating)
	}
	if mafiaRating.OverallElo != 980 || mafiaRating.MafiaElo != 980 {
		t.Fatalf("mafia rating = %#v, want overall/mafia 980", mafiaRating)
	}

	service.finishMafiaRoom(&room, "civilian")
	again, _ := store.MafiaPlayerRating(civilian.TelegramID)
	if again.OverallElo != civilianRating.OverallElo || again.RankedGamesPlayed != 1 {
		t.Fatalf("ELO processed more than once: before=%#v after=%#v", civilianRating, again)
	}
	results := store.MafiaGameResultsForUser(civilian.TelegramID)
	if len(results) != 1 || !results[0].IsRanked || !results[0].EloProcessed {
		t.Fatalf("ranked game result not saved correctly: %#v", results)
	}
}

func TestMafiaUnrankedGameSavesResultWithoutEloChange(t *testing.T) {
	store := newMafiaTestStore(t)
	service, room := startMafiaTestRoom(t, store, 6)
	room.Settings.RatingMode = MafiaRatingUnranked
	if err := store.SaveMafiaRoom(room); err != nil {
		t.Fatalf("SaveMafiaRoom: %v", err)
	}
	player := room.Players[0]

	service.finishMafiaRoom(&room, "civilian")
	rating, ok := store.MafiaPlayerRating(player.TelegramID)
	if !ok {
		t.Fatalf("rating missing")
	}
	if rating.OverallElo != 1000 || rating.RankedGamesPlayed != 0 {
		t.Fatalf("unranked changed rating: %#v", rating)
	}
	results := store.MafiaGameResultsForUser(player.TelegramID)
	if len(results) != 1 || results[0].IsRanked || !results[0].EloProcessed {
		t.Fatalf("unranked result not saved correctly: %#v", results)
	}
}

func TestMafiaAutoHostStartsVotingAfterDiscussionTimer(t *testing.T) {
	store := newMafiaTestStore(t)
	now := time.Now().UTC()
	room := MafiaRoom{
		ID:          "AUTO-DAY",
		OwnerID:     9600,
		GroupChatID: -1009600,
		Status:      MafiaStatusDayDiscussion,
		Settings:    defaultMafiaRoomSettings(),
		DayNumber:   1,
		Players: []MafiaPlayer{
			{TelegramID: 9600, DisplayName: "Owner", Role: MafiaRoleCivilian, Alive: true},
			{TelegramID: mafiaTestBotBaseID - 20, DisplayName: "Bot Mafia", Role: MafiaRoleMafia, Alive: true, IsTestBot: true},
			{TelegramID: mafiaTestBotBaseID - 21, DisplayName: "Bot One", Role: MafiaRoleCivilian, Alive: true, IsTestBot: true},
			{TelegramID: mafiaTestBotBaseID - 22, DisplayName: "Bot Two", Role: MafiaRoleCivilian, Alive: true, IsTestBot: true},
		},
		CreatedAt: now.Add(-10 * time.Minute).Unix(),
		UpdatedAt: now.Add(-10 * time.Minute).Unix(),
	}
	if err := store.SaveMafiaRoom(room); err != nil {
		t.Fatalf("SaveMafiaRoom: %v", err)
	}
	app := &App{store: store}
	if err := app.advanceMafiaRoomAsLaya(room.ID, now); err != nil {
		t.Fatalf("advanceMafiaRoomAsLaya: %v", err)
	}
	updated, ok := store.MafiaRoom(room.ID)
	if !ok {
		t.Fatalf("room not found")
	}
	if updated.Status != MafiaStatusVoting {
		t.Fatalf("status=%s, want voting", updated.Status)
	}
}
