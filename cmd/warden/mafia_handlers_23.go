package main

import (
	"fmt"
	"strings"
)

func mafiaPublishedWillText(room MafiaRoom, playerID int64) string {
	if playerID == 0 || !normalizeMafiaSettings(room.Settings).EnableWills {
		return ""
	}
	idx := mafiaRoomPlayerIndex(room, playerID)
	if idx < 0 {
		return ""
	}
	will := strings.TrimSpace(room.Players[idx].Will)
	if will == "" {
		return ""
	}
	if DetectMafiaSystemProof(will).IsRoleReveal || DetectMafiaRoleReveal(will).IsRoleReveal || looksLikeMafiaRevealCandidate(will) {
		return ""
	}
	return fmt.Sprintf("📜 Завещание игрока %s:\n\n%q", mafiaRoomPlayerLabel(room, room.Players[idx]), will)
}

func mafiaPostGameAchievementsText(room MafiaRoom) string {
	if room.Status != MafiaStatusFinished || !normalizeMafiaSettings(room.Settings).EnableAchievements {
		return ""
	}
	lines := []string{"🏆 Достижения после партии:"}
	for _, player := range room.Players {
		achievements := mafiaAchievementsForPlayer(player, room.Winner)
		if len(achievements) == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s — %s", mafiaRoomPlayerLabel(room, player), strings.Join(achievements, ", ")))
	}
	if len(lines) == 1 {
		return ""
	}
	return mafiaPhaseMessage("ДОСТИЖЕНИЯ", strings.Join(lines, "\n"))
}

func mafiaAchievementsForPlayer(player MafiaPlayer, winner string) []string {
	var achievements []string
	achievements = append(achievements, "Первая партия")
	if isMafiaTeam(player.Role) && winner == "mafia" {
		achievements = append(achievements, "Первый нож")
	}
	if player.Role == MafiaRoleDetective && winner == "civilian" {
		achievements = append(achievements, "Точная проверка")
	}
	if player.Role == MafiaRoleDoctor && winner == "civilian" {
		achievements = append(achievements, "Ночная смена")
	}
	if strings.TrimSpace(player.Will) != "" {
		achievements = append(achievements, "Последняя мысль")
	}
	if player.Role == MafiaRoleCivilian && !player.Alive && winner == "mafia" {
		achievements = append(achievements, "Ошибка города")
	}
	if winner == "mafia" && isMafiaTeam(player.Role) && player.Alive {
		achievements = append(achievements, "Маска не упала")
	}
	if winner == "civilian" && !isMafiaTeam(player.Role) && player.Alive {
		achievements = append(achievements, "Город стоит")
	}
	if ((winner == "mafia" && isMafiaTeam(player.Role)) || (winner == "civilian" && !isMafiaTeam(player.Role))) && !player.Alive {
		achievements = append(achievements, "Победа после смерти")
	}
	return achievements
}

func mafiaRoleRevealPunishmentText(room MafiaRoom, userID int64) string {
	idx := mafiaRoomPlayerIndex(room, userID)
	name := fmt.Sprintf("user %d", userID)
	alive := false
	warned := false
	if idx >= 0 {
		name = mafiaRoomPlayerLabel(room, room.Players[idx])
		alive = room.Players[idx].Alive
		warned = room.Players[idx].WarnedForReveal
	}
	if alive && warned {
		return "⚠️ Лайя заметила возможное раскрытие роли.\n\nИгрок " + name + " получает предупреждение. Повторное нарушение приведёт к выбыванию."
	}
	return "⚖️ Лайя зафиксировала раскрытие роли.\n\nИгрок " + name + " выбывает из игры."
}

func mafiaRolesRevealText(room MafiaRoom) string {
	lines := []string{"🎭 Роли игроков:"}
	for _, player := range room.Players {
		lines = append(lines, fmt.Sprintf("%s — %s", mafiaRoomPlayerLabel(room, player), mafiaRoleLabel(player.Role)))
	}
	return strings.Join(lines, "\n")
}

func mafiaRoleLabel(role MafiaRole) string {
	switch role {
	case MafiaRoleMafia:
		return "Мафия"
	case MafiaRoleDoctor:
		return "Доктор"
	case MafiaRoleDetective:
		return "Детектив"
	case MafiaRoleDon:
		return "Дон"
	case MafiaRoleSergeant:
		return "Сержант"
	case MafiaRoleBodyguard:
		return "Телохранитель"
	case MafiaRoleBlocker:
		return "Блокер"
	case MafiaRoleLawyer:
		return "Адвокат"
	case MafiaRoleJournalist:
		return "Журналист"
	case MafiaRoleMayor:
		return "Мэр"
	default:
		return "Мирный житель"
	}
}

func mafiaPlayerName(room MafiaRoom, userID int64) string {
	if idx := mafiaRoomPlayerIndex(room, userID); idx >= 0 {
		return mafiaRoomPlayerLabel(room, room.Players[idx])
	}
	return fmt.Sprintf("user %d", userID)
}

func mafiaPlayerPublicName(room MafiaRoom, userID int64) string {
	if idx := mafiaRoomPlayerIndex(room, userID); idx >= 0 {
		return mafiaRoomPlayerLabelWithUsername(room, room.Players[idx])
	}
	return fmt.Sprintf("user %d", userID)
}

func mafiaRoomPlayerLabel(room MafiaRoom, player MafiaPlayer) string {
	labels := mafiaRoomPlayerLabels(room)
	if label := strings.TrimSpace(labels[player.TelegramID]); label != "" {
		return label
	}
	return mafiaPlayerLabel(player)
}

func mafiaRoomPlayerLabelWithUsername(room MafiaRoom, player MafiaPlayer) string {
	label := mafiaRoomPlayerLabel(room, player)
	username := strings.TrimSpace(player.Username)
	if username == "" || player.IsTestBot {
		return label
	}
	username = strings.TrimPrefix(username, "@")
	if username == "" {
		return label
	}
	return fmt.Sprintf("%s(@%s)", label, username)
}

func mafiaRoomPlayerLabels(room MafiaRoom) map[int64]string {
	counts := make(map[string]int, len(room.Players))
	for _, player := range room.Players {
		counts[mafiaPlayerLabel(player)]++
	}
	seen := make(map[string]int, len(counts))
	labels := make(map[int64]string, len(room.Players))
	for _, player := range room.Players {
		base := mafiaPlayerLabel(player)
		seen[base]++
		label := base
		if counts[base] > 1 && seen[base] > 1 {
			label = fmt.Sprintf("%s #%d", base, seen[base])
		}
		labels[player.TelegramID] = label
	}
	return labels
}

func mafiaPlayerNameByID(roomID string, userID int64, store RuntimeStore) string {
	if store != nil {
		if room, ok := store.MafiaRoom(roomID); ok {
			return mafiaPlayerName(room, userID)
		}
	}
	return fmt.Sprintf("user %d", userID)
}
