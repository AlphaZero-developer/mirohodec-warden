package main

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) mafiaService() *MafiaService {
	if a == nil {
		return NewMafiaService(nil)
	}
	var cfg Config
	if loaded := a.cfg.Load(); loaded != nil {
		if typed, ok := loaded.(Config); ok {
			cfg = typed
		}
	}
	return NewMafiaServiceWithAI(a.store, a.currentAI(), cfg.AI)
}

func (a *App) ensureMafiaRegistered(c tele.Context) (MafiaRegisteredUser, error) {
	sender := c.Sender()
	if sender == nil || sender.ID == 0 {
		return MafiaRegisteredUser{}, errMafiaUserNotRegistered
	}
	if user, ok := a.store.MafiaUser(sender.ID); ok {
		if !mafiaProfileComplete(user) {
			return user, errMafiaProfileIncomplete
		}
		return user, nil
	}
	if chat := c.Chat(); chat != nil && chat.Type == tele.ChatPrivate {
		user, err := a.store.RegisterMafiaUser(sender, time.Now().UTC())
		if err != nil {
			return MafiaRegisteredUser{}, err
		}
		return user, errMafiaProfileIncomplete
	}
	return MafiaRegisteredUser{}, errMafiaUserNotRegistered
}

func (a *App) handleMafia(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	if room, ok := a.activeMafiaRoomForUser(sender.ID); ok {
		if err := c.Send(a.renderMafiaRoomForUser(room, sender.ID), a.mafiaMenuMarkup(c, mafiaRoomMarkup(room, sender.ID), sender.ID)); err != nil {
			return err
		}
		return a.maybeSendMafiaMiniAppEntry(c, room.ID)
	}
	user, ok := a.store.MafiaUser(sender.ID)
	if !ok || !mafiaProfileComplete(user) {
		if chat := c.Chat(); chat != nil && chat.Type == tele.ChatPrivate {
			return a.startMafiaRegistrationIntro(c)
		}
		return c.Send("🎭 Мафия\n\nПеред игрой нужно зарегистрироваться.\nЭто нужно, чтобы Лайя могла отправлять роли и личные игровые действия.", a.mafiaMenuMarkup(c, mafiaRegisterMarkup(c.Chat()), sender.ID))
	}
	rooms := a.store.MafiaRoomsForUser(sender.ID)
	if err := c.Send("🎭 Мафия\n\nВы не находитесь в активной комнате.", a.mafiaMenuMarkup(c, mafiaHomeMarkup(len(rooms) > 0, mafiaReadyQueueHas(a.store, sender.ID)), sender.ID)); err != nil {
		return err
	}
	return a.maybeSendMafiaMiniAppEntry(c, "")
}

func (a *App) handleMafiaBind(c tele.Context) error {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || chat.Type == tele.ChatPrivate {
		return c.Send("Комнату нужно привязывать в группе: /mafia_bind MF-4821")
	}
	roomID := strings.TrimSpace(strings.Join(c.Args(), " "))
	if roomID == "" {
		return c.Send("Использование: /mafia_bind MF-4821")
	}
	room, err := a.mafiaService().BindGroup(roomID, sender.ID, chat.ID, time.Now().UTC())
	defer func() {
		if err == nil {
			a.notifyMafiaPlayersMissingGroup(room)
		}
	}()
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	return c.Send(fmt.Sprintf("🔗 Группа подключена к комнате %s.\n\nТеперь дневное обсуждение и публичные события будут проходить здесь.", room.ID))
}

func (a *App) handleMafiaJoin(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	roomID := strings.TrimSpace(strings.Join(c.Args(), " "))
	if roomID == "" {
		return c.Send("Использование: /mafia_join MF-4821")
	}
	if _, err := a.ensureMafiaRegistered(c); err != nil {
		return a.handleMafiaRegistrationError(c, err)
	}
	alreadyInRoom := false
	if existing, ok := a.store.MafiaRoom(roomID); ok {
		alreadyInRoom = mafiaRoomPlayerIndex(existing, sender.ID) >= 0
	}
	room, err := a.mafiaService().JoinRoom(roomID, sender, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	if !alreadyInRoom {
		a.announceMafiaLobbyPlayerJoined(room, sender.ID)
	}
	a.sendMafiaGroupAccessIfMissing(sender.ID, room)
	return c.Send(a.renderMafiaRoomForUser(room, sender.ID), a.mafiaMenuMarkup(c, mafiaRoomMarkup(room, sender.ID), sender.ID))
}

func (a *App) handleMafiaChat(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	text := strings.TrimSpace(strings.Join(c.Args(), " "))
	if text == "" {
		return c.Send("Использование: /mafia_chat Сегодня убираем Shadow.")
	}
	room, ok := a.activeMafiaRoomForUser(sender.ID)
	if !ok {
		return c.Send("⛔ Активная комната мафии не найдена.")
	}
	_, msg, recipients, err := a.mafiaService().SendShadowMessage(room.ID, sender.ID, text, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	a.deliverMafiaShadowMessage(room.ID, msg, recipients)
	return c.Send("✅ Сообщение ушло в теневой канал.")
}

func (a *App) handleMafiaStatus(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	rooms := a.store.MafiaRoomsForUser(sender.ID)
	user, registered := a.store.MafiaUser(sender.ID)
	if len(rooms) == 0 {
		if registered && (user.Stats.Games > 0 || user.Stats.RoleReveals > 0) {
			return c.Send(mafiaStatsText(user))
		}
		return c.Send("🎭 У тебя пока нет комнат Мафии.")
	}
	lines := []string{"🎭 Твои комнаты:"}
	for _, room := range rooms {
		lines = append(lines, fmt.Sprintf("%s · %s · игроков %d/%d", room.ID, mafiaPhaseLabel(room.Status), len(room.Players), normalizeMafiaSettings(room.Settings).MaxPlayers))
	}
	if registered {
		lines = append(lines, "", mafiaStatsText(user))
	}
	return c.Send(strings.Join(lines, "\n"))
}

func (a *App) handleMafiaLeave(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	room, ok := a.activeMafiaRoomForUser(sender.ID)
	if !ok || room.Status != MafiaStatusLobby {
		return c.Send("⛔ Покинуть можно только комнату в лобби.")
	}
	idx := mafiaRoomPlayerIndex(room, sender.ID)
	if idx < 0 {
		return c.Send("⛔ Ты не состоишь в комнате.")
	}
	room.Players = append(room.Players[:idx], room.Players[idx+1:]...)
	room.UpdatedAt = time.Now().UTC().Unix()
	if err := a.store.SaveMafiaRoom(room); err != nil {
		return c.Send("⛔ " + err.Error())
	}
	return c.Send("🚪 Ты покинул комнату "+room.ID, a.mafiaMenuMarkup(c, nil, sender.ID))
}
