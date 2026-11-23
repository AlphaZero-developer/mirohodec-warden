package main

import (
	"context"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) ensureAIConfigured(cfg Config) {
	if !cfg.AI.Enabled {
		return
	}

	a.aiMu.Lock()
	defer a.aiMu.Unlock()
	if a.ai != nil {
		return
	}

	apiKey := strings.TrimSpace(os.Getenv("OPENROUTER_API_KEY"))
	if apiKey == "" {
		if a.log != nil {
			a.log.Warn("ai replies enabled but OPENROUTER_API_KEY is empty")
		}
		return
	}
	a.ai = NewOpenRouterClient(apiKey, nil)
	if a.log != nil {
		a.log.Info("ai replies enabled", zap.String("model", cfg.AI.Model))
	}
}

func (a *App) currentAI() AIService {
	if a == nil {
		return nil
	}
	a.aiMu.RLock()
	defer a.aiMu.RUnlock()
	return a.ai
}

func (a *App) Register() {
	a.bot.Use(a.trackAndGuard)

	a.bot.Handle("/start", a.handleAdmin)
	a.bot.Handle("/help", a.handleAdmin)
	a.bot.Handle("/admin", a.handleAdmin)
	a.bot.Handle("/auth", a.handleAuth)
	a.bot.Handle("/menu", a.handleLayaMenu)
	a.bot.Handle("/event", a.handleWeeklyEvent)
	a.bot.Handle("/week", a.handleWeeklyEvent)
	a.bot.Handle("/myid", a.handleMyID)
	a.bot.Handle("/id", a.handleMyID)
	a.bot.Handle("/chatid", a.handleChatID)
	a.bot.Handle("/topicid", a.handleTopicID)
	a.bot.Handle("/jail", a.handleJail)
	a.bot.Handle("/rehab", a.handleRehab)
	a.bot.Handle("/ward", a.handleWard)
	a.bot.Handle("/палата", a.handleWard)
	a.bot.Handle("/mute", a.handleMute)
	a.bot.Handle("/unmute", a.handleUnmute)
	a.bot.Handle("/banforever", a.handleBanForever)
	a.bot.Handle("/unbanforever", a.handleUnbanForever)
	a.bot.Handle("/unban", a.handleUnbanForever)
	a.bot.Handle("/all", a.handleAll)
	a.bot.Handle("/status", a.handleStatus)
	a.bot.Handle("/profile", a.handleProfile)
	a.bot.Handle("/профиль", a.handleProfile)
	a.bot.Handle("/passport", a.handlePassport)
	a.bot.Handle("/паспорт", a.handlePassport)
	a.bot.Handle("/activity", a.handleActivity)
	a.bot.Handle("/topactive", a.handleTopActive)
	a.bot.Handle("/karma", a.handleKarma)
	a.bot.Handle("/topkarma", a.handleTopKarma)
	a.bot.Handle("/shop", a.handleKarmaShop)
	a.bot.Handle("/buy", a.handleKarmaBuy)
	a.bot.Handle("/selfmute", a.handleSelfMute)
	a.bot.Handle("/selfjail", a.handleSelfJail)
	a.bot.Handle("/selfrehab", a.handleSelfJail)
	a.bot.Handle("/selfrelease", a.handleSelfRelease)
	a.bot.Handle("/selfban", a.handleSelfBan)
	a.bot.Handle("/register", a.handleLayaRegister)
	a.bot.Handle("/mafia", a.handleMafia)
	a.bot.Handle("/mafia_bind", a.handleMafiaBind)
	a.bot.Handle("/mafia_chat", a.handleMafiaChat)
	a.bot.Handle("/mafia_status", a.handleMafiaStatus)
	a.bot.Handle("/mafia_leave", a.handleMafiaLeave)
	a.bot.Handle("/mafia_join", a.handleMafiaJoin)
	a.bot.Handle("/mafia_find", a.handleMafiaFind)
	a.bot.Handle("/will", a.handleMafiaWill)
	a.bot.Handle("/warn", a.handleWarn)
	a.bot.Handle("/warnings", a.handleWarnings)
	a.bot.Handle("/unwarn", a.handleUnwarn)
	a.bot.Handle("/clearwarns", a.handleClearWarnings)
	a.bot.Handle("/spamlog", a.handleSpamLog)
	a.bot.Handle("/spamstatus", a.handleSpamStatus)
	a.bot.Handle("/spampardon", a.handleSpamPardon)
	a.bot.Handle("/addphoto", a.handleAddPhoto)
	a.bot.Handle("/photo", a.handlePhoto)
	a.bot.Handle("/photos", a.handlePhotos)
	a.bot.Handle("/delphoto", a.handleDelPhoto)
	a.bot.Handle("/addquote", a.handleAddQuote)
	a.bot.Handle("/quote", a.handleQuote)
	a.bot.Handle("/quotes", a.handleQuotes)
	a.bot.Handle("/delquote", a.handleDelQuote)
	a.bot.Handle("/faq", a.handleFAQ)
	a.bot.Handle("/faqs", a.handleFAQs)
	a.bot.Handle("/addfaq", a.handleAddFAQ)
	a.bot.Handle("/delfaq", a.handleDelFAQ)
	a.bot.Handle("/escape", a.handleEscape)
	a.bot.Handle("/bribe", a.handleBribe)
	a.bot.Handle("/job", a.handleJob)
	a.bot.Handle("/hire", a.handleJob)
	a.bot.Handle("/donate", a.handleDonate)
	a.bot.Handle("/idea", a.handleIdea)
	a.bot.Handle("/bug", a.handleBug)
	a.bot.Handle("/reportbug", a.handleBug)
	a.bot.Handle("/appeal", a.handleAppeal)
	a.bot.Handle("/clearhistory", a.handleClearHistory)
	a.bot.Handle("/aitest", a.handleAITest)
	a.bot.Handle("/cipher", a.handleReplyCipher)
	a.bot.Handle("/laya_cipher", a.handleReplyCipher)
	a.bot.Handle("/шифр", a.handleReplyCipher)
	a.bot.Handle("/reload", a.handleReload)
	a.bot.Handle(tele.OnCallback, a.handleCallback)
	a.bot.Handle(tele.OnText, a.handleText)
	a.bot.Handle(tele.OnEdited, func(c tele.Context) error { return nil })

	for _, endpoint := range []interface{}{
		tele.OnPhoto,
		tele.OnAudio,
		tele.OnAnimation,
		tele.OnDocument,
		tele.OnSticker,
		tele.OnVideo,
		tele.OnVoice,
		tele.OnVideoNote,
		tele.OnContact,
		tele.OnLocation,
		tele.OnVenue,
		tele.OnDice,
		tele.OnPoll,
		tele.OnMedia,
	} {
		a.bot.Handle(endpoint, a.handleNonTextAIMessage)
	}

	a.bot.Handle(tele.OnUserJoined, a.handleUserJoined)
	a.bot.Handle(tele.OnChatMember, a.handleChatMember)
	a.bot.Handle(tele.OnAddedToGroup, a.handleAddedToGroup)
	a.bot.Handle(tele.OnMyChatMember, a.handleMyChatMember)
}

