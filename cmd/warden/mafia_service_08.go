package main

import (
	"strings"
)

func mafiaEloRank(elo int) string {
	elo = mafiaNormalizedElo(elo)
	switch {
	case elo >= 1900:
		return "Лицо города"
	case elo >= 1700:
		return "Архитектор лжи"
	case elo >= 1500:
		return "Серый кардинал"
	case elo >= 1300:
		return "Игрок улиц"
	case elo >= 1100:
		return "Наблюдатель"
	case elo >= 900:
		return "Городской слух"
	default:
		return "Тень переулка"
	}
}

func mafiaAwardAchievements(existing []string, player MafiaPlayer, winner string, stats MafiaStats) []string {
	out := append([]string(nil), existing...)
	add := func(id string) { out = mafiaAddAchievement(out, id) }
	add("first_game")
	if isMafiaTeam(player.Role) && winner == "mafia" {
		add("first_knife")
	}
	if player.Role == MafiaRoleDetective && winner == "civilian" {
		add("sharp_check")
	}
	if player.Role == MafiaRoleDoctor && winner == "civilian" {
		add("night_shift")
	}
	if player.Will != "" {
		add("last_thought")
	}
	if player.Role == MafiaRoleCivilian && !player.Alive && winner == "mafia" {
		add("city_mistake")
	}
	if winner == "mafia" && isMafiaTeam(player.Role) && player.Alive {
		add("mask_survived")
	}
	if winner == "civilian" && !isMafiaTeam(player.Role) && player.Alive {
		add("city_stands")
	}
	if ((winner == "mafia" && isMafiaTeam(player.Role)) || (winner == "civilian" && !isMafiaTeam(player.Role))) && !player.Alive {
		add("posthumous_win")
	}
	if stats.Wins > 0 {
		add("first_win")
	}
	if stats.Games >= 5 {
		add("five_games")
	}
	if stats.Games >= 10 {
		add("ten_games")
	}
	if stats.Wins >= 5 {
		add("five_wins")
	}
	if stats.MafiaWins >= 3 {
		add("mafia_veteran")
	}
	if stats.CivilianWins >= 3 {
		add("civilian_veteran")
	}
	if stats.DetectiveWins >= 3 {
		add("detective_veteran")
	}
	if stats.DoctorWins >= 3 {
		add("doctor_veteran")
	}
	if stats.RankedGamesPlayed > 0 {
		add("ranked_debut")
	}
	return out
}

func mafiaAwardProgressAchievements(existing []string, stats MafiaStats) []string {
	out := append([]string(nil), existing...)
	add := func(id string) { out = mafiaAddAchievement(out, id) }
	if stats.Wins > 0 {
		add("first_win")
	}
	if stats.Games >= 5 {
		add("five_games")
	}
	if stats.Games >= 10 {
		add("ten_games")
	}
	if stats.Wins >= 5 {
		add("five_wins")
	}
	if stats.MafiaWins >= 3 {
		add("mafia_veteran")
	}
	if stats.CivilianWins >= 3 {
		add("civilian_veteran")
	}
	if stats.DetectiveWins >= 3 {
		add("detective_veteran")
	}
	if stats.DoctorWins >= 3 {
		add("doctor_veteran")
	}
	if stats.RankedGamesPlayed > 0 {
		add("ranked_debut")
	}
	return out
}

func mafiaAddAchievement(values []string, id string) []string {
	id = strings.TrimSpace(id)
	if id == "" {
		return values
	}
	for _, current := range values {
		if current == id {
			return values
		}
	}
	return append(values, id)
}

func mafiaAllAchievementIDs() []string {
	return []string{
		"first_game",
		"first_win",
		"ranked_debut",
		"first_knife",
		"mask_survived",
		"sharp_check",
		"night_shift",
		"last_thought",
		"city_stands",
		"posthumous_win",
		"city_mistake",
		"five_games",
		"ten_games",
		"five_wins",
		"mafia_veteran",
		"civilian_veteran",
		"detective_veteran",
		"doctor_veteran",
	}
}

func (s *MafiaService) playerActionRoom(roomID string, userID int64, role MafiaRole, status MafiaRoomStatus) (MafiaRoom, MafiaPlayer, error) {
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, MafiaPlayer{}, errMafiaRoomNotFound
	}
	if room.Status != status {
		return MafiaRoom{}, MafiaPlayer{}, errMafiaWrongPhase
	}
	idx := mafiaRoomPlayerIndex(room, userID)
	if idx < 0 {
		return MafiaRoom{}, MafiaPlayer{}, errMafiaPlayerNotFound
	}
	player := room.Players[idx]
	if !player.Alive {
		return MafiaRoom{}, MafiaPlayer{}, errMafiaPlayerDead
	}
	if player.Role != role {
		return MafiaRoom{}, MafiaPlayer{}, errMafiaWrongRole
	}
	return room, player, nil
}

func (s *MafiaService) ownerLobbyRoom(roomID string, ownerID int64) (MafiaRoom, error) {
	room, ok := s.store.MafiaRoom(roomID)
	if !ok {
		return MafiaRoom{}, errMafiaRoomNotFound
	}
	if room.OwnerID != ownerID {
		return MafiaRoom{}, errMafiaNotRoomOwner
	}
	if room.Status != MafiaStatusLobby {
		return MafiaRoom{}, errMafiaRoomNotLobby
	}
	room = normalizeMafiaRoomRecruitment(room)
	return room, nil
}
