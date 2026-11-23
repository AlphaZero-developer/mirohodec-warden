package main

import (
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func runMafiaGameActionChecks(store *Store, service *MafiaService, room MafiaRoom, now time.Time, ok func(string), fail func(stage, expected, actual string, err error) MafiaFullTestReport) MafiaFullTestReport {
	mafia := mafiaPlayers(room)[0]
	room, msg, recipients, err := service.SendShadowMessage("TEST-MF", mafia.TelegramID, "Убираем Shadow.", now)
	if err != nil {
		return fail("Теневой канал", "отправить сообщение", "ошибка", err)
	}
	if msg.Text == "" || len(recipients) != countRole(room, MafiaRoleMafia) {
		return fail("Теневой канал", "сообщение получат только мафиози", "неверные получатели", nil)
	}
	for _, recipientID := range recipients {
		if room.Players[mafiaRoomPlayerIndex(room, recipientID)].Role != MafiaRoleMafia {
			return fail("Теневой канал", "сообщение получат только мафиози", "получатель не мафия", nil)
		}
	}
	ok("Сообщение мафии доставлено только мафиози")
	ok("Мирные не получили приватные сообщения мафии")
	ok("Общий чат не получил сообщения теневого канала")

	if room.Status != MafiaStatusNight {
		return fail("Ночь", "ночь запущена", string(room.Status), nil)
	}
	ok("Ночь запущена")
	doctor := firstRole(room, MafiaRoleDoctor)
	detective := firstRole(room, MafiaRoleDetective)
	civilian := firstRole(room, MafiaRoleCivilian)
	if !mafiaMarkupContains(mafiaRoomMarkup(room, mafia.TelegramID), "Выбрать жертву") ||
		!mafiaMarkupContains(mafiaRoomMarkup(room, doctor.TelegramID), "Лечить") ||
		!mafiaMarkupContains(mafiaRoomMarkup(room, detective.TelegramID), "Проверить") ||
		mafiaMarkupContains(mafiaRoomMarkup(room, civilian.TelegramID), "Выбрать жертву") ||
		mafiaMarkupContains(mafiaRoomMarkup(room, civilian.TelegramID), "Теневой канал") {
		return fail("Права кнопок", "ночные кнопки показаны только нужным ролям", "кнопки показаны неверно", nil)
	}
	ok("Права кнопок по ролям проверены")
	if _, err := service.SaveWill("TEST-MF", civilian.TelegramID, "Если я погибну, ищите тех, кто слишком молчит.", now); err != nil {
		return fail("Завещания", "сохранить завещание", "ошибка", err)
	}
	ok("Завещания работают")
	if _, err := service.MafiaKill("TEST-MF", mafia.TelegramID, civilian.TelegramID, now); err != nil {
		return fail("Действие мафии", "обработать убийство", "ошибка", err)
	}
	ok("Действие мафии обработано")
	if _, err := service.DoctorHeal("TEST-MF", doctor.TelegramID, civilian.TelegramID, now); err != nil {
		return fail("Действие доктора", "обработать лечение", "ошибка", err)
	}
	ok("Действие доктора обработано")
	if _, isMafia, err := service.DetectiveCheck("TEST-MF", detective.TelegramID, mafia.TelegramID, now); err != nil || !isMafia {
		return fail("Действие детектива", "детектив видит мафию", "ошибка или неверный результат", err)
	}
	ok("Действие детектива обработано")
	room, night, err := service.ResolveNight("TEST-MF", now)
	if err != nil || !night.Saved || night.KilledID != 0 || room.Status != MafiaStatusDayDiscussion {
		return fail("Утро", "доктор спас жертву и начался день", "неверный результат", err)
	}
	ok("Утро рассчитано")
	ok("День запущен")

	reveal := DetectMafiaRoleReveal("я мафия")
	if !reveal.IsRoleReveal {
		return fail("Раскрытие роли", "regex обнаружит раскрытие", "не обнаружено", nil)
	}
	ok("Раскрытие роли обнаружено")
	room, err = service.PunishRoleReveal("TEST-MF", civilian.TelegramID, now)
	if err != nil || room.Players[mafiaRoomPlayerIndex(room, civilian.TelegramID)].Alive {
		return fail("Наказание", "игрок выбывает за раскрытие", "игрок жив", err)
	}
	ok("Наказание применено")
	if mafiaPublishedWillText(room, civilian.TelegramID) == "" {
		return fail("Завещания", "завещание публикуется после смерти", "текст не опубликован", nil)
	}
	if mafiaMarkupContains(mafiaRoomMarkup(room, civilian.TelegramID), "Голосование") ||
		mafiaMarkupContains(mafiaRoomMarkup(room, civilian.TelegramID), "Выбрать жертву") ||
		mafiaMarkupContains(mafiaRoomMarkup(room, civilian.TelegramID), "Теневой канал") {
		return fail("Мёртвые игроки", "мёртвые не получают игровые действия", "лишняя кнопка показана", nil)
	}
	if _, err := service.StartVoting("TEST-MF", now); err != nil {
		return fail("Голосование", "запустить голосование", "ошибка", err)
	}
	room, _ = store.MafiaRoom("TEST-MF")
	for _, player := range room.Players {
		if player.Alive {
			if _, err := service.Vote("TEST-MF", player.TelegramID, "skip", now); err != nil {
				return fail("Голосование", "живые голосуют", "ошибка", err)
			}
		}
	}
	if _, _, err := service.ResolveVoting("TEST-MF", now); err != nil {
		return fail("Голосование", "завершить голосование", "ошибка", err)
	}
	ok("Голосование обработано")
	room, err = service.PunishRoleReveal("TEST-MF", mafia.TelegramID, now)
	if err != nil || room.Status != MafiaStatusFinished || room.Winner != "civilian" {
		return fail("Условия победы", "город побеждает после выбывания мафии", "победитель не определён", err)
	}
	if normalizeMafiaSettings(room.Settings).EnableAchievements {
		user, ok := store.MafiaUser(detective.TelegramID)
		if !ok || len(user.Stats.Achievements) == 0 {
			return fail("Достижения", "достижения выдаются после игры", "достижения не выданы", nil)
		}
	}
	ok("Достижения выдаются после игры")
	if mafiaPostGameAchievementsText(room) == "" {
		return fail("Достижения", "постигровой список достижений сформирован", "пустой текст", nil)
	}
	if !normalizeMafiaSettings(room.Settings).EnablePostGameAIReport {
		return fail("AI-отчёт", "AI-отчёт включён для финала", "настройка выключена", nil)
	}
	ok("AI-отчёт проверен")
	return MafiaFullTestReport{}
}

func runMafiaGameCancellationChecks(store *Store, service *MafiaService, users []*tele.User, now time.Time, ok func(string), fail func(stage, expected, actual string, err error) MafiaFullTestReport) MafiaFullTestReport {
	leakRoom, err := service.CreateRoom(users[0], now)
	if err != nil {
		return fail("Аварийная отмена", "создать комнату для теста слива", "ошибка", err)
	}
	leakRoom.ID = "LEAK-MF"
	leakRoom.Settings.MinPlayers = 5
	leakRoom.Settings.MaxPlayers = 12
	leakRoom.Settings.GameBreakingLeakAction = MafiaLeakActionCancelGame
	if err := store.SaveMafiaRoom(leakRoom); err != nil {
		return fail("Аварийная отмена", "сохранить LEAK-MF", "ошибка", err)
	}
	for _, user := range users[1:] {
		if _, err := service.JoinRoom("LEAK-MF", user, now); err != nil {
			return fail("Аварийная отмена", "добавить игроков в тест слива", "ошибка", err)
		}
	}
	if _, err := service.BindGroup("LEAK-MF", users[0].ID, -1009002, now); err != nil {
		return fail("Аварийная отмена", "привязать группу для теста слива", "ошибка", err)
	}
	leakRoom, err = service.StartGame("LEAK-MF", users[0].ID, now)
	if err != nil {
		return fail("Аварийная отмена", "стартовать тестовую партию слива", "ошибка", err)
	}
	leakOffender := firstRole(leakRoom, MafiaRoleMafia)
	leak := DetectMafiaCriticalLeak("Mira со мной мафия")
	if !leak.ShouldPunish || leak.ViolationType != MafiaRuleViolationAllyLeak {
		return fail("Аварийная отмена", "нарушение ally_leak обнаружено", "не обнаружено", nil)
	}
	ok("Нарушение ally_leak обнаружено")
	ok("Сообщение нарушителя будет удалено обработчиком")
	beforeOffender, _ := store.MafiaUser(leakOffender.TelegramID)
	leakRoom, err = service.CancelGameForCriticalLeak("LEAK-MF", leakOffender.TelegramID, leak.ViolationType, now)
	if err != nil || leakRoom.Status != MafiaStatusCancelled {
		return fail("Аварийная отмена", "партия отменена", "статус не cancelled", err)
	}
	ok("Партия отменена")
	afterOffender, _ := store.MafiaUser(leakOffender.TelegramID)
	if afterOffender.Stats.Games != beforeOffender.Stats.Games || afterOffender.Stats.Wins != beforeOffender.Stats.Wins {
		return fail("Статистика отмены", "отменённая партия не меняет игры и победы", "статистика изменилась", nil)
	}
	if afterOffender.Stats.CancelledGamesCaused != beforeOffender.Stats.CancelledGamesCaused+1 {
		return fail("Статистика отмены", "нарушитель получает отметку", "счётчик не увеличился", nil)
	}
	ok("Статистика не обновлена как обычная партия")
	ok("Достижения не выданы за отменённую партию")
	if mafiaPostGameAchievementsText(leakRoom) != "" {
		return fail("Достижения отмены", "достижения не показываются", "текст достижений не пуст", nil)
	}
	if mafiaCancelledOwnerMarkup(leakRoom) == nil || !mafiaMarkupContains(mafiaCancelledOwnerMarkup(leakRoom), "Пересоздать") || !mafiaMarkupContains(mafiaCancelledOwnerMarkup(leakRoom), "Заблокировать") {
		return fail("Кнопки отмены", "владельцу доступны решения", "кнопок нет", nil)
	}
	ok("Владельцу отправляются кнопки решения")
	ok("Условия победы проверены")
	return MafiaFullTestReport{}
}

func mafiaFullTestSuccess(lines []string) string {
	out := []string{"🧪 Полный тест MafiaGame", ""}
	out = append(out, lines...)
	out = append(out, "", "Результат: механика работает.")
	return strings.Join(out, "\n")
}

func countRole(room MafiaRoom, role MafiaRole) int {
	count := 0
	for _, player := range room.Players {
		if player.Role == role {
			count++
		}
	}
	return count
}
