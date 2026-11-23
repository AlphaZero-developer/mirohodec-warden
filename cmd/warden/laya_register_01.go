package main

import (
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

const (
	layaRegisterTTL             = 10 * time.Minute
	layaRegisterMaxAddressRunes = 40
	layaAddressFilterPrompt     = ""
)

type layaRegisterStage int

const (
	layaRegisterStageGender layaRegisterStage = iota + 1
	layaRegisterStageAddress
)

type layaRegisterSession struct {
	UserID          int64
	ChatID          int64
	ThreadID        int
	PromptMessageID int
	Username        string
	DisplayName     string
	Gender          string
	Stage           layaRegisterStage
	StartedAt       time.Time
	UpdatedAt       time.Time
}

func (a *App) handleLayaRegister(c tele.Context) error {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || sender.ID == 0 {
		return c.Send(a.t("no_message"))
	}
	if chat.Type == tele.ChatPrivate {
		return c.Send("Регистрация обращения работает в группе. Напиши /register там, и я спрошу пол и как к тебе обращаться.")
	}
	if !a.isManagedChat(chat) {
		return c.Send(a.t("managed_chat_required"))
	}

	session := layaRegisterSession{
		UserID:      sender.ID,
		ChatID:      chat.ID,
		ThreadID:    msg.ThreadID,
		Username:    sender.Username,
		DisplayName: displayNameFromUser(sender),
		Stage:       layaRegisterStageGender,
		StartedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}
	a.setLayaRegisterSession(session)

	text := fmt.Sprintf("%s, записываю твой Telegram ID: %d.\nКакой у тебя пол? Ответь одним сообщением: мужской, женский или нейтральный.", displayNameFromUser(sender), sender.ID)
	sent, err := a.bot.Send(chat, text, &tele.SendOptions{ReplyTo: msg, ThreadID: msg.ThreadID})
	if err != nil {
		a.clearLayaRegisterSession(chat.ID, sender.ID)
		return err
	}
	if sent != nil {
		session.PromptMessageID = sent.ID
		a.setLayaRegisterSession(session)
	}
	if a.metrics != nil {
		a.metrics.IncModeration("laya_register_start")
	}
	return nil
}

func (a *App) maybeHandleLayaRegistrationAnswer(c tele.Context) bool {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || sender.ID == 0 || chat.Type == tele.ChatPrivate {
		return false
	}
	text := strings.TrimSpace(messageText(msg))
	if text == "" || commandName(text) != "" {
		return false
	}

	session, ok := a.layaRegisterSession(chat.ID, sender.ID)
	if !ok {
		return false
	}
	if time.Since(session.UpdatedAt) > layaRegisterTTL {
		a.clearLayaRegisterSession(chat.ID, sender.ID)
		return false
	}
	if session.ThreadID != msg.ThreadID {
		return false
	}

	switch session.Stage {
	case layaRegisterStageGender:
		return a.handleLayaRegisterGender(c, session, text)
	case layaRegisterStageAddress:
		return a.handleLayaRegisterAddress(c, session, text)
	default:
		a.clearLayaRegisterSession(chat.ID, sender.ID)
		return false
	}
}

func (a *App) handleLayaRegisterGender(c tele.Context, session layaRegisterSession, text string) bool {
	gender, ok := parseLayaGender(text)
	if !ok {
		a.replyLayaRegister(c, "Не поняла пол. Напиши: мужской, женский или нейтральный.")
		return true
	}
	session.Gender = gender
	session.Stage = layaRegisterStageAddress
	session.UpdatedAt = time.Now().UTC()
	a.setLayaRegisterSession(session)

	reply := "Приняла. Теперь напиши, как к тебе обращаться.\nМожно имя, ник или любое обращение."
	if sent := a.replyLayaRegister(c, reply); sent != nil {
		session.PromptMessageID = sent.ID
		a.setLayaRegisterSession(session)
	}
	return true
}

func (a *App) handleLayaRegisterAddress(c tele.Context, session layaRegisterSession, text string) bool {
	address, err := normalizeLayaAddress(text)
	if err != nil {
		a.replyLayaRegister(c, err.Error())
		return true
	}
	profile := LayaUserProfile{
		UserID:        session.UserID,
		Username:      session.Username,
		DisplayName:   session.DisplayName,
		Gender:        session.Gender,
		Address:       address,
		UpdatedUnix:   time.Now().UTC().Unix(),
		UpdatedChatID: session.ChatID,
	}
	if err := a.store.SetLayaUserProfile(profile); err != nil {
		if a.log != nil {
			a.log.Warn("save laya user profile failed", zap.Int64("user", session.UserID), zap.Error(err))
		}
		a.replyLayaRegister(c, "Не смогла сохранить профиль. Да, база данных выбрала драму.")
		return true
	}

	a.clearLayaRegisterSession(session.ChatID, session.UserID)
	a.clearAIConversation(c.Chat(), c.Sender())
	a.replyLayaRegister(c, fmt.Sprintf("Готово. Для ID %d сохранено: пол — %s, обращение — %s.", session.UserID, layaGenderLabel(session.Gender), address))
	if a.metrics != nil {
		a.metrics.IncModeration("laya_register_done")
	}
	return true
}

func (a *App) replyLayaRegister(c tele.Context, text string) *tele.Message {
	msg := c.Message()
	chat := c.Chat()
	if chat == nil || strings.TrimSpace(text) == "" {
		return nil
	}
	opts := &tele.SendOptions{}
	if msg != nil {
		opts.ReplyTo = msg
		opts.ThreadID = msg.ThreadID
	}
	sent, err := a.bot.Send(chat, text, opts)
	if err != nil && a.log != nil {
		a.log.Warn("send laya register reply failed", zap.Error(err))
	}
	return sent
}

func (a *App) setLayaRegisterSession(session layaRegisterSession) {
	if a == nil || session.ChatID == 0 || session.UserID == 0 {
		return
	}
	a.layaRegisterMu.Lock()
	defer a.layaRegisterMu.Unlock()
	if a.layaRegister == nil {
		a.layaRegister = make(map[string]layaRegisterSession)
	}
	a.layaRegister[layaRegisterKey(session.ChatID, session.UserID)] = session
}
