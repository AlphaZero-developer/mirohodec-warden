package main

import (
	"fmt"

	tele "gopkg.in/telebot.v3"
)

func mafiaVotePhaseKey(room MafiaRoom) string {
	if room.PhaseStartedAt > 0 {
		return idKey(room.PhaseStartedAt)
	}
	return fmt.Sprintf("%s-%d", room.Status, room.DayNumber)
}

func mafiaCriticalLeakCancelText() string {
	return "⛔ Партия отменена.\n\nЛайя обнаружила нарушение, которое сломало игру:\nкритическая информация была раскрыта в общем чате.\n\nПартия не будет записана в статистику.\nДостижения за неё не выдаются."
}

func mafiaCancelledOwnerText(room MafiaRoom, offender MafiaPlayer) string {
	return fmt.Sprintf("⚠️ Партия была отменена из-за критического слива.\n\nНарушитель: %s\n\nЧто сделать?", mafiaRoomPlayerLabel(room, offender))
}

func mafiaLeakOwnerDecisionText(room MafiaRoom, offender MafiaPlayer) string {
	return fmt.Sprintf("⚠️ Возможный критический слив.\n\nПартия приостановлена.\nРешение принимает создатель комнаты.\n\nНарушитель: %s", mafiaRoomPlayerLabel(room, offender))
}

func mafiaCancelledOwnerMarkup(room MafiaRoom) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(mafiaButton("🔁 Пересоздать комнату", "mafia:cancel_recreate:"+room.ID)),
		markup.Row(mafiaButton("🚫 Заблокировать игрока", "mafia:cancel_block:"+room.ID)),
		markup.Row(mafiaButton("📋 Показать состав", "mafia:players:"+room.ID)),
		markup.Row(mafiaButton("🗑 Закрыть комнату", "mafia:cancel_close:"+room.ID)),
	)
	return markup
}

func mafiaLeakDecisionMarkup(room MafiaRoom) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(mafiaButton("⛔ Отменить партию", "mafia:leak_cancel:"+room.ID)),
		markup.Row(mafiaButton("☠️ Убить нарушителя", "mafia:leak_kill:"+room.ID)),
		markup.Row(mafiaButton("✅ Продолжить игру", "mafia:leak_continue:"+room.ID)),
	)
	return markup
}

func mafiaSettingsText(room MafiaRoom) string {
	settings := normalizeMafiaSettings(room.Settings)
	return fmt.Sprintf("⚙️ Настройки комнаты %s\n\n☀️ День: %d сек.\n🌙 Ночь: %d сек.\n🗳 Голосование: %d сек.\n\n📜 Завещания: %s\n🏆 Достижения: %s\n🎭 AI-отчёт после игры: %s\n🤖 AI-боты: %s\n🚨 Критический слив: %s",
		room.ID,
		settings.DayDiscussionSeconds,
		settings.NightActionSeconds,
		settings.VotingSeconds,
		enabledMark(settings.EnableWills),
		enabledMark(settings.EnableAchievements),
		enabledMark(settings.EnablePostGameAIReport),
		enabledMark(settings.EnableAIBots),
		mafiaLeakActionLabel(settings.GameBreakingLeakAction),
	)
}

func mafiaSettingsMarkup(room MafiaRoom, userID int64) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	settings := normalizeMafiaSettings(room.Settings)
	markup.Inline(
		markup.Row(mafiaButton("⏱ Длительность фаз", "mafia:settings_time:"+room.ID)),
		markup.Row(mafiaButton("👥 Игроки и тест", "mafia:settings_players:"+room.ID)),
		markup.Row(mafiaButton("🎭 Дополнительные роли", "mafia:settings_roles:"+room.ID)),
		markup.Row(mafiaButton("📜 Завещания: "+enabledMark(settings.EnableWills), "mafia:toggle:"+room.ID+":wills")),
		markup.Row(mafiaButton("🏆 Достижения: "+enabledMark(settings.EnableAchievements), "mafia:toggle:"+room.ID+":achievements")),
		markup.Row(mafiaButton("🎭 AI-отчёт после игры: "+enabledMark(settings.EnablePostGameAIReport), "mafia:toggle:"+room.ID+":ai_report")),
		markup.Row(mafiaButton("🤖 AI-боты: "+enabledMark(settings.EnableAIBots), "mafia:toggle:"+room.ID+":ai_bots")),
		markup.Row(mafiaButton("🚨 Критический слив: "+mafiaLeakActionLabel(settings.GameBreakingLeakAction), "mafia:toggle:"+room.ID+":leak_action")),
		markup.Row(mafiaButton("🛡 Политика раскрытия ролей", "mafia:settings_reveal:"+room.ID)),
		markup.Row(mafiaButton("🗳 Голосование", "mafia:settings_voting:"+room.ID)),
		markup.Row(mafiaButton("⬅️ Назад", "mafia:status:"+room.ID)),
	)
	return markup
}

