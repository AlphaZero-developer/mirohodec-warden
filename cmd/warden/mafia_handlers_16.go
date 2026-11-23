package main

import (
	tele "gopkg.in/telebot.v3"
)

func mafiaRoomMarkup(room MafiaRoom, userID int64) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	switch room.Status {
	case MafiaStatusNight:
		return mafiaNightActionMarkup(room, userID)
	case MafiaStatusDayDiscussion:
		return mafiaDayMarkup(room, userID)
	case MafiaStatusVoting:
		idx := mafiaRoomPlayerIndex(room, userID)
		if idx < 0 || !room.Players[idx].Alive {
			if room.OwnerID == userID {
				return mafiaOwnerControlMarkup(room)
			}
			return mafiaDeadPlayerMarkup(room)
		}
		return mafiaVotingMarkup(room, room.OwnerID == userID)
	case MafiaStatusFinished:
		markup.Inline(
			markup.Row(mafiaButton("👤 Профили игроков", "mafia:profiles:"+room.ID)),
			markup.Row(mafiaButton("📜 Правила", "mafia:rules"), mafiaButton("⬅️ Назад", "mafia:back")),
		)
		return markup
	case MafiaStatusCancelled:
		if room.OwnerID == userID {
			return mafiaCancelledOwnerMarkup(room)
		}
		markup.Inline(
			markup.Row(mafiaButton("👤 Профили игроков", "mafia:profiles:"+room.ID)),
			markup.Row(mafiaButton("📜 Правила", "mafia:rules"), mafiaButton("⬅️ Назад", "mafia:back")),
		)
		return markup
	}
	if room.OwnerID == userID {
		var rows []tele.Row
		rows = append(rows, markup.Row(mafiaButton("▶️ Начать игру", "mafia:start:"+room.ID)))
		rows = append(rows, markup.Row(mafiaButton("👥 Игроки", "mafia:players:"+room.ID), mafiaButton("👤 Профили игроков", "mafia:profiles:"+room.ID)))
		rows = append(rows, markup.Row(mafiaButton("➕ Пригласить игроков", "mafia:invite:"+room.ID)))
		if room.GroupChatID == 0 {
			rows = append(rows, markup.Row(mafiaButton("🔗 Подключить группу", "mafia:bindhint")))
		}
		rows = append(rows, markup.Row(mafiaButton("⚙️ Настройки комнаты", "mafia:settings:"+room.ID)))
		rows = append(rows, markup.Row(mafiaButton("🧪 Полный тест механики", "mafia:test:"+room.ID)))
		rows = append(rows, markup.Row(mafiaButton("🗑 Закрыть комнату", "mafia:close:"+room.ID), mafiaButton("⬅️ Назад", "mafia:back")))
		markup.Inline(rows...)
		return markup
	}
	markup.Inline(
		markup.Row(mafiaButton("👥 Игроки", "mafia:players:"+room.ID), mafiaButton("👤 Профили игроков", "mafia:profiles:"+room.ID)),
		markup.Row(mafiaButton("📜 Правила", "mafia:rules")),
		markup.Row(mafiaButton("🚪 Покинуть комнату", "mafia:leave:"+room.ID), mafiaButton("⬅️ Назад", "mafia:back")),
	)
	return markup
}

func mafiaNightActionMarkup(room MafiaRoom, userID int64) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	settings := normalizeMafiaSettings(room.Settings)
	idx := mafiaRoomPlayerIndex(room, userID)
	if idx < 0 || !room.Players[idx].Alive {
		if room.OwnerID == userID {
			return mafiaOwnerControlMarkup(room)
		}
		return mafiaDeadPlayerMarkup(room)
	}
	player := room.Players[idx]
	var rows []tele.Row
	switch player.Role {
	case MafiaRoleMafia:
		rows = append(rows, markup.Row(mafiaButton("🗡 Выбрать жертву", "mafia:kill_menu:"+room.ID)))
		if settings.MafiaChatMode != MafiaChatDisabled {
			rows = append(rows, markup.Row(mafiaButton("💬 Теневой канал", "mafia:shadow:"+room.ID)))
		}
	case MafiaRoleDoctor:
		rows = append(rows, markup.Row(mafiaButton("💊 Лечить игрока", "mafia:heal_menu:"+room.ID)))
	case MafiaRoleDetective:
		rows = append(rows, markup.Row(mafiaButton("🔎 Проверить игрока", "mafia:check_menu:"+room.ID)))
	}
	rows = append(rows, markup.Row(mafiaButton("📜 Моя роль", "mafia:role:"+room.ID)))
	if settings.EnableWills {
		rows = append(rows, markup.Row(mafiaButton("📜 Завещание", "mafia:will:"+room.ID)))
	}
	if player.Role == MafiaRoleCivilian {
		rows = append(rows, markup.Row(mafiaButton("📖 Правила", "mafia:rules")))
	}
	rows = append(rows, markup.Row(mafiaButton("👤 Профили игроков", "mafia:profiles:"+room.ID)))
	rows = append(rows, markup.Row(mafiaButton("⏭ Скип фазы", mafiaSkipPhaseData(room))))
	if room.OwnerID == userID {
		rows = append(rows, markup.Row(mafiaButton("⛔ Завершить партию", "mafia:end:"+room.ID)))
	}
	markup.Inline(rows...)
	return markup
}

