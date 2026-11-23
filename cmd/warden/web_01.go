package main

import (
	"context"
	"crypto/subtle"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

const permanentWebSessionTTL = 10 * 365 * 24 * time.Hour

func (a *App) StartWeb(ctx context.Context, cfg WebConfig) {
	if !cfg.Enabled {
		return
	}
	token := strings.TrimSpace(cfg.AdminToken)
	if token == "" {
		if a.log != nil {
			a.log.Warn("web panel disabled: web.admin_token is empty")
		}
		return
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.webRequireAuth(cfg, a.webDashboard))
	mux.HandleFunc("/login", a.webLogin(cfg))
	mux.HandleFunc("/logout", a.webLogout)
	mux.HandleFunc("/users", a.webRequireAuth(cfg, a.webUsers))
	mux.HandleFunc("/supergroups", a.webRequireAuth(cfg, a.webSupergroups))
	mux.HandleFunc("/supergroups/add", a.webRequireAuth(cfg, a.webSupergroupAdd))
	mux.HandleFunc("/supergroups/update", a.webRequireAuth(cfg, a.webSupergroupUpdate))
	mux.HandleFunc("/supergroups/delete", a.webRequireAuth(cfg, a.webSupergroupDelete))
	mux.HandleFunc("/supergroups/debug", a.webRequireAuth(cfg, a.webSupergroupDebug))
	mux.HandleFunc("/supergroups/rehab-topic", a.webRequireAuth(cfg, a.webSupergroupRehabTopic))
	mux.HandleFunc("/rehab", a.webRequireAuth(cfg, a.webRehab))
	mux.HandleFunc("/rehab/add", a.webRequireAuth(cfg, a.webRehabAdd))
	mux.HandleFunc("/rehab/release", a.webRequireAuth(cfg, a.webRehabRelease))
	mux.HandleFunc("/moderation", a.webRequireAuth(cfg, a.webModeration))
	mux.HandleFunc("/warnings", a.webRequireAuth(cfg, a.webWarnings))
	mux.HandleFunc("/karma", a.webRequireAuth(cfg, a.webKarma))
	mux.HandleFunc("/faq", a.webRequireAuth(cfg, a.webFAQ))
	mux.HandleFunc("/faq/add", a.webRequireAuth(cfg, a.webFAQAdd))
	mux.HandleFunc("/faq/update", a.webRequireAuth(cfg, a.webFAQUpdate))
	mux.HandleFunc("/faq/delete", a.webRequireAuth(cfg, a.webFAQDelete))
	mux.HandleFunc("/quotes", a.webRequireAuth(cfg, a.webQuotes))
	mux.HandleFunc("/quotes/add", a.webRequireAuth(cfg, a.webQuoteAdd))
	mux.HandleFunc("/quotes/update", a.webRequireAuth(cfg, a.webQuoteUpdate))
	mux.HandleFunc("/quotes/delete", a.webRequireAuth(cfg, a.webQuoteDelete))

	server := &http.Server{Addr: cfg.Listen, Handler: mux}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	if a.log != nil {
		a.log.Info("web panel started", zap.String("listen", cfg.Listen))
	}
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed && a.log != nil {
		a.log.Warn("web panel stopped", zap.Error(err))
	}
}

func (a *App) webRequireAuth(cfg WebConfig, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("warden_session")
		if err != nil || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(webSessionHash(cfg.AdminToken))) != 1 {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next(w, r)
	}
}

func (a *App) webLogin(cfg WebConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if subtle.ConstantTimeCompare([]byte(strings.TrimSpace(r.FormValue("token"))), []byte(strings.TrimSpace(cfg.AdminToken))) == 1 {
				ttl := cfg.SessionTTL
				if ttl == 0 {
					ttl = 24 * time.Hour
				}
				if ttl < 0 {
					ttl = permanentWebSessionTTL
				}
				http.SetCookie(w, &http.Cookie{
					Name:     "warden_session",
					Value:    webSessionHash(cfg.AdminToken),
					Path:     "/",
					MaxAge:   int(ttl.Seconds()),
					HttpOnly: true,
					Secure:   webCookieSecure(cfg),
					SameSite: http.SameSiteLaxMode,
				})
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}
			a.webLayout(w, "Login", `<p class="error">Неверный токен.</p>`+loginForm())
			return
		}
		a.webLayout(w, "Login", loginForm())
	}
}

func (a *App) webLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "warden_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: webCookieSecure(a.config().Web), SameSite: http.SameSiteLaxMode})
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (a *App) webDashboard(w http.ResponseWriter, r *http.Request) {
	counts := a.store.Counts(time.Now().UTC())
	body := fmt.Sprintf(`<div class="grid">
<div class="card"><b>%d</b><span>known users</span></div>
<div class="card"><b>%d</b><span>psych center</span></div>
<div class="card"><b>%d</b><span>mutes</span></div>
<div class="card"><b>%d</b><span>forever bans</span></div>
</div>`, counts.KnownUsers, counts.ActiveJailed, counts.ActiveMuted, counts.ForeverBans)
	a.webLayout(w, "Dashboard", body)
}

