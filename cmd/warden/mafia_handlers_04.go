package main

import (
	"errors"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) respondMafiaCallback(c tele.Context, text string, alert bool) {
	if c == nil || c.Callback() == nil {
		return
	}
	if err := c.Respond(&tele.CallbackResponse{Text: text, ShowAlert: alert}); err != nil && a != nil && a.log != nil {
		a.log.Warn("callback respond failed", zap.Error(err))
	}
}

type mafiaVoteCallbackPayload struct {
	roomID   string
	phaseKey string
	target   string
}

func parseMafiaVoteCallbackData(data string) (mafiaVoteCallbackPayload, bool) {
	parts := strings.Split(strings.TrimSpace(data), ":")
	if len(parts) < 3 || parts[0] != "mafia" {
		return mafiaVoteCallbackPayload{}, false
	}
	payload := mafiaVoteCallbackPayload{roomID: strings.TrimSpace(parts[2]), target: "skip"}
	switch parts[1] {
	case "vote":
		switch len(parts) {
		case 4:
			payload.target = strings.TrimSpace(parts[3])
		case 5:
			payload.phaseKey = strings.TrimSpace(parts[3])
			payload.target = strings.TrimSpace(parts[4])
		default:
			return mafiaVoteCallbackPayload{}, false
		}
	case "voteskip":
		switch len(parts) {
		case 3:
		case 4:
			payload.phaseKey = strings.TrimSpace(parts[3])
		default:
			return mafiaVoteCallbackPayload{}, false
		}
	default:
		return mafiaVoteCallbackPayload{}, false
	}
	if payload.roomID == "" || payload.target == "" {
		return mafiaVoteCallbackPayload{}, false
	}
	return payload, true
}

func (a *App) callbackMafiaRegister(c tele.Context) error {
	if chat := c.Chat(); chat == nil || chat.Type != tele.ChatPrivate {
		return c.Send("⛔ Регистрация проходит в ЛС с Лайей. Открой бота и отправь /start.")
	}
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	return a.startMafiaRegistrationIntro(c)
}

func (a *App) startMafiaRegistrationIntro(c tele.Context) error {
	if chat := c.Chat(); chat == nil || chat.Type != tele.ChatPrivate {
		return c.Send("⛔ Регистрация проходит в ЛС с Лайей. Открой бота и отправь /start.")
	}
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	if !a.mafiaCanEditProfile(sender.ID) {
		return c.Send("⛔ Нельзя менять персонажа во время активной партии.\n\nДождитесь окончания игры.")
	}
	if _, err := a.store.RegisterMafiaUser(sender, time.Now().UTC()); err != nil {
		return c.Send("⛔ Не удалось зарегистрировать игрока.")
	}
	return c.Send(mafiaRegistrationIntroText(), mafiaRegistrationIntroMarkup())
}

func (a *App) callbackMafiaRegistrationIntroAck(c tele.Context) error {
	return a.startMafiaProfileRegistration(c)
}

const (
	mafiaProfileInputName          = "profile_name"
	mafiaProfileInputBackstory     = "profile_backstory"
	mafiaProfileInputEditName      = "profile_edit_name"
	mafiaProfileInputEditBackstory = "profile_edit_backstory"

	mafiaProfileNameMaxRunes      = 64
	mafiaProfileBackstoryMaxRunes = 500
)

func (a *App) handleMafiaRegistrationError(c tele.Context, err error) error {
	if errors.Is(err, errMafiaProfileIncomplete) {
		if chat := c.Chat(); chat != nil && chat.Type == tele.ChatPrivate {
			return a.startMafiaRegistrationIntro(c)
		}
		return c.Send("⛔ Сначала создайте персонажа в ЛС с Лайей.\n\nОткройте бота и нажмите /start.")
	}
	if errors.Is(err, errMafiaUserNotRegistered) {
		return c.Send("⛔ Сначала откройте ЛС с Лайей и отправьте /start.")
	}
	return c.Send("⛔ " + a.localizeMafiaErr(err))
}

func (a *App) startMafiaProfileRegistration(c tele.Context) error {
	if chat := c.Chat(); chat == nil || chat.Type != tele.ChatPrivate {
		return c.Send("⛔ Регистрация проходит в ЛС с Лайей. Открой бота и отправь /start.")
	}
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	if !a.mafiaCanEditProfile(sender.ID) {
		return c.Send("⛔ Нельзя менять персонажа во время активной партии.\n\nДождитесь окончания игры.")
	}
	user, err := a.store.RegisterMafiaUser(sender, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ Не удалось зарегистрировать игрока.")
	}
	settings := normalizeMafiaUserSettings(user.Settings)
	settings.ProfileInputMode = mafiaProfileInputName
	settings.PendingCharacterName = ""
	settings.GeneratedCharacterName = ""
	settings.GeneratedBackstory = ""
	if err := a.store.SetMafiaUserSettings(sender.ID, settings); err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	return c.Send(mafiaProfileRegistrationPromptText(), mafiaProfileRegistrationMarkup())
}
