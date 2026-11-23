package main

import (
	"sort"
)

func mafiaApplyCompletedGameStats(stats MafiaStats, snap MafiaGamePlayerSnapshot, room MafiaRoom, winningSide MafiaRatingSide, achievements bool) MafiaStats {
	stats.Games++
	finalPlayer := MafiaPlayer{TelegramID: snap.UserID, Role: snap.Role, Alive: true}
	if idx := mafiaRoomPlayerIndex(room, snap.UserID); idx >= 0 {
		finalPlayer = room.Players[idx]
	}
	if !finalPlayer.Alive {
		stats.Deaths++
	}
	if snap.Side == winningSide {
		stats.Wins++
		if snap.Side == MafiaRatingSideMafia {
			stats.MafiaWins++
		} else {
			stats.CivilianWins++
			if snap.Role == MafiaRoleDetective {
				stats.DetectiveWins++
			}
			if snap.Role == MafiaRoleDoctor {
				stats.DoctorWins++
			}
		}
	}
	if achievements {
		stats.Achievements = mafiaAwardAchievements(stats.Achievements, finalPlayer, string(winningSide), stats)
	}
	return stats
}

func mafiaGameResultFromRoom(room MafiaRoom, winningSide MafiaRatingSide, ranked bool, finishedAt int64, history []MafiaEloHistoryEntry) MafiaGameResult {
	startedAt := room.CreatedAt
	if startedAt <= 0 {
		startedAt = finishedAt
	}
	duration := finishedAt - startedAt
	if duration < 0 {
		duration = 0
	}
	return MafiaGameResult{
		ID:              room.ID,
		RoomID:          room.ID,
		WinningSide:     winningSide,
		IsRanked:        ranked,
		EloProcessed:    true,
		StartedAt:       startedAt,
		FinishedAt:      finishedAt,
		DurationSeconds: duration,
		PlayersSnapshot: append([]MafiaGamePlayerSnapshot(nil), room.PlayersSnapshot...),
		EloHistory:      append([]MafiaEloHistoryEntry(nil), history...),
	}
}

func mafiaUpsertGameResult(results []MafiaGameResult, result MafiaGameResult) []MafiaGameResult {
	if result.ID == "" {
		return results
	}
	for i := range results {
		if results[i].ID == result.ID {
			results[i] = result
			return results
		}
	}
	results = append(results, result)
	sort.SliceStable(results, func(i, j int) bool { return results[i].FinishedAt > results[j].FinishedAt })
	if len(results) > 1000 {
		return results[:1000]
	}
	return results
}

func mafiaGameResultIndex(results []MafiaGameResult, gameID string) int {
	for i := range results {
		if results[i].ID == gameID || results[i].RoomID == gameID {
			return i
		}
	}
	return -1
}

func mafiaGameResultsForUser(results []MafiaGameResult, userID int64) []MafiaGameResult {
	out := make([]MafiaGameResult, 0, 8)
	for _, result := range results {
		for _, player := range result.PlayersSnapshot {
			if player.UserID == userID {
				out = append(out, result)
				break
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].FinishedAt > out[j].FinishedAt })
	if len(out) > 30 {
		return out[:30]
	}
	return out
}

func mafiaRatingForState(state *MafiaState, userID int64) MafiaPlayerRating {
	mafiaEnsureRatingState(state)
	if rating, ok := state.PlayerRatings[idKey(userID)]; ok {
		return mafiaNormalizePlayerRating(rating)
	}
	if rec, ok := state.RegisteredUsers[idKey(userID)]; ok {
		return mafiaRatingFromStats(userID, rec.Stats)
	}
	return mafiaDefaultPlayerRating(userID)
}

func mafiaDefaultPlayerRating(userID int64) MafiaPlayerRating {
	return MafiaPlayerRating{
		UserID:      userID,
		OverallElo:  mafiaDefaultElo,
		CivilianElo: mafiaDefaultElo,
		MafiaElo:    mafiaDefaultElo,
		CurrentRank: mafiaEloRank(mafiaDefaultElo),
	}
}

func mafiaRatingFromStats(userID int64, stats MafiaStats) MafiaPlayerRating {
	overall := stats.OverallElo
	if overall <= 0 {
		overall = stats.Elo
	}
	overall = mafiaNormalizedElo(overall)
	civilian := mafiaNormalizedElo(stats.CivilianElo)
	mafia := mafiaNormalizedElo(stats.MafiaElo)
	if stats.CivilianElo <= 0 {
		civilian = overall
	}
	if stats.MafiaElo <= 0 {
		mafia = overall
	}
	return mafiaNormalizePlayerRating(MafiaPlayerRating{
		UserID:            userID,
		OverallElo:        overall,
		CivilianElo:       civilian,
		MafiaElo:          mafia,
		RankedGamesPlayed: stats.RankedGamesPlayed,
		RankedWins:        stats.RankedWins,
		RankedLosses:      stats.RankedLosses,
		CurrentRank:       mafiaEloRank(overall),
	})
}

func mafiaNormalizePlayerRating(rating MafiaPlayerRating) MafiaPlayerRating {
	rating.OverallElo = mafiaNormalizedElo(rating.OverallElo)
	rating.CivilianElo = mafiaNormalizedElo(rating.CivilianElo)
	rating.MafiaElo = mafiaNormalizedElo(rating.MafiaElo)
	rating.CurrentRank = mafiaEloRank(rating.OverallElo)
	return rating
}

func mafiaMirrorRatingIntoStats(stats MafiaStats, rating MafiaPlayerRating, overallDelta int, roleDelta int) MafiaStats {
	rating = mafiaNormalizePlayerRating(rating)
	stats.Elo = rating.OverallElo
	stats.OverallElo = rating.OverallElo
	stats.CivilianElo = rating.CivilianElo
	stats.MafiaElo = rating.MafiaElo
	stats.LastEloDelta = overallDelta
	stats.LastRoleEloDelta = roleDelta
	stats.RankedGamesPlayed = rating.RankedGamesPlayed
	stats.RankedWins = rating.RankedWins
	stats.RankedLosses = rating.RankedLosses
	return stats
}

func mafiaRoleElo(rating MafiaPlayerRating, side MafiaRatingSide) int {
	rating = mafiaNormalizePlayerRating(rating)
	if side == MafiaRatingSideMafia {
		return rating.MafiaElo
	}
	return rating.CivilianElo
}

func mafiaRatingSideForRole(role MafiaRole) MafiaRatingSide {
	if isMafiaTeam(role) {
		return MafiaRatingSideMafia
	}
	return MafiaRatingSideCivilian
}
