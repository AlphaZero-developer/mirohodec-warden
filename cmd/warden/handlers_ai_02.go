package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) maybeHandleLayaFas(c tele.Context) bool {
	msg := c.Message()
	chat := c.Chat()
	admin := c.Sender()
	if msg == nil || chat == nil || admin == nil || admin.IsBot || chat.Type == tele.ChatPrivate {
		return false
	}

	args, ok := layaFasArgs(messageText(msg), a.botUsername())
	if !ok {
		return false
	}

	if !a.isConfiguredAdmin(admin.ID) && !a.isTelegramGroupAdmin(chat, admin) {
		if err := c.Send(a.t("no_admin")); err != nil && a.log != nil {
			a.log.Warn("send no-admin notice failed", zap.Error(err))
		}
		return true
	}

	if msg.ReplyTo == nil || msg.ReplyTo.Sender == nil {
		if err := c.Send("Ответь на сообщение пользователя текстом: Лайя фас [duration] [reason]"); err != nil && a.log != nil {
			a.log.Warn("send laya fas usage failed", zap.Error(err))
		}
		return true
	}

	repliedUser := msg.ReplyTo.Sender
	target := targetUser{
		ID:       repliedUser.ID,
		Username: repliedUser.Username,
		Name:     displayNameFromUser(repliedUser),
	}
	if err := a.ensureModeratable(chat, target.ID); err != nil {
		if sendErr := c.Send(a.localizeErr(err)); sendErr != nil && a.log != nil {
			a.log.Warn("send laya fas moderatable error failed", zap.Error(sendErr))
		}
		return true
	}

	untilUnix, reason, err := a.layaFasMute(args)
	if err != nil {
		if sendErr := c.Send(a.localizeErr(err)); sendErr != nil && a.log != nil {
			a.log.Warn("send laya fas duration error failed", zap.Error(sendErr))
		}
		return true
	}

	if err := a.applyMute(chat, target, admin, untilUnix, reason); err != nil {
		if sendErr := c.Send(a.t("telegram_restrict_failed", err)); sendErr != nil && a.log != nil {
			a.log.Warn("send laya fas mute error failed", zap.Error(sendErr))
		}
		return true
	}
	a.metrics.IncModeration("laya_fas")

	cfg := a.config().AI
	cfg.SystemPrompt = strings.TrimSpace(cfg.SystemPrompt + "\n\n" + layaFasSystemPrompt)
	cfg.FallbackReply = "Фас так фас. Посиди в тишине, герой."
	targetText := messageText(msg.ReplyTo)
	if targetText == "" {
		targetText = target.Name
	}
	prompt := fmt.Sprintf("Цель: %s (%d).\nСообщение цели: %s\nСкажи ему короткую реплику в режиме фас.", target.Name, target.ID, targetText)
	reply, status, elapsed, aiErr := a.generateAIReply(cfg, admin, prompt)
	reply = trimRunes(reply, maxAIReplyChars(cfg))
	var sendErr error
	if _, sendErr = a.bot.Send(chat, reply, &tele.SendOptions{ReplyTo: msg.ReplyTo, ThreadID: msg.ThreadID}); sendErr != nil {
		if a.log != nil {
			a.log.Warn("send laya fas reply failed", zap.Int64("chat", chat.ID), zap.Int64("target", target.ID), zap.Error(sendErr))
		}
	}

	if aiErr == nil {
		if sendErr != nil {
			status = "ответ получен, отправка в Telegram не удалась"
		} else {
			status = "ответ отправлен"
		}
	}
	fields := []logField{
		logField{"Цель", fmt.Sprintf("%s (%d)", target.Name, target.ID)},
		logField{"Чат", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)},
		logField{"Мут", a.formatUntil(untilUnix)},
		logField{"Причина", textOrDash(reason)},
		logField{"AI", status},
		logField{"Время AI", elapsed.Truncate(time.Millisecond).String()},
		logField{"Выдал", displayNameFromUser(admin)},
	}
	if sendErr != nil {
		fields = append(fields, logField{"Отправка ответа", shortText(sendErr.Error(), 500)})
	}
	a.sendOwnerLog("Лайя фас", fields...)
	return true
}

func (a *App) handleAITest(c tele.Context) error {
	if !a.requireAdmin(c) {
		return nil
	}

	cfg := a.config()
	prompt := strings.TrimSpace(strings.Join(c.Args(), " "))
	if prompt == "" {
		prompt = "проверка"
	}

	reply, status, elapsed, err := a.generateAIReply(cfg.AI, c.Sender(), prompt)
	activeAI := a.applyLayaRuntimeConfig(cfg.AI)
	lines := []string{
		"AI test",
		fmt.Sprintf("enabled: %t", cfg.AI.Enabled),
		fmt.Sprintf("model_config: %s", cfg.AI.Model),
		fmt.Sprintf("model_active: %s", activeAI.Model),
		fmt.Sprintf("timeout_active: %s soft / %s hard", activeAI.SoftTimeout, activeAI.Timeout),
		fmt.Sprintf("key_in_process: %t", strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY")) != ""),
		fmt.Sprintf("client_ready: %t", a.currentAI() != nil),
		fmt.Sprintf("status: %s", status),
		fmt.Sprintf("duration: %s", elapsed.Truncate(time.Millisecond)),
	}
	if err != nil {
		var apiErr aiAPIError
		if errors.As(err, &apiErr) {
			lines = append(lines, fmt.Sprintf("openrouter_status: %d", apiErr.StatusCode))
			if strings.TrimSpace(apiErr.Message) != "" {
				lines = append(lines, "openrouter_message: "+shortText(apiErr.Message, 300))
			}
		} else {
			lines = append(lines, "error: "+shortText(err.Error(), 300))
		}
	} else {
		lines = append(lines, "reply: "+shortText(reply, 700))
	}

	return c.Send(strings.Join(lines, "\n"))
}

func (a *App) handleClearHistory(c tele.Context) error {
	msg := c.Message()
	chat := c.Chat()
	sender := c.Sender()
	if msg == nil || chat == nil || sender == nil || sender.IsBot {
		return nil
	}

	a.clearAIConversation(chat, sender)
	if a.metrics != nil {
		a.metrics.IncModeration("clear_ai_history")
	}
	return c.Send("История общения с Лайей очищена.", a.layaInlineMarkup(chat, sender))
}

func (a *App) generateAIReply(cfg AIConfig, sender *tele.User, text string) (string, string, time.Duration, error) {
	return a.generateAIReplyWithMessages(cfg, sender, []AIMessage{{Role: "user", Content: text}}, text)
}

func (a *App) generateAIReplyForMessage(cfg AIConfig, chat *tele.Chat, sender *tele.User, text string) (string, string, time.Duration, error) {
	return a.generateAIReplyForTelegramMessage(cfg, chat, sender, nil, text)
}
