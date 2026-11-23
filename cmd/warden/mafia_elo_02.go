package main

import (
	"fmt"
	"math"
	"time"
)

func mafiaApplyGameResultToState(state *MafiaState, room MafiaRoom, winner string, finishedAt time.Time) (MafiaRoom, MafiaGameResult, []MafiaEloHistoryEntry, bool, error) {
	mafiaEnsureRatingState(state)
	winningSide, ok := mafiaRatingSideFromWinner(winner)
	if !ok {
		return MafiaRoom{}, MafiaGameResult{}, nil, false, errMafiaWrongPhase
	}
	if finishedAt.IsZero() {
		finishedAt = time.Now().UTC()
	}
	finishedUnix := finishedAt.UTC().Unix()
	if idx := mafiaGameResultIndex(state.GameResults, room.ID); idx >= 0 && state.GameResults[idx].EloProcessed {
		room.Status = MafiaStatusFinished
		room.Winner = winner
		room.FinishedAt = state.GameResults[idx].FinishedAt
		room.EloProcessed = true
		room.GameResult = state.GameResults[idx]
		return room, state.GameResults[idx], state.GameResults[idx].EloHistory, false, nil
	}
	if room.EloProcessed {
		result := room.GameResult
		if result.ID == "" {
			result = mafiaGameResultFromRoom(room, winningSide, normalizeMafiaSettings(room.Settings).RatingMode != MafiaRatingUnranked, finishedUnix, nil)
		}
		return room, result, result.EloHistory, false, nil
	}

	settings := normalizeMafiaSettings(room.Settings)
	isRanked := settings.RatingMode != MafiaRatingUnranked
	snapshot := mafiaSnapshotForState(state, room)
	mafiaRating, civilianRating := mafiaTeamAverageRatings(snapshot)
	history := []MafiaEloHistoryEntry{}

	for i := range snapshot {
		snapshot[i].Won = snapshot[i].Side == winningSide
	}
	for _, snap := range snapshot {
		if snap.IsTestBot {
			continue
		}
		key := idKey(snap.UserID)
		rec, ok := state.RegisteredUsers[key]
		if !ok {
			continue
		}
		rating := mafiaNormalizePlayerRating(state.PlayerRatings[key])
		if rating.UserID == 0 {
			rating = mafiaRatingFromStats(snap.UserID, rec.Stats)
		}
		stats := rec.Stats
		stats = mafiaApplyCompletedGameStats(stats, snap, room, winningSide, settings.EnableAchievements)
		oldOverall := rating.OverallElo
		oldRole := mafiaRoleElo(rating, snap.Side)
		delta := 0
		expected := 0.0
		teamRating := civilianRating
		opponentRating := mafiaRating
		if snap.Side == MafiaRatingSideMafia {
			teamRating = mafiaRating
			opponentRating = civilianRating
		}
		expected = mafiaExpectedResult(teamRating, opponentRating)
		if isRanked {
			result := 0.0
			if snap.Won {
				result = 1
			}
			delta = int(math.Round(float64(mafiaRatingKFactor(rating.RankedGamesPlayed)) * (result - expected)))
			rating.OverallElo = mafiaClampElo(rating.OverallElo + delta)
			if snap.Side == MafiaRatingSideMafia {
				rating.MafiaElo = mafiaClampElo(rating.MafiaElo + delta)
			} else {
				rating.CivilianElo = mafiaClampElo(rating.CivilianElo + delta)
			}
			rating.RankedGamesPlayed++
			if snap.Won {
				rating.RankedWins++
			} else {
				rating.RankedLosses++
			}
			rating.CurrentRank = mafiaEloRank(rating.OverallElo)
			rating.UpdatedAt = finishedUnix
			state.PlayerRatings[key] = rating
			entry := MafiaEloHistoryEntry{
				ID:                 fmt.Sprintf("%s:%d", room.ID, snap.UserID),
				UserID:             snap.UserID,
				GameID:             room.ID,
				Role:               snap.Role,
				Side:               snap.Side,
				OldOverallElo:      oldOverall,
				NewOverallElo:      rating.OverallElo,
				DeltaOverall:       delta,
				OldRoleElo:         oldRole,
				NewRoleElo:         mafiaRoleElo(rating, snap.Side),
				DeltaRole:          delta,
				TeamRating:         teamRating,
				OpponentTeamRating: opponentRating,
				ExpectedResult:     expected,
				CreatedAt:          finishedUnix,
			}
			history = append(history, entry)
			stats = mafiaMirrorRatingIntoStats(stats, rating, delta, delta)
		} else {
			stats = mafiaMirrorRatingIntoStats(stats, rating, 0, 0)
		}
		if settings.EnableAchievements {
			stats.Achievements = mafiaAwardProgressAchievements(stats.Achievements, stats)
		}
		rec.Stats = stats
		state.RegisteredUsers[key] = rec
	}

	room.Status = MafiaStatusFinished
	room.Winner = winner
	room.FinishedAt = finishedUnix
	room.EloProcessed = true
	room.PlayersSnapshot = snapshot
	result := mafiaGameResultFromRoom(room, winningSide, isRanked, finishedUnix, history)
	room.GameResult = result
	state.GameResults = mafiaUpsertGameResult(state.GameResults, result)
	if isRanked {
		state.EloHistory = append(state.EloHistory, history...)
		if len(state.EloHistory) > 5000 {
			state.EloHistory = state.EloHistory[len(state.EloHistory)-5000:]
		}
	}
	return room, result, history, true, nil
}

func mafiaSnapshotForState(state *MafiaState, room MafiaRoom) []MafiaGamePlayerSnapshot {
	if len(room.PlayersSnapshot) > 0 {
		out := append([]MafiaGamePlayerSnapshot(nil), room.PlayersSnapshot...)
		for i := range out {
			if out[i].Side == "" {
				out[i].Side = mafiaRatingSideForRole(out[i].Role)
			}
			if out[i].StartingOverallElo <= 0 || out[i].StartingRoleElo <= 0 {
				rating := mafiaRatingForState(state, out[i].UserID)
				out[i].StartingOverallElo = rating.OverallElo
				out[i].StartingRoleElo = mafiaRoleElo(rating, out[i].Side)
			}
		}
		return out
	}
	out := make([]MafiaGamePlayerSnapshot, 0, len(room.Players))
	for _, player := range room.Players {
		side := mafiaRatingSideForRole(player.Role)
		rating := mafiaRatingForState(state, player.TelegramID)
		out = append(out, MafiaGamePlayerSnapshot{
			UserID:             player.TelegramID,
			Username:           player.Username,
			DisplayName:        player.DisplayName,
			Role:               player.Role,
			Side:               side,
			IsTestBot:          player.IsTestBot,
			StartingOverallElo: rating.OverallElo,
			StartingRoleElo:    mafiaRoleElo(rating, side),
		})
	}
	return out
}
