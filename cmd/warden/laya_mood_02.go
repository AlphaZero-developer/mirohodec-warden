package main

import (
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

var layaCreatorTargetWords = map[string]bool{
	"creator":     true,
	"developer":   true,
	"owner":       true,
	"владелец":    true,
	"владельца":   true,
	"разраб":      true,
	"разработчик": true,
	"создатель":   true,
	"создателя":   true,
	"создателю":   true,
}

var layaBrotherTargetWords = map[string]bool{
	"bro":      true,
	"brother":  true,
	"брат":     true,
	"брата":    true,
	"брате":    true,
	"братец":   true,
	"братика":  true,
	"братишка": true,
	"братишку": true,
	"братишки": true,
}

var dumbQuestionPhrases = []string{
	"тупой вопрос",
	"тупые вопросы",
	"глупый вопрос",
	"дурацкий вопрос",
	"идиотский вопрос",
	"dumb question",
	"stupid question",
	"silly question",
}

var tinyQuestionWords = map[string]bool{
	"а":    true,
	"как":  true,
	"ну":   true,
	"че":   true,
	"чего": true,
	"что":  true,
	"why":  true,
	"what": true,
	"how":  true,
}

func (a *App) applyLayaMood(cfg AIConfig, event string) AIConfig {
	state := a.activeLayaRuntimeState()
	if state.MoodMode == "off" {
		return cfg
	}
	mood := ""
	if state.MoodMode == "custom" {
		mood = state.MoodPrompt
	} else {
		mood = randomString(layaMoodPrompts)
	}
	if mood == "" {
		return cfg
	}
	extra := mood + "\nЭто только оттенок характера: сохраняй смысл запроса, не нарушай правила модерации и отвечай без лишней многословности."
	if strings.TrimSpace(event) != "" {
		extra += "\nКонтекст ответа: " + strings.TrimSpace(event) + "."
	}
	cfg.SystemPrompt = strings.TrimSpace(cfg.SystemPrompt + "\n\n" + extra)
	return cfg
}

func (a *App) layaMoodReply() string {
	key := randomString(layaMoodReplyKeys)
	reply := strings.TrimSpace(a.t(key))
	if reply == "" || reply == key {
		return "Настроение: сарказм заряжен, терпение на минималках."
	}
	return reply
}

func (a *App) layaReactionText(kind string) string {
	keys := layaReactionKeys[kind]
	if len(keys) == 0 {
		keys = []string{"laya_reaction_default"}
	}
	key := randomString(keys)
	reply := strings.TrimSpace(a.t(key))
	if reply == "" || reply == key {
		reply = strings.TrimSpace(a.t("laya_reaction_default"))
	}
	return reply
}

func (a *App) maybeSendLayaReaction(chat *tele.Chat, msg *tele.Message, kind string) bool {
	if a == nil || a.bot == nil || chat == nil || msg == nil {
		return false
	}
	reply := a.layaReactionText(kind)
	if strings.TrimSpace(reply) == "" {
		return false
	}
	if !a.allowLayaReaction(chat.ID, msg.ThreadID, kind, time.Now().UTC()) {
		return false
	}

	opts := &tele.SendOptions{ThreadID: msg.ThreadID}
	if kind != "spam" {
		opts.ReplyTo = msg
	}
	opts.ReplyMarkup = a.layaInlineMarkup(chat, msg.Sender)
	if _, err := a.bot.Send(chat, reply, opts); err != nil {
		if a.log != nil {
			a.log.Warn("send laya reaction failed",
				zap.Int64("chat", chat.ID),
				zap.Int("thread", msg.ThreadID),
				zap.String("kind", kind),
				zap.Error(err),
			)
		}
		return false
	}
	if a.log != nil {
		a.log.Info("laya reaction sent",
			zap.Int64("chat", chat.ID),
			zap.Int("thread", msg.ThreadID),
			zap.String("kind", kind),
		)
	}
	if a.metrics != nil {
		a.metrics.IncModeration("laya_reaction_" + kind)
	}
	return true
}
