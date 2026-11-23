package main

import (
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) callbackMafiaAddTestBot(c tele.Context, roomID string, count int) error {
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	updated, added, err := a.mafiaService().AddTestBots(room.ID, sender.ID, count, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	return c.Send(mafiaTestBotsChangedText(updated, len(added), "добавлено"), mafiaPlayerSettingsMarkup(updated))
}

func (a *App) callbackMafiaFillTestBots(c tele.Context, roomID string) error {
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	updated, added, err := a.mafiaService().FillTestBotsToMinimum(room.ID, sender.ID, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	if len(added) == 0 {
		return c.Send("🤖 Боты не понадобились.\n\nВ комнате уже достаточно игроков для текущего минимума.", mafiaPlayerSettingsMarkup(updated))
	}
	return c.Send(mafiaTestBotsChangedText(updated, len(added), "добавлено"), mafiaPlayerSettingsMarkup(updated))
}

func (a *App) callbackMafiaClearTestBots(c tele.Context, roomID string) error {
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	updated, removed, err := a.mafiaService().RemoveTestBots(room.ID, sender.ID, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	if removed == 0 {
		return c.Send("🤖 В комнате нет тест-ботов.", mafiaPlayerSettingsMarkup(updated))
	}
	return c.Send(mafiaTestBotsChangedText(updated, removed, "убрано"), mafiaPlayerSettingsMarkup(updated))
}

func (a *App) callbackMafiaToggleSetting(c tele.Context, data string) error {
	parts := strings.Split(data, ":")
	if len(parts) != 4 {
		return c.Send("⛔ Устаревшая кнопка.")
	}
	roomID, key := parts[2], parts[3]
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	updated, err := a.mafiaService().UpdateLobbySettings(room.ID, sender.ID, time.Now().UTC(), func(settings MafiaRoomSettings) MafiaRoomSettings {
		switch key {
		case "wills":
			settings.EnableWills = !settings.EnableWills
		case "achievements":
			settings.EnableAchievements = !settings.EnableAchievements
		case "ai_report":
			settings.EnablePostGameAIReport = !settings.EnablePostGameAIReport
		case "ai_bots":
			settings.EnableAIBots = !settings.EnableAIBots
		case "vote_change":
			settings.AllowVoteChange = !settings.AllowVoteChange
		case "tie_random":
			settings.TieMode = MafiaTieRandomBetweenLeaders
		case "tie_nobody":
			settings.TieMode = MafiaTieNobodyDies
		case "reveal_allow":
			settings.RoleRevealPolicy = MafiaRoleRevealAllowClaims
			settings.AllowRoleReveal = true
			settings.PunishRoleReveal = false
		case "reveal_system":
			settings.RoleRevealPolicy = MafiaRoleRevealBanSystemProof
			settings.AllowRoleReveal = true
			settings.PunishRoleReveal = true
		case "reveal_all":
			settings.RoleRevealPolicy = MafiaRoleRevealBanAllClaims
			settings.AllowRoleReveal = false
			settings.PunishRoleReveal = true
		case "leak_action":
			settings.GameBreakingLeakAction = mafiaNextLeakAction(settings.GameBreakingLeakAction)
		}
		return settings
	})
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	if strings.HasPrefix(key, "tie_") || key == "vote_change" {
		return c.Send(mafiaVotingSettingsText(updated), mafiaVotingSettingsMarkup(updated))
	}
	if strings.HasPrefix(key, "reveal_") {
		return c.Send(mafiaRevealSettingsText(updated), mafiaRevealSettingsMarkup(updated))
	}
	return c.Send(mafiaSettingsText(updated), mafiaSettingsMarkup(updated, sender.ID))
}

func (a *App) callbackMafiaToggleOptionalRole(c tele.Context, data string) error {
	parts := strings.Split(data, ":")
	if len(parts) != 4 {
		return c.Send("⛔ Устаревшая кнопка.")
	}
	roomID, role := parts[2], parts[3]
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	updated, err := a.mafiaService().UpdateLobbySettings(room.ID, sender.ID, time.Now().UTC(), func(settings MafiaRoomSettings) MafiaRoomSettings {
		switch role {
		case "don":
			settings.OptionalRoles.EnableDon = !settings.OptionalRoles.EnableDon
		case "sergeant":
			settings.OptionalRoles.EnableSergeant = !settings.OptionalRoles.EnableSergeant
		case "bodyguard":
			settings.OptionalRoles.EnableBodyguard = !settings.OptionalRoles.EnableBodyguard
		case "blocker":
			settings.OptionalRoles.EnableBlocker = !settings.OptionalRoles.EnableBlocker
		case "lawyer":
			settings.OptionalRoles.EnableLawyer = !settings.OptionalRoles.EnableLawyer
		case "journalist":
			settings.OptionalRoles.EnableJournalist = !settings.OptionalRoles.EnableJournalist
		case "mayor":
			settings.OptionalRoles.EnableMayor = !settings.OptionalRoles.EnableMayor
		}
		return settings
	})
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	return c.Send(mafiaRoleSettingsText(), mafiaRoleSettingsMarkup(updated))
}

func (a *App) callbackMafiaFullTest(c tele.Context, roomID string) error {
	if strings.TrimSpace(roomID) == "" {
		sender := c.Sender()
		if sender != nil {
			if room, ok := a.activeMafiaRoomForUser(sender.ID); ok {
				roomID = room.ID
			}
		}
	}
	if strings.TrimSpace(roomID) == "" {
		return c.Send("⛔ Полный тест доступен только создателю комнаты в лобби.")
	}
	if _, _, _, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby}); !ok {
		return nil
	}
	report := RunMafiaFullTest()
	return c.Send(report.Text)
}

func mafiaRegisterMarkup(chat *tele.Chat) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	registerData := "mafia:register"
	if chat != nil && chat.Type != tele.ChatPrivate {
		registerData = "mafia:register_hint"
	}
	markup.Inline(
		markup.Row(mafiaButton("✅ Зарегистрироваться", registerData)),
		markup.Row(mafiaButton("❓ Что такое Мафия?", "mafia:rules"), mafiaButton("⬅️ Назад", "mafia:back")),
	)
	return markup
}

func mafiaButton(text, data string) tele.Btn {
	return tele.Btn{Text: text, Data: data}
}

func mafiaSkipPhaseData(room MafiaRoom) string {
	return "mafia:skip_phase:" + room.ID + ":" + string(room.Status)
}