func mafiaTimeSettingsText(room MafiaRoom) string {
	settings := normalizeMafiaSettings(room.Settings)
	return fmt.Sprintf("⏱ Длительность фаз\n\n☀️ День: %d сек.\n🌙 Ночь: %d сек.\n🗳 Голосование: %d сек.", settings.DayDiscussionSeconds, settings.NightActionSeconds, settings.VotingSeconds)
}

func mafiaTimeSettingsMarkup(room MafiaRoom) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(mafiaButton("☀️ День -30", "mafia:time:"+room.ID+":day:-30"), mafiaButton("☀️ День +30", "mafia:time:"+room.ID+":day:+30")),
		markup.Row(mafiaButton("🌙 Ночь -15", "mafia:time:"+room.ID+":night:-15"), mafiaButton("🌙 Ночь +15", "mafia:time:"+room.ID+":night:+15")),
		markup.Row(mafiaButton("🗳 Голосование -15", "mafia:time:"+room.ID+":vote:-15"), mafiaButton("🗳 Голосование +15", "mafia:time:"+room.ID+":vote:+15")),
		markup.Row(mafiaButton("⚡ Быстрый режим", "mafia:time:"+room.ID+":preset:fast")),
		markup.Row(mafiaButton("⚖️ Обычный режим", "mafia:time:"+room.ID+":preset:normal")),
		markup.Row(mafiaButton("Длинная партия", "mafia:time:"+room.ID+":preset:long")),
		markup.Row(mafiaButton("⬅️ Назад", "mafia:settings:"+room.ID)),
	)
	return markup
}

func mafiaPlayerSettingsText(room MafiaRoom) string {
	settings := normalizeMafiaSettings(room.Settings)
	return fmt.Sprintf("👥 Игроки и тест\n\nИгроков: %d/%d\nМинимум для старта: %d\nТест-ботов: %d\n\nМинимум можно менять только в лобби. Самое маленькое значение — 4 игрока.",
		len(room.Players),
		settings.MaxPlayers,
		settings.MinPlayers,
		mafiaTestBotCount(room),
	)
}

func mafiaPlayerSettingsMarkup(room MafiaRoom) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	settings := normalizeMafiaSettings(room.Settings)
	markup.Inline(
		markup.Row(mafiaButton("Минимум -1", "mafia:player_limit:"+room.ID+":min:-1"), mafiaButton("Минимум +1", "mafia:player_limit:"+room.ID+":min:+1")),
		markup.Row(mafiaButton("Максимум -1", "mafia:player_limit:"+room.ID+":max:-1"), mafiaButton("Максимум +1", "mafia:player_limit:"+room.ID+":max:+1")),
		markup.Row(mafiaButton("🤖 Добавить тест-бота", "mafia:add_bot:"+room.ID)),
		markup.Row(mafiaButton(fmt.Sprintf("🤖 Добрать до %d", settings.MinPlayers), "mafia:fill_bots:"+room.ID)),
		markup.Row(mafiaButton("🧹 Убрать тест-ботов", "mafia:clear_bots:"+room.ID)),
		markup.Row(mafiaButton("⬅️ Назад", "mafia:settings:"+room.ID)),
	)
	return markup
}

func mafiaTestBotsChangedText(room MafiaRoom, count int, action string) string {
	settings := normalizeMafiaSettings(room.Settings)
	return fmt.Sprintf("🤖 Тест-боты: %s %d.\n\nИгроков: %d/%d\nМинимум для старта: %d", action, count, len(room.Players), settings.MaxPlayers, settings.MinPlayers)
}

func mafiaRoleSettingsText() string {
	return "🎭 Дополнительные роли\n\nБазовые роли всегда включены:\n— Мафия\n— Мирный житель\n— Доктор\n— Детектив"
}

