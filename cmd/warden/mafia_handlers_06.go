package main

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) callbackMafiaProfileGenerate(c tele.Context) error {
	if chat := c.Chat(); chat == nil || chat.Type != tele.ChatPrivate {
		return c.Send("⛔ Персонаж настраивается в ЛС с Лайей.")
	}
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	user, err := a.store.RegisterMafiaUser(sender, time.Now().UTC())
	if err != nil {
		return c.Send("⛔ Не удалось зарегистрировать игрока.")
	}
	if !a.mafiaCanEditProfile(sender.ID) {
		return c.Send("⛔ Нельзя менять персонажа во время активной партии.\n\nДождитесь окончания игры.")
	}
	profile := randomMafiaProfile(sender.ID)
	settings := normalizeMafiaUserSettings(user.Settings)
	clearMafiaProfileInput(&settings)
	settings.GeneratedCharacterName = profile.CharacterName
	settings.GeneratedBackstory = profile.Backstory
	if err := a.store.SetMafiaUserSettings(sender.ID, settings); err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	return c.Send(mafiaGeneratedProfileText(profile), mafiaProfileGeneratedMarkup())
}

func (a *App) callbackMafiaProfileUseGenerated(c tele.Context) error {
	sender := c.Sender()
	if sender == nil {
		return c.Send(a.t("no_sender"))
	}
	user, ok := a.store.MafiaUser(sender.ID)
	if !ok {
		return a.startMafiaProfileRegistration(c)
	}
	if !a.mafiaCanEditProfile(sender.ID) {
		return c.Send("⛔ Нельзя менять персонажа во время активной партии.\n\nДождитесь окончания игры.")
	}
	settings := normalizeMafiaUserSettings(user.Settings)
	profile := MafiaProfile{
		CharacterName: strings.TrimSpace(settings.GeneratedCharacterName),
		Backstory:     strings.TrimSpace(settings.GeneratedBackstory),
		CreatedAt:     user.Profile.CreatedAt,
	}
	if profile.CharacterName == "" || profile.Backstory == "" {
		return a.callbackMafiaProfileGenerate(c)
	}
	if err := a.store.SetMafiaProfile(sender.ID, profile); err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	clearMafiaProfileInput(&settings)
	if err := a.store.SetMafiaUserSettings(sender.ID, settings); err != nil {
		return c.Send("⛔ " + a.localizeMafiaErr(err))
	}
	updated, _ := a.store.MafiaUser(sender.ID)
	return c.Send(mafiaProfileCreatedText(updated), mafiaProfileMarkup(true))
}

func (a *App) mafiaCanEditProfile(userID int64) bool {
	room, ok := a.activeMafiaRoomForUser(userID)
	if !ok {
		return true
	}
	switch room.Status {
	case MafiaStatusLobby, MafiaStatusFinished, MafiaStatusCancelled:
		return true
	default:
		return false
	}
}

func clearMafiaProfileInput(settings *MafiaUserSettings) {
	if settings == nil {
		return
	}
	settings.ProfileInputMode = ""
	settings.PendingCharacterName = ""
	settings.GeneratedCharacterName = ""
	settings.GeneratedBackstory = ""
}

func validateMafiaCharacterName(text string) (string, string) {
	name := strings.TrimSpace(text)
	switch {
	case name == "":
		return "", "⚠️ Имя не может быть пустым.\n\nВведите любое имя персонажа."
	case len([]rune(name)) > mafiaProfileNameMaxRunes:
		return "", "⚠️ Имя слишком длинное.\n\nСократите его, чтобы Лайя могла нормально показывать его в игре."
	default:
		return name, ""
	}
}

func validateMafiaBackstory(text string) (string, string) {
	backstory := strings.TrimSpace(text)
	switch {
	case backstory == "":
		return "", "⚠️ Предыстория не может быть пустой.\n\nНапишите хотя бы пару слов."
	case len([]rune(backstory)) > mafiaProfileBackstoryMaxRunes:
		return "", "⚠️ Предыстория слишком длинная.\n\nСократите её, чтобы она нормально отображалась в партии."
	default:
		return backstory, ""
	}
}

