package main

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) handleMafiaWill(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	room, ok := a.activeMafiaRoomForUser(sender.ID)
	if !ok {
		return c.Send("⛔ Активная комната Мафии не найдена.")
	}
	settings := normalizeMafiaSettings(room.Settings)
	if !settings.EnableWills {
		return c.Send("⛔ Завещания в этой комнате выключены.")
	}
	text := strings.TrimSpace(strings.Join(c.Args(), " "))
	if text == "" {
		return c.Send("📜 Завещание\n\nНапиши так:\n/will текст, который Лайя опубликует после твоей смерти.")
	}
	if DetectMafiaSystemProof(text).IsRoleReveal || DetectMafiaRoleReveal(text).IsRoleReveal || looksLikeMafiaRevealCandidate(text) {
		return c.Send("⛔ Завещание похоже на мета-слив роли или системной информации. Лайя такое не сохранит.")
	}
	if _, err := a.mafiaService().SaveWill(room.ID, sender.ID, text, time.Now().UTC()); err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	return c.Send("✅ Завещание сохранено.\n\nОно будет опубликовано, если ты погибнешь.")
}

func (a *App) handleMafiaFind(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	args := c.Args()
	if len(args) < 2 {
		return c.Send("🔎 Поиск игрока\n\nИспользование:\n/mafia_find MF-4821 имя или username")
	}
	roomID := normalizeMafiaRoomID(args[0])
	room, _, _, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	query := strings.TrimSpace(strings.Join(args[1:], " "))
	results := a.searchMafiaUsersForInvite(room, query, 8)
	if len(results) == 0 {
		return c.Send("🔎 Ничего не нашла.\n\nПопробуйте другое имя или username.")
	}
	lines := []string{"🔎 Найдено:"}
	markup := &tele.ReplyMarkup{}
	var rows []tele.Row
	for i, user := range results {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, mafiaRegisteredUserLabel(user)))
		shortName := mafiaShortButtonName(mafiaUserCharacterName(user))
		rows = append(rows, markup.Row(
			mafiaButton("✅ Пригласить "+shortName, "mafia:invite_user:"+room.ID+":"+idKey(user.TelegramID)),
			mafiaButton("⭐ В избранные", "mafia:favorite_add:"+room.ID+":"+idKey(user.TelegramID)),
		))
	}
	rows = append(rows, markup.Row(mafiaButton("⬅️ Назад", "mafia:invite_menu:"+room.ID)))
	markup.Inline(rows...)
	return c.Send(strings.Join(lines, "\n"), markup)
}

func (a *App) handleCallback(c tele.Context) error {
	cb := c.Callback()
	if cb == nil {
		return nil
	}
	data := strings.TrimSpace(cb.Data)
	unique := strings.TrimSpace(cb.Unique)
	if strings.HasPrefix(unique, "mafia") && !strings.HasPrefix(data, "mafia") {
		data = unique
	}
	if data == "" && unique != "" {
		data = unique
	}
	data = strings.TrimLeft(strings.TrimSpace(data), "\f")
	data = normalizeMafiaCallbackData(data)
	if strings.HasPrefix(data, "laya:") {
		if err := c.Respond(&tele.CallbackResponse{}); err != nil && a.log != nil {
			a.log.Warn("callback respond failed", zap.Error(err))
		}
		return a.handleLayaCallback(c, data)
	}
	if !strings.HasPrefix(data, "mafia:") {
		return nil
	}
	if !mafiaCallbackNeedsManualResponse(data) {
		if err := c.Respond(&tele.CallbackResponse{}); err != nil && a.log != nil {
			a.log.Warn("callback respond failed", zap.Error(err))
		}
	}
	return a.routeMafiaCallback(c, data)
}
