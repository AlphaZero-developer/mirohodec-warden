package main

import (
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func mafiaProfileCreatedText(user MafiaRegisteredUser) string {
	return fmt.Sprintf("✅ Персонаж создан.\n\nИмя: %s\n\nПредыстория:\n%s\n\nТеперь вас можно приглашать в комнаты Мафии.", mafiaUserCharacterName(user), strings.TrimSpace(user.Profile.Backstory))
}

func mafiaGeneratedProfileText(profile MafiaProfile) string {
	return fmt.Sprintf("🎲 Лайя предлагает персонажа:\n\nИмя: %s\n\nПредыстория:\n%s\n\nИспользовать этого персонажа?", strings.TrimSpace(profile.CharacterName), strings.TrimSpace(profile.Backstory))
}

func mafiaAchievementsText(user MafiaRegisteredUser) string {
	list := mafiaAchievementListText(user.Stats.Achievements, 0)
	if list == "" {
		list = "Пока достижений нет.\n\nСыграйте партию до конца, и Лайя выдаст первые отметки после финала."
	}
	return fmt.Sprintf("🏆 Достижения MafiaGame\n\nИгрок: %s\nПолучено: %d\n\n%s", mafiaUserCharacterName(user), len(user.Stats.Achievements), list)
}

func mafiaAchievementListText(ids []string, limit int) string {
	if len(ids) == 0 {
		return ""
	}
	lines := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		lines = append(lines, "• "+mafiaAchievementLabel(id))
		if limit > 0 && len(lines) >= limit {
			remaining := len(ids) - len(lines)
			if remaining > 0 {
				lines = append(lines, fmt.Sprintf("• ещё %d", remaining))
			}
			break
		}
	}
	return strings.Join(lines, "\n")
}

func mafiaAchievementLabel(id string) string {
	switch id {
	case "first_game":
		return "Первая партия — сыграть первую игру"
	case "first_knife":
		return "Первый нож — победить за мафию"
	case "sharp_check":
		return "Точная проверка — победить за детектива"
	case "night_shift":
		return "Ночная смена — победить за доктора"
	case "last_thought":
		return "Последняя мысль — оставить завещание"
	case "city_mistake":
		return "Ошибка города — погибнуть мирным при победе мафии"
	case "first_win":
		return "Первая победа — выиграть любую партию"
	case "ranked_debut":
		return "В рейтинге — завершить первую ranked-партию"
	case "mask_survived":
		return "Маска не упала — победить за мафию и выжить"
	case "city_stands":
		return "Город стоит — победить за город и выжить"
	case "posthumous_win":
		return "Победа после смерти — выиграть, не дожив до финала"
	case "five_games":
		return "Пять ночей — сыграть 5 партий"
	case "ten_games":
		return "Десять дел — сыграть 10 партий"
	case "five_wins":
		return "Серия доверия — выиграть 5 партий"
	case "mafia_veteran":
		return "Тень с опытом — выиграть 3 раза за мафию"
	case "civilian_veteran":
		return "Голос города — выиграть 3 раза за город"
	case "detective_veteran":
		return "Синий след — выиграть 3 раза за детектива"
	case "doctor_veteran":
		return "Дежурный врач — выиграть 3 раза за доктора"
	default:
		return id
	}
}

func randomMafiaProfile(userID int64) MafiaProfile {
	names := []string{
		"Виктор Сухарь",
		"Арчибальд Крюк",
		"София Туманова",
		"Марта Лис",
		"Григорий Безсонов",
		"Алиса Дымова",
		"Павел Сомов",
		"Ева Чердак",
	}
	backstories := []string{
		"Бывший бухгалтер, который однажды исчез на три дня и вернулся с чужими документами.",
		"Ночной кондитер, уверенный, что лучшие алиби пахнут корицей.",
		"Скрипачка из старого театра, которая слишком внимательно слушает чужие паузы.",
		"Почтальон, знающий все адреса города и ни одного спокойного секрета.",
		"Бывший нотариус, который слишком много знал о чужих завещаниях.",
		"Смотритель музея, который уверяет, что все экспонаты двигаются только по правилам.",
		"Бариста, запоминающий людей по заказам и слишком долгим взглядам.",
		"Таксист, который видел город ночью и предпочитает не уточнять подробности.",
	}
	seed := time.Now().UnixNano() + userID
	if seed < 0 {
		seed = -seed
	}
	return MafiaProfile{
		CharacterName: names[int(seed%int64(len(names)))],
		Backstory:     backstories[int((seed/7)%int64(len(backstories)))],
	}
}

func mafiaProfileRegistrationMarkup() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(mafiaButton("🎲 Сгенерировать персонажа", "mafia:profile_generate")),
		markup.Row(mafiaButton("⬅️ Назад", "mafia:back")),
	)
	return markup
}

func mafiaProfileMarkup(canEdit bool) *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	var rows []tele.Row
	if canEdit {
		rows = append(rows,
			markup.Row(mafiaButton("✏️ Изменить имя", "mafia:profile_edit_name")),
			markup.Row(mafiaButton("📜 Изменить предысторию", "mafia:profile_edit_backstory")),
			markup.Row(mafiaButton("🎲 Сгенерировать заново", "mafia:profile_generate")),
		)
	}
	rows = append(rows, markup.Row(mafiaButton("🏆 Достижения", "mafia:achievements")))
	rows = append(rows, markup.Row(mafiaButton("🎭 Играть", "mafia:back")))
	markup.Inline(rows...)
	return markup
}

func mafiaAchievementsMarkup() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(mafiaButton("👤 Мой персонаж", "mafia:profile")),
		markup.Row(mafiaButton("⬅️ Назад", "mafia:back")),
	)
	return markup
}

func mafiaProfileGeneratedMarkup() *tele.ReplyMarkup {
	markup := &tele.ReplyMarkup{}
	markup.Inline(
		markup.Row(mafiaButton("✅ Использовать", "mafia:profile_use_generated")),
		markup.Row(mafiaButton("🔁 Сгенерировать ещё", "mafia:profile_generate_again")),
		markup.Row(mafiaButton("✏️ Ввести вручную", "mafia:profile_manual_input")),
	)
	return markup
}

type mafiaCallbackGuard struct {
	RequiresPlayer bool
	RequiresOwner  bool
	RequiresAlive  bool
	RequiredRole   MafiaRole
	RequiredPhase  MafiaRoomStatus
	AllowOwnerOnly bool
}
