package main

import (
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func (a *App) callbackMafiaInviteUser(c tele.Context, data string) error {
	parts := strings.Split(data, ":")
	if len(parts) != 4 {
		return c.Send("⛔ Устаревшая кнопка.")
	}
	roomID := parts[2]
	targetID := mustParseInt64(parts[3])
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	if err := a.sendDirectMafiaInvite(room, sender.ID, targetID); err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	return c.Send("✅ Приглашение отправлено.", mafiaInviteMenuMarkup(room))
}

func (a *App) callbackMafiaFavoriteAdd(c tele.Context, data string) error {
	parts := strings.Split(data, ":")
	if len(parts) != 4 {
		return c.Send("⛔ Устаревшая кнопка.")
	}
	roomID := parts[2]
	targetID := mustParseInt64(parts[3])
	if _, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby}); !ok {
		return nil
	} else if err := a.mafiaService().AddFavorite(sender.ID, targetID); err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	return c.Send("⭐ Игрок добавлен в избранные.")
}

func (a *App) callbackMafiaPlayerSearch(c tele.Context, roomID string) error {
	if _, _, _, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby}); !ok {
		return nil
	}
	return c.Send("🔎 Найти игрока\n\nВведите команду с именем или username:\n/mafia_find " + normalizeMafiaRoomID(roomID) + " запрос")
}

func (a *App) callbackMafiaStartVoting(c tele.Context, roomID string) error {
	baseRoom, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusDayDiscussion})
	if !ok {
		return nil
	}
	return c.Send("🎭 Лайя ведёт партию сама.\n\nГолосование начнётся автоматически, когда закончится время обсуждения.", a.mafiaMenuMarkup(c, mafiaRoomMarkup(baseRoom, sender.ID), sender.ID))
}

func (a *App) callbackMafiaResolveNight(c tele.Context, roomID string) error {
	baseRoom, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusNight})
	if !ok {
		return nil
	}
	return c.Send("🎭 Лайя ведёт партию сама.\n\nНочь завершится автоматически: когда все действия сделаны или истечёт таймер.", a.mafiaMenuMarkup(c, mafiaRoomMarkup(baseRoom, sender.ID), sender.ID))
}

func (a *App) callbackMafiaResolveVoting(c tele.Context, roomID string) error {
	baseRoom, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusVoting})
	if !ok {
		return nil
	}
	return c.Send("🎭 Лайя ведёт партию сама.\n\nГолосование завершится автоматически: когда все проголосуют или истечёт таймер.", a.mafiaMenuMarkup(c, mafiaRoomMarkup(baseRoom, sender.ID), sender.ID))
}

func (a *App) callbackMafiaPlayers(c tele.Context, roomID string) error {
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{})
	if !ok {
		return nil
	}
	if room.OwnerID != sender.ID && mafiaRoomPlayerIndex(room, sender.ID) < 0 {
		return c.Send("⛔ Вы не состоите в этой комнате.")
	}
	lines := []string{"👥 Игроки " + room.ID + ":"}
	labels := mafiaRoomPlayerLabels(room)
	for _, player := range room.Players {
		status := "жив"
		if !player.Alive {
			status = "мертв"
		}
		label := labels[player.TelegramID]
		if player.IsTestBot {
			label += " 🤖"
		} else if room.OwnerID == sender.ID && strings.TrimSpace(player.Username) != "" {
			label = fmt.Sprintf("%s — @%s", label, player.Username)
		}
		lines = append(lines, fmt.Sprintf("• %s — %s", label, status))
	}
	return c.Send(strings.Join(lines, "\n"))
}

func (a *App) callbackMafiaAlivePlayers(c tele.Context, roomID string) error {
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{})
	if !ok {
		return nil
	}
	if room.OwnerID != sender.ID && mafiaRoomPlayerIndex(room, sender.ID) < 0 {
		return c.Send("⛔ Вы не состоите в этой комнате.")
	}
	lines := []string{"👥 Живые игроки " + room.ID + ":"}
	labels := mafiaRoomPlayerLabels(room)
	for _, player := range room.Players {
		if player.Alive {
			lines = append(lines, "• "+labels[player.TelegramID])
		}
	}
	return c.Send(strings.Join(lines, "\n"))
}

