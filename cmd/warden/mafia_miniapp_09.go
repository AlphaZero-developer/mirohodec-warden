package main

import (
	"fmt"
	wardenweb "mirohodec-warden/internal/web"
	"sort"
	"strings"
	"time"
)

func mafiaMiniAchievements(unlocked []string) []wardenweb.MafiaMiniAppAchievement {
	seen := map[string]bool{}
	for _, id := range unlocked {
		seen[strings.TrimSpace(id)] = true
	}
	ids := mafiaAllAchievementIDs()
	out := make([]wardenweb.MafiaMiniAppAchievement, 0, len(ids))
	for _, id := range ids {
		title, description := mafiaMiniAchievementParts(id)
		out = append(out, wardenweb.MafiaMiniAppAchievement{
			ID:          id,
			Title:       title,
			Description: description,
			Unlocked:    seen[id],
		})
	}
	return out
}

func mafiaMiniAchievementParts(id string) (string, string) {
	label := mafiaAchievementLabel(id)
	parts := strings.SplitN(label, " — ", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return label, ""
}

func (s webPanelService) mafiaMiniHistory(userID int64) []wardenweb.MafiaMiniAppHistoryEntry {
	rooms := s.app.store.MafiaRoomsForUser(userID)
	results := s.app.store.MafiaGameResultsForUser(userID)
	out := make([]wardenweb.MafiaMiniAppHistoryEntry, 0, len(rooms)+len(results))
	seen := map[string]bool{}
	for _, result := range results {
		out = append(out, mafiaMiniHistoryFromResult(result, userID))
		seen[result.RoomID] = true
		seen[result.ID] = true
	}
	for _, room := range rooms {
		if room.Status != MafiaStatusFinished && room.Status != MafiaStatusCancelled {
			continue
		}
		if seen[room.ID] {
			continue
		}
		idx := mafiaRoomPlayerIndex(room, userID)
		role := MafiaRole("")
		if idx >= 0 {
			role = room.Players[idx].Role
		}
		date := room.UpdatedAt
		if date == 0 {
			date = room.CreatedAt
		}
		result := mafiaMiniWinnerText(room.Winner)
		if room.Status == MafiaStatusCancelled {
			result = "Партия отменена."
		}
		out = append(out, wardenweb.MafiaMiniAppHistoryEntry{
			ID:           room.ID,
			Date:         date,
			Result:       result,
			Role:         string(role),
			RoleLabel:    mafiaMiniRoleLabel(role),
			Winner:       room.Winner,
			Players:      len(room.Players),
			DurationText: mafiaMiniDurationText(room.CreatedAt, date),
			IsRanked:     normalizeMafiaSettings(room.Settings).RatingMode != MafiaRatingUnranked,
			WinningSide:  room.Winner,
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Date > out[j].Date })
	if len(out) > 30 {
		return out[:30]
	}
	return out
}

func mafiaMiniHistoryFromResult(result MafiaGameResult, userID int64) wardenweb.MafiaMiniAppHistoryEntry {
	role := MafiaRole("")
	players := len(result.PlayersSnapshot)
	eloDelta := 0
	for _, player := range result.PlayersSnapshot {
		if player.UserID == userID {
			role = player.Role
			break
		}
	}
	for _, entry := range result.EloHistory {
		if entry.UserID == userID {
			eloDelta = entry.DeltaOverall
			break
		}
	}
	resultText := mafiaMiniWinnerText(string(result.WinningSide))
	return wardenweb.MafiaMiniAppHistoryEntry{
		ID:           result.ID,
		Date:         result.FinishedAt,
		Result:       resultText,
		Role:         string(role),
		RoleLabel:    mafiaMiniRoleLabel(role),
		Winner:       string(result.WinningSide),
		Players:      players,
		DurationText: mafiaMiniDurationSecondsText(result.DurationSeconds),
		IsRanked:     result.IsRanked,
		WinningSide:  string(result.WinningSide),
		EloDelta:     eloDelta,
	}
}

func mafiaMiniFavoriteRole(stats MafiaStats) string {
	switch {
	case stats.MafiaWins >= stats.DoctorWins && stats.MafiaWins >= stats.DetectiveWins && stats.MafiaWins > 0:
		return "Мафия"
	case stats.DoctorWins >= stats.DetectiveWins && stats.DoctorWins > 0:
		return "Доктор"
	case stats.DetectiveWins > 0:
		return "Детектив"
	default:
		return ""
	}
}

func mafiaMiniDurationText(start, end int64) string {
	if start <= 0 || end <= start {
		return ""
	}
	return mafiaMiniDurationSecondsText(end - start)
}

func mafiaMiniDurationSecondsText(seconds int64) string {
	if seconds <= 0 {
		return ""
	}
	minutes := int(time.Duration(seconds) * time.Second / time.Minute)
	if minutes <= 0 {
		return "меньше минуты"
	}
	if minutes < 60 {
		return fmt.Sprintf("%d мин.", minutes)
	}
	return fmt.Sprintf("%d ч. %d мин.", minutes/60, minutes%60)
}

func mafiaMiniCanWriteChannel(room MafiaRoom, viewerIdx int, isHost bool, channel string) bool {
	if isHost && (channel == mafiaMiniChannelHost || channel == mafiaMiniChannelPublic || channel == mafiaMiniChannelDead) {
		return true
	}
	if viewerIdx < 0 {
		return false
	}
	player := room.Players[viewerIdx]
	switch channel {
	case mafiaMiniChannelPublic:
		ok, _ := mafiaMiniCanWritePublic(room, player, viewerIdx, isHost)
		return ok
	case mafiaMiniChannelDead:
		return !player.Alive
	default:
		return false
	}
}

func mafiaMiniCanWritePublic(room MafiaRoom, player MafiaPlayer, viewerIdx int, isHost bool) (bool, string) {
	if isHost {
		return true, ""
	}
	if viewerIdx < 0 {
		return false, "Ты не участник партии."
	}
	if !player.Alive {
		return false, "Мёртвые читают общий чат без права писать."
	}
	switch room.Status {
	case MafiaStatusLobby, MafiaStatusDayDiscussion, MafiaStatusVoting:
		return true, ""
	case MafiaStatusNight, MafiaStatusRoleDistribution:
		return false, "Сейчас ночь. Общий чат молчит."
	default:
		return false, "В этой фазе писать нельзя."
	}
}
