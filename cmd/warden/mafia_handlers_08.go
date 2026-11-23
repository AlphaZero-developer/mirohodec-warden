package main

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) requireMafiaCallback(c tele.Context, roomID string, guard mafiaCallbackGuard) (MafiaRoom, *MafiaPlayer, *tele.User, bool) {
	sender := c.Sender()
	if sender == nil || sender.ID == 0 {
		_ = c.Send(a.t("no_sender"))
		return MafiaRoom{}, nil, nil, false
	}
	if _, ok := a.store.MafiaUser(sender.ID); !ok {
		_ = c.Send("⛔ Сначала зарегистрируйтесь у Лайи в ЛС через /start.")
		return MafiaRoom{}, nil, sender, false
	}
	room, ok := a.store.MafiaRoom(roomID)
	if !ok {
		_ = c.Send("⛔ Комната не найдена.")
		return MafiaRoom{}, nil, sender, false
	}
	if guard.RequiresOwner && room.OwnerID != sender.ID {
		_ = c.Send("⛔ Это действие доступно только создателю комнаты.")
		return MafiaRoom{}, nil, sender, false
	}
	if guard.RequiredPhase != "" && room.Status != guard.RequiredPhase {
		_ = c.Send("⛔ Сейчас неподходящая фаза игры.")
		return MafiaRoom{}, nil, sender, false
	}
	if room.PendingLeakOffenderID != 0 && guard.RequiredPhase != "" {
		_ = c.Send("⚠️ Партия приостановлена из-за возможного критического слива. Сначала нужно решение создателя комнаты.")
		return MafiaRoom{}, nil, sender, false
	}
	var player *MafiaPlayer
	if idx := mafiaRoomPlayerIndex(room, sender.ID); idx >= 0 {
		player = &room.Players[idx]
	}
	if guard.RequiresPlayer && player == nil {
		_ = c.Send("⛔ Вы не состоите в этой комнате.")
		return MafiaRoom{}, nil, sender, false
	}
	if guard.RequiresAlive && (player == nil || !player.Alive) {
		_ = c.Send("⛔ Мёртвые не могут выполнить это действие.")
		return MafiaRoom{}, nil, sender, false
	}
	if guard.RequiredRole != "" && (player == nil || player.Role != guard.RequiredRole) {
		_ = c.Send("⛔ Это действие недоступно вашей роли.")
		return MafiaRoom{}, nil, sender, false
	}
	return room, player, sender, true
}

