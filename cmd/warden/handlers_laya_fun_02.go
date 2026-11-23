package main

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func parseLayaFunRequest(text, botUsername string) (layaFunRequest, bool) {
	fields := strings.Fields(text)
	if len(fields) < 2 {
		return layaFunRequest{}, false
	}

	first := normalizeLayaCommandToken(fields[0])
	if first != "лайя" && first != "laya" {
		botUsername = normalizeUsername(botUsername)
		if botUsername == "" || normalizeUsername(first) != botUsername {
			return layaFunRequest{}, false
		}
	}

	cmd := normalizeLayaCommandToken(fields[1])
	rest := strings.TrimSpace(strings.Join(fields[2:], " "))
	switch cmd {
	case "фишки", "команды", "help":
		return layaFunRequest{Kind: layaFunHelp, Rest: rest, Name: "help"}, true
	case "настроение", "вайб", "mood":
		return layaFunRequest{Kind: layaFunMood, Rest: rest, Name: "mood"}, true
	case "монетка", "coin":
		return layaFunRequest{Kind: layaFunCoin, Rest: rest, Name: "coin"}, true
	case "шар", "оракул", "8ball":
		return layaFunRequest{Kind: layaFunBall, Rest: rest, Name: "ball"}, true
	case "выбери", "реши", "choose":
		return layaFunRequest{Kind: layaFunChoice, Rest: rest, Name: "choice"}, true
	case "оцени", "рейтинг", "rate":
		return layaFunRequest{Kind: layaFunRate, Rest: rest, Name: "rate"}, true
	case "шанс", "вероятность", "chance":
		return layaFunRequest{Kind: layaFunChance, Rest: rest, Name: "chance"}, true
	case "вердикт", "приговор", "verdict":
		return layaFunRequest{Kind: layaFunVerdict, Rest: rest, Name: "verdict"}, true
	case "совет", "advice":
		return layaFunRequest{Kind: layaFunAdvice, Rest: rest, Name: "advice"}, true
	case "слух", "сплетня", "rumor":
		return layaFunRequest{Kind: layaFunRumor, Rest: rest, Name: "rumor"}, true
	case "кто", "who":
		return layaFunRequest{Kind: layaFunWho, Rest: rest, Name: "who"}, true
	case "пара", "совместимость", "ship":
		return layaFunRequest{Kind: layaFunPair, Rest: rest, Name: "pair"}, true
	case "титул", "кличка", "title":
		return layaFunRequest{Kind: layaFunTitle, Rest: rest, Name: "title"}, true
	case "извинись", "извини", "apologize", "sorry":
		return layaFunRequest{Kind: layaFunApology, Rest: rest, Name: "apology"}, true
	case "прожарь", "жарь", "roast":
		return layaFunRequest{Kind: layaFunRoast, Rest: rest, Name: "roast"}, true
	case "похвали", "комплимент", "compliment":
		return layaFunRequest{Kind: layaFunCompliment, Rest: rest, Name: "compliment"}, true
	default:
		return layaFunRequest{}, false
	}
}

func layaFunHelpText() string {
	return "Фишки: Лайя шар, выбери, оцени, шанс, вердикт, совет, слух, кто, пара, титул, прожарь, похвали, монетка, настроение. Админы ещё могут: Лайя фас, Лайя извинись."
}

