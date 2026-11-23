package main

import (
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func mafiaHomeMarkup(showRooms bool, queued bool) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := []tele.Row{
		markup.Row(mafiaButton("➕ Создать комнату", "mafia:create")),
		markup.Row(mafiaButton("🚪 Войти по коду", "mafia:joinhint")),
		markup.Row(mafiaButton("👤 Мой персонаж", "mafia:profile"), mafiaButton("🏆 Достижения", "mafia:achievements")),
	}
	if showRooms {
		rows = append(rows, markup.Row(mafiaButton("📋 Мои комнаты", "mafia:rooms")))
	}
	if queued {
		rows = append(rows, markup.Row(mafiaButton("🚫 Убрать меня из очереди", "mafia:queue_leave")))
	} else {
		rows = append(rows, markup.Row(mafiaButton("🎮 Я готов играть", "mafia:queue_join")))
	}
	rows = append(rows, markup.Row(mafiaButton("🔔 Настройки приглашений", "mafia:invite_settings")))
	rows = append(rows, markup.Row(mafiaButton("📖 Правила", "mafia:rules"), mafiaButton("⬅️ Назад", "mafia:back")))
	markup.Inline(rows...)
	return markup
}

func mafiaReadyQueueHas(store RuntimeStore, userID int64) bool {
	if store == nil {
		return false
	}
	for _, entry := range store.MafiaReadyQueue() {
		if entry.TelegramID == userID {
			return true
		}
	}
	return false
}

func mafiaInviteMenuMarkup(room MafiaRoom) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(mafiaButton("📢 Открыть набор в группе", "mafia:recruit_open:"+room.ID)),
		markup.Row(mafiaButton("⚡ Автонабор из очереди", "mafia:queue_autofill:"+room.ID)),
		markup.Row(mafiaButton("🔔 Разослать приглашение активным", "mafia:invite_active:"+room.ID)),
		markup.Row(mafiaButton("⭐ Пригласить избранных", "mafia:invite_favorites:"+room.ID)),
		markup.Row(mafiaButton("🔎 Найти игрока", "mafia:player_search:"+room.ID), mafiaButton("📋 Заявки игроков", "mafia:requests:"+room.ID)),
		markup.Row(mafiaButton("Режим: "+mafiaRecruitmentModeLabel(room.RecruitmentMode), "mafia:recruit_mode:"+room.ID+":"+mafiaOtherRecruitmentMode(room.RecruitmentMode))),
		markup.Row(mafiaButton("⬅️ Назад", "mafia:status:"+room.ID)),
	)
	return markup
}

func mafiaRecruitmentPostMarkup(room MafiaRoom) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(mafiaButton("🎭 Хочу играть", "mafia:join_request:"+room.ID)))
	return markup
}

func mafiaRecruitmentPostText(room MafiaRoom) string {
	settings := normalizeMafiaSettings(room.Settings)
	free := settings.MaxPlayers - len(room.Players)
	if free < 0 {
		free = 0
	}
	return fmt.Sprintf("🎭 Открыт набор в Мафию\n\nКомната: %s\nИгроков: %d/%d\nСвободных мест: %d\n\nНажмите кнопку ниже, если хотите участвовать.", room.ID, len(room.Players), settings.MaxPlayers, free)
}

func mafiaRequestsText(room MafiaRoom) string {
	settings := normalizeMafiaSettings(room.Settings)
	free := settings.MaxPlayers - len(room.Players)
	if free < 0 {
		free = 0
	}
	lines := []string{fmt.Sprintf("📋 Заявки в комнату %s", room.ID), "", fmt.Sprintf("Свободных мест: %d", free), ""}
	pending := mafiaPendingJoinRequests(room)
	if len(pending) == 0 {
		lines = append(lines, "Ожидающих заявок нет.")
		return strings.Join(lines, "\n")
	}
	for i, req := range pending {
		lines = append(lines, fmt.Sprintf("%d. %s", i+1, mafiaJoinRequestLabel(req)))
	}
	return strings.Join(lines, "\n")
}