func mafiaDayMarkup(room MafiaRoom, userID int64) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	settings := normalizeMafiaSettings(room.Settings)
	idx := mafiaRoomPlayerIndex(room, userID)
	if idx < 0 || !room.Players[idx].Alive {
		if room.OwnerID == userID {
			return mafiaOwnerControlMarkup(room)
		}
		return mafiaDeadPlayerMarkup(room)
	}
	var rows []tele.Row
	rows = append(rows, markup.Row(mafiaButton("🗳 Голосование", "mafia:voting_hint:"+room.ID)))
	rows = append(rows, markup.Row(mafiaButton("📜 Моя роль", "mafia:role:"+room.ID)))
	if settings.EnableWills {
		rows = append(rows, markup.Row(mafiaButton("📜 Завещание", "mafia:will:"+room.ID)))
	}
	rows = append(rows, markup.Row(mafiaButton("👥 Живые игроки", "mafia:alive:"+room.ID), mafiaButton("👤 Профили игроков", "mafia:profiles:"+room.ID)))
	rows = append(rows, markup.Row(mafiaButton("📖 Правила", "mafia:rules")))
	rows = append(rows, markup.Row(mafiaButton("⏭ Скип фазы", mafiaSkipPhaseData(room))))
	if room.OwnerID == userID {
		rows = append(rows, markup.Row(mafiaButton("⛔ Завершить партию", "mafia:end:"+room.ID)))
	}
	markup.Inline(rows...)
	return markup
}

func mafiaDeadPlayerMarkup(room MafiaRoom) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	settings := normalizeMafiaSettings(room.Settings)
	rows := []tele.Row{
		markup.Row(mafiaButton("👥 Живые игроки", "mafia:alive:"+room.ID), mafiaButton("👤 Профили игроков", "mafia:profiles:"+room.ID)),
		markup.Row(mafiaButton("📖 Правила", "mafia:rules")),
	}
	if settings.EnableWills {
		rows = append(rows, markup.Row(mafiaButton("📜 Моё завещание", "mafia:will:"+room.ID)))
	}
	markup.Inline(rows...)
	return markup
}

func mafiaOwnerControlMarkup(room MafiaRoom) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	rows := []tele.Row{
		markup.Row(mafiaButton("👥 Живые игроки", "mafia:alive:"+room.ID), mafiaButton("👤 Профили игроков", "mafia:profiles:"+room.ID)),
		markup.Row(mafiaButton("📖 Правила", "mafia:rules")),
	}
	rows = append(rows, markup.Row(mafiaButton("⛔ Завершить партию", "mafia:end:"+room.ID)))
	rows = append(rows, markup.Row(mafiaButton("⬅️ Назад", "mafia:back")))
	markup.Inline(rows...)
	return markup
}

func mafiaTargetMarkup(room MafiaRoom, actorID int64, action string) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	var rows []tele.Row
	actorIdx := mafiaRoomPlayerIndex(room, actorID)
	labels := mafiaRoomPlayerLabels(room)
	for _, target := range room.Players {
		if !target.Alive {
			continue
		}
		if action == "kill" && actorIdx >= 0 && target.Role == MafiaRoleMafia {
			continue
		}
		rows = append(rows, markup.Row(mafiaButton(labels[target.TelegramID], "mafia:"+action+":"+room.ID+":"+idKey(target.TelegramID))))
	}
	rows = append(rows, markup.Row(mafiaButton("⬅️ Назад", "mafia:status:"+room.ID)))
	markup.Inline(rows...)
	return markup
}

func mafiaVotingMarkup(room MafiaRoom, includeResolve bool) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	var rows []tele.Row
	labels := mafiaRoomPlayerLabels(room)
	phaseKey := mafiaVotePhaseKey(room)
	for _, player := range room.Players {
		if !player.Alive {
			continue
		}
		rows = append(rows, markup.Row(mafiaButton(labels[player.TelegramID], "mafia:vote:"+room.ID+":"+phaseKey+":"+idKey(player.TelegramID))))
	}
	rows = append(rows, markup.Row(mafiaButton("Пропустить", "mafia:voteskip:"+room.ID+":"+phaseKey)))
	rows = append(rows, markup.Row(mafiaButton("👤 Профили игроков", "mafia:profiles:"+room.ID)))
	rows = append(rows, markup.Row(mafiaButton("📊 Статус игры", "mafia:status:"+room.ID)))
	rows = append(rows, markup.Row(mafiaButton("⏭ Скип фазы", mafiaSkipPhaseData(room))))
	if includeResolve {
		rows = append(rows, markup.Row(mafiaButton("⛔ Завершить партию", "mafia:end:"+room.ID)))
	}
	markup.Inline(rows...)
	return markup
}
