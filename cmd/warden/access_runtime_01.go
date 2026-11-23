package main

import (
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) requireAdmin(c tele.Context) bool {
	if a.isAdmin(c) {
		return true
	}
	if err := c.Send(a.t("no_admin")); err != nil {
		a.log.Warn("send no-admin notice failed", zap.Error(err))
	}
	return false
}

func (a *App) shouldIgnoreCommand(c tele.Context) bool {
	cfg := a.config()
	if !cfg.Security.IgnoreNonAdminCommands {
		return false
	}

	msg := c.Message()
	if msg == nil || strings.TrimSpace(msg.Text) == "" {
		return false
	}

	cmd := commandName(msg.Text)
	if cmd == "" {
		return false
	}
	if cmd == "auth" {
		return false
	}
	if a.isPublicCommand(cmd) {
		return false
	}
	if chat := c.Chat(); chat != nil && chat.Type == tele.ChatPrivate && a.store != nil && a.store.HasOwner() && !a.isAdmin(c) {
		return true
	}
	if cmd == "myid" || cmd == "id" {
		chat := c.Chat()
		return chat != nil && chat.Type != tele.ChatPrivate && !a.isAdmin(c)
	}
	return !a.isAdmin(c)
}

func (a *App) handleUnknownCommand(c tele.Context) bool {
	msg := c.Message()
	if msg == nil {
		return false
	}
	cmd := commandName(msg.Text)
	if cmd == "" || knownCommand(cmd) {
		return false
	}
	if err := c.Send("я не хочу больше с тобой разговаривать"); err != nil && a.log != nil {
		a.log.Warn("send unknown command reply failed", zap.Error(err))
	}
	return true
}

func commandName(text string) string {
	text = strings.TrimSpace(text)
	if !strings.HasPrefix(text, "/") {
		return ""
	}
	first := strings.Fields(text)
	if len(first) == 0 {
		return ""
	}
	cmd := strings.TrimPrefix(first[0], "/")
	if at := strings.Index(cmd, "@"); at >= 0 {
		cmd = cmd[:at]
	}
	return strings.ToLower(cmd)
}

func knownCommand(cmd string) bool {
	switch strings.ToLower(strings.TrimPrefix(strings.TrimSpace(cmd), "/")) {
	case "start", "help", "admin", "auth", "myid", "id", "chatid", "topicid",
		"menu",
		"jail", "rehab", "ward", "палата", "mute", "unmute", "banforever", "unbanforever", "unban",
		"all", "status", "profile", "профиль", "passport", "паспорт", "activity", "topactive", "karma", "topkarma",
		"shop", "buy", "event", "week", "selfmute", "selfjail", "selfrehab", "selfban",
		"register", "mafia", "mafia_bind", "mafia_chat", "mafia_status", "mafia_leave", "mafia_join", "mafia_find", "will",
		"warn", "warnings", "unwarn", "clearwarns",
		"spamlog", "spamstatus", "spampardon",
		"addphoto", "photo", "photos", "delphoto",
		"addquote", "quote", "quotes", "delquote",
		"faq", "faqs", "addfaq", "delfaq",
		"escape", "bribe", "job", "hire", "donate",
		"idea", "bug", "reportbug", "appeal", "clearhistory", "aitest", "cipher", "laya_cipher", "шифр", "reload":
		return true
	default:
		return false
	}
}

func isAppealCommandMessage(msg *tele.Message) bool {
	return msg != nil && commandName(msg.Text) == "appeal"
}

func (a *App) userHasActiveRestriction(chatID, userID int64, now time.Time) bool {
	if a == nil || a.store == nil {
		return false
	}
	if _, ok := a.store.ActiveJail(chatID, userID, now); ok {
		return true
	}
	if _, ok := a.store.ActiveMute(chatID, userID, now); ok {
		return true
	}
	if _, ok := a.store.Ban(chatID, userID); ok {
		return true
	}
	return false
}

func (a *App) deleteRestrictedAppealMessage(c tele.Context, chat *tele.Chat, sender *tele.User) {
	if c == nil {
		return
	}
	if err := c.Delete(); err != nil {
		if a != nil && a.log != nil {
			chatID := int64(0)
			userID := int64(0)
			if chat != nil {
				chatID = chat.ID
			}
			if sender != nil {
				userID = sender.ID
			}
			a.log.Warn("delete restricted appeal message failed", zap.Int64("chat", chatID), zap.Int64("user", userID), zap.Error(err))
		}
		return
	}
	if a != nil && a.metrics != nil {
		a.metrics.IncModeration("appeal_delete")
	}
}

func (a *App) isAdmin(c tele.Context) bool {
	user := c.Sender()
	if user == nil {
		return false
	}
	if a.isConfiguredAdmin(user.ID) {
		return true
	}

	chat := c.Chat()
	if !a.config().Security.TrustTelegramAdmins || chat == nil || chat.Type == tele.ChatPrivate {
		return false
	}

	member, err := a.bot.ChatMemberOf(chat, user)
	if err != nil {
		a.log.Warn("admin check failed", zap.Int64("chat", chat.ID), zap.Int64("user", user.ID), zap.Error(err))
		return false
	}
	return member.Role == tele.Creator || member.Role == tele.Administrator
}

func (a *App) ensureModeratable(chat *tele.Chat, userID int64) error {
	if a.bot.Me != nil && userID == a.bot.Me.ID {
		return errors.New(a.t("err_self"))
	}
	if a.isProtectedUser(userID) {
		return errors.New(a.t("err_protected_user"))
	}
	if !a.config().Security.ProtectAdmins {
		return nil
	}
	if a.isConfiguredAdmin(userID) {
		return errors.New(a.t("err_config_admin"))
	}

	member, err := a.bot.ChatMemberOf(chat, &tele.User{ID: userID})
	if err != nil {
		return nil
	}
	if member.Role == tele.Creator || member.Role == tele.Administrator {
		return errors.New(a.t("err_telegram_admin"))
	}
	return nil
}