func (a *App) sendLayaApology(c tele.Context, rest string) bool {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil {
		return false
	}
	if !a.isLayaCreator(sender.ID) {
		_ = a.sendLayaFunReplyErr(c, "Эту команду может использовать только создатель Лайи.")
		return true
	}
	if chat.Type == tele.ChatPrivate {
		_ = c.Send("Это работает в группе: ответь на сообщение человека фразой «Лайя извинись перед ним».")
		return true
	}
	target, tail, err := a.targetFromArgs(chat.ID, c, layaApologyArgs(rest))
	if err != nil {
		_ = a.sendLayaFunReplyErr(c, "Укажи пользователя ответом, @username или id: «Лайя извинись перед @user».")
		return true
	}

	reason := strings.Join(layaApologyReasonArgs(tail), " ")
	reply := layaApologyText(target.Name, reason)
	replyTo := msg
	if msg.ReplyTo != nil {
		replyTo = msg.ReplyTo
	}
	if _, err := a.bot.Send(chat, reply, &tele.SendOptions{ReplyTo: replyTo, ThreadID: msg.ThreadID}); err != nil {
		if a.log != nil {
			a.log.Warn("send laya apology failed", zap.Int64("chat", chat.ID), zap.Int64("user", target.ID), zap.Error(err))
		}
		return true
	}
	if a.metrics != nil {
		a.metrics.IncModeration("laya_apology")
	}
	return true
}

func layaApologyArgs(rest string) []string {
	fields := strings.Fields(strings.TrimSpace(rest))
	for len(fields) > 0 {
		token := normalizeLayaCommandToken(strings.Trim(fields[0], ".,:;!?()[]{}<>«»\"'"))
		switch token {
		case "перед", "для", "к", "ко", "ним", "ней", "нему", "нею", "ему", "ей", "него", "нее":
			fields = fields[1:]
		default:
			return fields
		}
	}
	return fields
}

func layaApologyReasonArgs(args []string) []string {
	for len(args) > 0 {
		token := normalizeLayaCommandToken(strings.Trim(args[0], ".,:;!?()[]{}<>«»\"'"))
		switch token {
		case "за", "из-за", "изза", "потому", "что", "ним", "ней", "нему", "нею", "ему", "ей", "него", "нее":
			args = args[1:]
		default:
			return args
		}
	}
	return args
}

func layaApologyText(name, reason string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "слушай"
	}
	reason = strings.TrimSpace(reason)
	if reason != "" {
		return fmt.Sprintf("%s, прости. Я накосячила: %s. Исправляюсь и больше не буду делать вид, что так и задумано.", name, reason)
	}
	return fmt.Sprintf("%s, прости. Я накосячила и признаю это без театра. Исправляюсь.", name)
}

func layaCoinReply() string {
	if randomIndex(2) == 0 {
		return "Орёл. Ну хоть кто-то сегодня выглядит уверенно."
	}
	return "Решка. Символично, конечно."
}

func layaBallReply(rest string, msg *tele.Message) string {
	if strings.TrimSpace(rest) == "" && msg != nil && msg.ReplyTo != nil {
		rest = messageText(msg.ReplyTo)
	}
	if strings.TrimSpace(rest) == "" {
		return "Сначала вопрос задай. Шар не читает пустоту, он её осуждает."
	}
	return randomString([]string{
		"Да. Но не радуйся слишком громко.",
		"Нет. И, честно, это даже к лучшему.",
		"Скорее да. Удивительно, но бывает.",
		"Скорее нет. Реальность решила не рисковать.",
		"Спроси позже. Сейчас вселенная делает вид, что занята.",
		"Шансы есть. Небольшие, как твоя осторожность.",
		"Ответ мутный. Прямо как план.",
	})
}

func layaChoiceReply(rest string) string {
	options := layaChoiceOptions(rest)
	if len(options) < 2 {
		return "Дай варианты через «или», запятую или слэш. Я выбираю, а не вытаскиваю смысл пинцетом."
	}
	chosen := options[randomIndex(len(options))]
	return fmt.Sprintf("Выбираю: %s. Да, ответственность теперь на тебе.", chosen)
}

func layaRateReply(rest string, msg *tele.Message, sender *tele.User) string {
	target := shortText(layaFunTarget(rest, msg, sender), 120)
	score := randomIndex(11)
	return fmt.Sprintf("%s — %d/10. %s", target, score, layaScoreComment(score))
}

func layaChanceReply(rest string, msg *tele.Message, sender *tele.User) string {
	target := shortText(layaFunTarget(rest, msg, sender), 140)
	score := randomIndex(101)
	return fmt.Sprintf("Шанс, что %s: %d%%. %s", target, score, layaChanceComment(score))
}
