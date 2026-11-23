package main

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) maybeHandleMafiaGroupMessage(c tele.Context) bool {
	if a == nil || a.store == nil || c == nil {
		return false
	}
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || chat.Type == tele.ChatPrivate {
		return false
	}
	room, ok := a.mafiaRoomForGroupMessage(chat.ID, sender.ID)
	if !ok || !mafiaRoomStatusAcceptsPublicDiscussion(room.Status) {
		return false
	}
	text := messageText(msg)
	if room.PendingLeakOffenderID != 0 {
		a.tryDeleteMafiaMessage(c, "mafia_paused_delete")
		_ = c.Send("⚠️ Партия приостановлена из-за возможного критического слива. Ждём решение создателя комнаты.")
		return true
	}
	if mafiaRoomStatusHasActiveParty(room.Status) {
		if mafiaIsForwardedMessage(msg) {
			a.tryDeleteMafiaMessage(c, "mafia_active_forward_delete")
			if a.shouldSendMafiaGroupNotice(room, sender.ID, "forward_block", time.Now().UTC()) {
				_ = c.Send("⛔ Во время партии пересланные сообщения удаляются.\nПишите своими словами, без пересылок из ЛС и других чатов.")
			}
			return true
		}
		if mafiaIsMediaMessage(msg) {
			a.tryDeleteMafiaMessage(c, "mafia_active_media_delete")
			if a.shouldSendMafiaGroupNotice(room, sender.ID, "media_block", time.Now().UTC()) {
				_ = c.Send("⛔ Во время партии медиа отключены.\nДо финала в общем чате остаётся только текст.")
			}
			return true
		}
	}
	if mafiaRoomStatusBlocksPublicTalk(room.Status) {
		a.tryDeleteMafiaMessage(c, "mafia_night_silence_delete")
		if idx := mafiaRoomPlayerIndex(room, sender.ID); idx >= 0 {
			if leak := DetectMafiaCriticalLeak(text); leak.ShouldPunish && mafiaCriticalLeakAppliesToPlayer(leak, room.Players[idx]) {
				a.handleMafiaCriticalLeak(c, room, room.Players[idx], leak)
				return true
			}
		}
		if a.shouldSendMafiaGroupNotice(room, sender.ID, "night_silence", time.Now().UTC()) {
			_ = c.Send("🌙 Ночь. Общий чат молчит до утра.\nДействия проходят в личных сообщениях с Лайей.")
		}
		return true
	}
	idx := mafiaRoomPlayerIndex(room, sender.ID)
	if idx < 0 {
		return false
	}
	if !room.Players[idx].Alive {
		a.tryDeleteMafiaMessage(c, "mafia_dead_talking_delete")
		if a.shouldSendMafiaGroupNotice(room, sender.ID, "dead_talking", time.Now().UTC()) {
			_ = c.Send("⚰️ Мёртвые не участвуют в обсуждении.")
		}
		return true
	}
	if mafiaIsGroupSkipPhaseText(text) {
		a.handleMafiaGroupPhaseSkip(c, room, room.Players[idx])
		return true
	}
	room = a.recordMafiaAIBotSuspicionAndMaybeTalk(room, msg, sender, text)
	if leak := DetectMafiaCriticalLeak(text); leak.ShouldPunish && mafiaCriticalLeakAppliesToPlayer(leak, room.Players[idx]) {
		a.tryDeleteMafiaMessage(c, "mafia_critical_leak_delete")
		a.handleMafiaCriticalLeak(c, room, room.Players[idx], leak)
		return true
	}
	if room.Status != MafiaStatusDayDiscussion {
		return false
	}
	settings := normalizeMafiaSettings(room.Settings)
	if !settings.PunishRoleReveal {
		return false
	}
	reveal := DetectMafiaSystemProof(text)
	if !reveal.IsRoleReveal && settings.RoleRevealPolicy == MafiaRoleRevealBanAllClaims {
		reveal = DetectMafiaRoleReveal(text)
	}
	if !reveal.IsRoleReveal && settings.RoleRevealPolicy == MafiaRoleRevealBanAllClaims {
		reveal = a.detectMafiaRoleRevealAI(text, room.Status)
	}
	if !reveal.IsRoleReveal {
		return false
	}
	if reveal.Source == MafiaRevealSourceAI && reveal.Confidence < 0.9 {
		if reveal.Confidence >= 0.7 {
			a.sendMafiaDM(room.OwnerID, fmt.Sprintf("⚖️ Лайя обнаружила возможное раскрытие роли.\n\nИгрок: %s\nСообщение: %q\nУверенность: %.0f%%\n\nПроверь вручную.", mafiaRoomPlayerLabel(room, room.Players[idx]), text, reveal.Confidence*100), nil)
		}
		return false
	}
	updated, err := a.mafiaService().PunishRoleReveal(room.ID, sender.ID, time.Now().UTC())
	if err != nil {
		if a.log != nil {
			a.log.Warn("mafia role reveal punishment failed", zap.Error(err))
		}
		return false
	}
	a.syncMafiaGroupRestrictions(updated)
	_ = c.Send(mafiaRoleRevealPunishmentText(updated, sender.ID))
	if idx := mafiaRoomPlayerIndex(updated, sender.ID); idx >= 0 && !updated.Players[idx].Alive {
		a.notifyMafiaPlayerEliminated(updated, sender.ID, "role_reveal")
	}
	a.announceMafiaWill(updated, sender.ID)
	if updated.Status == MafiaStatusFinished {
		a.announceMafiaFinished(updated)
	}
	return true
}

