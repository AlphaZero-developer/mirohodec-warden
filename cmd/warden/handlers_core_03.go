package main

import (
	"crypto/subtle"
	"fmt"
	"strings"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) handleAuth(c tele.Context) error {
	msg := c.Message()
	chat := c.Chat()
	if msg == nil || chat == nil || c.Sender() == nil {
		return c.Send(a.t("no_message"))
	}
	if chat.Type != tele.ChatPrivate {
		return c.Send(a.t("auth_private_only"))
	}
	if a.store != nil && a.store.HasOwner() && !a.isAdmin(c) {
		return nil
	}

	key := strings.TrimSpace(strings.Join(c.Args(), " "))
	expected := strings.TrimSpace(a.config().Security.AuthKey)
	if expected == "" {
		return c.Send(a.t("auth_disabled"))
	}
	if key == "" {
		return c.Send(a.t("usage_auth"))
	}
	if subtle.ConstantTimeCompare([]byte(key), []byte(expected)) != 1 {
		return c.Send(a.t("auth_invalid"))
	}

	return a.registerOwner(c)
}

func (a *App) handleText(c tele.Context) error {
	msg := c.Message()
	chat := c.Chat()
	if msg == nil || chat == nil {
		return nil
	}
	if chat.Type != tele.ChatPrivate {
		if a.maybeHandleMafiaGroupMessage(c) {
			return nil
		}
		return nil
	}

	key := strings.TrimSpace(msg.Text)
	expected := strings.TrimSpace(a.config().Security.AuthKey)
	if key == "" {
		return nil
	}
	if strings.HasPrefix(key, "/") {
		return nil
	}
	if expected != "" && subtle.ConstantTimeCompare([]byte(key), []byte(expected)) == 1 {
		if a.store != nil && a.store.HasOwner() && !a.isAdmin(c) {
			return nil
		}
		return a.registerOwner(c)
	}
	if a.handleMafiaProfileInput(c) {
		return nil
	}
	if a.handleLayaReplyKeyboard(c) {
		return nil
	}
	if a.handleMafiaReplyKeyboard(c) {
		return nil
	}
	if a.maybeHandleLayaFun(c) {
		return nil
	}
	return a.handlePrivateAIMessage(c)
}

func (a *App) registerOwner(c tele.Context) error {
	chat := c.Chat()
	sender := c.Sender()
	if chat == nil || sender == nil {
		return c.Send(a.t("no_message"))
	}

	if err := a.store.RegisterOwner(sender, chat); err != nil {
		a.log.Warn("owner auth save failed", zap.Int64("user", sender.ID), zap.Error(err))
		return c.Send(a.t("auth_save_failed", err))
	}
	a.metrics.IncModeration("auth")
	a.sendOwnerLog("Owner auth", logField{"Owner", fmt.Sprintf("%s (%d)", displayNameFromUser(sender), sender.ID)}, logField{"Log chat", fmt.Sprintf("%d", chat.ID)})
	a.sendWebPanelLink("auth")
	a.sendMafiaMiniAppLinkToCreator("auth")
	go a.syncCommandMenus()
	return c.Send(a.t("auth_done"))
}

func (a *App) syncCommandMenus() {
	if err := syncBotCommandMenus(a.bot, a.store, a.log); err != nil && a.log != nil {
		a.log.Warn("sync bot command menus failed", zap.Error(err))
	}
}

func (a *App) handleMyID(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}

	lines := []string{
		a.t("myid_user", sender.ID),
	}
	if chat := c.Chat(); chat != nil {
		lines = append(lines, a.t("myid_chat", chat.ID))
		if msg := c.Message(); msg != nil {
			lines = append(lines, a.t("myid_topic", msg.ThreadID))
		}
	}
	lines = append(lines, a.t("myid_hint"))
	return c.Send(strings.Join(lines, "\n"))
}

func (a *App) handleChatID(c tele.Context) error {
	chat := c.Chat()
	if chat == nil {
		return c.Send(a.t("no_chat"))
	}
	return c.Send(a.t("chat_id", chat.ID))
}

func (a *App) handleTopicID(c tele.Context) error {
	msg := c.Message()
	if msg == nil {
		return c.Send(a.t("no_message"))
	}
	return c.Send(a.t("topic_id", msg.ThreadID))
}

func (a *App) handleReload(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}
	if err := a.reloadNow(); err != nil {
		return c.Send(a.t("reload_failed", err))
	}
	a.sendWebPanelLink("reload")
	a.sendMafiaMiniAppLinkToCreator("reload")
	return c.Send(a.t("reload_done"))
}

func (a *App) handleUserJoined(c tele.Context) error {
	msg := c.Message()
	chat := c.Chat()
	a.rememberManagedChat(chat)
	if msg == nil || chat == nil || !a.isManagedChat(chat) {
		return nil
	}

	for _, user := range joinedUsersFromMessage(msg) {
		a.handleJoinedUserEvent(chat, user, msg.Sender, nil, false, msg.ThreadID, "service_message")
	}
	return nil
}

func (a *App) handleChatMember(c tele.Context) error {
	upd := c.ChatMember()
	if upd == nil || upd.Chat == nil || upd.NewChatMember == nil || upd.NewChatMember.User == nil {
		return nil
	}
	a.rememberManagedChat(upd.Chat)
	if !a.isManagedChat(upd.Chat) {
		return nil
	}

	if chatMemberBecamePresent(upd.OldChatMember, upd.NewChatMember) {
		a.handleJoinedUserEvent(upd.Chat, upd.NewChatMember.User, upd.Sender, upd.InviteLink, upd.ViaFolderLink, 0, "chat_member")
	}
	return nil
}
