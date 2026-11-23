package main

import (
	"errors"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) callbackMafiaActionTargetMenu(c tele.Context, roomID, action string) error {
	requiredRole := MafiaRoleMafia
	title := "🗡 Выберите жертву:"
	prefix := "kill"
	switch action {
	case "heal":
		requiredRole = MafiaRoleDoctor
		title = "💊 Кого лечить этой ночью?"
		prefix = "heal"
	case "check":
		requiredRole = MafiaRoleDetective
		title = "🔎 Кого проверить этой ночью?"
		prefix = "check"
	}
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{
		RequiresPlayer: true,
		RequiresAlive:  true,
		RequiredRole:   requiredRole,
		RequiredPhase:  MafiaStatusNight,
	})
	if !ok {
		return nil
	}
	return c.Send(title, mafiaTargetMarkup(room, sender.ID, prefix))
}

func (a *App) callbackMafiaNightAction(c tele.Context, data, action string) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	parts := strings.Split(data, ":")
	if len(parts) != 4 {
		return c.Send("⛔ Устаревшая кнопка.")
	}
	roomID := parts[2]
	targetID := mustParseInt64(parts[3])
	requiredRole := MafiaRoleMafia
	if action == "heal" {
		requiredRole = MafiaRoleDoctor
	} else if action == "check" {
		requiredRole = MafiaRoleDetective
	}
	if _, _, _, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{
		RequiresPlayer: true,
		RequiresAlive:  true,
		RequiredRole:   requiredRole,
		RequiredPhase:  MafiaStatusNight,
	}); !ok {
		return nil
	}
	var err error
	var checked bool
	switch action {
	case "kill":
		_, err = a.mafiaService().MafiaKill(roomID, sender.ID, targetID, time.Now().UTC())
	case "heal":
		_, err = a.mafiaService().DoctorHeal(roomID, sender.ID, targetID, time.Now().UTC())
	case "check":
		_, checked, err = a.mafiaService().DetectiveCheck(roomID, sender.ID, targetID, time.Now().UTC())
	}
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	if action == "check" {
		result := "не мафия"
		if checked {
			result = "мафия"
		}
		return c.Send("🔎 Результат проверки:\n\n" + mafiaPlayerNameByID(roomID, targetID, a.store) + " — " + result + ".")
	}
	return c.Send("✅ Действие принято.")
}

func (a *App) handleMafiaVoteCallbackPress(c tele.Context, data string) bool {
	sender := c.Sender()
	if sender == nil {
		a.respondMafiaCallback(c, a.t("no_sender"), true)
		return true
	}
	payload, ok := parseMafiaVoteCallbackData(data)
	if !ok {
		a.respondMafiaCallback(c, "⛔ Устаревшая кнопка голосования.", true)
		return true
	}
	if a == nil || a.store == nil {
		a.respondMafiaCallback(c, "⛔ Хранилище игры сейчас недоступно.", true)
		return true
	}
	if _, ok := a.store.MafiaUser(sender.ID); !ok {
		a.respondMafiaCallback(c, "⛔ Сначала зарегистрируйтесь у Лайи в ЛС через /start.", true)
		return true
	}
	room, ok := a.store.MafiaRoom(payload.roomID)
	if !ok {
		a.respondMafiaCallback(c, "⛔ Комната не найдена.", true)
		return true
	}
	if room.Status != MafiaStatusVoting {
		a.respondMafiaCallback(c, "⛔ Сейчас не фаза голосования.", true)
		return true
	}
	if room.PendingLeakOffenderID != 0 {
		a.respondMafiaCallback(c, "⚠️ Партия приостановлена. Сначала нужно решение создателя комнаты.", true)
		return true
	}
	idx := mafiaRoomPlayerIndex(room, sender.ID)
	if idx < 0 {
		a.respondMafiaCallback(c, "⛔ Вы не состоите в этой комнате.", true)
		return true
	}
	if !room.Players[idx].Alive {
		a.respondMafiaCallback(c, "⚰️ Мёртвые не голосуют.", true)
		return true
	}
	if payload.phaseKey != "" && payload.phaseKey != mafiaVotePhaseKey(room) {
		a.respondMafiaCallback(c, "⚠️ Эта кнопка голосования уже устарела. Откройте актуальное голосование через статус игры.", true)
		return true
	}

	now := time.Now().UTC()
	updated, err := a.mafiaService().Vote(payload.roomID, sender.ID, payload.target, now)
	if err != nil {
		if errors.Is(err, errMafiaVoteExists) {
			a.respondMafiaCallback(c, "⚠️ Ты уже голосовал.", true)
			return true
		}
		a.respondMafiaCallback(c, "⛔ "+a.localizeMafiaErr(err), true)
		return true
	}

	ready := mafiaVotingReadyForLaya(updated)
	responseText := "✅ Голос принят."
	if ready {
		responseText = "✅ Голос принят. Все проголосовали, Лайя завершает голосование."
	}
	a.respondMafiaCallback(c, responseText, false)
	a.sendMafiaGroup(updated, mafiaPhaseMessage("ГОЛОСОВАНИЕ", mafiaVotingProgressText(updated, updated.Votes)), nil)
	if ready {
		if err := a.advanceMafiaRoomAsLaya(updated.ID, now); err != nil {
			_ = c.Send("⛔ " + a.localizeMafiaErr(err))
		}
	}
	return true
}
