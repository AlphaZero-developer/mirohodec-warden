package main

import (
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) callbackMafiaBlockCancelledOffender(c tele.Context, roomID string) error {
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true})
	if !ok {
		return nil
	}
	if room.CancelledBy == 0 {
		return c.Send("⛔ Нарушитель не найден.")
	}
	if err := a.store.AddMafiaHostBlacklist(sender.ID, room.CancelledBy, "критический слив в партии "+room.ID, time.Now().UTC()); err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	return c.Send("🚫 Игрок заблокирован для ваших комнат.")
}

func (a *App) callbackMafiaLeakCancel(c tele.Context, roomID string) error {
	room, _, _, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true})
	if !ok {
		return nil
	}
	reason := room.PendingLeakType
	if reason == "" {
		reason = MafiaRuleViolationGameBreakingLeak
	}
	offenderID := room.PendingLeakOffenderID
	if offenderID == 0 {
		return c.Send("⛔ Нет ожидающего решения по сливу.")
	}
	updated, err := a.mafiaService().CancelGameForCriticalLeak(room.ID, offenderID, reason, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	a.syncMafiaGroupRestrictions(updated)
	a.sendMafiaGroup(updated, mafiaCriticalLeakCancelText(), nil)
	return c.Send("⛔ Партия отменена.", mafiaCancelledOwnerMarkup(updated))
}

func (a *App) callbackMafiaLeakKill(c tele.Context, roomID string) error {
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true})
	if !ok {
		return nil
	}
	updated, err := a.mafiaService().KillOffenderForCriticalLeak(room.ID, sender.ID, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	a.syncMafiaGroupRestrictions(updated)
	a.sendMafiaGroup(updated, "☠️ Нарушитель выбывает из партии.\n\nИгра продолжается по решению создателя комнаты.", nil)
	if updated.Status == MafiaStatusFinished {
		a.announceMafiaFinished(updated)
	}
	return c.Send(a.renderMafiaGameStatus(updated, sender.ID), a.mafiaMenuMarkup(c, mafiaRoomMarkup(updated, sender.ID), sender.ID))
}

func (a *App) callbackMafiaLeakContinue(c tele.Context, roomID string) error {
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true})
	if !ok {
		return nil
	}
	updated, err := a.mafiaService().ClearCriticalLeakPending(room.ID, sender.ID, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	a.syncMafiaGroupRestrictions(updated)
	a.sendMafiaGroup(updated, "✅ Игра продолжается по решению создателя комнаты.", nil)
	return c.Send(a.renderMafiaGameStatus(updated, sender.ID), a.mafiaMenuMarkup(c, mafiaRoomMarkup(updated, sender.ID), sender.ID))
}

func (a *App) callbackMafiaSettings(c tele.Context, roomID string) error {
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true})
	if !ok {
		return nil
	}
	if room.Status != MafiaStatusLobby {
		return c.Send("⚙️ Настройки комнаты\n\nИгра уже началась.\nКритические настройки нельзя менять во время партии.")
	}
	return c.Send(mafiaSettingsText(room), mafiaSettingsMarkup(room, sender.ID))
}

func (a *App) callbackMafiaSettingsTime(c tele.Context, roomID string) error {
	room, _, _, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	return c.Send(mafiaTimeSettingsText(room), mafiaTimeSettingsMarkup(room))
}

func (a *App) callbackMafiaSettingsPlayers(c tele.Context, roomID string) error {
	room, _, _, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	return c.Send(mafiaPlayerSettingsText(room), mafiaPlayerSettingsMarkup(room))
}

func (a *App) callbackMafiaSettingsRoles(c tele.Context, roomID string) error {
	room, _, _, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	return c.Send(mafiaRoleSettingsText(), mafiaRoleSettingsMarkup(room))
}

func (a *App) callbackMafiaSettingsVoting(c tele.Context, roomID string) error {
	room, _, _, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	return c.Send(mafiaVotingSettingsText(room), mafiaVotingSettingsMarkup(room))
}

func (a *App) callbackMafiaSettingsReveal(c tele.Context, roomID string) error {
	room, _, _, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	return c.Send(mafiaRevealSettingsText(room), mafiaRevealSettingsMarkup(room))
}

func (a *App) callbackMafiaTimeAdjust(c tele.Context, data string) error {
	parts := strings.Split(data, ":")
	if len(parts) != 5 {
		return c.Send("⛔ Устаревшая кнопка.")
	}
	roomID, field, op := parts[2], parts[3], parts[4]
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	updated, err := a.mafiaService().UpdateLobbySettings(room.ID, sender.ID, time.Now().UTC(), func(settings MafiaRoomSettings) MafiaRoomSettings {
		switch field {
		case "preset":
			settings = mafiaApplyTimePreset(settings, op)
		case "day":
			settings.DayDiscussionSeconds = clampInt(settings.DayDiscussionSeconds+mustParseInt(op), 60, 600)
		case "night":
			settings.NightActionSeconds = clampInt(settings.NightActionSeconds+mustParseInt(op), 30, 300)
		case "vote":
			settings.VotingSeconds = clampInt(settings.VotingSeconds+mustParseInt(op), 30, 180)
		}
		return settings
	})
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	return c.Send(mafiaTimeSettingsText(updated), mafiaTimeSettingsMarkup(updated))
}

func (a *App) callbackMafiaPlayerLimitAdjust(c tele.Context, data string) error {
	parts := strings.Split(data, ":")
	if len(parts) != 5 {
		return c.Send("⛔ Устаревшая кнопка.")
	}
	roomID, field, op := parts[2], parts[3], parts[4]
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	updated, err := a.mafiaService().UpdateLobbySettings(room.ID, sender.ID, time.Now().UTC(), func(settings MafiaRoomSettings) MafiaRoomSettings {
		delta := mustParseInt(op)
		switch field {
		case "min":
			settings.MinPlayers = clampInt(settings.MinPlayers+delta, mafiaMinPlayersLimit, settings.MaxPlayers)
		case "max":
			minMaxPlayers := mafiaMinPlayersLimit
			if len(room.Players) > minMaxPlayers {
				minMaxPlayers = len(room.Players)
			}
			settings.MaxPlayers = clampInt(settings.MaxPlayers+delta, minMaxPlayers, mafiaMaxPlayersLimit)
			if settings.MinPlayers > settings.MaxPlayers {
				settings.MinPlayers = settings.MaxPlayers
			}
		}
		return settings
	})
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	return c.Send(mafiaPlayerSettingsText(updated), mafiaPlayerSettingsMarkup(updated))
}