func (a *App) StartCleanup(ctx context.Context) {
	for {
		interval := mustDuration(a.config().Moderation.CleanupInterval, time.Minute)
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case now := <-timer.C:
			now = now.UTC()
			a.failExpiredRehabReleaseExams(now)
			expiredRehab := a.expiredRehabRecords(now)
			releasedRehab := make([]expiredRehabRecord, 0, len(expiredRehab))
			for _, item := range expiredRehab {
				chat := &tele.Chat{ID: item.chatID, Type: tele.ChatSuperGroup}
				if a.maybeStartRehabReleaseExam(chat, item.rec, "timeout", nil) {
					continue
				}
				releasedRehab = append(releasedRehab, item)
			}
			removed := a.store.CleanupExpired(now)
			if removed > 0 {
				a.log.Info("cleanup removed expired records", zap.Int("removed", removed))
				for _, item := range releasedRehab {
					duration := a.rehabReleaseDurationText(item.rec, true, now)
					a.hideRehabEscapeCommand(item.chatID, item.rec.UserID)
					a.clearRehabEscapeCooldown(item.chatID, item.rec.UserID)
					a.logRehabRelease(item.chatID, item.rec, duration, "cleanup")
					a.sendRehabReleaseNotice(&tele.Chat{ID: item.chatID, Type: tele.ChatSuperGroup}, item.rec, duration, 0)
					a.schedulePostRehabCheck(item.chatID, item.rec, now)
				}
			}
		}
	}
}
