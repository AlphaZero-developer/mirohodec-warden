package main

import (
	"math/rand"
	"strings"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) maybeReplyToLayaMention(c tele.Context) bool {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || chat.Type == tele.ChatPrivate {
		return false
	}
	if a.layaIsIgnoring(chat.ID, sender.ID) {
		return true
	}
	text := strings.TrimSpace(messageText(msg))
	inputText := layaMessageUserText(msg)
	addressed := mentionsLaya(text, a.botUsername())
	addressedByReply := false
	if !addressed && msg.ReplyTo != nil && msg.ReplyTo.Sender != nil && a.bot != nil && a.bot.Me != nil {
		addressedByReply = msg.ReplyTo.Sender.ID == a.bot.Me.ID && !isForumTopicRootReply(msg)
		addressed = addressedByReply
	}
	if !layaMessageHasUserContent(msg) || commandName(text) != "" || !addressed {
		return false
	}
	if !a.aiWhitelistedChat(chat) {
		a.sendNonWhitelistedLayaChatReply(c)
		return true
	}
	if text != "" && isDumbLayaQuestion(text, a.botUsername()) {
		a.maybeSendLayaReaction(chat, msg, "dumb_question")
		return true
	}
	if text != "" && a.maybeReplyToDirectLayaInsult(c, text, addressedByReply) {
		return true
	}
	if text != "" && a.maybeReplyToHardLayaRequest(c, text) {
		return true
	}
	if a.maybeReplyAIToLayaMention(c, inputText) {
		return true
	}

	quoteText := ""
	if a.quotesEnabledForChat(chat) {
		if rec, ok := a.quotes.Random(""); ok {
			quoteText = a.sanitizeRandomLayaReply(sender, a.formatQuote(rec))
		}
	}
	quoteCfg := a.activeQuoteConfigForChat(chat)
	if a.photosEnabledForChat(chat) && quoteCfg.PhotoChance > 0 && rand.Float64() <= quoteCfg.PhotoChance {
		file, caption, ok, err := a.randomPhotoMedia("")
		if err != nil {
			a.log.Warn("mention photo lookup failed", zap.Error(err))
		}
		if ok {
			if quoteText != "" {
				caption = quoteText
			} else {
				caption = a.sanitizeRandomLayaReply(sender, caption)
			}
			if _, err := a.bot.Send(chat, &tele.Photo{File: file, Caption: trimRunes(caption, 1000)}, &tele.SendOptions{ReplyTo: msg, ThreadID: msg.ThreadID, ReplyMarkup: a.layaInlineMarkup(chat, sender)}); err != nil {
				a.log.Warn("mention photo reply failed", zap.Error(err))
			} else {
				a.metrics.IncModeration("mention_reply_photo")
				return true
			}
		}
	}
	if quoteText == "" {
		quoteText = "я здесь."
	}
	if _, err := a.bot.Send(chat, quoteText, &tele.SendOptions{ReplyTo: msg, ThreadID: msg.ThreadID, ReplyMarkup: a.layaInlineMarkup(chat, sender)}); err != nil {
		a.log.Warn("mention reply failed", zap.Error(err))
		return false
	}
	a.metrics.IncModeration("mention_reply")
	return true
}

const nonWhitelistedLayaChatReply = "Извините, я работаю только там, где нужно моему создателю."

func (a *App) maybeReplyToNonWhitelistedLayaChat(c tele.Context) bool {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || chat.Type == tele.ChatPrivate {
		return false
	}
	if a.aiWhitelistedChat(chat) || !layaMessageHasUserContent(msg) {
		return false
	}
	text := strings.TrimSpace(messageText(msg))
	if commandName(text) != "" {
		return false
	}
	addressed := mentionsLaya(text, a.botUsername())
	if !addressed && msg.ReplyTo != nil && msg.ReplyTo.Sender != nil && a.bot != nil && a.bot.Me != nil {
		addressed = msg.ReplyTo.Sender.ID == a.bot.Me.ID && !isForumTopicRootReply(msg)
	}
	if !addressed {
		return false
	}
	a.sendNonWhitelistedLayaChatReply(c)
	return true
}

func (a *App) sendNonWhitelistedLayaChatReply(c tele.Context) {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	opts := &tele.SendOptions{}
	if msg != nil {
		opts.ReplyTo = msg
		opts.ThreadID = msg.ThreadID
	}
	if chat != nil && sender != nil {
		opts.ReplyMarkup = a.layaInlineMarkup(chat, sender)
	}
	var err error
	if a != nil && a.bot != nil && chat != nil {
		_, err = a.bot.Send(chat, nonWhitelistedLayaChatReply, opts)
	} else {
		err = c.Send(nonWhitelistedLayaChatReply)
	}
	if err != nil && a != nil && a.log != nil {
		chatID := int64(0)
		if chat != nil {
			chatID = chat.ID
		}
		a.log.Warn("send non-whitelisted laya chat reply failed", zap.Int64("chat", chatID), zap.Error(err))
	}
	if a != nil && a.metrics != nil {
		a.metrics.IncModeration("laya_ai_not_whitelisted")
	}
}
