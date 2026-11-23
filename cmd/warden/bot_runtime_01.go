package main

import (
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

var allowedUpdates = []string{
	"message",
	"edited_message",
	"callback_query",
	"chat_member",
	"my_chat_member",
	"chat_join_request",
}

func buildPoller(cfg Config) tele.Poller {
	if strings.EqualFold(cfg.Telegram.Mode, "webhook") {
		webhook := &tele.Webhook{
			Listen:         cfg.Telegram.Webhook.Listen,
			MaxConnections: cfg.Telegram.Webhook.MaxConnections,
			AllowedUpdates: allowedUpdates,
			SecretToken:    cfg.Telegram.Webhook.SecretToken,
		}
		if cfg.Telegram.Webhook.PublicURL != "" {
			webhook.Endpoint = &tele.WebhookEndpoint{
				PublicURL: cfg.Telegram.Webhook.PublicURL,
				Cert:      cfg.Telegram.Webhook.EndpointCert,
			}
		}
		if cfg.Telegram.Webhook.TLSCert != "" && cfg.Telegram.Webhook.TLSKey != "" {
			webhook.TLS = &tele.WebhookTLS{
				Cert: cfg.Telegram.Webhook.TLSCert,
				Key:  cfg.Telegram.Webhook.TLSKey,
			}
		}
		return webhook
	}

	return &tele.LongPoller{
		Timeout:        mustDuration(cfg.Telegram.PollerTimeout, 10*time.Second),
		AllowedUpdates: allowedUpdates,
	}
}

func newBotWithRetry(cfg Config, logger *AsyncLogger) (*tele.Bot, error) {
	const attempts = 5
	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		bot, err := tele.NewBot(tele.Settings{
			Token:     cfg.Telegram.Token,
			Poller:    buildPoller(cfg),
			ParseMode: tele.ModeDefault,
			Client:    telegramHTTPClient(),
		})
		if err == nil {
			return bot, nil
		}
		lastErr = err
		if attempt < attempts {
			logger.Warn("create bot attempt failed",
				zap.Int("attempt", attempt),
				zap.Int("max_attempts", attempts),
				zap.Error(err),
			)
			time.Sleep(time.Duration(attempt*2) * time.Second)
		}
	}
	return nil, lastErr
}

func telegramHTTPClient() *http.Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.TLSHandshakeTimeout = 30 * time.Second
	transport.ResponseHeaderTimeout = 30 * time.Second
	return &http.Client{
		Timeout:   90 * time.Second,
		Transport: transport,
	}
}

func syncBotCommandMenus(bot *tele.Bot, store RuntimeStore, logger *AsyncLogger) error {
	if bot == nil {
		return nil
	}

	languages := []string{"", "ru", "en"}
	resetScopes := []tele.CommandScope{
		{Type: tele.CommandScopeDefault},
		{Type: tele.CommandScopeAllPrivateChats},
		{Type: tele.CommandScopeAllGroupChats},
		{Type: tele.CommandScopeAllChatAdmin},
	}
	for _, scope := range resetScopes {
		for _, lang := range languages {
			if err := deleteBotCommands(bot, scope, lang); err != nil {
				if logger != nil {
					logger.Warn("delete bot commands failed", zap.String("scope", scope.Type), zap.String("lang", lang), zap.Error(err))
				}
			}
		}
	}

	for _, scope := range []tele.CommandScope{
		{Type: tele.CommandScopeAllGroupChats},
		{Type: tele.CommandScopeAllChatAdmin},
	} {
		if err := bot.SetCommands(publicBotCommands(), scope); err != nil {
			return err
		}
	}
	if store != nil {
		if store.HasOwner() {
			if err := bot.SetCommands(publicBotCommands(), tele.CommandScope{Type: tele.CommandScopeAllPrivateChats}); err != nil {
				return err
			}
		} else if err := bot.SetCommands(setupPrivateBotCommands(), tele.CommandScope{Type: tele.CommandScopeAllPrivateChats}); err != nil {
			return err
		}
		if ownerChatID := store.LogChatID(); ownerChatID != 0 {
			if err := bot.SetCommands(ownerPrivateBotCommands(), tele.CommandScope{Type: tele.CommandScopeChat, ChatID: ownerChatID}); err != nil {
				return err
			}
		}
		for _, settings := range store.ManagedChats(true) {
			scope := tele.CommandScope{Type: tele.CommandScopeChat, ChatID: settings.ChatID}
			if !settings.DebugCommands || !settings.Enabled {
				if err := deleteBotCommands(bot, scope, ""); err != nil && logger != nil {
					logger.Warn("delete chat commands failed", zap.Int64("chat", settings.ChatID), zap.Error(err))
				}
				continue
			}
			if err := bot.SetCommands(debugChatBotCommands(), scope); err != nil {
				return err
			}
		}
	} else if err := bot.SetCommands(setupPrivateBotCommands(), tele.CommandScope{Type: tele.CommandScopeAllPrivateChats}); err != nil {
		return err
	}
	if err := syncActiveRehabEscapeCommands(bot, store, logger); err != nil {
		return err
	}
	return nil
}

func deleteBotCommands(bot *tele.Bot, scope tele.CommandScope, lang string) error {
	if lang == "" {
		return bot.DeleteCommands(scope)
	}
	return bot.DeleteCommands(lang, scope)
}

func publicBotCommands() []tele.Command {
	return []tele.Command{
		{Text: "profile", Description: "профиль"},
		{Text: "passport", Description: "паспорт участника"},
		{Text: "karma", Description: "karma"},
		{Text: "topkarma", Description: "top karma"},
		{Text: "menu", Description: "меню Лайи"},
		{Text: "event", Description: "событие недели"},
		{Text: "register", Description: "настроить обращение Лайи"},
		{Text: "activity", Description: "активность участника"},
		{Text: "topactive", Description: "топ активных участников"},
		{Text: "mafia", Description: "Мафия через Лайю"},
		{Text: "mafia_find", Description: "найти игрока для Мафии"},
		{Text: "will", Description: "завещание в MafiaGame"},
		{Text: "quote", Description: "случайная цитата Лайи"},
		{Text: "photo", Description: "случайное фото Лайи"},
		{Text: "faq", Description: "FAQ Лайи"},
		{Text: "job", Description: "заявка на работу"},
		{Text: "donate", Description: "донат владельцу"},
		{Text: "idea", Description: "сообщить идею директору"},
		{Text: "bug", Description: "сообщить об ошибке"},
		{Text: "appeal", Description: "обжаловать решение"},
		{Text: "clearhistory", Description: "очистить историю общения с Лайей"},
		{Text: "selfmute", Description: "самомут на время"},
		{Text: "selfjail", Description: "самому уйти в психцентр"},
		{Text: "selfban", Description: "самобан на время"},
	}
}

func setupPrivateBotCommands() []tele.Command {
	return []tele.Command{
		{Text: "auth", Description: "зарегистрировать владельца"},
		{Text: "menu", Description: "меню Лайи"},
		{Text: "event", Description: "событие недели"},
		{Text: "myid", Description: "показать твой user id"},
		{Text: "clearhistory", Description: "очистить историю общения с Лайей"},
	}
}
