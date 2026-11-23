package main

import (
	"errors"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func (a *App) isConfiguredAdmin(userID int64) bool {
	if a.store != nil && a.store.IsOwner(userID) {
		return true
	}
	adminIDs, _ := a.adminIDs.Load().(map[int64]struct{})
	_, ok := adminIDs[userID]
	return ok
}

func (a *App) isLayaFather(userID int64) bool {
	if userID == 0 {
		return false
	}
	if userID == a.activeLayaConfig().FatherUserID {
		return true
	}
	rel, ok := a.layaRelationshipForUserID(userID)
	return layaRelationshipAllowsRole(rel, ok, layaRoleFather)
}

func (a *App) isLayaMother(userID int64) bool {
	if userID == 0 {
		return false
	}
	if userID == a.activeLayaConfig().MotherUserID {
		return true
	}
	rel, ok := a.layaRelationshipForUserID(userID)
	return layaRelationshipAllowsRole(rel, ok, layaRoleMother)
}

func (a *App) isLayaBrother(userID int64) bool {
	if userID == 0 {
		return false
	}
	if userID == a.activeLayaConfig().BrotherUserID {
		return true
	}
	rel, ok := a.layaRelationshipForUserID(userID)
	return layaRelationshipAllowsRole(rel, ok, layaRoleBrother)
}

func (a *App) isLayaCreator(userID int64) bool {
	if userID == 0 {
		return false
	}
	if userID == a.activeLayaConfig().CreatorUserID {
		return true
	}
	rel, ok := a.layaRelationshipForUserID(userID)
	return layaRelationshipAllowsRole(rel, ok, layaRoleCreator)
}

func (a *App) isPublicCommand(cmd string) bool {
	cmd = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(cmd), "/"))
	if cmd == "" {
		return false
	}
	if cmd == "profile" || cmd == "профиль" {
		return true
	}
	for _, allowed := range a.config().Security.PublicCommands {
		if strings.ToLower(strings.TrimSpace(allowed)) == cmd {
			return true
		}
	}
	return false
}

func (a *App) isProtectedUser(userID int64) bool {
	for _, id := range a.config().Security.ProtectedUserIDs {
		if id == userID {
			return true
		}
	}
	return false
}

func (a *App) isSpamExempt(chatID, userID int64) bool {
	if a.isJailExempt(userID) {
		return true
	}
	if a.store != nil {
		if _, ok := a.store.ScriptwriterExemption(chatID, userID); ok {
			return true
		}
	}
	for _, id := range a.config().Spam.ExemptUserIDs {
		if id == userID {
			return true
		}
	}
	return false
}

func (a *App) isJailExempt(userID int64) bool {
	return a.isConfiguredAdmin(userID) || a.isProtectedUser(userID)
}

func (a *App) isManagedChat(chat *tele.Chat) bool {
	if chat == nil {
		return false
	}
	if a.store != nil {
		if settings, ok := a.store.ChatSettings(chat.ID); ok {
			return settings.Enabled
		}
	}
	cfg := a.config()
	if cfg.Chat.ID != 0 {
		return chat.ID == cfg.Chat.ID
	}
	return chat.Type == tele.ChatGroup || chat.Type == tele.ChatSuperGroup
}

func (a *App) commandChat(c tele.Context) (*tele.Chat, error) {
	chat := c.Chat()
	if chat != nil && a.isManagedChat(chat) {
		return chat, nil
	}
	if cfg := a.config(); cfg.Chat.ID != 0 {
		return &tele.Chat{ID: cfg.Chat.ID, Type: tele.ChatSuperGroup}, nil
	}
	return nil, errors.New(a.t("managed_chat_required"))
}

type targetUser struct {
	ID       int64
	Username string
	Name     string
	IsBot    bool
}

func (a *App) targetFromArgsOrSender(chatID int64, c tele.Context, args []string) (targetUser, []string, error) {
	if msg := c.Message(); msg != nil && msg.ReplyTo != nil && msg.ReplyTo.Sender != nil {
		return a.targetFromArgs(chatID, c, args)
	}
	if len(args) > 0 {
		return a.targetFromArgs(chatID, c, args)
	}
	sender := c.Sender()
	if sender == nil {
		return targetUser{}, nil, errors.New("missing target")
	}
	return targetUser{
		ID:       sender.ID,
		Username: sender.Username,
		Name:     displayNameFromUser(sender),
		IsBot:    sender.IsBot,
	}, args, nil
}
