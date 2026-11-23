package main

import (
	wardenweb "mirohodec-warden/internal/web"
	"strings"
)

func mafiaMiniCanSeeMafiaChannel(room MafiaRoom, player MafiaPlayer, viewerIdx int) bool {
	if viewerIdx < 0 || !isMafiaTeam(player.Role) {
		return false
	}
	settings := normalizeMafiaSettings(room.Settings)
	if settings.MafiaChatMode == MafiaChatDisabled {
		return false
	}
	if settings.MafiaChatMode == MafiaChatNightOnly && room.Status != MafiaStatusNight && room.Status != MafiaStatusRoleDistribution {
		return false
	}
	return player.Alive || settings.DeadMafiaCanReadChat
}

func mafiaMiniVisibleChannels(room MafiaRoom, viewerIdx int, isHost bool) map[string]bool {
	player := MafiaPlayer{}
	if viewerIdx >= 0 {
		player = room.Players[viewerIdx]
	}
	out := map[string]bool{mafiaMiniChannelPublic: true, mafiaMiniChannelSystem: true}
	if isHost || mafiaMiniCanSeeMafiaChannel(room, player, viewerIdx) {
		out[mafiaMiniChannelMafia] = true
	}
	if isHost || (viewerIdx >= 0 && !player.Alive) {
		out[mafiaMiniChannelDead] = true
	}
	if isHost {
		out[mafiaMiniChannelHost] = true
	}
	return out
}

func mafiaMiniCanSeeChannel(room MafiaRoom, viewerIdx int, isHost bool, channel string) bool {
	return mafiaMiniVisibleChannels(room, viewerIdx, isHost)[strings.ToUpper(channel)]
}

func mafiaMiniAliveTargets(room MafiaRoom, excludeID int64, allowMafia bool, excludeMafia bool) []wardenweb.MafiaMiniAppActionTarget {
	targets := make([]wardenweb.MafiaMiniAppActionTarget, 0, len(room.Players))
	for _, player := range room.Players {
		if !player.Alive || player.TelegramID == excludeID {
			continue
		}
		if excludeMafia && isMafiaTeam(player.Role) {
			continue
		}
		if !allowMafia && isMafiaTeam(player.Role) {
			continue
		}
		targets = append(targets, wardenweb.MafiaMiniAppActionTarget{
			ID:       player.TelegramID,
			Key:      idKey(player.TelegramID),
			Name:     mafiaRoomPlayerLabel(room, player),
			Subtitle: mafiaMiniTargetSubtitle(player),
		})
	}
	return targets
}

func mafiaMiniTargetSubtitle(player MafiaPlayer) string {
	if player.IsTestBot {
		return "бот"
	}
	if player.Username != "" {
		return "@" + player.Username
	}
	return ""
}

func mafiaMiniPlayerActionSubmitted(room MafiaRoom, player MafiaPlayer) bool {
	if !player.Alive {
		return false
	}
	switch room.Status {
	case MafiaStatusNight:
		switch player.Role {
		case MafiaRoleMafia:
			return room.NightActions.MafiaVotes[idKey(player.TelegramID)] != 0
		case MafiaRoleDoctor:
			return room.NightActions.DoctorTargetID != 0
		case MafiaRoleDetective:
			_, ok := room.NightActions.DetectiveChecks[idKey(player.TelegramID)]
			return ok
		}
	case MafiaStatusVoting:
		return strings.TrimSpace(room.Votes[idKey(player.TelegramID)]) != ""
	}
	return false
}

func mafiaMiniDetectiveKnowsRole(room MafiaRoom, detectiveID int64, targetID int64) bool {
	if detectiveID == 0 || targetID == 0 {
		return false
	}
	if room.NightActions.DetectiveChecks[idKey(detectiveID)] == targetID {
		return true
	}
	known := room.DetectiveKnownRoles[idKey(detectiveID)]
	if len(known) == 0 {
		return false
	}
	return known[idKey(targetID)] != ""
}

func (s webPanelService) mafiaMiniFinalResult(room MafiaRoom, viewerID int64, viewerIdx int) *wardenweb.MafiaMiniAppFinalResult {
	if room.Status != MafiaStatusFinished || room.Winner == "" || viewerIdx < 0 {
		return nil
	}
	result := room.GameResult
	if result.ID == "" {
		if results := s.app.store.MafiaGameResultsForUser(viewerID); len(results) > 0 {
			for _, candidate := range results {
				if candidate.RoomID == room.ID || candidate.ID == room.ID {
					result = candidate
					break
				}
			}
		}
	}
	if result.ID == "" {
		result.ID = room.ID
		result.RoomID = room.ID
		result.IsRanked = normalizeMafiaSettings(room.Settings).RatingMode != MafiaRatingUnranked
		result.PlayersSnapshot = room.PlayersSnapshot
	}
	winningSide, ok := mafiaRatingSideFromWinner(room.Winner)
	if !ok {
		return nil
	}
	if result.WinningSide != "" {
		winningSide = result.WinningSide
	}
	player := room.Players[viewerIdx]
	side := mafiaRatingSideForRole(player.Role)
	snapshotOverall := mafiaDefaultElo
	snapshotRole := mafiaDefaultElo
	for _, snap := range result.PlayersSnapshot {
		if snap.UserID == viewerID {
			player.Role = snap.Role
			side = snap.Side
			snapshotOverall = mafiaNormalizedElo(snap.StartingOverallElo)
			snapshotRole = mafiaNormalizedElo(snap.StartingRoleElo)
			break
		}
	}
	if snapshotOverall == mafiaDefaultElo && len(result.PlayersSnapshot) == 0 {
		if rating, ok := s.app.store.MafiaPlayerRating(viewerID); ok {
			snapshotOverall = rating.OverallElo
			snapshotRole = mafiaRoleElo(rating, side)
		}
	}
	overall := wardenweb.MafiaMiniAppEloLine{Label: "Overall ELO", Old: snapshotOverall, New: snapshotOverall}
	roleLine := wardenweb.MafiaMiniAppEloLine{Label: mafiaMiniRoleEloLabel(side), Old: snapshotRole, New: snapshotRole}
	for _, entry := range result.EloHistory {
		if entry.UserID != viewerID {
			continue
		}
		overall.Old = entry.OldOverallElo
		overall.New = entry.NewOverallElo
		overall.Delta = entry.DeltaOverall
		roleLine.Old = entry.OldRoleElo
		roleLine.New = entry.NewRoleElo
		roleLine.Delta = entry.DeltaRole
		break
	}
	rank := mafiaEloRank(overall.New)
	if rating, ok := s.app.store.MafiaPlayerRating(viewerID); ok {
		rank = rating.CurrentRank
	}
	message := ""
	if !result.IsRanked {
		message = "Рейтинг не изменился: партия была без рейтинга."
	}
	return &wardenweb.MafiaMiniAppFinalResult{
		GameID:      room.ID,
		WinningSide: string(winningSide),
		WinnerLabel: mafiaMiniWinnerText(string(winningSide)),
		Role:        string(player.Role),
		RoleLabel:   mafiaMiniRoleLabel(player.Role),
		PlayerWon:   side == winningSide,
		Ranked:      result.IsRanked,
		Rank:        rank,
		Overall:     overall,
		RoleElo:     roleLine,
		Message:     message,
	}
}
