package main

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) sendMafiaDM(userID int64, text string, markup *tele.ReplyMarkup) {
	if a == nil || a.bot == nil || userID <= 0 || strings.TrimSpace(text) == "" {
		return
	}
	user := &tele.User{ID: userID}
	if markup != nil {
		_, _ = a.bot.Send(user, text, markup)
		return
	}
	_, _ = a.bot.Send(user, text)
}

func (a *App) notifyMafiaPlayersMissingGroup(room MafiaRoom) {
	for _, player := range room.Players {
		a.sendMafiaGroupAccessIfMissing(player.TelegramID, room)
	}
}

func (a *App) sendMafiaGroupAccessIfMissing(userID int64, room MafiaRoom) {
	if a == nil || a.bot == nil || userID <= 0 || room.GroupChatID == 0 {
		return
	}
	if a.mafiaUserInGameGroup(room, userID) {
		return
	}
	a.sendMafiaDM(userID, a.mafiaGroupAccessText(room), nil)
}

func (a *App) mafiaUserInGameGroup(room MafiaRoom, userID int64) bool {
	if a == nil || a.bot == nil || userID <= 0 || room.GroupChatID == 0 {
		return true
	}
	member, err := a.bot.ChatMemberOf(&tele.Chat{ID: room.GroupChatID}, &tele.User{ID: userID})
	if err != nil {
		if a.log != nil {
			a.log.Warn("check mafia group member failed", zap.Int64("chat", room.GroupChatID), zap.Int64("user", userID), zap.Error(err))
		}
		return false
	}
	if member == nil {
		return false
	}
	switch member.Role {
	case tele.Left, tele.Kicked:
		return false
	case tele.Restricted:
		return member.Member
	default:
		return true
	}
}

func (a *App) mafiaGroupAccessText(room MafiaRoom) string {
	title, link := a.mafiaGroupTitleAndLink(room)
	titleLine := ""
	if strings.TrimSpace(title) != "" {
		titleLine = "\nЧат: " + title
	}
	if strings.TrimSpace(link) != "" {
		return fmt.Sprintf("📍 Игра проходит в привязанном чате.%s\n\n%s\n\nДневное обсуждение и публичные голосования будут там.", titleLine, link)
	}
	return fmt.Sprintf("📍 Игра проходит в привязанном чате.%s\n\nЛайя не смогла получить ссылку автоматически. Попросите создателя комнаты добавить вас в группу.", titleLine)
}

func (a *App) mafiaGroupTitleAndLink(room MafiaRoom) (string, string) {
	if a == nil || a.bot == nil || room.GroupChatID == 0 {
		return "", ""
	}
	chat := &tele.Chat{ID: room.GroupChatID}
	title := ""
	link := ""
	if info, err := a.bot.ChatByID(room.GroupChatID); err == nil && info != nil {
		title = strings.TrimSpace(info.Title)
		if strings.TrimSpace(info.Username) != "" {
			link = "https://t.me/" + strings.TrimPrefix(strings.TrimSpace(info.Username), "@")
		}
		if link == "" && strings.TrimSpace(info.InviteLink) != "" {
			link = strings.TrimSpace(info.InviteLink)
		}
	} else if err != nil && a.log != nil {
		a.log.Warn("fetch mafia group info failed", zap.Int64("chat", room.GroupChatID), zap.Error(err))
	}
	if link == "" {
		if invite, err := a.bot.InviteLink(chat); err == nil {
			link = strings.TrimSpace(invite)
		} else if a.log != nil {
			a.log.Warn("export mafia group invite link failed", zap.Int64("chat", room.GroupChatID), zap.Error(err))
		}
	}
	if link == "" {
		if invite, err := a.bot.CreateInviteLink(chat, &tele.ChatInviteLink{Name: "MafiaGame " + room.ID}); err == nil && invite != nil {
			link = strings.TrimSpace(invite.InviteLink)
		} else if err != nil && a.log != nil {
			a.log.Warn("create mafia group invite link failed", zap.Int64("chat", room.GroupChatID), zap.Error(err))
		}
	}
	return title, link
}

func mafiaRoleText(player MafiaPlayer) string {
	switch player.Role {
	case MafiaRoleMafia:
		return "🎭 Твоя роль: Мафия.\n\nТы просыпаешься ночью, выбираешь жертву и можешь писать в теневой канал.\nНе раскрывай роль напрямую.\n\nТвой псевдоним в теневом канале: " + player.MafiaAlias
	case MafiaRoleDoctor:
		return "🎭 Твоя роль: Доктор.\n\nКаждую ночь ты можешь выбрать одного игрока для лечения."
	case MafiaRoleDetective:
		return "🎭 Твоя роль: Детектив.\n\nКаждую ночь ты можешь проверить одного игрока."
	default:
		return "🎭 Твоя роль: Мирный житель.\n\nДнём обсуждай и голосуй. Роль напрямую не раскрывай."
	}
}

func mafiaNightResultText(room MafiaRoom, result MafiaNightResult) string {
	if result.Saved {
		return "☀️ Утро наступило.\n\nНочью кто-то должен был умереть. Но смерть прошла мимо."
	}
	if result.KilledID != 0 {
		return mafiaNightKillResultText(room, result.KilledID)
	}
	return "☀️ Утро наступило.\n\nНочь прошла тихо. Слишком тихо."
}

func mafiaNightKillResultText(room MafiaRoom, killedID int64) string {
	playerName := mafiaPlayerPublicName(room, killedID)
	template := strings.TrimSpace(normalizeMafiaSettings(room.Settings).NightKillMessage)
	if template == "" {
		template = defaultMafiaNightKillMessage()
	}
	replacer := strings.NewReplacer(
		"{player}", playerName,
		"{name}", playerName,
		"{room}", room.ID,
		"{day}", fmt.Sprint(room.DayNumber),
	)
	return replacer.Replace(template)
}

func mafiaAlivePlayersCount(room MafiaRoom) int {
	count := 0
	for _, player := range room.Players {
		if player.Alive {
			count++
		}
	}
	return count
}

func mafiaRosterText(room MafiaRoom) string {
	alive := make([]string, 0, len(room.Players))
	dead := make([]string, 0, len(room.Players))
	for _, player := range room.Players {
		label := mafiaRoomPlayerLabelWithUsername(room, player)
		if player.Alive {
			alive = append(alive, label)
		} else {
			dead = append(dead, label)
		}
	}
	lines := []string{
		fmt.Sprintf("👥 Живые игроки (%d):", len(alive)),
		mafiaRosterNames(alive),
	}
	if len(dead) > 0 {
		lines = append(lines, "", fmt.Sprintf("⚰️ Выбыли (%d):", len(dead)), mafiaRosterNames(dead))
	}
	return strings.Join(lines, "\n")
}

func mafiaRosterNames(names []string) string {
	if len(names) == 0 {
		return "— никого"
	}
	lines := make([]string, 0, len(names))
	for i, name := range names {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, name))
	}
	return strings.Join(lines, "\n")
}