func (a *App) recordMafiaAIBotSuspicionAndMaybeTalk(room MafiaRoom, msg *tele.Message, sender *tele.User, text string) MafiaRoom {
	if a == nil || msg == nil || sender == nil || strings.TrimSpace(text) == "" {
		return room
	}
	if room.Status != MafiaStatusDayDiscussion && room.Status != MafiaStatusVoting {
		return room
	}
	service := a.mafiaService()
	updated, remembered, err := service.RecordAIBotSuspicion(room.ID, sender.ID, msg.ID, text, time.Now().UTC())
	if err != nil {
		if a.log != nil {
			a.log.Warn("record mafia ai bot suspicion failed", zap.String("room", room.ID), zap.Int64("user", sender.ID), zap.Error(err))
		}
		return room
	}
	if !remembered {
		return updated
	}
	updated, botMessage, err := service.MaybeAddAIBotDiscussionMessage(room.ID, time.Now().UTC())
	if err != nil {
		if a.log != nil {
			a.log.Warn("send mafia ai bot thought failed", zap.String("room", room.ID), zap.Error(err))
		}
		return updated
	}
	if botMessage != nil && strings.TrimSpace(botMessage.Text) != "" {
		a.sendMafiaGroup(updated, mafiaAIBotMessageLine(*botMessage), nil)
	}
	return updated
}

func (a *App) tryDeleteMafiaMessage(c tele.Context, action string) {
	if c == nil {
		return
	}
	if err := c.Delete(); err != nil {
		if a != nil && a.log != nil {
			chatID := int64(0)
			userID := int64(0)
			if chat := c.Chat(); chat != nil {
				chatID = chat.ID
			}
			if sender := c.Sender(); sender != nil {
				userID = sender.ID
			}
			a.log.Warn("delete mafia message failed", zap.Int64("chat", chatID), zap.Int64("user", userID), zap.String("action", action), zap.Error(err))
		}
		return
	}
	if a != nil && a.metrics != nil {
		a.metrics.IncModeration(action)
	}
}

func mafiaRoomStatusAcceptsPublicDiscussion(status MafiaRoomStatus) bool {
	switch status {
	case MafiaStatusNight, MafiaStatusDayDiscussion, MafiaStatusVoting, MafiaStatusRoleDistribution:
		return true
	default:
		return false
	}
}

func mafiaRoomStatusHasActiveParty(status MafiaRoomStatus) bool {
	switch status {
	case MafiaStatusRoleDistribution, MafiaStatusNight, MafiaStatusDayDiscussion, MafiaStatusVoting:
		return true
	default:
		return false
	}
}
