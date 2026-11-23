package main

import (
	"os"
	"sync"
	"sync/atomic"
	"time"

	tele "gopkg.in/telebot.v3"
)

type App struct {
	cfg          atomic.Value
	configPath   string
	configMod    time.Time
	reloadMu     sync.Mutex
	bot          *tele.Bot
	store        RuntimeStore
	photos       PhotoRepository
	quotes       QuoteRepository
	faq          FAQRepository
	aiMu         sync.RWMutex
	ai           AIService
	aiHistoryMu  sync.Mutex
	aiHistory    map[string][]AIMessage
	aiNoRepeat   map[string][]string
	aiReplyLinks map[string]layaAIReplyLink
	aiBridgeNext map[string][]AIMessage
	aiBridgeText map[string]string
	aiWebLogMu   sync.Mutex
	aiWebLogNext uint64
	aiWebLogs    []aiWebLogEntry
	aiCacheMu    sync.Mutex
	promptCache  map[string]string
	log          *AsyncLogger
	metrics      *Metrics
	adminIDs     atomic.Value

	warnMu            sync.Mutex
	lastWarning       map[string]time.Time
	photoMu           sync.Mutex
	lastPhoto         map[string]time.Time
	autoQuoteMu       sync.Mutex
	lastAutoQuote     map[string]time.Time
	faqMu             sync.Mutex
	lastFAQEntry      map[string]time.Time
	lastFAQUser       map[string]time.Time
	ideaMu            sync.Mutex
	lastIdeaUser      map[string]time.Time
	lastIdeaChat      map[string]time.Time
	appealMu          sync.Mutex
	lastAppeal        map[string]time.Time
	rehabMu           sync.Mutex
	rehabNudgeSlots   map[string]string
	rehabDMBlocked    map[string]bool
	rehabBrainwashLog map[string][]string
	rehabWardProgress map[string]rehabWardProgress
	rehabEscapeMu     sync.Mutex
	rehabEscapeWait   map[string]rehabEscapeCooldownState
	postRehabMu       sync.Mutex
	postRehabChecks   map[string]postRehabCheck
	releaseExamMu     sync.Mutex
	releaseExams      map[string]rehabReleaseExamSession
	layaReactionMu    sync.Mutex
	lastLayaReaction  map[string]time.Time
	layaRegisterMu    sync.Mutex
	layaRegister      map[string]layaRegisterSession
	layaContextMu     sync.Mutex
	layaChatContext   map[string][]layaChatContextMessage
	layaSafetyMu      sync.Mutex
	lastLayaSafetyAI  map[string]time.Time
	discussionMu      sync.Mutex
	joinMu            sync.Mutex
	lastJoinEvents    map[string]time.Time
	botAddedMu        sync.Mutex
	lastBotAddedChats map[int64]time.Time
	mafiaNoticeMu     sync.Mutex
	lastMafiaNotice   map[string]time.Time
	spamMu            sync.Mutex
	spamBuckets       map[string]*spamBucket
	spamLogs          []spamLogEntry
}

func NewApp(configPath string, cfg Config, bot *tele.Bot, store RuntimeStore, photos PhotoRepository, quotes QuoteRepository, faq FAQRepository, ai AIService, logger *AsyncLogger, metrics *Metrics) *App {
	app := &App{
		configPath:        configPath,
		bot:               bot,
		store:             store,
		photos:            photos,
		quotes:            quotes,
		faq:               faq,
		ai:                ai,
		aiHistory:         make(map[string][]AIMessage),
		aiNoRepeat:        make(map[string][]string),
		aiReplyLinks:      make(map[string]layaAIReplyLink),
		aiBridgeNext:      make(map[string][]AIMessage),
		aiBridgeText:      make(map[string]string),
		promptCache:       make(map[string]string),
		log:               logger,
		metrics:           metrics,
		lastWarning:       make(map[string]time.Time),
		lastPhoto:         make(map[string]time.Time),
		lastAutoQuote:     make(map[string]time.Time),
		lastFAQEntry:      make(map[string]time.Time),
		lastFAQUser:       make(map[string]time.Time),
		lastIdeaUser:      make(map[string]time.Time),
		lastIdeaChat:      make(map[string]time.Time),
		lastAppeal:        make(map[string]time.Time),
		rehabNudgeSlots:   make(map[string]string),
		rehabDMBlocked:    make(map[string]bool),
		rehabBrainwashLog: make(map[string][]string),
		rehabWardProgress: make(map[string]rehabWardProgress),
		rehabEscapeWait:   make(map[string]rehabEscapeCooldownState),
		postRehabChecks:   make(map[string]postRehabCheck),
		releaseExams:      make(map[string]rehabReleaseExamSession),
		lastLayaReaction:  make(map[string]time.Time),
		layaRegister:      make(map[string]layaRegisterSession),
		layaChatContext:   make(map[string][]layaChatContextMessage),
		lastLayaSafetyAI:  make(map[string]time.Time),
		lastJoinEvents:    make(map[string]time.Time),
		lastBotAddedChats: make(map[int64]time.Time),
		lastMafiaNotice:   make(map[string]time.Time),
		spamBuckets:       make(map[string]*spamBucket),
	}
	app.applyConfig(cfg)
	if stat, err := os.Stat(configPath); err == nil {
		app.configMod = stat.ModTime()
	}
	return app
}

func (a *App) t(key string, args ...interface{}) string {
	return tr(a.config().I18n.Locale, key, args...)
}

func (a *App) localizeErr(err error) string {
	if err == nil {
		return ""
	}
	switch err.Error() {
	case "duration is required":
		return a.t("duration_required")
	case "Telegram treats restrictions under 30 seconds as forever; use 30s or more":
		return a.t("duration_too_short")
	case "missing target":
		return a.t("target_missing")
	case "target must be a numeric user_id or a reply", "target must be a numeric user_id, @username, t.me link, or a reply":
		return a.t("target_numeric")
	case "target username is unknown":
		return a.t("target_username_unknown")
	default:
		return err.Error()
	}
}

func (a *App) formatUntil(untilUnix int64) string {
	if untilUnix == 0 {
		return a.t("forever")
	}
	return formatUnix(untilUnix)
}

func (a *App) formatMaybeUnix(ts int64) string {
	if ts == 0 {
		return a.t("never")
	}
	return formatUnix(ts)
}

func (a *App) applyConfig(cfg Config) {
	a.cfg.Store(cfg)
	adminIDs := make(map[int64]struct{}, len(cfg.Security.AdminIDs))
	for _, id := range cfg.Security.AdminIDs {
		adminIDs[id] = struct{}{}
	}
	a.adminIDs.Store(adminIDs)
	if a.log != nil {
		a.log.SetLevel(cfg.Logging.Level)
	}
	a.ensureAIConfigured(cfg)
}
