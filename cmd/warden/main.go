package main

import (
	"context"
	"flag"
	"fmt"
	wardenweb "mirohodec-warden/internal/web"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func main() {
	os.Exit(run())
}

func run() int {
	configPath := flag.String("config", defaultConfigPath, "path to YAML config")
	flag.Parse()

	if err := loadDotEnv(".env"); err != nil {
		fmt.Fprintf(os.Stderr, "load .env: %v\n", err)
		return 1
	}

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		return 1
	}

	logger, err := NewAsyncLogger(cfg.Logging)
	if err != nil {
		fmt.Fprintf(os.Stderr, "create logger: %v\n", err)
		return 1
	}
	defer logger.Close()

	store, photoStore, quoteStore, faqStore, err := openRepositories(cfg, logger)
	if err != nil {
		logger.Error("open repositories failed", zap.Error(err))
		return 1
	}
	defer func() {
		if err := store.Flush(); err != nil {
			logger.Warn("store flush failed", zap.Error(err))
		}
		if err := photoStore.Flush(); err != nil {
			logger.Warn("photo database flush failed", zap.Error(err))
		}
		if err := quoteStore.Flush(); err != nil {
			logger.Warn("quote database flush failed", zap.Error(err))
		}
		if err := faqStore.Flush(); err != nil {
			logger.Warn("faq database flush failed", zap.Error(err))
		}
	}()

	bot, err := newBotWithRetry(cfg, logger)
	if err != nil {
		logger.Error("create bot failed", zap.Error(err))
		return 1
	}
	if !strings.EqualFold(cfg.Telegram.Mode, "webhook") {
		if err := bot.RemoveWebhook(false); err != nil {
			logger.Warn("remove webhook before polling failed", zap.Error(err))
		}
	}

	metrics := NewMetrics(store)
	app := NewApp(*configPath, cfg, bot, store, photoStore, quoteStore, faqStore, nil, logger, metrics)
	app.Register()

	for _, scope := range []tele.CommandScope{
		{Type: tele.CommandScopeDefault},
		{Type: tele.CommandScopeAllGroupChats},
		{Type: tele.CommandScopeAllChatAdmin},
	} {
		if err := bot.DeleteCommands(scope); err != nil {
			logger.Warn("delete group commands failed", zap.String("scope", scope.Type), zap.Error(err))
		}
	}
	commands := setupPrivateBotCommands()
	if err := bot.SetCommands(commands, tele.CommandScope{Type: tele.CommandScopeAllPrivateChats}); err != nil {
		logger.Warn("set private commands failed", zap.Error(err))
	}
	if err := syncBotCommandMenus(bot, store, logger); err != nil {
		logger.Warn("sync bot command menus failed", zap.Error(err))
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go app.StartCleanup(ctx)
	go app.StartReloader(ctx)
	go app.StartMafiaAutoHost(ctx)
	go app.StartRehabReminders(ctx)
	go app.StartRehabTopicTicker(ctx)
	go app.StartPostRehabChecks(ctx)
	go metrics.Start(ctx, cfg.Metrics, logger)
	go func() {
		if cfg.Web.Enabled {
			if err := wardenweb.Start(newWebPanelConfig(ctx, cfg, app, logger)); err != nil {
				logger.Error("web server failed", zap.Error(err))
			}
		}
	}()
	go func() {
		<-ctx.Done()
		bot.Stop()
	}()

	if bot.Me != nil {
		logger.Info("bot started", zap.String("username", bot.Me.Username), zap.String("mode", cfg.Telegram.Mode))
	} else {
		logger.Info("bot started", zap.String("mode", cfg.Telegram.Mode))
	}
	bot.Start()
	return 0
}

func openRepositories(cfg Config, logger *AsyncLogger) (RuntimeStore, PhotoRepository, QuoteRepository, FAQRepository, error) {
	if strings.EqualFold(cfg.Storage.Driver, "postgres") {
		pg, err := NewPostgresStore(cfg, logger)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		return pg, &PostgresPhotoStore{base: pg}, &PostgresQuoteStore{base: pg}, &PostgresFAQStore{base: pg}, nil
	}

	store, err := NewStore(cfg.Storage.Path, logger)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	photoStore, err := NewPhotoStore(cfg.Photos.Path, logger)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	quoteStore, err := NewQuoteStore(cfg.Quotes.Path, logger)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	faqStore, err := NewFAQStore(cfg.FAQ.Path, logger)
	if err != nil {
		return nil, nil, nil, nil, err
	}
	return store, photoStore, quoteStore, faqStore, nil
}
