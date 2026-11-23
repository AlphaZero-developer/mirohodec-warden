package main

import (
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) replyInRehab(c tele.Context, jail ModerationRecord) {
	msg := c.Message()
	chat := c.Chat()
	if msg == nil || chat == nil {
		return
	}
	now := time.Now().UTC()
	replyMarkup := a.layaInlineMarkup(chat, c.Sender())
	jail, wardMessage, wardHandled, wardRelease := a.handleRehabWardMessage(chat.ID, jail, now)
	if wardHandled {
		if wardRelease {
			if a.maybeStartRehabReleaseExam(chat, jail, "vip_review", msg) {
				if strings.TrimSpace(wardMessage) != "" {
					wardMessage += "\nПеред выходом из психиатрического центра назначен экзамен."
				}
			} else {
				rec, released := a.store.RemoveJail(chat.ID, jail.UserID)
				a.hideRehabEscapeCommand(chat.ID, jail.UserID)
				a.clearRehabEscapeCooldown(chat.ID, jail.UserID)
				a.resetRehabWardProgress(chat.ID, jail)
				if !released {
					rec = jail
				}
				duration := a.rehabReleaseDurationText(rec, released, now)
				a.logRehabRelease(chat.ID, rec, duration, "vip_review")
				a.sendRehabReleaseNotice(chat, rec, duration, msg.ThreadID)
				a.schedulePostRehabCheck(chat.ID, rec, now)
				if strings.TrimSpace(wardMessage) != "" {
					wardMessage += "\n" + a.t("rehab_release_notice", duration)
				}
			}
		}
		if strings.TrimSpace(wardMessage) != "" {
			if _, err := a.bot.Send(chat, wardMessage, &tele.SendOptions{ReplyTo: msg, ThreadID: msg.ThreadID, ReplyMarkup: replyMarkup}); err != nil {
				a.log.Warn("rehab ward state reply failed", zap.Error(err))
			}
		}
		a.metrics.IncModeration("rehab_ward_state")
		return
	}
	if a.shouldSkipQuietWardReply(chat.ID, jail) {
		a.metrics.IncModeration("rehab_quiet_skip")
		return
	}
	if reply, handled := rehabWardReply(jail); handled {
		if strings.TrimSpace(reply) == "" {
			return
		}
		if _, err := a.bot.Send(chat, reply, &tele.SendOptions{ReplyTo: msg, ThreadID: msg.ThreadID, ReplyMarkup: replyMarkup}); err != nil {
			a.log.Warn("rehab ward reply failed", zap.Error(err))
			return
		}
		a.metrics.IncModeration("rehab_ward_reply")
		return
	}
	if reply, ok := a.rehabBrainwashReply(jail); ok {
		if _, err := a.bot.Send(chat, reply, &tele.SendOptions{ReplyTo: msg, ThreadID: msg.ThreadID, ReplyMarkup: replyMarkup}); err != nil {
			a.log.Warn("rehab brainwash reply failed", zap.Error(err))
			return
		}
		a.metrics.IncModeration("rehab_brainwash_reply")
		return
	}
	if a.shouldUseHardRehabReply(chat.ID, jail.UserID) {
		reply := strings.TrimSpace(a.config().Rehab.HarshReply)
		if reply == "" {
			reply = a.layaReactionText("rehab")
		}
		if _, err := a.bot.Send(chat, reply, &tele.SendOptions{ReplyTo: msg, ThreadID: msg.ThreadID, ReplyMarkup: replyMarkup}); err != nil {
			a.log.Warn("hard rehab reply failed", zap.Error(err))
			return
		}
		a.metrics.IncModeration("hard_rehab_reply")
		return
	}
	var quoteText string
	if a.quotesEnabledForChat(chat) {
		if rec, ok := a.quotes.Random(""); ok {
			quoteText = a.sanitizeRandomLayaReply(&tele.User{ID: jail.UserID}, a.formatQuote(rec))
		}
	}
	if a.photosEnabledForChat(chat) {
		file, caption, ok, err := a.randomPhotoMedia("")
		if err != nil {
			a.log.Warn("rehab photo lookup failed", zap.Error(err))
		}
		if ok {
			if quoteText != "" {
				caption = quoteText
				if strings.TrimSpace(caption) != "" {
					caption = trimRunes(caption, 1000)
				}
			} else {
				caption = a.sanitizeRandomLayaReply(&tele.User{ID: jail.UserID}, caption)
			}
			if _, err := a.bot.Send(chat, &tele.Photo{File: file, Caption: caption}, &tele.SendOptions{ReplyTo: msg, ThreadID: msg.ThreadID, ReplyMarkup: replyMarkup}); err != nil {
				a.log.Warn("rehab photo reply failed", zap.Error(err))
			} else {
				a.metrics.IncModeration("rehab_reply_photo")
				return
			}
		}
	}
	if quoteText == "" {
		quoteText = a.layaReactionText("rehab")
	}
	if _, err := a.bot.Send(chat, quoteText, &tele.SendOptions{ReplyTo: msg, ThreadID: msg.ThreadID, ReplyMarkup: replyMarkup}); err != nil {
		a.log.Warn("rehab text reply failed", zap.Error(err))
		return
	}
	a.metrics.IncModeration("rehab_reply")
}
