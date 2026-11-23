package main

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

const (
	layaRoastSystemPrompt      = `Режим "прожарь": дай короткий язвительный подкол по цели, но оставайся Лайей. Без угроз, травли, дискриминационных оскорблений, деанона, сексуальных унижений и призывов к вреду. Не объясняй правила.`
	layaComplimentSystemPrompt = `Режим "кривой комплимент": дай короткий комплимент с язвительным хвостиком, но без жестокости. Оставайся Лайей, отвечай живо и по-русски.`
)

type layaFunKind int

const (
	layaFunUnknown layaFunKind = iota
	layaFunHelp
	layaFunMood
	layaFunCoin
	layaFunBall
	layaFunChoice
	layaFunRate
	layaFunRoast
	layaFunCompliment
	layaFunChance
	layaFunVerdict
	layaFunAdvice
	layaFunRumor
	layaFunWho
	layaFunPair
	layaFunTitle
	layaFunApology
)

type layaFunRequest struct {
	Kind layaFunKind
	Rest string
	Name string
}

func (a *App) maybeHandleLayaFun(c tele.Context) bool {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || sender.IsBot {
		return false
	}

	req, ok := parseLayaFunRequest(messageText(msg), a.botUsername())
	if !ok {
		return false
	}

	switch req.Kind {
	case layaFunHelp:
		return a.sendLayaFunReply(c, layaFunHelpText())
	case layaFunMood:
		return a.sendLayaFunReply(c, a.layaMoodReply())
	case layaFunCoin:
		return a.sendLayaFunReply(c, layaCoinReply())
	case layaFunBall:
		return a.sendLayaFunReply(c, layaBallReply(req.Rest, msg))
	case layaFunChoice:
		return a.sendLayaFunReply(c, layaChoiceReply(req.Rest))
	case layaFunRate:
		return a.sendLayaFunReply(c, layaRateReply(req.Rest, msg, sender))
	case layaFunChance:
		return a.sendLayaFunReply(c, layaChanceReply(req.Rest, msg, sender))
	case layaFunVerdict:
		return a.sendLayaFunReply(c, layaVerdictReply(req.Rest, msg, sender))
	case layaFunAdvice:
		return a.sendLayaFunReply(c, layaAdviceReply(req.Rest))
	case layaFunRumor:
		return a.sendLayaFunReply(c, a.layaRumorReply(c, req.Rest))
	case layaFunWho:
		return a.sendLayaFunReply(c, a.layaWhoReply(c, req.Rest))
	case layaFunPair:
		return a.sendLayaFunReply(c, a.layaPairReply(c))
	case layaFunTitle:
		return a.sendLayaFunReply(c, layaTitleReply(req.Rest, msg, sender))
	case layaFunApology:
		return a.sendLayaApology(c, req.Rest)
	case layaFunRoast:
		return a.sendLayaFunAIReply(c, req, layaRoastSystemPrompt, "Я бы прожарила, но оно само уже хрустит.")
	case layaFunCompliment:
		return a.sendLayaFunAIReply(c, req, layaComplimentSystemPrompt, "Неплохо. Почти подозрительно.")
	default:
		return false
	}
}

func (a *App) sendLayaFunAIReply(c tele.Context, req layaFunRequest, systemExtra, fallback string) bool {
	msg := c.Message()
	sender := c.Sender()
	if msg == nil || sender == nil {
		return false
	}

	cfg := a.config().AI
	cfg.SystemPrompt = strings.TrimSpace(cfg.SystemPrompt + "\n\n" + systemExtra)
	cfg.FallbackReply = fallback

	target := layaFunTarget(req.Rest, msg, sender)
	prompt := fmt.Sprintf("Цель: %s\nОтветь одной-двумя короткими фразами.", target)
	reply, status, elapsed, aiErr := a.generateAIReply(cfg, sender, prompt)
	reply = trimRunes(reply, maxAIReplyChars(cfg))

	sendErr := a.sendLayaFunReplyErr(c, reply)
	if aiErr == nil {
		if sendErr != nil {
			status = "ответ получен, отправка в Telegram не удалась"
		} else {
			status = "ответ отправлен"
		}
	}
	if sendErr != nil && a.log != nil {
		a.log.Warn("send laya fun ai reply failed", zap.String("feature", req.Name), zap.Error(sendErr))
	}

	chat := c.Chat()
	if chat != nil {
		a.sendAIMessageLog("Фишка Лайи: "+req.Name, chat, msg, sender, target, reply, status, elapsed, firstNonNil(aiErr, sendErr))
	}
	if a.metrics != nil {
		a.metrics.IncModeration("laya_fun_" + req.Name)
	}
	return true
}

func (a *App) sendLayaFunReply(c tele.Context, reply string) bool {
	if err := a.sendLayaFunReplyErr(c, reply); err != nil {
		if a.log != nil {
			a.log.Warn("send laya fun reply failed", zap.Error(err))
		}
		return false
	}
	if a.metrics != nil {
		a.metrics.IncModeration("laya_fun")
	}
	return true
}

func (a *App) sendLayaFunReplyErr(c tele.Context, reply string) error {
	msg := c.Message()
	chat := c.Chat()
	if msg == nil || chat == nil || strings.TrimSpace(reply) == "" {
		return nil
	}
	_, err := a.bot.Send(chat, reply, &tele.SendOptions{ReplyTo: msg, ThreadID: msg.ThreadID, ReplyMarkup: a.layaInlineMarkup(chat, c.Sender())})
	return err
}