func mafiaRequestsMarkup(room MafiaRoom) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	var rows []tele.Row
	for _, req := range mafiaPendingJoinRequests(room) {
		shortName := mafiaShortButtonName(req.CharacterName)
		rows = append(rows,
			markup.Row(mafiaButton("✅ Принять "+shortName, "mafia:request_accept:"+room.ID+":"+idKey(req.TelegramID))),
			markup.Row(mafiaButton("❌ Отклонить "+shortName, "mafia:request_decline:"+room.ID+":"+idKey(req.TelegramID))),
		)
	}
	rows = append(rows, markup.Row(mafiaButton("⚡ Принять первых до лимита", "mafia:request_accept_first:"+room.ID)))
	rows = append(rows, markup.Row(mafiaButton("❌ Отклонить все", "mafia:request_decline_all:"+room.ID), mafiaButton("🔒 Закрыть набор", "mafia:recruit_close:"+room.ID)))
	rows = append(rows, markup.Row(mafiaButton("⬅️ Назад", "mafia:invite_menu:"+room.ID)))
	markup.Inline(rows...)
	return markup
}

func mafiaInviteDMMarkup(room MafiaRoom) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(mafiaButton("🎭 Хочу играть", "mafia:join_request:"+room.ID)),
		markup.Row(mafiaButton("🚫 Не сейчас", "mafia:invite_decline:"+room.ID), mafiaButton("🔕 Не звать меня", "mafia:invite_disable")),
	)
	return markup
}

func mafiaInviteText(room MafiaRoom) string {
	settings := normalizeMafiaSettings(room.Settings)
	free := settings.MaxPlayers - len(room.Players)
	if free < 0 {
		free = 0
	}
	return fmt.Sprintf("🎭 Лайя открыла набор в Мафию\n\nКомната: %s\nСвободных мест: %d\n\nХотите участвовать?", room.ID, free)
}

func mafiaInviteSettingsText(user MafiaRegisteredUser) string {
	settings := normalizeMafiaUserSettings(user.Settings)
	return "🔔 Настройки приглашений\n\nПолучать приглашения в Мафию: " + enabledMark(settings.AllowMafiaInvites)
}

func mafiaInviteSettingsMarkup(user MafiaRegisteredUser) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	settings := normalizeMafiaUserSettings(user.Settings)
	markup.Inline(
		markup.Row(mafiaButton("🔔 Приглашения: "+enabledMark(settings.AllowMafiaInvites), "mafia:invite_toggle")),
		markup.Row(mafiaButton("⬅️ Назад", "mafia:back")),
	)
	return markup
}

func mafiaPendingJoinRequests(room MafiaRoom) []MafiaJoinRequest {
	var out []MafiaJoinRequest
	for _, req := range room.JoinRequests {
		if req.Status == MafiaJoinRequestPending {
			out = append(out, req)
		}
	}
	return out
}

func mafiaJoinRequestLabel(req MafiaJoinRequest) string {
	if strings.TrimSpace(req.Username) != "" {
		return fmt.Sprintf("%s — @%s", req.CharacterName, req.Username)
	}
	return req.CharacterName
}

func mafiaRegisteredUserLabel(user MafiaRegisteredUser) string {
	if strings.TrimSpace(user.Username) != "" {
		return fmt.Sprintf("%s — @%s", mafiaUserCharacterName(user), user.Username)
	}
	return mafiaUserCharacterName(user)
}

func mafiaShortButtonName(name string) string {
	fields := strings.Fields(strings.TrimSpace(name))
	if len(fields) == 0 {
		return "игрока"
	}
	runes := []rune(fields[0])
	if len(runes) > 14 {
		return string(runes[:14])
	}
	return fields[0]
}

func mafiaRecruitmentModeLabel(mode MafiaRecruitmentMode) string {
	if mode == MafiaRecruitmentAutoAccept {
		return "автоприём"
	}
	return "заявки"
}

func mafiaOtherRecruitmentMode(mode MafiaRecruitmentMode) string {
	if mode == MafiaRecruitmentAutoAccept {
		return string(MafiaRecruitmentRequests)
	}
	return string(MafiaRecruitmentAutoAccept)
}
