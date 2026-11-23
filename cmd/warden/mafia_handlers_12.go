package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) callbackMafiaVote(c tele.Context, data string) error {
	if a.handleMafiaVoteCallbackPress(c, data) {
		return nil
	}
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	parts := strings.Split(data, ":")
	if len(parts) < 3 {
		return c.Send("⛔ Устаревшая кнопка.")
	}
	roomID := parts[2]
	target := "skip"
	buttonPhaseKey := ""
	if strings.HasPrefix(data, "mafia:vote:") {
		switch len(parts) {
		case 4:
			target = parts[3]
		case 5:
			buttonPhaseKey = parts[3]
			target = parts[4]
		default:
			return c.Send("⛔ Устаревшая кнопка.")
		}
	} else {
		switch len(parts) {
		case 3:
		case 4:
			buttonPhaseKey = parts[3]
		default:
			return c.Send("⛔ Устаревшая кнопка.")
		}
	}
	room, _, _, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{
		RequiresPlayer: true,
		RequiresAlive:  true,
		RequiredPhase:  MafiaStatusVoting,
	})
	if !ok {
		return nil
	}
	if buttonPhaseKey != "" && buttonPhaseKey != mafiaVotePhaseKey(room) {
		return c.Send("⚠️ Эта кнопка голосования уже устарела.\n\nОткройте актуальное голосование через меню Мафии или нажмите «📊 Статус игры».")
	}
	now := time.Now().UTC()
	updated, err := a.mafiaService().Vote(roomID, sender.ID, target, now)
	if err != nil {
		if errors.Is(err, errMafiaVoteExists) {
			return c.Send("⚠️ Ты уже голосовал.")
		}
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	a.sendMafiaGroup(updated, mafiaPhaseMessage("ГОЛОСОВАНИЕ", mafiaVotingProgressText(updated, updated.Votes)), nil)
	if mafiaVotingReadyForLaya(updated) {
		if err := a.advanceMafiaRoomAsLaya(updated.ID, now); err != nil {
			return c.Send("⛔ " + a.localizeMafiaErr(err))
		}
		return c.Send("✅ Голос принят. Все проголосовали, Лайя завершает голосование.")
	}
	return c.Send("✅ Голос принят.")
}

func (a *App) callbackMafiaSkipPhase(c tele.Context, data string) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	parts := strings.Split(data, ":")
	roomID := strings.TrimSpace(data)
	var buttonPhase MafiaRoomStatus
	if len(parts) >= 1 {
		roomID = parts[0]
	}
	if len(parts) >= 2 {
		buttonPhase = MafiaRoomStatus(parts[1])
	}
	room, _, _, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{
		RequiresPlayer: true,
		RequiresAlive:  true,
	})
	if !ok {
		return nil
	}
	if buttonPhase != "" && room.Status != buttonPhase {
		return c.Send("⚠️ Эта кнопка уже устарела. Откройте актуальное меню Мафии.")
	}
	if !mafiaAutoHostStatus(room.Status) {
		return c.Send("⛔ Сейчас эту фазу нельзя скипнуть.")
	}
	now := time.Now().UTC()
	updated, votes, needed, reached, err := a.mafiaService().VotePhaseSkip(room.ID, sender.ID)
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	if reached {
		a.sendMafiaGroup(updated, fmt.Sprintf("⏭ Голосование за скип фазы прошло.\n\nГолосов: %d/%d.\nЛайя переводит игру дальше.", votes, needed), nil)
		if err := a.advanceMafiaRoomAsLaya(updated.ID, now); err != nil {
			return c.Send("⛔ " + a.localizeMafiaErr(err))
		}
		return c.Send("⏭ Скип фазы принят. Лайя переводит игру дальше.")
	}
	return c.Send(fmt.Sprintf("⏭ Голос за скип фазы принят.\n\nГолосов: %d/%d.", votes, needed), a.mafiaMenuMarkup(c, mafiaRoomMarkup(updated, sender.ID), sender.ID))
}

func (a *App) callbackMafiaLeaveRoom(c tele.Context, roomID string) error {
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresPlayer: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	idx := mafiaRoomPlayerIndex(room, sender.ID)
	room.Players = append(room.Players[:idx], room.Players[idx+1:]...)
	room.UpdatedAt = time.Now().UTC().Unix()
	if err := a.store.SaveMafiaRoom(room); err != nil {
		return c.Send("⛔ " + err.Error())
	}
	return c.Send("🚪 Ты покинул комнату "+room.ID, a.mafiaMenuMarkup(c, nil, sender.ID))
}

func (a *App) callbackMafiaCloseRoom(c tele.Context, roomID string) error {
	room, _, _, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	if err := a.store.DeleteMafiaRoom(room.ID); err != nil {
		return c.Send("⛔ " + err.Error())
	}
	return c.Send("🗑 Комната закрыта: "+room.ID, a.mafiaMenuMarkup(c, nil, room.OwnerID))
}

func (a *App) callbackMafiaEndGame(c tele.Context, roomID string) error {
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true})
	if !ok {
		return nil
	}
	if room.Status == MafiaStatusLobby {
		return a.callbackMafiaCloseRoom(c, room.ID)
	}
	updated, err := a.mafiaService().EndGameByOwner(room.ID, sender.ID, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	a.syncMafiaGroupRestrictions(updated)
	a.sendMafiaGroup(updated, "⛔ Партия завершена создателем комнаты.\n\nСтатистика и достижения за неё не выдаются.", nil)
	return c.Send("⛔ Партия завершена создателем комнаты.\n\nСтатистика и достижения за неё не выдаются.", a.mafiaMenuMarkup(c, mafiaRoomMarkup(updated, sender.ID), sender.ID))
}

func (a *App) callbackMafiaCloseCancelledRoom(c tele.Context, roomID string) error {
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true})
	if !ok {
		return nil
	}
	if room.Status != MafiaStatusCancelled {
		return c.Send("⛔ Закрыть через эту кнопку можно только отменённую партию.")
	}
	if err := a.store.DeleteMafiaRoom(room.ID); err != nil {
		return c.Send("⛔ " + err.Error())
	}
	return c.Send("🗑 Отменённая партия закрыта: "+room.ID, a.mafiaMenuMarkup(c, nil, sender.ID))
}

func (a *App) callbackMafiaRecreateCancelledRoom(c tele.Context, roomID string) error {
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true})
	if !ok {
		return nil
	}
	if room.Status != MafiaStatusCancelled {
		return c.Send("⛔ Пересоздать можно только отменённую партию.")
	}
	next, err := a.mafiaService().RecreateCancelledRoom(room.ID, sender.ID, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	return c.Send("🔁 Новая комната создана.\n\nНастройки сохранены.\nНарушитель не добавлен.", a.mafiaMenuMarkup(c, mafiaRoomMarkup(next, sender.ID), sender.ID))
}
