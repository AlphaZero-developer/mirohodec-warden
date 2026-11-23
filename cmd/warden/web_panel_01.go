package main

import (
	"context"
	"errors"
	wardenweb "mirohodec-warden/internal/web"
	"sort"
	"strconv"
	"strings"
	"time"
)

type webPanelService struct {
	app *App
}

func newWebPanelConfig(ctx context.Context, cfg Config, app *App, logger *AsyncLogger) wardenweb.Config {
	return wardenweb.Config{
		Web:     toInternalWebConfig(cfg.Web),
		Context: ctx,
		Logger:  logger,
		Service: webPanelService{app: app},
		OnStarted: func() {
			app.sendWebPanelLink("startup")
			app.sendMafiaMiniAppLinkToCreator("startup")
		},
	}
}

func toInternalWebConfig(cfg WebConfig) wardenweb.WebConfig {
	return wardenweb.WebConfig{
		Enabled:              cfg.Enabled,
		Listen:               cfg.Listen,
		PublicURL:            cfg.PublicURL,
		MafiaMiniAppDir:      cfg.MafiaMiniAppDir,
		AdminToken:           cfg.AdminToken,
		SessionTTL:           cfg.SessionTTL,
		BehindProxy:          cfg.BehindProxy,
		CookieSecure:         cfg.CookieSecure,
		SendPanelLinkToOwner: cfg.SendPanelLinkToOwner,
	}
}

func (s webPanelService) Dashboard(ctx context.Context) (wardenweb.Dashboard, error) {
	a := s.app
	if a == nil || a.store == nil {
		return wardenweb.Dashboard{}, errors.New("app store is not ready")
	}
	cfg := a.config()
	counts := a.store.Counts(time.Now().UTC())
	out := wardenweb.Dashboard{
		Mode:         cfg.Telegram.Mode,
		State:        "running",
		PublicURL:    strings.TrimSpace(cfg.Web.PublicURL),
		KnownUsers:   counts.KnownUsers,
		ActiveJailed: counts.ActiveJailed,
		ActiveMuted:  counts.ActiveMuted,
		ForeverBans:  counts.ForeverBans,
	}
	out.SleepingOwners = len(a.store.ActiveOwnerSleeps())
	if a.bot != nil && a.bot.Me != nil {
		switch {
		case strings.TrimSpace(a.bot.Me.Username) != "":
			out.BotName = "@" + a.bot.Me.Username
		case strings.TrimSpace(a.bot.Me.FirstName) != "":
			out.BotName = a.bot.Me.FirstName
		default:
			out.BotName = strconv.FormatInt(a.bot.Me.ID, 10)
		}
	}
	if a.faq != nil {
		out.FAQCount = len(a.faq.List())
	}
	if a.quotes != nil {
		out.QuoteCount = len(a.quotes.List(""))
	}
	if a.photos != nil {
		out.PhotoCount = a.photos.Count()
	}
	return out, nil
}

func (s webPanelService) ChatSelection(ctx context.Context, selected int64) (wardenweb.ChatSelection, error) {
	a := s.app
	if a == nil || a.store == nil {
		return wardenweb.ChatSelection{}, errors.New("app store is not ready")
	}
	current := s.resolveChatID(selected)
	ids := a.store.ChatIDs()
	if len(ids) == 0 && a.config().Chat.ID != 0 {
		ids = []int64{a.config().Chat.ID}
	}
	if current != 0 && !containsInt64(ids, current) {
		ids = append(ids, current)
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	}
	options := make([]wardenweb.ChatOption, 0, len(ids))
	for _, id := range ids {
		options = append(options, wardenweb.ChatOption{
			ID:       id,
			Label:    a.chatSelectLabel(id),
			Selected: id == current,
		})
	}
	return wardenweb.ChatSelection{CurrentID: current, Options: options}, nil
}

func (s webPanelService) Groups(ctx context.Context) ([]wardenweb.Group, error) {
	a := s.app
	if a == nil || a.store == nil {
		return nil, errors.New("app store is not ready")
	}
	settings := a.webKnownChatSettings()
	groups := make([]wardenweb.Group, 0, len(settings))
	for _, group := range settings {
		topics := s.knownTopics(group.ChatID)
		groups = append(groups, wardenweb.Group{
			ID:                    group.ChatID,
			Title:                 group.Title,
			Enabled:               group.Enabled,
			RehabTopicID:          group.RehabTopicID,
			SpamNoRehabAction:     group.SpamNoRehabAction,
			DebugCommands:         group.DebugCommands,
			KnownTopics:           topics,
			Settings:              s.groupSettingRows(group, topics),
			AIEnabledMode:         group.AIEnabledMode,
			SpamEnabledMode:       group.SpamEnabledMode,
			WarnEnabledMode:       group.WarnEnabledMode,
			KarmaEnabledMode:      group.KarmaEnabledMode,
			FAQEnabledMode:        group.FAQEnabledMode,
			QuotesEnabledMode:     group.QuotesEnabledMode,
			PhotosEnabledMode:     group.PhotosEnabledMode,
			WarnExpireAfter:       group.WarnExpireAfter,
			WarnMuteThreshold:     group.WarnMuteThreshold,
			WarnMuteDuration:      group.WarnMuteDuration,
			WarnRehabThreshold:    group.WarnRehabThreshold,
			WarnRehabDuration:     group.WarnRehabDuration,
			KarmaMessagesPerPoint: group.KarmaMessagesPerPoint,
			KarmaDailyLimit:       group.KarmaDailyLimit,
			KarmaManualCooldown:   group.KarmaManualCooldown,
		})
	}
	return groups, nil
}

func (s webPanelService) AddGroup(ctx context.Context, input wardenweb.GroupInput) error {
	return s.upsertGroup(input, ChatSettings{})
}

func (s webPanelService) UpdateGroup(ctx context.Context, input wardenweb.GroupInput) error {
	old, _ := s.app.store.ChatSettings(input.ChatID)
	return s.upsertGroup(input, old)
}

func (s webPanelService) DeleteGroup(ctx context.Context, chatID int64) error {
	if chatID == 0 {
		return errors.New("chat_id is required")
	}
	if err := s.app.store.DeleteChatSettings(chatID); err != nil {
		return err
	}
	s.syncMenus()
	return nil
}

func (s webPanelService) SetGroupDebug(ctx context.Context, chatID int64, debug bool) error {
	settings, ok := s.app.store.ChatSettings(chatID)
	if !ok {
		return errors.New("group not found")
	}
	settings.DebugCommands = debug
	if err := s.app.store.UpsertChatSettings(settings); err != nil {
		return err
	}
	s.syncMenus()
	return nil
}
