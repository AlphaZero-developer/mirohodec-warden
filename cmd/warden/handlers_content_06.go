package main

import (
	"math/rand"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) maybeAutoReplyQuote(c tele.Context) {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || sender.IsBot || chat.Type == tele.ChatPrivate {
		return
	}
	quoteCfg := a.activeQuoteConfigForChat(chat)
	if !quoteCfg.Enabled || !quoteCfg.AutoReply {
		return
	}
	text := messageText(msg)
	if text == "" || commandName(text) != "" {
		return
	}

	chance := quoteCfg.AutoReplyChance
	if chance <= 0 {
		return
	}
	now := time.Now().UTC()
	if member, ok := a.store.Member(chat.ID, sender.ID); ok && member.QuoteBoostUntilUnix > now.Unix() && chance < 0.25 {
		chance = 0.25
	}
	chance = weeklyEventQuoteChance(chance, now)
	if chance < 1 && rand.Float64() > chance {
		return
	}

	cooldown := mustDuration(quoteCfg.AutoReplyCooldown, 30*time.Minute)
	key := idKey(chat.ID)
	a.autoQuoteMu.Lock()
	last := a.lastAutoQuote[key]
	if now.Sub(last) < cooldown {
		a.autoQuoteMu.Unlock()
		return
	}
	a.lastAutoQuote[key] = now
	a.autoQuoteMu.Unlock()

	rec, ok := a.quotes.Random("")
	if !ok {
		return
	}

	target, ok := a.store.RandomActiveMember(chat.ID, mustDuration(quoteCfg.ActiveUserWindow, 7*24*time.Hour), now, sender.ID)
	if !ok {
		target = memberFromUser(sender, now.Unix())
		target.LastMessageID = msg.ID
		target.LastThreadID = msg.ThreadID
	}

	replyTo := msg
	threadID := msg.ThreadID
	if target.LastMessageID != 0 {
		replyTo = &tele.Message{ID: target.LastMessageID, Chat: chat, ThreadID: target.LastThreadID}
		if target.LastThreadID != 0 {
			threadID = target.LastThreadID
		}
	}

	quoteRecipient := userFromMemberRecord(target)
	quoteText := a.filterRandomChatMessage(quoteRecipient, a.formatQuote(rec))
	photoCfg := a.activePhotoConfigForChat(chat)
	if photoCfg.Enabled && quoteCfg.PhotoChance > 0 && rand.Float64() <= quoteCfg.PhotoChance {
		file, caption, ok, err := a.randomPhotoMedia("")
		if err != nil {
			a.log.Warn("auto quote photo lookup failed", zap.Error(err))
		}
		if ok {
			photoCaption := quoteText
			if caption != "" {
				caption = a.sanitizeRandomLayaReply(quoteRecipient, caption)
				photoCaption = quoteText + "\n\n" + caption
			}
			if _, err := a.bot.Send(chat, &tele.Photo{File: file, Caption: trimRunes(photoCaption, 1000)}, &tele.SendOptions{ReplyTo: replyTo, ThreadID: threadID, ReplyMarkup: a.layaInlineMarkup(chat, sender)}); err != nil {
				a.log.Warn("auto quote photo reply failed", zap.Error(err))
			} else {
				a.metrics.IncModeration("auto_quote_photo")
			}
			return
		}
	}

	if _, err := a.bot.Send(chat, quoteText, &tele.SendOptions{ReplyTo: replyTo, ThreadID: threadID, ReplyMarkup: a.layaInlineMarkup(chat, sender)}); err != nil {
		a.log.Warn("auto quote reply failed", zap.Error(err))
		return
	}
	a.metrics.IncModeration("auto_quote")
}

func (a *App) maybeAutoReplyPhoto(c tele.Context) {
	msg := c.Message()
	chat := c.Chat()
	if msg == nil || chat == nil {
		return
	}
	cfg := a.activePhotoConfigForChat(chat)
	if !cfg.Enabled || !cfg.AutoReply {
		return
	}

	text := strings.TrimSpace(msg.Text)
	caption := strings.TrimSpace(msg.Caption)
	if strings.HasPrefix(text, "/") || strings.HasPrefix(caption, "/") {
		return
	}

	tag := ""
	if len(cfg.Triggers) > 0 {
		found := false
		haystack := strings.ToLower(text + " " + caption)
		for _, trigger := range cfg.Triggers {
			trigger = strings.ToLower(strings.TrimSpace(trigger))
			if trigger == "" {
				continue
			}
			if strings.Contains(haystack, trigger) {
				found = true
				break
			}
		}
		if !found {
			return
		}
	}

	chance := cfg.AutoReplyChance
	if chance <= 0 {
		chance = 1
	}
	if chance < 1 && rand.Float64() > chance {
		return
	}

	cooldown := mustDuration(cfg.Cooldown, 30*time.Second)
	key := idKey(chat.ID)
	now := time.Now().UTC()
	a.photoMu.Lock()
	last := a.lastPhoto[key]
	if now.Sub(last) < cooldown {
		a.photoMu.Unlock()
		return
	}
	a.lastPhoto[key] = now
	a.photoMu.Unlock()

	if err := a.sendRandomPhotoQuiet(c, tag, false); err != nil {
		a.log.Warn("auto photo reply failed", zap.Error(err))
	}
}

func (a *App) sendRandomPhoto(c tele.Context, tag string, preferRepliedMessage bool) error {
	return a.sendRandomPhotoFromBase(c, tag, preferRepliedMessage, true)
}

func (a *App) sendRandomPhotoQuiet(c tele.Context, tag string, preferRepliedMessage bool) error {
	return a.sendRandomPhotoFromBase(c, tag, preferRepliedMessage, false)
}
