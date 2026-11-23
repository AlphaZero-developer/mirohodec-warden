package main

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

type layaForwardRestrictionIntent int

const (
	layaForwardRestrictionNone layaForwardRestrictionIntent = iota
	layaForwardRestrictionBlock
	layaForwardRestrictionAllow
)

func (a *App) maybeHandleLayaForwardRestriction(c tele.Context) bool {
	msg := c.Message()
	chat := c.Chat()
	admin := c.Sender()
	if msg == nil || chat == nil || admin == nil || admin.IsBot || chat.Type == tele.ChatPrivate || !a.isManagedChat(chat) {
		return false
	}

	intent, ok := parseLayaForwardRestrictionIntent(messageText(msg), a.botUsername())
	if !ok {
		return false
	}

	if !a.isConfiguredAdmin(admin.ID) && !a.isTelegramGroupAdmin(chat, admin) {
		if err := c.Send(a.t("no_admin")); err != nil && a.log != nil {
			a.log.Warn("send no-admin notice failed", zap.Error(err))
		}
		return true
	}

	target, ok := layaForwardRestrictionReplyTarget(msg)
	if !ok {
		if err := c.Send("Ответь на сообщение пользователя: Лайя, этому запрещено пересылать сообщения. Снять запрет: Лайя, этому можно пересылать сообщения."); err != nil && a.log != nil {
			a.log.Warn("send laya forward restriction usage failed", zap.Error(err))
		}
		return true
	}
	if err := a.ensureModeratable(chat, target.ID); err != nil {
		if sendErr := c.Send(a.localizeErr(err)); sendErr != nil && a.log != nil {
			a.log.Warn("send laya forward restriction moderatable error failed", zap.Error(sendErr))
		}
		return true
	}

	now := time.Now().UTC()
	switch intent {
	case layaForwardRestrictionBlock:
		a.store.SetForwardRestriction(chat.ID, ModerationRecord{
			UserID:      target.ID,
			Username:    target.Username,
			DisplayName: target.Name,
			Reason:      "forward restriction",
			ByAdminID:   admin.ID,
			ByAdminName: displayNameFromUser(admin),
			SinceUnix:   now.Unix(),
			UntilUnix:   0,
		})
		if a.metrics != nil {
			a.metrics.IncModeration("forward_restrict")
		}
		a.sendOwnerLog("Forward restriction",
			logField{"User", fmt.Sprintf("%s (%d)", target.Name, target.ID)},
			logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)},
			logField{"Action", "block forwarded/via-bot messages"},
			logField{"By", displayNameFromUser(admin)},
		)
		if err := c.Send(fmt.Sprintf("Поняла. %s теперь без пересылок: forward и via-bot сообщения буду удалять сразу.", target.Name)); err != nil && a.log != nil {
			a.log.Warn("send laya forward restriction reply failed", zap.Error(err))
		}
	case layaForwardRestrictionAllow:
		_, existed := a.store.RemoveForwardRestriction(chat.ID, target.ID)
		if a.metrics != nil {
			a.metrics.IncModeration("forward_unrestrict")
		}
		a.sendOwnerLog("Forward restriction removed",
			logField{"User", fmt.Sprintf("%s (%d)", target.Name, target.ID)},
			logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)},
			logField{"Action", "allow forwarded/via-bot messages"},
			logField{"By", displayNameFromUser(admin)},
		)
		reply := fmt.Sprintf("Сняла запрет на пересылку для %s.", target.Name)
		if !existed {
			reply = fmt.Sprintf("У %s и так не было запрета на пересылку.", target.Name)
		}
		if err := c.Send(reply); err != nil && a.log != nil {
			a.log.Warn("send laya forward restriction removal reply failed", zap.Error(err))
		}
	}
	return true
}

func (a *App) maybeDeleteForwardRestrictedMessage(c tele.Context) bool {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || !a.isManagedChat(chat) || !isForwardRestrictionPayload(msg) {
		return false
	}

	rec, matchedUserID, ok := a.forwardRestrictionForMessage(chat.ID, sender, msg, time.Now().UTC())
	if !ok {
		return false
	}

	if err := c.Delete(); err != nil {
		if a.log != nil {
			a.log.Warn("delete forward-restricted message failed",
				zap.Int64("chat", chat.ID),
				zap.Int64("sender", userID(sender)),
				zap.Int64("restricted_user", matchedUserID),
				zap.Error(err),
			)
		}
		if a.metrics != nil {
			a.metrics.IncModeration("forward_restricted_delete_failed")
		}
		return true
	}
	if a.metrics != nil {
		a.metrics.IncModeration("forward_restricted_delete")
	}
	if a.log != nil {
		a.log.Info("forward-restricted message deleted",
			zap.Int64("chat", chat.ID),
			zap.Int64("sender", userID(sender)),
			zap.Int64("restricted_user", matchedUserID),
			zap.String("restricted_name", rec.DisplayName),
			zap.String("kind", forwardRestrictionPayloadKind(msg)),
		)
	}
	return true
}

func (a *App) forwardRestrictionForMessage(chatID int64, sender *tele.User, msg *tele.Message, now time.Time) (ModerationRecord, int64, bool) {
	if a == nil || a.store == nil || msg == nil {
		return ModerationRecord{}, 0, false
	}
	if sender != nil && sender.ID != 0 {
		if rec, ok := a.store.ActiveForwardRestriction(chatID, sender.ID, now); ok {
			return rec, sender.ID, true
		}
	}
	for _, id := range forwardedOriginalUserIDs(msg) {
		if rec, ok := a.store.ActiveForwardRestriction(chatID, id, now); ok {
			return rec, id, true
		}
	}
	return ModerationRecord{}, 0, false
}

func parseLayaForwardRestrictionIntent(text, botUsername string) (layaForwardRestrictionIntent, bool) {
	text = strings.TrimSpace(text)
	if text == "" || !mentionsLaya(text, botUsername) {
		return layaForwardRestrictionNone, false
	}
	normalized := " " + normalizeFAQText(text) + " "
	if !containsAnyNormalized(normalized, " пересыл", " форвард", " forward") {
		return layaForwardRestrictionNone, false
	}

	targetRef := containsAnyNormalized(normalized,
		" этому ", " этому пользователю ", " ему ", " ей ", " пользователю ", " юзеру ", " этому юзеру ", " данному ",
		" this ", " user ", " him ", " her ",
	)
	blockImperative := containsAnyNormalized(normalized, " запрети", " запрещай", " запрешай", " запретить", " forbid", " ban")
	allowImperative := containsAnyNormalized(normalized, " разреши", " сними", " снять", " убери", " верни", " allow", " unban")
	blockState := targetRef && containsAnyNormalized(normalized, " запрещ", " запреш", " нельзя", " запрет")
	allowState := targetRef && containsAnyNormalized(normalized, " разреш", " можно")

	switch {
	case allowImperative || allowState:
		return layaForwardRestrictionAllow, true
	case blockImperative || blockState:
		return layaForwardRestrictionBlock, true
	default:
		return layaForwardRestrictionNone, false
	}
}

func containsAnyNormalized(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