func mafiaRegistrationIntroText() string {
	return "🎭 Как работает MafiaGame\n\n1. Игра идёт в подключённой группе. Там проходят день, обсуждение, голосование и объявления Лайи.\n\n2. Личные действия идут в ЛС с Лайей. Роли, лечение, проверки и теневой канал мафии не публикуются в группе.\n\n3. Ночью общий чат молчит и закрыт: Лайя ограничивает сообщения до утра. Писать можно только там, где Лайя дала личные кнопки.\n\n4. Мёртвые игроки больше не пишут в общий чат, не голосуют и не делают действия.\n\n5. Нельзя доказывать роль через Лайю: пересылать ЛС, скринить роль, писать про кнопки или теневой канал. В этой комнате также нельзя прямо писать свою роль: «я мафия», «я доктор», «я детектив», «я мирный».\n\n6. Обычные обвинения разрешены: можно подозревать, блефовать и голосовать против кого угодно.\n\n7. Персонаж нужен для атмосферы. Имя и предыстория не влияют на роль, баланс, проверки, победу или поражение.\n\nПосле ознакомления создайте персонажа: имя и короткую предысторию."
}

func mafiaRegistrationIntroMarkup() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(mafiaButton("✅ Ознакомился", "mafia:register_intro_ack")),
		markup.Row(mafiaButton("📖 Правила", "mafia:rules"), mafiaButton("⬅️ Назад", "mafia:back")),
	)
	return markup
}

func mafiaProfileRegistrationPromptText() string {
	return "🎭 Регистрация в Мафии\n\nВведите вымышленное имя персонажа.\n\nЖелательно придумайте оригинальное имя, а не просто оскорбление. Так играть будет смешнее и атмосфернее.\n\nПримеры:\n— Арчибальд Крюк\n— София Туманова\n— Григорий Безсонов\n— Марта Лис"
}

func mafiaProfileBackstoryPromptText() string {
	return "📜 Предыстория персонажа\n\nТеперь напишите короткую предысторию персонажа.\n\nОна не влияет на роль и нужна только для атмосферы партии.\n\nПример:\n\"Бывший аптекарь, который слишком хорошо разбирается в ядах. Уверяет, что теперь работает честно.\""
}

func mafiaProfileText(user MafiaRegisteredUser) string {
	settings := normalizeMafiaUserSettings(user.Settings)
	stats := user.Stats
	winrate := 0
	if stats.Games > 0 {
		winrate = int(float64(stats.Wins) / float64(stats.Games) * 100)
	}
	telegram := strings.TrimSpace(user.DisplayName)
	if telegram == "" {
		telegram = fmt.Sprintf("user %d", user.TelegramID)
	}
	if strings.TrimSpace(user.Username) != "" {
		telegram = fmt.Sprintf("%s (@%s)", telegram, strings.TrimPrefix(strings.TrimSpace(user.Username), "@"))
	}
	invites := "включены"
	if !settings.AllowMafiaInvites {
		invites = "выключены"
	} else if settings.InviteOnlyFromFavorites {
		invites = "только от избранных"
	}
	discussionChat := "не запомнен"
	if settings.LastDiscussionGroupChatID != 0 {
		discussionChat = fmt.Sprintf("запомнен (%d)", settings.LastDiscussionGroupChatID)
	}
	registeredAt := "неизвестно"
	if user.RegisteredAt > 0 {
		registeredAt = time.Unix(user.RegisteredAt, 0).In(time.Local).Format("02.01.2006")
	}
	achievements := mafiaAchievementListText(stats.Achievements, 6)
	if achievements == "" {
		achievements = "пока нет"
	}
	backstory := strings.TrimSpace(user.Profile.Backstory)
	if backstory == "" {
		backstory = "не указана"
	}
	return fmt.Sprintf("👤 Мой персонаж\n\nИмя: %s\nTelegram: %s\nВ MafiaGame с: %s\n\nПредыстория:\n%s\n\n📊 Игрок\nИгр: %d · Побед: %d · Винрейт: %d%% · Смертей: %d\nПобеды: мафия %d · город %d · детектив %d · доктор %d\nРаскрытий роли: %d · Отмен по вине игрока: %d\n\n🏆 Достижения: %d\n%s\n\n🔔 Приглашения: %s\n💬 Чат обсуждений: %s",
		mafiaUserCharacterName(user),
		telegram,
		registeredAt,
		backstory,
		stats.Games,
		stats.Wins,
		winrate,
		stats.Deaths,
		stats.MafiaWins,
		stats.CivilianWins,
		stats.DetectiveWins,
		stats.DoctorWins,
		stats.RoleReveals,
		stats.CancelledGamesCaused,
		len(stats.Achievements),
		achievements,
		invites,
		discussionChat,
	)
}
