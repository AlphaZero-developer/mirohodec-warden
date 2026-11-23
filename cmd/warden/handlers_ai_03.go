package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) generateAIReplyForTelegramMessage(cfg AIConfig, chat *tele.Chat, sender *tele.User, msg *tele.Message, text string) (string, string, time.Duration, error) {
	started := time.Now()
	meta := aiWebLogMetaFromTelegram(chat, msg)
	cfg.SystemPrompt = strings.TrimSpace(cfg.SystemPrompt + "\n\n" + layaMemorySystemPrompt)
	cfg.SystemPrompt = appendPrompt(cfg.SystemPrompt, a.layaSocialSafetyPrompt(chat, sender))
	if a.layaTelegramMessageHasExternalBotPeer(sender, msg) {
		cfg.SystemPrompt = appendPrompt(cfg.SystemPrompt, layaBotConversationPrompt)
	}
	baseCfg := cfg
	history := a.aiConversationHistory(chat, sender)
	a.clearPendingLayaBridgeSnapshot(chat, sender)
	repeatMemory := appendLayaNoRepeatMemory(history, a.aiNoRepeatReplies(chat, sender))
	current := a.layaTelegramMessageContext(chat, sender, msg, text)
	messages := append([]AIMessage(nil), history...)
	if bridgeText, bridgeHistory := a.layaReplyBridgeContext(chat, msg, sender); bridgeText != "" {
		repeatMemory = appendLayaNoRepeatMemory(repeatMemory, recentAssistantReplyTexts(bridgeHistory, maxAIConversationHistory))
		messages = append(messages, AIMessage{Role: "user", Content: bridgeText})
		a.setPendingLayaBridgeContext(chat, sender, bridgeText, bridgeHistory)
		a.rememberLayaBridgeCarrierMessage(chat, msg, sender, bridgeText, bridgeHistory)
	}
	cfg.SystemPrompt = appendPrompt(cfg.SystemPrompt, layaVisibleReplyOnlyPrompt())
	cfg.SystemPrompt = appendPrompt(cfg.SystemPrompt, layaNoRepeatMemoryPrompt(repeatMemory, 8))
	messages = append(messages, AIMessage{Role: "user", Content: current})

	reply, status, elapsed, err := a.generateAIReplyWithMessagesWithinMeta(cfg, sender, messages, text, 0, meta.withStage("primary"))
	if err != nil || (!isRepeatedAIReply(reply, repeatMemory) && !layaLooksLikeInternalLoopReply(reply)) {
		return a.formatAIReplyForDelivery(baseCfg, chat, sender, msg, reply), status, elapsed, err
	}

	rejected := []string{reply}
	for attempt := 0; attempt < maxLayaAntiLoopRetries; attempt++ {
		remaining := remainingLayaAIRetryBudget(baseCfg, started)
		if remaining < minLayaAIRetryWait {
			break
		}
		retryCfg := cfg
		retryCfg.SystemPrompt = appendPrompt(retryCfg.SystemPrompt, layaAntiLoopPrompt(repeatMemory, rejected))
		retryCfg.SystemPrompt = appendPrompt(retryCfg.SystemPrompt, layaAntiLoopDiversityPrompt(attempt))
		reply, status, elapsed, err = a.generateAIReplyWithMessagesWithinMeta(retryCfg, sender, layaAntiLoopMessagesForAttempt(messages, current, attempt), text, remaining, meta.withStage(fmt.Sprintf("anti-loop retry %d", attempt+1)))
		if err != nil {
			a.rememberAINoRepeatReplies(chat, sender, rejected...)
			return a.formatAIReplyForDelivery(baseCfg, chat, sender, msg, reply), status, elapsed, err
		}
		if !isRepeatedAIReplyAgainst(reply, repeatMemory, rejected) && !layaLooksLikeInternalLoopReply(reply) {
			a.rememberAINoRepeatReplies(chat, sender, rejected...)
			return a.formatAIReplyForDelivery(baseCfg, chat, sender, msg, reply), status, elapsed, nil
		}
		rejected = append(rejected, reply)
	}

	a.rememberAINoRepeatReplies(chat, sender, rejected...)
	a.clearAIConversation(chat, sender)
	a.clearPendingLayaBridgeSnapshot(chat, sender)
	reply, status, elapsed, err = a.generateAIReplyAfterSilentAntiLoopReset(baseCfg, chat, sender, current, text, rejected, elapsed, started, meta)
	return a.formatAIReplyForDelivery(baseCfg, chat, sender, msg, reply), status, elapsed, err
}

func (a *App) generateAIReplyAfterSilentAntiLoopReset(cfg AIConfig, chat *tele.Chat, sender *tele.User, current, logText string, rejected []string, lastElapsed time.Duration, started time.Time, meta aiWebLogMeta) (string, string, time.Duration, error) {
	elapsed := lastElapsed
	resetRejected := append([]string(nil), rejected...)
	for attempt := 0; attempt < maxLayaSilentResetTries; attempt++ {
		remaining := remainingLayaAIRetryBudget(cfg, started)
		if remaining < minLayaAIRetryWait {
			break
		}
		resetCfg := cfg
		resetCfg.SystemPrompt = appendPrompt(resetCfg.SystemPrompt, layaSilentAntiLoopResetPrompt(resetRejected))
		resetCfg.SystemPrompt = appendPrompt(resetCfg.SystemPrompt, layaAntiLoopDiversityPrompt(attempt))
		reply, status, elapsed, err := a.generateAIReplyWithMessagesWithinMeta(resetCfg, sender, []AIMessage{{Role: "user", Content: current}}, logText, remaining, meta.withStage(fmt.Sprintf("silent reset %d", attempt+1)))
		if err != nil {
			return reply, status, elapsed, err
		}
		if !isRepeatedAIReplyAgainst(reply, nil, resetRejected) && !layaLooksLikeInternalLoopReply(reply) {
			return reply, "ответ получен", elapsed, nil
		}
		resetRejected = append(resetRejected, reply)
		a.rememberAINoRepeatReplies(chat, sender, reply)
	}
	return trimRunes(aiFallbackReply(cfg), maxAIReplyChars(cfg)), "ответ получен", elapsed, nil
}

