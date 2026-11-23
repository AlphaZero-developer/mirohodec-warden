package main

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) sendJoinLog(chat *tele.Chat, user, actor *tele.User, inviteLink *tele.ChatInviteLink, viaFolder bool, source string) {
	fields := []logField{
		{Name: "User", Value: fmt.Sprintf("%s (%d)", displayNameFromUser(user), user.ID)},
		{Name: "Chat", Value: fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)},
	}
	if actor != nil {
		fields = append(fields, logField{Name: "Invited by", Value: fmt.Sprintf("%s (%d)", displayNameFromUser(actor), actor.ID)})
	}
	if inviteLink != nil {
		if inviteLink.Creator != nil {
			fields = append(fields, logField{Name: "Invite link creator", Value: fmt.Sprintf("%s (%d)", displayNameFromUser(inviteLink.Creator), inviteLink.Creator.ID)})
		}
		if strings.TrimSpace(inviteLink.Name) != "" {
			fields = append(fields, logField{Name: "Invite link", Value: inviteLink.Name})
		}
	}
	if viaFolder {
		fields = append(fields, logField{Name: "Via", Value: "chat folder invite"})
	}
	if strings.TrimSpace(source) != "" {
		fields = append(fields, logField{Name: "Source", Value: source})
	}
	a.sendOwnerLog("Join log", fields...)
}

func (a *App) sendWelcomeReaction(chat *tele.Chat, user, actor *tele.User, threadID int) {
	if a == nil || a.bot == nil || chat == nil || user == nil {
		return
	}
	name := displayNameFromUser(user)
	fallback := fmt.Sprintf("Привет, %s. Заходи, только без героизма.", name)
	prompt := fmt.Sprintf("Событие: в чат %q зашел пользователь %s. Поздоровайся коротко в стиле Лайи: приветствие плюс один колкий, но не злой комментарий.", textOrDash(chat.Title), name)
	reply := a.generateLayaEventReply(actor, prompt, fallback)
	if _, err := a.bot.Send(chat, reply, &tele.SendOptions{ThreadID: threadID}); err != nil {
		if a.log != nil {
			a.log.Warn("send welcome reaction failed", zap.Int64("chat", chat.ID), zap.Int64("user", user.ID), zap.Error(err))
		}
		return
	}
	if a.metrics != nil {
		a.metrics.IncModeration("welcome_reaction")
	}
}

func (a *App) handleAddedToGroup(c tele.Context) error {
	chat := c.Chat()
	actor := c.Sender()
	a.rememberManagedChat(chat)
	if chat == nil || !a.isManagedChat(chat) {
		return nil
	}
	threadID := 0
	if msg := c.Message(); msg != nil {
		threadID = msg.ThreadID
	}
	a.handleBotAddedEvent(chat, actor, threadID, "service_message")
	return nil
}

func (a *App) handleMyChatMember(c tele.Context) error {
	upd := c.ChatMember()
	if upd == nil || upd.Chat == nil || upd.NewChatMember == nil || upd.NewChatMember.User == nil {
		return nil
	}
	a.rememberManagedChat(upd.Chat)
	if !a.isManagedChat(upd.Chat) || !a.isSelfUser(upd.NewChatMember.User) {
		return nil
	}
	if chatMemberBecamePresent(upd.OldChatMember, upd.NewChatMember) {
		a.handleBotAddedEvent(upd.Chat, upd.Sender, 0, "my_chat_member")
	}
	return nil
}

func (a *App) isSelfUser(user *tele.User) bool {
	return a != nil && a.bot != nil && a.bot.Me != nil && user != nil && user.ID == a.bot.Me.ID
}

func (a *App) handleBotAddedEvent(chat *tele.Chat, actor *tele.User, threadID int, source string) {
	if chat == nil || !a.claimBotAdded(chat.ID, time.Now().UTC()) {
		return
	}
	fields := []logField{
		{Name: "Chat", Value: fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)},
	}
	if actor != nil {
		fields = append(fields, logField{Name: "Added by", Value: fmt.Sprintf("%s (%d)", displayNameFromUser(actor), actor.ID)})
	}
	if source != "" {
		fields = append(fields, logField{Name: "Source", Value: source})
	}
	a.sendOwnerLog("Bot added", fields...)
	a.sendBotAddedReaction(chat, actor, threadID)
}

func (a *App) claimBotAdded(chatID int64, now time.Time) bool {
	a.botAddedMu.Lock()
	defer a.botAddedMu.Unlock()
	if last := a.lastBotAddedChats[chatID]; !last.IsZero() && now.Sub(last) < time.Minute {
		return false
	}
	a.lastBotAddedChats[chatID] = now
	return true
}

func (a *App) sendBotAddedReaction(chat *tele.Chat, actor *tele.User, threadID int) {
	fallback := "Я сейчас создателю всё расскажу. Он вам такой разбор устроит, что психиатрический центр покажется санаторием."
	prompt := fmt.Sprintf("Событие: тебя добавили в чат %q. Скажи короткую реакцию в стиле Лайи. Смысл: ты сейчас все расскажешь создателю, а он устроит им жесткий разбор. Без угроз физической расправы.", textOrDash(chat.Title))
	reply := a.generateLayaEventReply(actor, prompt, fallback)
	if _, err := a.bot.Send(chat, reply, &tele.SendOptions{ThreadID: threadID}); err != nil {
		if a.log != nil {
			a.log.Warn("send bot-added reaction failed", zap.Int64("chat", chat.ID), zap.Error(err))
		}
		return
	}
	if a.metrics != nil {
		a.metrics.IncModeration("bot_added_reaction")
	}
}

func (a *App) generateLayaEventReply(actor *tele.User, prompt, fallback string) string {
	fallback = strings.TrimSpace(fallback)
	if fallback == "" {
		fallback = "Ну привет. Уже интересно, кто это придумал."
	}
	cfg := a.config().AI
	if !cfg.Enabled || a.currentAI() == nil {
		return fallback
	}
	cfg.SystemPrompt = strings.TrimSpace(cfg.SystemPrompt + "\n\n" + layaEventSystemPrompt)
	cfg.FallbackReply = fallback
	cfg.MaxReplyChars = minInt(maxAIReplyChars(cfg), 500)
	reply, _, _, err := a.generateAIReply(cfg, actor, prompt)
	if err != nil || strings.TrimSpace(reply) == "" {
		return fallback
	}
	return trimRunes(reply, maxAIReplyChars(cfg))
}
