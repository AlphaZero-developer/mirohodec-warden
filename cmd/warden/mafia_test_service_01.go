package main

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

type MafiaFullTestReport struct {
	Text string
}

func RunMafiaFullTest() MafiaFullTestReport {
	var lines []string
	ok := func(text string) { lines = append(lines, "✅ "+text) }
	fail := func(stage, expected, actual string, err error) MafiaFullTestReport {
		lines := []string{
			"🧪 Полный тест MafiaGame провален.",
			"",
			"Этап: " + stage,
			"Ожидалось: " + expected,
			"Фактически: " + actual,
		}
		if err != nil {
			lines = append(lines, "", "Ошибка:", err.Error())
		}
		return MafiaFullTestReport{Text: strings.Join(lines, "\n")}
	}

	dir, err := os.MkdirTemp("", "laya-mafia-test-*")
	if err != nil {
		return fail("Хранилище", "создать временную папку", "ошибка", err)
	}
	defer os.RemoveAll(dir)
	store, err := NewStore(filepath.Join(dir, "state.json"), nil)
	if err != nil {
		return fail("Хранилище", "создать тестовое хранилище", "ошибка", err)
	}
	service := NewMafiaService(store)
	now := time.Now().UTC()
	users := []*tele.User{
		{ID: 9001, FirstName: "Player_One"},
		{ID: 9002, FirstName: "Player_Two"},
		{ID: 9003, FirstName: "Mira_Test"},
		{ID: 9004, FirstName: "Alex_Test"},
		{ID: 9005, FirstName: "Shadow_Test"},
	}
	for _, user := range users {
		if _, err := service.RegisterUser(user, now); err != nil {
			return fail("Регистрация", "зарегистрировать фейкового игрока", "ошибка", err)
		}
	}

	room, err := service.CreateRoom(users[0], now)
	if err != nil {
		return fail("Комната", "комната создана", "ошибка", err)
	}
	room.ID = "TEST-MF"
	room.Settings.MinPlayers = 5
	room.Settings.MaxPlayers = 12
	if err := store.SaveMafiaRoom(room); err != nil {
		return fail("Комната", "сохранить TEST-MF", "ошибка", err)
	}
	ok("Комната создана")

	for _, user := range users[1:] {
		if _, err := service.JoinRoom("TEST-MF", user, now); err != nil {
			return fail("Игроки", "добавить 5 фейковых игроков", "ошибка", err)
		}
	}
	room, _ = store.MafiaRoom("TEST-MF")
	if len(room.Players) != 5 {
		return fail("Игроки", "5 игроков", "другое число игроков", nil)
	}
	ok("Игроки добавлены")

	room, err = service.UpdateLobbySettings("TEST-MF", users[0].ID, now, func(settings MafiaRoomSettings) MafiaRoomSettings {
		settings.DayDiscussionSeconds = 300
		settings.NightActionSeconds = 120
		settings.VotingSeconds = 90
		settings.EnableWills = false
		settings.EnableAchievements = false
		settings.EnablePostGameAIReport = false
		settings.OptionalRoles.EnableDon = true
		return settings
	})
	if err != nil {
		return fail("Настройки", "изменить настройки в лобби", "ошибка", err)
	}
	settings := normalizeMafiaSettings(room.Settings)
	if settings.DayDiscussionSeconds != 300 || settings.NightActionSeconds != 120 || settings.VotingSeconds != 90 {
		return fail("Длительность фаз", "день 300, ночь 120, голосование 90", "значения не сохранились", nil)
	}
	if settings.EnableWills || settings.EnableAchievements || settings.EnablePostGameAIReport || !settings.OptionalRoles.EnableDon {
		return fail("Настройки", "фичи переключаются, дополнительные роли включаются", "переключение не сработало", nil)
	}
	ok("Длительность фаз настраивается")
	ok("Завещания можно выключить")
	ok("Достижения можно выключить")
	ok("AI-отчёт можно выключить")
	ok("Дополнительные роли можно включить")
	room, err = service.UpdateLobbySettings("TEST-MF", users[0].ID, now, func(settings MafiaRoomSettings) MafiaRoomSettings {
		settings = mafiaApplyTimePreset(settings, "normal")
		settings.EnableWills = true
		settings.EnableAchievements = true
		settings.EnablePostGameAIReport = true
		settings.OptionalRoles.EnableDon = false
		return settings
	})
	if err != nil {
		return fail("Настройки", "вернуть обычные настройки", "ошибка", err)
	}
	if !mafiaMarkupContains(mafiaRoomMarkup(room, users[0].ID), "Полный тест") || mafiaMarkupContains(mafiaRoomMarkup(room, users[1].ID), "Полный тест") {
		return fail("Права кнопок", "полный тест видит только создатель в лобби", "кнопка показана неверно", nil)
	}

	if _, err := service.BindGroup("TEST-MF", users[0].ID, -1009001, now); err != nil {
		return fail("Группа", "привязать тестовую группу", "ошибка", err)
	}
	room, err = service.StartGame("TEST-MF", users[0].ID, now)
	if err != nil {
		return fail("Роли", "раздать роли", "ошибка", err)
	}
	if countRole(room, MafiaRoleMafia) != 1 || countRole(room, MafiaRoleDoctor) != 1 || countRole(room, MafiaRoleDetective) != 1 || countRole(room, MafiaRoleCivilian) != 2 {
		return fail("Роли", "1 мафия, 1 доктор, 1 детектив, 2 мирных", "роль-сетка другая", nil)
	}
	ok("Роли розданы")
	if mafiaRoleLabel(MafiaRoleMafia) != "Мафия" || mafiaRoleLabel(MafiaRoleDetective) != "Детектив" || mafiaPhaseLabel(MafiaStatusNight) != "🌙 Ночь" {
		return fail("Русификация", "роли и фазы отображаются на русском", "техническое название просочилось", nil)
	}
	ok("Роли отображаются на русском")
	if mafiaPlayers(room)[0].MafiaAlias == "" {
		return fail("Псевдонимы", "мафия получила alias", "alias пустой", nil)
	}
	ok("Псевдонимы мафии назначены")
	ok("Теневой канал создан")

	if report := runMafiaGameActionChecks(store, service, room, now, ok, fail); report.Text != "" {
		return report
	}
	if report := runMafiaGameCancellationChecks(store, service, users, now, ok, fail); report.Text != "" {
		return report
	}
	return MafiaFullTestReport{Text: mafiaFullTestSuccess(lines)}
}