func layaAntiLoopMessagesForAttempt(messages []AIMessage, current string, attempt int) []AIMessage {
	if attempt < 2 {
		return messages
	}
	current = strings.TrimSpace(current)
	if current == "" {
		return messages
	}
	return []AIMessage{{Role: "user", Content: current}}
}

func (a *App) generateAIReplyWithMessages(cfg AIConfig, sender *tele.User, messages []AIMessage, logText string) (string, string, time.Duration, error) {
	return a.generateAIReplyWithMessagesWithin(cfg, sender, messages, logText, 0)
}

func (a *App) generateAIReplyWithMessagesWithin(cfg AIConfig, sender *tele.User, messages []AIMessage, logText string, maxWait time.Duration) (string, string, time.Duration, error) {
	reply, status, elapsed, err := a.generateAIReplyWithMessagesWithinMeta(cfg, sender, messages, logText, maxWait, aiWebLogMeta{})
	return a.formatAIReplyForDelivery(cfg, nil, sender, nil, reply), status, elapsed, err
}

func (a *App) generateAIReplyWithMessagesWithinMeta(cfg AIConfig, sender *tele.User, messages []AIMessage, logText string, maxWait time.Duration, meta aiWebLogMeta) (string, string, time.Duration, error) {
	started := time.Now()
	cfg = a.applyLayaRuntimeConfig(cfg)
	cfg = a.applyLayaSenderIdentityConfig(cfg, sender)
	cfg = a.applyLayaRelationshipConfig(cfg, sender)
	cfg = a.applyLayaUserProfileConfig(cfg, sender)

	cfg.SystemPrompt = appendPrompt(cfg.SystemPrompt, layaIdentitySystemPrompt)
	cfg.SystemPrompt = appendPrompt(cfg.SystemPrompt, layaQualitySystemPrompt)
	cfg = a.applyLayaMood(cfg, layaAIReplyContext(cfg.SystemPrompt))
	cfg.SystemPrompt = appendPrompt(cfg.SystemPrompt, layaBoundarySystemPrompt)
	cfg.SystemPrompt = a.cachedSystemPrompt(cfg.SystemPrompt)
	record := func(rawReply, finalReply, status string, elapsed time.Duration, err error) {
		a.recordAIWebLog(meta, cfg, sender, messages, logText, rawReply, finalReply, status, elapsed, err)
	}
	ai := a.currentAI()
	if ai == nil {
		elapsed := time.Since(started)
		status := "AI недоступен: нет OPENROUTER_API_KEY"
		fallback := trimRunes(aiFallbackReply(cfg), maxAIReplyChars(cfg))
		a.logAIReplyFailure(sender, logText, status, elapsed, errAIUnavailable)
		record("", fallback, status, elapsed, errAIUnavailable)
		return fallback, status, elapsed, errAIUnavailable
	}

	hardTimeout := mustDuration(cfg.Timeout, 20*time.Second)
	softTimeout := effectiveAISoftTimeout(cfg, hardTimeout)
	if maxWait > 0 && maxWait < softTimeout {
		softTimeout = maxWait
	}
	ctx, cancel := context.WithTimeout(context.Background(), hardTimeout)
	resultCh := make(chan aiReplyResult, 1)
	go func() {
		reply, err := a.replyWithAIWithRetries(ctx, ai, cfg, messages, logText)
		if err == nil {
			reply = trimRunes(reply, maxAIReplyChars(cfg))
		}
		resultCh <- aiReplyResult{reply: reply, err: err}
	}()

	var result aiReplyResult
	select {
	case result = <-resultCh:
		cancel()
	case <-time.After(softTimeout):
		cancel()
		status := "AI медлит, отправлен быстрый fallback"
		elapsed := time.Since(started)
		fallback := trimRunes(aiFallbackReply(cfg), maxAIReplyChars(cfg))
		a.logAIReplyFailure(sender, logText, status, elapsed, errAISoftTimeout)
		record("", fallback, status, elapsed, errAISoftTimeout)
		return fallback, status, elapsed, errAISoftTimeout
	}

	reply, err := result.reply, result.err
	elapsed := time.Since(started)
	rawReply := strings.TrimSpace(reply)
	if err != nil {
		status := "AI недоступен"
		if errors.Is(err, errAIEmptyReply) {
			status = "AI вернул пустой ответ"
		}
		fallback := trimRunes(aiFallbackReply(cfg), maxAIReplyChars(cfg))
		a.logAIReplyFailure(sender, logText, status, elapsed, err)
		record(rawReply, fallback, status, elapsed, err)
		return fallback, status, elapsed, err
	}

	reply = polishLayaReply(rawReply)
	if reply == "" {
		status := "AI вернул пустой ответ"
		fallback := trimRunes(aiFallbackReply(cfg), maxAIReplyChars(cfg))
		a.logAIReplyFailure(sender, logText, status, elapsed, errAIEmptyReply)
		record(rawReply, fallback, status, elapsed, errAIEmptyReply)
		return fallback, status, elapsed, errAIEmptyReply
	}
	reply = trimRunes(reply, maxAIReplyChars(cfg))
	record(rawReply, reply, "ответ получен", elapsed, nil)
	return reply, "ответ получен", elapsed, nil
}