func (a *App) callbackMafiaPlayerProfiles(c tele.Context, roomID string) error {
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{})
	if !ok {
		return nil
	}
	if room.OwnerID != sender.ID && mafiaRoomPlayerIndex(room, sender.ID) < 0 {
		return c.Send("⛔ Вы не состоите в этой комнате.")
	}
	labels := mafiaRoomPlayerLabels(room)
	lines := []string{"👤 Профили игроков " + room.ID + ":"}
	for _, player := range room.Players {
		backstory := strings.TrimSpace(player.Backstory)
		if backstory == "" {
			backstory = "Предыстория не указана."
		}
		lines = append(lines, "", "👤 "+labels[player.TelegramID], backstory)
	}
	return c.Send(strings.Join(lines, "\n"))
}

func (a *App) callbackMafiaShadow(c tele.Context, roomID string) error {
	room, player, _, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresPlayer: true, RequiredRole: MafiaRoleMafia})
	if !ok {
		return nil
	}
	if !player.Alive && !normalizeMafiaSettings(room.Settings).DeadMafiaCanReadChat {
		return c.Send("⚰️ Доступ закрыт.\n\nМёртвые не говорят из тени.")
	}
	settings := normalizeMafiaSettings(room.Settings)
	if settings.MafiaChatMode == MafiaChatDisabled || (settings.MafiaChatMode == MafiaChatNightOnly && room.Status != MafiaStatusNight) {
		return c.Send("💬 Теневой канал доступен только ночью.")
	}
	lines := []string{"💬 Теневой канал комнаты " + room.ID, "", "Последние сообщения:"}
	start := len(room.MafiaPrivateChat) - 8
	if start < 0 {
		start = 0
	}
	if len(room.MafiaPrivateChat) == 0 {
		lines = append(lines, "Пока пусто.")
	}
	for _, msg := range room.MafiaPrivateChat[start:] {
		lines = append(lines, fmt.Sprintf("%s: %s", msg.SenderAlias, msg.Text))
	}
	lines = append(lines, "", "Напиши: /mafia_chat текст сообщения")
	return c.Send(strings.Join(lines, "\n"))
}

func (a *App) callbackMafiaStatusRoom(c tele.Context, roomID string) error {
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{})
	if !ok {
		return nil
	}
	if mafiaRoomPlayerIndex(room, sender.ID) < 0 && room.OwnerID != sender.ID {
		return c.Send("⛔ Вы не состоите в этой комнате.")
	}
	return c.Send(a.renderMafiaGameStatus(room, sender.ID), a.mafiaMenuMarkup(c, mafiaRoomMarkup(room, sender.ID), sender.ID))
}

func (a *App) callbackMafiaRole(c tele.Context, roomID string) error {
	room, player, _, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresPlayer: true})
	if !ok {
		return nil
	}
	return c.Send(mafiaRoleText(*player), a.mafiaMenuMarkup(c, mafiaRoomMarkup(room, player.TelegramID), player.TelegramID))
}

func (a *App) callbackMafiaWill(c tele.Context, roomID string) error {
	room, player, _, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresPlayer: true})
	if !ok {
		return nil
	}
	if !normalizeMafiaSettings(room.Settings).EnableWills {
		return c.Send("⛔ Завещания в этой комнате выключены.")
	}
	text := "Пока пусто."
	if strings.TrimSpace(player.Will) != "" {
		text = "\"" + strings.TrimSpace(player.Will) + "\""
	}
	return c.Send("📜 Завещание\n\nТекущее:\n" + text + "\n\nЧтобы изменить, отправь:\n/will текст завещания")
}