func (a *App) callbackMafiaCreate(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	if _, err := a.ensureMafiaRegistered(c); err != nil {
		return a.handleMafiaRegistrationError(c, err)
	}
	room, err := a.mafiaService().CreateRoom(sender, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	a.syncMafiaGroupPlayerTitles(room)
	groupNote := "Теперь подключите группу или пригласите игроков."
	if room.GroupChatID != 0 {
		groupNote = "Лайя вспомнила прошлый чат обсуждений и подключила его автоматически.\nМожно сразу приглашать игроков или запускать партию."
	}
	return c.Send(fmt.Sprintf("🎭 Комната создана.\n\nКод комнаты: %s\nСтатус: Лобби\nИгроков: %d/%d\nГруппа: %s\n\n%s", room.ID, len(room.Players), room.Settings.MaxPlayers, mafiaGroupStatus(room), groupNote), a.mafiaMenuMarkup(c, mafiaRoomMarkup(room, sender.ID), sender.ID))
}

func (a *App) callbackMafiaStart(c tele.Context, roomID string) error {
	baseRoom, _, sender, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	room, err := a.mafiaService().StartGame(roomID, sender.ID, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ "+a.mafiaStartErrorText(baseRoom, err), a.mafiaMenuMarkup(c, mafiaRoomMarkup(baseRoom, sender.ID), sender.ID))
	}
	a.announceMafiaGameStart(room)
	return c.Send(a.renderMafiaGameStatus(room, sender.ID), a.mafiaMenuMarkup(c, mafiaRoomMarkup(room, sender.ID), sender.ID))
}

func (a *App) mafiaStartErrorText(room MafiaRoom, err error) string {
	if err == nil {
		return ""
	}
	if err.Error() == "mafia group is not bound" {
		return fmt.Sprintf("сначала подключите группу.\n\nВ нужной группе отправьте:\n/mafia_bind %s", room.ID)
	}
	if strings.HasPrefix(err.Error(), "need at least ") {
		settings := normalizeMafiaSettings(room.Settings)
		return fmt.Sprintf("нужно минимум %d игрока.\n\nСейчас: %d/%d. Добавьте игроков или тест-ботов в настройках комнаты.", settings.MinPlayers, len(room.Players), settings.MinPlayers)
	}
	return a.localizeMafiaErr(err)
}

func (a *App) callbackMafiaRooms(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	rooms := a.store.MafiaRoomsForUser(sender.ID)
	if len(rooms) == 0 {
		return c.Send("🎭 У тебя пока нет комнат Мафии.")
	}
	lines := []string{"📋 Мои комнаты:"}
	for _, room := range rooms {
		lines = append(lines, fmt.Sprintf("%s · %s · %d/%d", room.ID, mafiaPhaseLabel(room.Status), len(room.Players), normalizeMafiaSettings(room.Settings).MaxPlayers))
	}
	return c.Send(strings.Join(lines, "\n"))
}

func (a *App) callbackMafiaReadyQueueJoin(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	if _, err := a.ensureMafiaRegistered(c); err != nil {
		return a.handleMafiaRegistrationError(c, err)
	}
	if _, err := a.mafiaService().JoinReadyQueue(sender, time.Now().UTC()); err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	return c.Send("🎮 Ты в очереди готовых игроков.\n\nКогда создатель комнаты включит автонабор, Лайя сможет добавить тебя в лобби.", a.mafiaMenuMarkup(c, mafiaHomeMarkup(len(a.store.MafiaRoomsForUser(sender.ID)) > 0, true), sender.ID))
}

func (a *App) callbackMafiaReadyQueueLeave(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	if err := a.mafiaService().LeaveReadyQueue(sender.ID); err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	return c.Send("🚫 Лайя убрала тебя из очереди готовых игроков.", a.mafiaMenuMarkup(c, mafiaHomeMarkup(len(a.store.MafiaRoomsForUser(sender.ID)) > 0, false), sender.ID))
}

func (a *App) callbackMafiaInviteSettings(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	user, ok := a.store.MafiaUser(sender.ID)
	if !ok {
		return c.Send("⛔ Сначала зарегистрируйтесь у Лайи.")
	}
	return c.Send(mafiaInviteSettingsText(user), mafiaInviteSettingsMarkup(user))
}

func (a *App) callbackMafiaInviteToggle(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	settings, err := a.mafiaService().ToggleInvitePermission(sender.ID)
	if err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	user, _ := a.store.MafiaUser(sender.ID)
	user.Settings = settings
	return c.Send(mafiaInviteSettingsText(user), mafiaInviteSettingsMarkup(user))
}

func (a *App) callbackMafiaInviteDisable(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	if err := a.mafiaService().SetInvitePermission(sender.ID, false); err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	return c.Send("🔕 Хорошо. Лайя больше не будет присылать вам приглашения в Мафию.\n\nВы сможете включить их обратно в настройках.")
}

func (a *App) callbackMafiaInviteMenu(c tele.Context, roomID string) error {
	room, _, _, ok := a.requireMafiaCallback(c, roomID, mafiaCallbackGuard{RequiresOwner: true, RequiredPhase: MafiaStatusLobby})
	if !ok {
		return nil
	}
	return c.Send("➕ Пригласить игроков\n\nРежим набора: "+mafiaRecruitmentModeLabel(room.RecruitmentMode)+"\n\nВыберите способ набора:", a.mafiaInviteMenuReplyMarkup(c, mafiaInviteMenuMarkup(room), room))
}
