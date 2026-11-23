package main

import (
	"fmt"
	"strings"

	tele "gopkg.in/telebot.v3"
)

func layaChanceComment(score int) string {
	switch {
	case score <= 10:
		return "Даже надежда отвернулась."
	case score <= 35:
		return "Теоретически можно, практически смешно."
	case score <= 65:
		return "Монетка нервно зависла в воздухе."
	case score <= 90:
		return "Выглядит подозрительно реально."
	default:
		return "Почти неизбежно. Готовь лицо попроще."
	}
}

func layaVerdictReply(rest string, msg *tele.Message, sender *tele.User) string {
	target := shortText(layaFunTarget(rest, msg, sender), 140)
	return fmt.Sprintf("%s — %s", target, randomString([]string{
		"виновно в самоуверенности.",
		"условно допущено к существованию.",
		"отправляется подумать над поведением.",
		"получает предупреждающий взгляд.",
		"не оправдано, но мне лень оформлять документы.",
		"оставлено под наблюдением. Очень пристальным.",
	}))
}

func layaAdviceReply(rest string) string {
	topic := strings.TrimSpace(rest)
	prefix := "Совет"
	if topic != "" {
		prefix = "Совет про " + shortText(topic, 90)
	}
	return prefix + ": " + randomString([]string{
		"не делай вид, что это был план. Сделай новый.",
		"если сомневаешься, сначала перестань шуметь.",
		"выбери простое решение. Сложное тебя уже выбрало.",
		"не спорь с реальностью без запасного выхода.",
		"пауза перед ответом иногда выглядит как интеллект. Пользуйся.",
		"убери лишнее. Особенно уверенность.",
	})
}

func (a *App) layaRumorReply(c tele.Context, rest string) string {
	target := strings.TrimSpace(rest)
	if target == "" {
		if rec, ok := a.randomLayaMember(c, 0); ok {
			target = rec.DisplayName
		}
	}
	if target == "" {
		target = "этот чат"
	}
	return fmt.Sprintf("Слух: %s %s", shortText(target, 90), randomString([]string{
		"сегодня слишком уверенно молчит.",
		"что-то задумал, но пока сам не понял что.",
		"делает вид, что контролирует ситуацию.",
		"подозрительно близко к нормальности.",
		"уже на тонкой грани между планом и импровизацией.",
	}))
}

func (a *App) layaWhoReply(c tele.Context, rest string) string {
	chat := c.Chat()
	sender := c.Sender()
	if chat == nil || sender == nil || chat.Type == tele.ChatPrivate {
		return "В личке выбирать особо некого. Кроме тебя, но это слишком легко."
	}
	rec, ok := a.randomLayaMember(c, sender.ID)
	if !ok {
		return "Я бы выбрала, но чат пока слишком пустой для красивого обвинения."
	}
	question := strings.TrimSpace(rest)
	if question == "" {
		question = "сегодня главный подозреваемый"
	}
	return fmt.Sprintf("%s — %s. Выглядит достаточно убедительно.", shortText(question, 100), rec.DisplayName)
}

func (a *App) layaPairReply(c tele.Context) string {
	msg := c.Message()
	sender := c.Sender()
	if sender == nil {
		return "Пара распалась ещё до начала. Эффектно."
	}

	first := displayNameFromUser(sender)
	second := ""
	if msg != nil && msg.ReplyTo != nil && msg.ReplyTo.Sender != nil && msg.ReplyTo.Sender.ID != sender.ID {
		second = displayNameFromUser(msg.ReplyTo.Sender)
	} else if rec, ok := a.randomLayaMember(c, sender.ID); ok {
		second = rec.DisplayName
	}
	if second == "" {
		return "Для пары нужен ещё кто-то активный. Не романтично, зато честно."
	}

	score := randomIndex(101)
	return fmt.Sprintf("%s + %s: совместимость %d%%. %s", first, second, score, layaPairComment(score))
}

func layaPairComment(score int) string {
	switch {
	case score <= 20:
		return "Лучше общайтесь через переводчика здравого смысла."
	case score <= 45:
		return "Искра есть. Где-то вдалеке. Возможно, это короткое замыкание."
	case score <= 70:
		return "Работать может, если не трогать руками."
	case score <= 90:
		return "Подозрительно неплохо. Я слежу."
	default:
		return "Слишком идеально. Значит, где-то подвох."
	}
}

func layaTitleReply(rest string, msg *tele.Message, sender *tele.User) string {
	target := shortText(layaFunTarget(rest, msg, sender), 80)
	return fmt.Sprintf("%s теперь %s.", target, randomString([]string{
		"министр сомнительных решений",
		"архитектор внезапной уверенности",
		"магистр громких выводов",
		"хранитель странного вайба",
		"специалист по красивым ошибкам",
		"почётный наблюдатель хаоса",
		"главный по драматичным паузам",
	}))
}

func layaScoreComment(score int) string {
	switch {
	case score <= 2:
		return "Смело. В плохом смысле."
	case score <= 4:
		return "Живёт, но лучше бы с планом."
	case score <= 6:
		return "Терпимо. Уже почти не стыдно."
	case score <= 8:
		return "Нормально. На этом редком чуде и остановимся."
	default:
		return "Редкий случай, когда я почти довольна."
	}
}

func layaFunTarget(rest string, msg *tele.Message, sender *tele.User) string {
	if strings.TrimSpace(rest) != "" {
		return strings.TrimSpace(rest)
	}
	if msg != nil && msg.ReplyTo != nil {
		name := "это"
		if msg.ReplyTo.Sender != nil {
			name = displayNameFromUser(msg.ReplyTo.Sender)
		}
		if text := strings.TrimSpace(messageText(msg.ReplyTo)); text != "" {
			return fmt.Sprintf("%s: %s", name, shortText(text, 300))
		}
		return name
	}
	if sender != nil {
		return displayNameFromUser(sender)
	}
	return "это"
}

func layaChoiceOptions(rest string) []string {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return nil
	}
	normalized := strings.NewReplacer(",", " или ", "/", " или ", "|", " или ", ";", " или ").Replace(rest)
	if options := layaChoiceOptionsByWord(normalized); len(options) >= 2 {
		return options
	}
	return nil
}
