package main

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) callbackMafiaRecruitmentOpen(c tele.Context, roomID string) error {
	baseRoom, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	if baseRoom.GroupChatID == 0 {
		return c.Send("⛔ Сначала подключите группу к комнате.")
	}
	room, err := a.mafiaService().OpenRecruitment(baseRoom.ID, sender.ID, baseRoom.RecruitmentMode, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	a.sendMafiaGroup(room, mafiaRecruitmentPostText(room), mafiaRecruitmentPostMarkup(room))
	return c.Send("📢 Набор открыт и опубликован в группе.", mafiaInviteMenuMarkup(room))
}

func (a *App) callbackMafiaRecruitmentClose(c tele.Context, roomID string) error {
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	updated, err := a.mafiaService().CloseRecruitment(room.ID, sender.ID, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	return c.Send("🔒 Набор закрыт.", mafiaInviteMenuMarkup(updated))
}

func (a *App) callbackMafiaRecruitmentMode(c tele.Context, data string) error {
	parts := strings.Split(data, ":")
	if len(parts) != 4 {
		return c.Send("⛔ Устаревшая кнопка.")
	}
	roomID := parts[2]
	mode := MafiaRecruitmentMode(parts[3])
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	updated, err := a.mafiaService().SetRecruitmentMode(room.ID, sender.ID, mode, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	return c.Send("✅ Режим набора: "+mafiaRecruitmentModeLabel(updated.RecruitmentMode), mafiaInviteMenuMarkup(updated))
}

func (a *App) callbackMafiaJoinRequest(c tele.Context, roomID string) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	user, ok := a.store.MafiaUser(sender.ID)
	if !ok {
		return c.Send("⛔ Сначала откройте ЛС с Лайей и отправьте /start.")
	}
	if !mafiaProfileComplete(user) {
		return c.Send("⛔ Сначала создайте персонажа в ЛС с Лайей.\n\nОткройте бота и нажмите /start.")
	}
	room, added, err := a.mafiaService().SubmitJoinRequest(roomID, sender, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	if added {
		a.announceMafiaLobbyPlayerJoined(room, sender.ID)
		a.sendMafiaGroupAccessIfMissing(sender.ID, room)
		a.sendMafiaDM(sender.ID, fmt.Sprintf("🎭 Вы добавлены в комнату Мафии.\n\nКомната: %s\nОжидайте начала игры.", room.ID), mafiaRoomMarkup(room, sender.ID))
		a.sendMafiaDM(room.OwnerID, fmt.Sprintf("✅ %s автоматически вошёл в комнату %s.", mafiaUserCharacterName(user), room.ID), nil)
		return c.Send("✅ Вы добавлены в комнату " + room.ID + ".")
	}
	a.sendMafiaDM(room.OwnerID, fmt.Sprintf("📋 Новая заявка в комнату %s: %s", room.ID, mafiaUserCharacterName(user)), mafiaRequestsMarkup(room))
	return c.Send("📋 Заявка отправлена создателю комнаты.")
}

func (a *App) callbackMafiaRequests(c tele.Context, roomID string) error {
	room, _, _, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	return c.Send(mafiaRequestsText(room), mafiaRequestsMarkup(room))
}

func (a *App) callbackMafiaRequestDecision(c tele.Context, data string, accept bool) error {
	parts := strings.Split(data, ":")
	if len(parts) != 4 {
		return c.Send("⛔ Устаревшая кнопка.")
	}
	roomID := parts[2]
	targetID := mustParseInt64(parts[3])
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	var err error
	if accept {
		room, err = a.mafiaService().AcceptJoinRequest(room.ID, sender.ID, targetID, time.Now().UTC())
		if err == nil {
			a.announceMafiaLobbyPlayerJoined(room, targetID)
			a.sendMafiaGroupAccessIfMissing(targetID, room)
			a.sendMafiaDM(targetID, fmt.Sprintf("🎭 Вы добавлены в комнату Мафии.\n\nКомната: %s\nОжидайте начала игры.", room.ID), mafiaRoomMarkup(room, targetID))
		}
	} else {
		room, err = a.mafiaService().DeclineJoinRequest(room.ID, sender.ID, targetID, time.Now().UTC())
		if err == nil {
			a.sendMafiaDM(targetID, "🚫 Заявку в комнату "+room.ID+" отклонили.", nil)
		}
	}
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	return c.Send(mafiaRequestsText(room), mafiaRequestsMarkup(room))
}

func (a *App) callbackMafiaRequestAcceptFirst(c tele.Context, roomID string) error {
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	updated, added, err := a.mafiaService().AcceptFirstJoinRequests(room.ID, sender.ID, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	for _, userID := range added {
		a.sendMafiaGroupAccessIfMissing(userID, updated)
		a.sendMafiaDM(userID, fmt.Sprintf("🎭 Вы добавлены в комнату Мафии.\n\nКомната: %s\nОжидайте начала игры.", updated.ID), mafiaRoomMarkup(updated, userID))
	}
	a.announceMafiaLobbyPlayersAdded(updated, added)
	return c.Send(fmt.Sprintf("⚡ Принято игроков: %d.", len(added)), mafiaRequestsMarkup(updated))
}

func (a *App) callbackMafiaRequestDeclineAll(c tele.Context, roomID string) error {
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	updated, err := a.mafiaService().DeclineAllJoinRequests(room.ID, sender.ID, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	return c.Send("❌ Все ожидающие заявки отклонены.", mafiaRequestsMarkup(updated))
}

func (a *App) callbackMafiaQueueAutoFill(c tele.Context, roomID string) error {
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	updated, added, err := a.mafiaService().AutoFillFromQueue(room.ID, sender.ID, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	for _, userID := range added {
		a.sendMafiaGroupAccessIfMissing(userID, updated)
		a.sendMafiaDM(userID, fmt.Sprintf("🎭 Вы добавлены в комнату Мафии.\n\nКомната: %s\nОжидайте начала игры.", updated.ID), mafiaRoomMarkup(updated, userID))
	}
	a.announceMafiaLobbyPlayersAdded(updated, added)
	return c.Send(fmt.Sprintf("⚡ Автонабор завершён.\n\nДобавлено игроков: %d\nКомната: %d/%d", len(added), len(updated.Players), normalizeMafiaSettings(updated.Settings).MaxPlayers), mafiaInviteMenuMarkup(updated))
}

func (a *App) callbackMafiaInviteBroadcast(c tele.Context, roomID string, favoritesOnly bool) error {
	room, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	updated, recipients, err := a.mafiaService().PrepareInviteBroadcast(room.ID, sender.ID, favoritesOnly, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	for _, user := range recipients {
		a.sendMafiaDM(user.TelegramID, mafiaInviteText(updated), mafiaInviteDMMarkup(updated))
	}
	if len(recipients) == 0 {
		return c.Send("🔔 Подходящих игроков для приглашения не найдено.", mafiaInviteMenuMarkup(updated))
	}
	return c.Send(fmt.Sprintf("🔔 Приглашения отправлены.\n\nПолучателей: %d", len(recipients)), mafiaInviteMenuMarkup(updated))
}
