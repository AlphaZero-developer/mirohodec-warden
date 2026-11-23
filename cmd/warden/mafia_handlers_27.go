package main

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func mafiaRoomStatusBlocksPublicTalk(status MafiaRoomStatus) bool {
	switch status {
	case MafiaStatusNight, MafiaStatusRoleDistribution:
		return true
	default:
		return false
	}
}

func mafiaIsForwardedMessage(msg *tele.Message) bool {
	return msg != nil && (isForwardedMessage(msg) || msg.AutomaticForward)
}

func mafiaIsMediaMessage(msg *tele.Message) bool {
	if msg == nil {
		return false
	}
	return msg.Audio != nil ||
		msg.Document != nil ||
		msg.Photo != nil ||
		msg.Sticker != nil ||
		msg.Voice != nil ||
		msg.VideoNote != nil ||
		msg.Video != nil ||
		msg.Animation != nil ||
		msg.Contact != nil ||
		msg.Location != nil ||
		msg.Venue != nil ||
		msg.Poll != nil ||
		msg.Game != nil ||
		msg.Dice != nil ||
		msg.AlbumID != ""
}

func mafiaCriticalLeakAppliesToPlayer(leak MafiaRuleCheckResult, player MafiaPlayer) bool {
	if leak.ViolationType == MafiaRuleViolationAllyLeak {
		return isMafiaTeam(player.Role)
	}
	if leak.ViolationType == MafiaRuleViolationGameBreakingLeak && strings.Contains(leak.Reason, "состава мафии") {
		return isMafiaTeam(player.Role)
	}
	return true
}

func (a *App) handleMafiaCriticalLeak(c tele.Context, room MafiaRoom, offender MafiaPlayer, leak MafiaRuleCheckResult) {
	settings := normalizeMafiaSettings(room.Settings)
	now := time.Now().UTC()
	switch settings.GameBreakingLeakAction {
	case MafiaLeakActionHostDecision:
		updated, err := a.mafiaService().MarkCriticalLeakPending(room.ID, offender.TelegramID, leak.ViolationType, now)
		if err != nil {
			if a.log != nil {
				a.log.Warn("mark mafia critical leak pending failed", zap.Error(err))
			}
			return
		}
		a.syncMafiaGroupRestrictions(updated)
		a.sendMafiaGroup(updated, "⚠️ Возможный критический слив.\n\nПартия приостановлена.\nРешение принимает создатель комнаты.", nil)
		a.sendMafiaDM(updated.OwnerID, mafiaLeakOwnerDecisionText(updated, offender), mafiaLeakDecisionMarkup(updated))
	case MafiaLeakActionKillOffender:
		updated, err := a.mafiaService().MarkCriticalLeakPending(room.ID, offender.TelegramID, leak.ViolationType, now)
		if err == nil {
			updated, err = a.mafiaService().KillOffenderForCriticalLeak(updated.ID, updated.OwnerID, now)
		}
		if err != nil {
			if a.log != nil {
				a.log.Warn("kill mafia critical leak offender failed", zap.Error(err))
			}
			return
		}
		a.syncMafiaGroupRestrictions(updated)
		a.sendMafiaGroup(updated, "☠️ Лайя обнаружила критический слив.\n\nНарушитель выбывает из партии.", nil)
		if updated.Status == MafiaStatusFinished {
			a.announceMafiaFinished(updated)
		}
	default:
		updated, err := a.mafiaService().CancelGameForCriticalLeak(room.ID, offender.TelegramID, leak.ViolationType, now)
		if err != nil {
			if a.log != nil {
				a.log.Warn("cancel mafia game for critical leak failed", zap.Error(err))
			}
			return
		}
		a.syncMafiaGroupRestrictions(updated)
		a.sendMafiaGroup(updated, mafiaCriticalLeakCancelText(), nil)
		a.sendMafiaDM(updated.OwnerID, mafiaCancelledOwnerText(updated, offender), mafiaCancelledOwnerMarkup(updated))
	}
}

func (a *App) deliverMafiaShadowMessage(roomID string, msg MafiaPrivateMessage, recipients []int64) {
	if a == nil || a.bot == nil {
		return
	}
	for _, userID := range recipients {
		if userID <= 0 {
			continue
		}
		text := fmt.Sprintf("💬 Теневой канал %s\n\n%s: %s", roomID, msg.SenderAlias, msg.Text)
		if userID == msg.SenderID {
			text = fmt.Sprintf("💬 Теневой канал %s\n\nВы: %s", roomID, msg.Text)
		}
		if _, err := a.bot.Send(&tele.User{ID: userID}, text); err != nil && a.log != nil {
			a.log.Warn("send mafia shadow message failed", zap.Int64("user", userID), zap.Error(err))
		}
	}
}

func (a *App) localizeMafiaErr(err error) string {
	switch {
	case errors.Is(err, errMafiaUserNotRegistered):
		return "пользователь не зарегистрирован. Нужно открыть ЛС с Лайей и нажать /start."
	case errors.Is(err, errMafiaProfileIncomplete):
		return "сначала создайте персонажа Мафии в ЛС с Лайей."
	case errors.Is(err, errMafiaRoomNotFound):
		return "комната не найдена."
	case errors.Is(err, errMafiaRoomNotLobby):
		return "комната уже не в лобби."
	case strings.HasPrefix(err.Error(), "need at least "):
		var minPlayers int
		if _, scanErr := fmt.Sscanf(err.Error(), "need at least %d players", &minPlayers); scanErr == nil && minPlayers > 0 {
			return fmt.Sprintf("нужно минимум %d игрока. Добавьте игроков или тест-ботов в настройках комнаты.", minPlayers)
		}
		return "нужно больше игроков. Добавьте игроков или тест-ботов в настройках комнаты."
	case err.Error() == "mafia group is not bound":
		return "сначала подключите группу. В нужной группе отправьте /mafia_bind " + "КОД_КОМНАТЫ" + "."
	case errors.Is(err, errMafiaNotRoomOwner):
		return "это может сделать только владелец комнаты."
	case errors.Is(err, errMafiaWrongPhase):
		return "сейчас не та фаза игры."
	case errors.Is(err, errMafiaWrongRole):
		return "у вас нет доступа к этому действию."
	case errors.Is(err, errMafiaPlayerDead):
		return "мёртвые не выполняют это действие."
	case errors.Is(err, errMafiaShadowUnavailable):
		return "теневой канал сейчас недоступен."
	case errors.Is(err, errMafiaWillsDisabled):
		return "завещания в этой комнате выключены."
	case errors.Is(err, errMafiaSettingsLocked):
		return "настройки можно менять только в лобби."
	case errors.Is(err, errMafiaRecruitmentClosed):
		return "набор в эту комнату закрыт."
	case errors.Is(err, errMafiaRoomFull):
		return "комната уже заполнена."
	case errors.Is(err, errMafiaRequestExists):
		return "вы уже подали заявку."
	case errors.Is(err, errMafiaInviteLimited):
		return "нельзя так часто рассылать приглашения. Попробуйте открыть набор в группе или использовать очередь готовых игроков."
	case errors.Is(err, errMafiaNightActionExists):
		return "ночное действие уже принято."
	case errors.Is(err, errMafiaInvitesDisabled):
		return "у этого игрока отключены приглашения в Мафию."
	case errors.Is(err, errMafiaHostBlocked):
		return "вы не можете подать заявку в комнаты этого создателя."
	default:
		return err.Error()
	}
}
