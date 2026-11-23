package main

import (
	"fmt"
	wardenweb "mirohodec-warden/internal/web"
	"sort"
	"strings"
)

func (s webPanelService) mafiaMiniAvatar(userID int64) (string, string, string) {
	if userID == 0 || s.app == nil || s.app.store == nil {
		return "", "", ""
	}
	rec, ok := s.app.store.MafiaUser(userID)
	if !ok {
		return "", "", ""
	}
	return rec.Profile.AvatarURL, rec.Profile.AvatarText, rec.Profile.AvatarTone
}

func mafiaMiniDisplayName(user mafiaMiniAuthUser) string {
	name := strings.TrimSpace(strings.TrimSpace(user.FirstName) + " " + strings.TrimSpace(user.LastName))
	if name == "" && strings.TrimSpace(user.Username) != "" {
		name = "@" + strings.TrimPrefix(strings.TrimSpace(user.Username), "@")
	}
	if name == "" {
		name = fmt.Sprintf("user %d", user.ID)
	}
	return name
}

func (s webPanelService) mafiaMiniProfile(userID int64) wardenweb.MafiaMiniAppProfile {
	rec, ok := s.app.store.MafiaUser(userID)
	if !ok {
		return wardenweb.MafiaMiniAppProfile{UserID: userID, DisplayName: fmt.Sprintf("user %d", userID), StatusText: "Профиль ещё не создан"}
	}
	stats := rec.Stats
	activeRoomID := ""
	for _, room := range s.app.store.MafiaRoomsForUser(userID) {
		if room.Status != MafiaStatusFinished && room.Status != MafiaStatusCancelled {
			activeRoomID = room.ID
			break
		}
	}
	name := mafiaUserCharacterName(rec)
	losses := stats.Games - stats.Wins
	if losses < 0 {
		losses = 0
	}
	status := "Свободен"
	if activeRoomID != "" {
		status = "В партии " + activeRoomID
	}
	rating, ok := s.app.store.MafiaPlayerRating(userID)
	if !ok {
		rating = mafiaRatingFromStats(userID, stats)
	}
	rating = mafiaNormalizePlayerRating(rating)
	return wardenweb.MafiaMiniAppProfile{
		UserID:        rec.TelegramID,
		Username:      rec.Username,
		DisplayName:   name,
		CharacterName: rec.Profile.CharacterName,
		AvatarURL:     rec.Profile.AvatarURL,
		AvatarText:    rec.Profile.AvatarText,
		AvatarTone:    rec.Profile.AvatarTone,
		Games:         stats.Games,
		Wins:          stats.Wins,
		Losses:        losses,
		Deaths:        stats.Deaths,
		FavoriteRole:  mafiaMiniFavoriteRole(stats),
		Stats:         mafiaMiniProfileStats(stats),
		Elo:           rating.OverallElo,
		Rank:          rating.CurrentRank,
		LastEloDelta:  stats.LastEloDelta,
		OverallElo:    rating.OverallElo,
		CivilianElo:   rating.CivilianElo,
		MafiaElo:      rating.MafiaElo,
		Achievements:  mafiaMiniAchievements(stats.Achievements),
		ActiveRoomID:  activeRoomID,
		StatusText:    status,
	}
}

func mafiaMiniProfileStats(stats MafiaStats) wardenweb.MafiaMiniAppProfileStats {
	losses := stats.Games - stats.Wins
	if losses < 0 {
		losses = 0
	}
	rating := mafiaRatingFromStats(0, stats)
	elo := rating.OverallElo
	winRate := 0
	if stats.Games > 0 {
		winRate = int(float64(stats.Wins)/float64(stats.Games)*100 + 0.5)
	}
	activity := stats.Games*8 + stats.Wins*4 + len(stats.Achievements)*5
	if activity > 100 {
		activity = 100
	}
	return wardenweb.MafiaMiniAppProfileStats{
		Games:                stats.Games,
		Wins:                 stats.Wins,
		Losses:               losses,
		Deaths:               stats.Deaths,
		WinRate:              winRate,
		Elo:                  elo,
		LastEloDelta:         stats.LastEloDelta,
		OverallElo:           elo,
		CivilianElo:          rating.CivilianElo,
		MafiaElo:             rating.MafiaElo,
		LastRoleEloDelta:     stats.LastRoleEloDelta,
		RankedGamesPlayed:    stats.RankedGamesPlayed,
		RankedWins:           stats.RankedWins,
		RankedLosses:         stats.RankedLosses,
		RoleReveals:          stats.RoleReveals,
		CancelledGamesCaused: stats.CancelledGamesCaused,
		MafiaWins:            stats.MafiaWins,
		CivilianWins:         stats.CivilianWins,
		DetectiveWins:        stats.DetectiveWins,
		DoctorWins:           stats.DoctorWins,
		Achievements:         len(stats.Achievements),
		ActivityScore:        activity,
	}
}

func (s webPanelService) mafiaMiniLeaderboard(limit int) []wardenweb.MafiaMiniAppLeaderboardEntry {
	if limit <= 0 {
		limit = 10
	}
	ratings := s.app.store.MafiaPlayerRatings()
	usersByID := map[int64]MafiaRegisteredUser{}
	for _, user := range s.app.store.MafiaUsers() {
		usersByID[user.TelegramID] = user
	}
	filtered := make([]MafiaPlayerRating, 0, limit)
	for _, rating := range ratings {
		if rating.RankedGamesPlayed > 0 {
			filtered = append(filtered, mafiaNormalizePlayerRating(rating))
		}
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		if filtered[i].OverallElo == filtered[j].OverallElo {
			if filtered[i].RankedWins == filtered[j].RankedWins {
				return filtered[i].UserID < filtered[j].UserID
			}
			return filtered[i].RankedWins > filtered[j].RankedWins
		}
		return filtered[i].OverallElo > filtered[j].OverallElo
	})
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}
	out := make([]wardenweb.MafiaMiniAppLeaderboardEntry, 0, len(filtered))
	for _, rating := range filtered {
		user := usersByID[rating.UserID]
		winRate := 0
		if rating.RankedGamesPlayed > 0 {
			winRate = int(float64(rating.RankedWins)/float64(rating.RankedGamesPlayed)*100 + 0.5)
		}
		out = append(out, wardenweb.MafiaMiniAppLeaderboardEntry{
			UserID:            rating.UserID,
			DisplayName:       mafiaUserCharacterName(user),
			Username:          user.Username,
			AvatarURL:         user.Profile.AvatarURL,
			AvatarText:        user.Profile.AvatarText,
			AvatarTone:        user.Profile.AvatarTone,
			Elo:               rating.OverallElo,
			Rank:              rating.CurrentRank,
			OverallElo:        rating.OverallElo,
			CivilianElo:       rating.CivilianElo,
			MafiaElo:          rating.MafiaElo,
			RankedGamesPlayed: rating.RankedGamesPlayed,
			RankedWins:        rating.RankedWins,
			RankedLosses:      rating.RankedLosses,
			WinRate:           winRate,
			CurrentRank:       rating.CurrentRank,
			Wins:              user.Stats.Wins,
			Games:             user.Stats.Games,
		})
	}
	return out
}
