package main

import (
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) trackAndGuard(next tele.HandlerFunc) tele.HandlerFunc {
	return func(c tele.Context) error {
		a.metrics.IncUpdate()

		msg := c.Message()
		chat := c.Chat()
		sender := c.Sender()
		a.rememberManagedChat(chat)

		if c.Callback() != nil {
			return next(c)
		}

		if chat != nil && sender != nil && a.isManagedChat(chat) {
			threadID := 0
			if msg != nil {
				threadID = msg.ThreadID
				a.metrics.IncMessage()
			}
			messageID := 0
			if msg != nil {
				messageID = msg.ID
			}
			a.store.TrackMember(chat.ID, sender, threadID, messageID, msg != nil)
			karmaCfg := a.activeKarmaConfigForChat(chat.ID)
			if msg != nil && karmaCfg.Enabled && countableKarmaMessage(msg) {
				now := time.Now().UTC()
				if _, awarded := a.store.MaybeAwardActivityKarma(chat.ID, sender.ID, karmaCfg.ActivityMessagesPerPoint, karmaCfg.ActivityDailyLimit, now); awarded {
					a.maybeAwardWeeklyEventKarma(chat, sender, msg, now)
				}
				a.maybeUnlockSocialAchievements(chat, sender, now)
			}
		}

		mafiaDiscussionSilence := false
		if msg != nil && chat != nil && sender != nil && chat.Type != tele.ChatPrivate {
			if a.maybeHandleMafiaGroupMessage(c) {
				return nil
			}
			mafiaDiscussionSilence = a.shouldSilenceMafiaDiscussionMessage(chat, msg)
		}

		if msg != nil && chat != nil && a.isManagedChat(chat) && !mafiaDiscussionSilence {
			a.maybeHandleDiscussionAutoReply(c)
		}

		if msg != nil && chat != nil && sender != nil && a.isManagedChat(chat) {
			if a.maybeHandleLayaScriptwriterExemption(c) {
				return nil
			}
			if a.maybeHandleLayaIgnoreCommand(c) {
				return nil
			}
			if a.maybeDeleteForwardRestrictedMessage(c) {
				return nil
			}
			if a.spamGuard(c) {
				return nil
			}
			if jail, ok := a.store.ActiveJail(chat.ID, sender.ID, time.Now().UTC()); ok {
				rehabTopicID := a.rehabTopicID(chat)
				if msg.ThreadID != rehabTopicID && !a.isJailExempt(sender.ID) {
					if isAppealCommandMessage(msg) {
						return next(c)
					}
					if err := c.Delete(); err != nil {
						a.log.Warn("delete jailed message failed", zap.Int64("chat", chat.ID), zap.Int64("user", sender.ID), zap.Error(err))
					}
					a.metrics.IncDeletedJailedMessage()
					a.warnJailedUser(chat, sender, jail)
					return nil
				}
				if rehabTopicID != 0 && msg.ThreadID == rehabTopicID && !a.isJailExempt(sender.ID) {
					if a.maybeHandleRehabReleaseExamAnswer(c, jail) {
						return nil
					}
					if a.maybeHandleRehabEscape(c, jail) {
						return nil
					}
					if a.maybeHandleRehabBribe(c, jail) {
						return nil
					}
					if isOwnerReportCommandMessage(msg) {
						return next(c)
					}
					if !a.layaIsIgnoring(chat.ID, sender.ID) {
						a.replyInRehab(c, jail)
					}
				}
			}
		}

		if msg != nil && chat != nil && sender != nil && a.isManagedChat(chat) {
			if a.maybeHandlePostRehabAnswer(c) {
				return nil
			}
		}

		if a.handleUnknownCommand(c) {
			return nil
		}

		if a.shouldIgnoreCommand(c) {
			return nil
		}

		if mafiaDiscussionSilence {
			return nil
		}

		if msg != nil && chat != nil && sender != nil {
			a.rememberLayaChatContext(chat, msg, sender)
		}

		if msg != nil && chat != nil && sender != nil {
			if a.maybeHandleOwnerSleepCommand(c) {
				return nil
			}
			if !a.layaIsIgnoring(chat.ID, sender.ID) && a.maybeHandleSleepingOwnerMention(c) {
				return nil
			}
		}

		if msg != nil && chat != nil && sender != nil && a.isManagedChat(chat) {
			if a.layaIsIgnoring(chat.ID, sender.ID) {
				return next(c)
			}
			if a.maybeHandleLayaRegistrationAnswer(c) {
				return nil
			}
			if a.maybeHandleManualKarma(c) {
				return nil
			}
			if a.store != nil {
				if _, jailed := a.store.ActiveJail(chat.ID, sender.ID, time.Now().UTC()); jailed && msg.ThreadID == a.rehabTopicID(chat) {
					return next(c)
				}
			}
			if a.maybeReactToProtectedInsult(c) {
				return nil
			}
			if a.maybeHandleLayaForwardRestriction(c) {
				return nil
			}
			if a.maybeHandleLayaFas(c) {
				return nil
			}
			if a.maybeHandleLayaFun(c) {
				return nil
			}
			if a.maybeReplyToNonWhitelistedLayaChat(c) {
				return nil
			}
			if !a.maybeAutoReplyFAQ(c) && !a.maybeReplyToLayaMention(c) {
				a.maybeAutoReplyQuote(c)
				a.maybeAutoReplyPhoto(c)
			}
		}

		return next(c)
	}
}