func (a *App) webUsers(w http.ResponseWriter, r *http.Request) {
	chatID := a.webChatID(r)
	rows := strings.Builder{}
	for _, rec := range a.store.TopActive(chatID, 100) {
		rows.WriteString(fmt.Sprintf("<tr><td>%d</td><td>%s</td><td>%d</td><td>%d</td><td>%d</td><td>%s</td></tr>",
			rec.ID, esc(rec.DisplayName), rec.Messages, messagesToday(rec), a.store.Karma(chatID, rec.ID), esc(a.formatMaybeUnix(rec.LastMessage))))
	}
	a.webLayout(w, "Users", chatSelector(a, r)+`<table><thead><tr><th>ID</th><th>Name</th><th>Total</th><th>Today</th><th>Karma</th><th>Last message</th></tr></thead><tbody>`+rows.String()+`</tbody></table>`)
}

func (a *App) webSupergroups(w http.ResponseWriter, r *http.Request) {
	var body strings.Builder
	body.WriteString(`<section class="panel"><h2>Add supergroup</h2><form class="form-grid" method="post" action="/supergroups/add">
<input name="chat_id" placeholder="-1001234567890" required>
<input name="title" placeholder="Название чата">
<input name="rehab_topic_id" placeholder="psych center topic id">
<select name="spam_no_rehab_action"><option value="mute_on_delete">mute on delete-worthy spam</option><option value="normal">normal spam thresholds</option></select>
<label><input type="checkbox" name="enabled" checked> enabled</label>
<button>Add</button></form></section>`)
	body.WriteString(`<div class="cards">`)
	for _, settings := range a.webKnownChatSettings() {
		formID := fmt.Sprintf("chat_%d", settings.ChatID)
		body.WriteString(fmt.Sprintf(`<article class="item-card">
<header><b>%s</b><span>%d</span></header>
<form id="%s" method="post" action="/supergroups/update" class="stack">
<input type="hidden" name="chat_id" value="%d">
<label>Title<input name="title" value="%s"></label>
<label>Psych center topic<input name="rehab_topic_id" value="%d"></label>
<label>Spam without psych center<select name="spam_no_rehab_action"><option value="mute_on_delete"%s>mute on delete-worthy spam</option><option value="normal"%s>normal thresholds</option></select></label>
<label><input type="checkbox" name="enabled" %s> enabled</label>
<p class="muted">Debug menu: %s</p>
</form>
<div class="actions"><button form="%s">Save</button>
<form method="post" action="/supergroups/debug"><input type="hidden" name="chat_id" value="%d"><input type="hidden" name="debug" value="%t"><button>%s</button></form>
<form method="post" action="/supergroups/delete"><input type="hidden" name="chat_id" value="%d"><button class="danger">Delete</button></form></div>
%s</article>`,
			esc(chatSettingsTitle(settings)), settings.ChatID, formID, settings.ChatID, esc(settings.Title), settings.RehabTopicID,
			selected(settings.SpamNoRehabAction == spamNoRehabMuteOnDelete || settings.SpamNoRehabAction == ""), selected(settings.SpamNoRehabAction == "normal"),
			checked(settings.Enabled), debugModeText(settings.DebugCommands), formID, settings.ChatID, !settings.DebugCommands, debugButton(settings.DebugCommands), settings.ChatID, a.knownTopicButtons(settings.ChatID)))
	}
	body.WriteString(`</div>`)
	a.webLayout(w, "Supergroups", body.String())
}

func (a *App) webSupergroupAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		_ = a.store.UpsertChatSettings(chatSettingsFromRequest(r, ChatSettings{}))
		go a.syncCommandMenus()
	}
	http.Redirect(w, r, "/supergroups", http.StatusFound)
}

func (a *App) webSupergroupUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		chatID, _ := strconv.ParseInt(strings.TrimSpace(r.FormValue("chat_id")), 10, 64)
		old, _ := a.store.ChatSettings(chatID)
		_ = a.store.UpsertChatSettings(chatSettingsFromRequest(r, old))
		go a.syncCommandMenus()
	}
	http.Redirect(w, r, "/supergroups", http.StatusFound)
}

func (a *App) webSupergroupDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		chatID, _ := strconv.ParseInt(strings.TrimSpace(r.FormValue("chat_id")), 10, 64)
		_ = a.store.DeleteChatSettings(chatID)
		go a.syncCommandMenus()
	}
	http.Redirect(w, r, "/supergroups", http.StatusFound)
}
