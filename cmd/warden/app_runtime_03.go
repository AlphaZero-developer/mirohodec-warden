package main

import (
	"context"
	"os"
	"time"

	"go.uber.org/zap"
)

func (a *App) StartReloader(ctx context.Context) {
	for {
		interval := mustDuration(a.config().Moderation.ReloadInterval, 30*time.Second)
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			if changed, err := a.reloadConfigIfChanged(); err != nil {
				a.metrics.IncReload("config", "error")
				a.log.Warn("config hot reload failed", zap.Error(err))
			} else if changed {
				a.metrics.IncReload("config", "success")
			}
			if changed, err := a.store.ReloadIfChanged(); err != nil {
				a.metrics.IncReload("state", "error")
				a.log.Warn("state hot reload failed", zap.Error(err))
			} else if changed {
				a.metrics.IncReload("state", "success")
				a.log.Info("state hot reloaded")
			}
			if changed, err := a.photos.ReloadIfChanged(); err != nil {
				a.metrics.IncReload("photos", "error")
				a.log.Warn("photo database hot reload failed", zap.Error(err))
			} else if changed {
				a.metrics.IncReload("photos", "success")
				a.log.Info("photo database hot reloaded")
			}
			if changed, err := a.quotes.ReloadIfChanged(); err != nil {
				a.metrics.IncReload("quotes", "error")
				a.log.Warn("quote database hot reload failed", zap.Error(err))
			} else if changed {
				a.metrics.IncReload("quotes", "success")
				a.log.Info("quote database hot reloaded")
			}
			if changed, err := a.faq.ReloadIfChanged(); err != nil {
				a.metrics.IncReload("faq", "error")
				a.log.Warn("faq database hot reload failed", zap.Error(err))
			} else if changed {
				a.metrics.IncReload("faq", "success")
				a.log.Info("faq database hot reloaded")
			}
		}
	}
}

func (a *App) reloadConfigIfChanged() (bool, error) {
	a.reloadMu.Lock()
	defer a.reloadMu.Unlock()

	stat, err := os.Stat(a.configPath)
	if err != nil {
		return false, err
	}
	if !stat.ModTime().After(a.configMod) {
		return false, nil
	}
	cfg, err := LoadConfig(a.configPath)
	if err != nil {
		return false, err
	}
	old := a.config()
	cfg.Telegram.Token = old.Telegram.Token
	cfg.Telegram.Mode = old.Telegram.Mode
	cfg.Telegram.Webhook = old.Telegram.Webhook
	cfg.Storage = old.Storage
	cfg.Photos.Path = old.Photos.Path
	cfg.Quotes.Path = old.Quotes.Path
	cfg.FAQ.Path = old.FAQ.Path
	cfg.Metrics.Listen = old.Metrics.Listen
	cfg.Metrics.Path = old.Metrics.Path
	a.applyConfig(cfg)
	a.configMod = stat.ModTime()
	a.log.Info("config hot reloaded")
	return true, nil
}

func (a *App) reloadNow() error {
	a.reloadMu.Lock()
	defer a.reloadMu.Unlock()

	cfg, err := LoadConfig(a.configPath)
	if err != nil {
		a.metrics.IncReload("config", "error")
		return err
	}
	old := a.config()
	cfg.Telegram.Token = old.Telegram.Token
	cfg.Telegram.Mode = old.Telegram.Mode
	cfg.Telegram.Webhook = old.Telegram.Webhook
	cfg.Storage = old.Storage
	cfg.Photos.Path = old.Photos.Path
	cfg.Quotes.Path = old.Quotes.Path
	cfg.FAQ.Path = old.FAQ.Path
	cfg.Metrics.Listen = old.Metrics.Listen
	cfg.Metrics.Path = old.Metrics.Path
	a.applyConfig(cfg)
	if stat, err := os.Stat(a.configPath); err == nil {
		a.configMod = stat.ModTime()
	}
	a.metrics.IncReload("config", "success")

	if err := a.store.Reload(); err != nil {
		a.metrics.IncReload("state", "error")
		return err
	}
	a.metrics.IncReload("state", "success")
	if err := a.photos.Reload(); err != nil {
		a.metrics.IncReload("photos", "error")
		return err
	}
	a.metrics.IncReload("photos", "success")
	if err := a.quotes.Reload(); err != nil {
		a.metrics.IncReload("quotes", "error")
		return err
	}
	a.metrics.IncReload("quotes", "success")
	if err := a.faq.Reload(); err != nil {
		a.metrics.IncReload("faq", "error")
		return err
	}
	a.metrics.IncReload("faq", "success")
	return nil
}