func mafiaRoleSettingsMarkup(room MafiaRoom) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	roles := normalizeMafiaSettings(room.Settings).OptionalRoles
	markup.Inline(
		markup.Row(mafiaButton("🎩 Дон: "+enabledMark(roles.EnableDon), "mafia:toggle_role:"+room.ID+":don")),
		markup.Row(mafiaButton("🛡 Сержант: "+enabledMark(roles.EnableSergeant), "mafia:toggle_role:"+room.ID+":sergeant")),
		markup.Row(mafiaButton("Телохранитель: "+enabledMark(roles.EnableBodyguard), "mafia:toggle_role:"+room.ID+":bodyguard")),
		markup.Row(mafiaButton("🚫 Блокер: "+enabledMark(roles.EnableBlocker), "mafia:toggle_role:"+room.ID+":blocker")),
		markup.Row(mafiaButton("⚖️ Адвокат: "+enabledMark(roles.EnableLawyer), "mafia:toggle_role:"+room.ID+":lawyer")),
		markup.Row(mafiaButton("📰 Журналист: "+enabledMark(roles.EnableJournalist), "mafia:toggle_role:"+room.ID+":journalist")),
		markup.Row(mafiaButton("🏛 Мэр: "+enabledMark(roles.EnableMayor), "mafia:toggle_role:"+room.ID+":mayor")),
		markup.Row(mafiaButton("⬅️ Назад", "mafia:settings:"+room.ID)),
	)
	return markup
}

func mafiaVotingSettingsText(room MafiaRoom) string {
	settings := normalizeMafiaSettings(room.Settings)
	return "🗳 Настройки голосования\n\nСмена голоса: " + enabledMark(settings.AllowVoteChange) + "\nНичья: " + mafiaTieModeLabel(settings.TieMode)
}

func mafiaVotingSettingsMarkup(room MafiaRoom) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	settings := normalizeMafiaSettings(room.Settings)
	markup.Inline(
		markup.Row(mafiaButton("🔁 Смена голоса: "+enabledMark(settings.AllowVoteChange), "mafia:toggle:"+room.ID+":vote_change")),
		markup.Row(mafiaButton("⚖️ Ничья: никто не умирает", "mafia:toggle:"+room.ID+":tie_nobody")),
		markup.Row(mafiaButton("🎲 Ничья: случайный из лидеров", "mafia:toggle:"+room.ID+":tie_random")),
		markup.Row(mafiaButton("⬅️ Назад", "mafia:settings:"+room.ID)),
	)
	return markup
}

func mafiaRevealSettingsText(room MafiaRoom) string {
	settings := normalizeMafiaSettings(room.Settings)
	return fmt.Sprintf("🛡 Правила про роли\n\nТекущий режим: %s\n\n%s\n\nРежимы:\n\n🟢 Свободный\nМожно заявлять роль как угодно. Лайя всё равно удаляет пересылки, медиа и критический слив союзников.\n\n🟡 Обычный\nМожно блефовать и говорить «я доктор», «я детектив», «я проверил Софию». Нельзя доказывать роль через Лайю: скрин, пересылка ЛС, «у меня кнопка лечения», «вижу теневой канал».\n\n🔴 Строгий\nНельзя прямо заявлять свою роль вообще: «я мафия», «я доктор», «я детектив», «я мирный». Обычные подозрения других игроков разрешены.", mafiaRevealPolicyLabel(settings.RoleRevealPolicy), mafiaRevealPolicyDescription(settings.RoleRevealPolicy))
}

func mafiaRevealSettingsMarkup(room MafiaRoom) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	settings := normalizeMafiaSettings(room.Settings)
	markup.Inline(
		markup.Row(mafiaButton(mafiaRevealPolicyButton("🟢 Свободный режим", settings.RoleRevealPolicy == MafiaRoleRevealAllowClaims), "mafia:toggle:"+room.ID+":reveal_allow")),
		markup.Row(mafiaButton(mafiaRevealPolicyButton("🟡 Обычный режим", settings.RoleRevealPolicy == MafiaRoleRevealBanSystemProof), "mafia:toggle:"+room.ID+":reveal_system")),
		markup.Row(mafiaButton(mafiaRevealPolicyButton("🔴 Строгий режим", settings.RoleRevealPolicy == MafiaRoleRevealBanAllClaims), "mafia:toggle:"+room.ID+":reveal_all")),
		markup.Row(mafiaButton("⬅️ Назад", "mafia:settings:"+room.ID)),
	)
	return markup
}

func mafiaRevealPolicyButton(label string, current bool) string {
	if current {
		return label + " ✅"
	}
	return label
}
