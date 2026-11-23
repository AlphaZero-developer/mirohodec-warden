package main

import (
	"math"
)

func mafiaRatingSideFromWinner(winner string) (MafiaRatingSide, bool) {
	switch winner {
	case "mafia":
		return MafiaRatingSideMafia, true
	case "civilian", "city", "town":
		return MafiaRatingSideCivilian, true
	default:
		return "", false
	}
}

func mafiaTeamAverageRatings(snapshot []MafiaGamePlayerSnapshot) (float64, float64) {
	mafiaSum := 0
	mafiaCount := 0
	civilianSum := 0
	civilianCount := 0
	for _, player := range snapshot {
		elo := mafiaNormalizedElo(player.StartingOverallElo)
		if player.Side == MafiaRatingSideMafia {
			mafiaSum += elo
			mafiaCount++
		} else {
			civilianSum += elo
			civilianCount++
		}
	}
	return mafiaAverageOrDefault(mafiaSum, mafiaCount), mafiaAverageOrDefault(civilianSum, civilianCount)
}

func mafiaAverageOrDefault(sum int, count int) float64 {
	if count <= 0 {
		return mafiaDefaultElo
	}
	return float64(sum) / float64(count)
}

func mafiaExpectedResult(teamRating float64, opponentTeamRating float64) float64 {
	return 1 / (1 + math.Pow(10, (opponentTeamRating-teamRating)/400))
}

func mafiaRatingKFactor(rankedGames int) int {
	switch {
	case rankedGames < 10:
		return 40
	case rankedGames > 50:
		return 16
	default:
		return 24
	}
}

func mafiaClampElo(value int) int {
	if value < 0 {
		return 0
	}
	return value
}

func mafiaEnsureRatingState(state *MafiaState) {
	if state.RegisteredUsers == nil {
		state.RegisteredUsers = map[string]MafiaRegisteredUser{}
	}
	if state.Rooms == nil {
		state.Rooms = map[string]MafiaRoom{}
	}
	if state.PlayerRatings == nil {
		state.PlayerRatings = map[string]MafiaPlayerRating{}
	}
	if state.EloHistory == nil {
		state.EloHistory = []MafiaEloHistoryEntry{}
	}
	if state.GameResults == nil {
		state.GameResults = []MafiaGameResult{}
	}
}
