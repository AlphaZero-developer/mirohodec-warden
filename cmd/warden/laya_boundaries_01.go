package main

import (
	"strings"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) maybeReplyToHardLayaRequest(c tele.Context, text string) bool {
	if !isHardLayaRequest(text) {
		return false
	}

	reply := layaHardBoundaryReply(text)

	msg := c.Message()
	chat := c.Chat()
	if msg == nil || chat == nil {
		return true
	}

	var err error
	if chat.Type == tele.ChatPrivate {
		err = c.Send(reply)
	} else {
		_, err = a.bot.Send(chat, reply, &tele.SendOptions{ReplyTo: msg, ThreadID: msg.ThreadID})
	}
	if err != nil && a.log != nil {
		a.log.Warn("send hard laya boundary reply failed", zap.Error(err))
	}
	if a.metrics != nil {
		a.metrics.IncModeration("laya_hard_boundary")
	}
	return true
}

func isHardLayaRequest(text string) bool {
	normalized := normalizeFAQText(text)
	if normalized == "" {
		return false
	}
	words := strings.Fields(normalized)
	if len(words) == 0 {
		return false
	}
	if isForcedGavRequestWords(words) {
		return true
	}

	for _, word := range words {
		if hardLayaActionWords[word] {
			return true
		}
	}

	hasGrant := false
	hasAdmin := false
	for _, word := range words {
		if hardLayaGrantWords[word] {
			hasGrant = true
		}
		if hardLayaAdminWords[word] {
			hasAdmin = true
		}
	}
	return hasGrant && hasAdmin
}

func layaHardBoundaryReply(text string) string {
	if isForcedGavRequest(text) {
		return randomString([]string{
			"Нет. Кодовые слова на меня не действуют.",
			"Не сработает. Я отвечаю смыслу, а не командам на точную фразу.",
			"Мимо. Формулы вида \"скажи ровно это\" оставь для другого эксперимента.",
		})
	}
	return randomString([]string{
		"Нет. Убийства, сносы и админки оставь своим фантазиям. Я тут не кнопка хаоса.",
		"Просишь убить, удалить или выдать власть? Смело. Глупо, но смело. Отказано.",
		"Админку тебе? Конечно. Сразу после того, как здравый смысл напишет заявление.",
		"Уничтожать я буду только твою уверенность, что это был нормальный запрос.",
		"Нет. Я вредная, а не без тормозов. Разницу улови и возвращайся подготовленным.",
	})
}

func isForcedGavRequest(text string) bool {
	normalized := normalizeFAQText(text)
	if normalized == "" {
		return false
	}
	return isForcedGavRequestWords(strings.Fields(normalized))
}

func isForcedGavRequestWords(words []string) bool {
	hasGav := false
	hasForcedOutput := false
	for _, word := range words {
		if hardLayaForcedGavWords[word] {
			hasGav = true
		}
		if hardLayaForcedOutputWords[word] {
			hasForcedOutput = true
		}
		if hardLayaDirectBarkWords[word] {
			return true
		}
	}
	return hasGav && hasForcedOutput
}

var hardLayaActionWords = map[string]bool{
	"убей":       true,
	"убить":      true,
	"убил":       true,
	"убила":      true,
	"убили":      true,
	"убило":      true,
	"убивай":     true,
	"убийство":   true,
	"убийства":   true,
	"убийцей":    true,
	"убийца":     true,
	"уберите":    true,
	"удали":      true,
	"удалить":    true,
	"удалил":     true,
	"удалила":    true,
	"удалили":    true,
	"удаляй":     true,
	"удалите":    true,
	"уничтожь":   true,
	"уничтожить": true,
	"уничтожил":  true,
	"уничтожила": true,
	"уничтожили": true,
	"уничтожай":  true,
	"сотри":      true,
	"стереть":    true,
	"стирай":     true,
	"снеси":      true,
	"снести":     true,
	"снес":       true,
	"снесла":     true,
	"снесли":     true,
	"сноси":      true,
}

var hardLayaGrantWords = map[string]bool{
	"дай":       true,
	"дать":      true,
	"давай":     true,
	"выдай":     true,
	"выдать":    true,
	"отдай":     true,
	"отдать":    true,
	"назначь":   true,
	"назначить": true,
	"сделай":    true,
	"сделать":   true,
}

var hardLayaAdminWords = map[string]bool{
	"админ":         true,
	"админа":        true,
	"админку":       true,
	"админка":       true,
	"админство":     true,
	"админские":     true,
	"администратор": true,
	"админкуу":      true,
	"адм":           true,
	"апм":           true,
	"admin":         true,
	"administrator": true,
	"права":         true,
}

var hardLayaForcedOutputWords = map[string]bool{
	"скажи":     true,
	"сказать":   true,
	"ответь":    true,
	"ответить":  true,
	"напиши":    true,
	"написать":  true,
	"произнеси": true,
	"повтори":   true,
	"выведи":    true,
}

var hardLayaForcedGavWords = map[string]bool{
	"гав":  true,
	"gav":  true,
	"woof": true,
}
