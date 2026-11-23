package main

import (
	"strings"

	tele "gopkg.in/telebot.v3"
)

func (a *App) handleMafiaProfileInput(c tele.Context) bool {
	if a == nil || a.store == nil || c == nil {
		return false
	}
	chat := c.Chat()
	msg := c.Message()
	sender := c.Sender()
	if chat == nil || chat.Type != tele.ChatPrivate || msg == nil || sender == nil {
		return false
	}
	user, ok := a.store.MafiaUser(sender.ID)
	if !ok {
		return false
	}
	settings := normalizeMafiaUserSettings(user.Settings)
	mode := strings.TrimSpace(settings.ProfileInputMode)
	if mode == "" {
		return false
	}
	text := strings.TrimSpace(messageText(msg))
	switch mode {
	case mafiaProfileInputName:
		name, warn := validateMafiaCharacterName(text)
		if warn != "" {
			_ = c.Send(warn)
			return true
		}
		settings.PendingCharacterName = name
		settings.ProfileInputMode = mafiaProfileInputBackstory
		if err := a.store.SetMafiaUserSettings(sender.ID, settings); err != nil {
			_ = c.Send("⛔ " + a.localizeMafiaErr(err))
			return true
		}
		_ = c.Send(mafiaProfileBackstoryPromptText())
		return true
	case mafiaProfileInputBackstory:
		backstory, warn := validateMafiaBackstory(text)
		if warn != "" {
			_ = c.Send(warn)
			return true
		}
		name := strings.TrimSpace(settings.PendingCharacterName)
		if name == "" {
			settings.ProfileInputMode = mafiaProfileInputName
			_ = a.store.SetMafiaUserSettings(sender.ID, settings)
			_ = c.Send("⚠️ Имя персонажа потерялось.\n\nВведите имя ещё раз.")
			return true
		}
		user.Profile.CharacterName = name
		user.Profile.Backstory = backstory
		if err := a.store.SetMafiaProfile(sender.ID, user.Profile); err != nil {
			_ = c.Send("⛔ " + a.localizeMafiaErr(err))
			return true
		}
		clearMafiaProfileInput(&settings)
		_ = a.store.SetMafiaUserSettings(sender.ID, settings)
		updated, _ := a.store.MafiaUser(sender.ID)
		_ = c.Send(mafiaProfileCreatedText(updated), mafiaProfileMarkup(true))
		return true
	case mafiaProfileInputEditName:
		if !a.mafiaCanEditProfile(sender.ID) {
			clearMafiaProfileInput(&settings)
			_ = a.store.SetMafiaUserSettings(sender.ID, settings)
			_ = c.Send("⛔ Нельзя менять персонажа во время активной партии.\n\nДождитесь окончания игры.")
			return true
		}
		name, warn := validateMafiaCharacterName(text)
		if warn != "" {
			_ = c.Send(warn)
			return true
		}
		user.Profile.CharacterName = name
		if err := a.store.SetMafiaProfile(sender.ID, user.Profile); err != nil {
			_ = c.Send("⛔ " + a.localizeMafiaErr(err))
			return true
		}
		clearMafiaProfileInput(&settings)
		_ = a.store.SetMafiaUserSettings(sender.ID, settings)
		updated, _ := a.store.MafiaUser(sender.ID)
		_ = c.Send("✅ Имя обновлено.\n\n"+mafiaProfileText(updated), mafiaProfileMarkup(true))
		return true
	case mafiaProfileInputEditBackstory:
		if !a.mafiaCanEditProfile(sender.ID) {
			clearMafiaProfileInput(&settings)
			_ = a.store.SetMafiaUserSettings(sender.ID, settings)
			_ = c.Send("⛔ Нельзя менять персонажа во время активной партии.\n\nДождитесь окончания игры.")
			return true
		}
		backstory, warn := validateMafiaBackstory(text)
		if warn != "" {
			_ = c.Send(warn)
			return true
		}
		user.Profile.Backstory = backstory
		if err := a.store.SetMafiaProfile(sender.ID, user.Profile); err != nil {
			_ = c.Send("⛔ " + a.localizeMafiaErr(err))
			return true
		}
		clearMafiaProfileInput(&settings)
		_ = a.store.SetMafiaUserSettings(sender.ID, settings)
		updated, _ := a.store.MafiaUser(sender.ID)
		_ = c.Send("✅ Предыстория обновлена.\n\n"+mafiaProfileText(updated), mafiaProfileMarkup(true))
		return true
	default:
		clearMafiaProfileInput(&settings)
		_ = a.store.SetMafiaUserSettings(sender.ID, settings)
		return false
	}
}

func (a *App) callbackMafiaProfileOpen(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	user, ok := a.store.MafiaUser(sender.ID)
	if !ok || !mafiaProfileComplete(user) {
		return a.startMafiaRegistrationIntro(c)
	}
	return c.Send(mafiaProfileText(user), mafiaProfileMarkup(a.mafiaCanEditProfile(sender.ID)))
}

func (a *App) callbackMafiaAchievements(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	user, ok := a.store.MafiaUser(sender.ID)
	if !ok {
		return a.startMafiaRegistrationIntro(c)
	}
	return c.Send(mafiaAchievementsText(user), mafiaAchievementsMarkup())
}

func (a *App) callbackMafiaProfileEditName(c tele.Context) error {
	return a.startMafiaProfileEdit(c, mafiaProfileInputEditName, "✏️ Введите новое имя персонажа.")
}

func (a *App) callbackMafiaProfileEditBackstory(c tele.Context) error {
	return a.startMafiaProfileEdit(c, mafiaProfileInputEditBackstory, "📜 Введите новую короткую предысторию персонажа.")
}

func (a *App) startMafiaProfileEdit(c tele.Context, mode, prompt string) error {
	if chat := c.Chat(); chat == nil || chat.Type != tele.ChatPrivate {
		return c.Send("⛔ Персонаж настраивается в ЛС с Лайей.")
	}
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	user, ok := a.store.MafiaUser(sender.ID)
	if !ok || !mafiaProfileComplete(user) {
		return a.startMafiaProfileRegistration(c)
	}
	if !a.mafiaCanEditProfile(sender.ID) {
		return c.Send("⛔ Нельзя менять персонажа во время активной партии.\n\nДождитесь окончания игры.")
	}
	settings := normalizeMafiaUserSettings(user.Settings)
	clearMafiaProfileInput(&settings)
	settings.ProfileInputMode = mode
	if err := a.store.SetMafiaUserSettings(sender.ID, settings); err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	return c.Send(prompt)
}
