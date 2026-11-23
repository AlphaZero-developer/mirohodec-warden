package web

import (
	"html/template"
	"net/http"
)

type server struct {
	cfg       Config
	templates *template.Template
	mafiaHub  *mafiaWSHub
}

type pageData struct {
	Title     string
	Active    string
	Error     string
	Message   string
	PublicURL string

	Dashboard  Dashboard
	Chats      ChatSelection
	Groups     []Group
	Rehab      RehabPage
	Users      []UserRow
	Moderation ModerationPage
	Warnings   []WarningRow
	Karma      []KarmaRow
	OwnerSleep OwnerSleepPage
	Laya       LayaControlPage
	LayaUsers  LayaUsersPage
	AILogs     AILogPage
	Discussion DiscussionPage
	Content    ContentControlPage
	FAQs       []FAQRow
	Quotes     []QuoteRow
	Photos     []PhotoRow
	PhotoTag   string
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticSubtree()))))
	mux.HandleFunc("/mafia-mini-app", s.mafiaMiniApp)
	mux.HandleFunc("/mafia-mini-app/", s.mafiaMiniApp)
	mux.HandleFunc("/", s.dashboard)
	mux.HandleFunc("/login", s.login)
	mux.HandleFunc("/logout", s.logout)
	mux.HandleFunc("/reload", s.requireAuth(s.reload))
	mux.HandleFunc("/settings", s.requireAuth(s.settings))
	mux.HandleFunc("/supergroups", s.requireAuth(s.supergroups))
	mux.HandleFunc("/supergroups/add", s.requireAuth(s.supergroupAdd))
	mux.HandleFunc("/supergroups/update", s.requireAuth(s.supergroupUpdate))
	mux.HandleFunc("/supergroups/delete", s.requireAuth(s.supergroupDelete))
	mux.HandleFunc("/supergroups/debug", s.requireAuth(s.supergroupDebug))
	mux.HandleFunc("/supergroups/rehab-topic", s.requireAuth(s.supergroupRehabTopic))
	mux.HandleFunc("/rehab", s.requireAuth(s.rehab))
	mux.HandleFunc("/rehab/add", s.requireAuth(s.rehabAdd))
	mux.HandleFunc("/rehab/release", s.requireAuth(s.rehabRelease))
	mux.HandleFunc("/rehab/brainwash", s.requireAuth(s.rehabBrainwash))
	mux.HandleFunc("/rehab/exam", s.requireAuth(s.rehabReleaseExam))
	mux.HandleFunc("/users", s.requireAuth(s.users))
	mux.HandleFunc("/moderation", s.requireAuth(s.moderation))
	mux.HandleFunc("/moderation/ban", s.requireAuth(s.foreverBan))
	mux.HandleFunc("/warnings", s.requireAuth(s.warnings))
	mux.HandleFunc("/karma", s.requireAuth(s.karma))
	mux.HandleFunc("/sleep", s.requireAuth(s.ownerSleep))
	mux.HandleFunc("/sleep/clear", s.requireAuth(s.ownerSleepClear))
	mux.HandleFunc("/laya", s.requireAuth(s.laya))
	mux.HandleFunc("/laya/update", s.requireAuth(s.layaUpdate))
	mux.HandleFunc("/laya/say", s.requireAuth(s.layaSay))
	mux.HandleFunc("/laya/users", s.requireAuth(s.layaUsers))
	mux.HandleFunc("/laya/users/update", s.requireAuth(s.layaUserUpdate))
	mux.HandleFunc("/laya/cache/clear", s.requireAuth(s.layaCacheClear))
	mux.HandleFunc("/ai-logs", s.requireAuth(s.aiLogs))
	mux.HandleFunc("/ai-logs/clear", s.requireAuth(s.aiLogsClear))
	mux.HandleFunc("/discussion", s.requireAuth(s.discussion))
	mux.HandleFunc("/discussion/update", s.requireAuth(s.discussionUpdate))
	mux.HandleFunc("/content", s.requireAuth(s.content))
	mux.HandleFunc("/content/update", s.requireAuth(s.contentUpdate))
	mux.HandleFunc("/faq", s.requireAuth(s.faq))
	mux.HandleFunc("/faq/add", s.requireAuth(s.faqAdd))
	mux.HandleFunc("/faq/update", s.requireAuth(s.faqUpdate))
	mux.HandleFunc("/faq/delete", s.requireAuth(s.faqDelete))
	mux.HandleFunc("/quotes", s.requireAuth(s.quotes))
	mux.HandleFunc("/quotes/add", s.requireAuth(s.quoteAdd))
	mux.HandleFunc("/quotes/update", s.requireAuth(s.quoteUpdate))
	mux.HandleFunc("/quotes/delete", s.requireAuth(s.quoteDelete))
	mux.HandleFunc("/photos", s.requireAuth(s.photos))
	mux.HandleFunc("/photos/add", s.requireAuth(s.photoAdd))
	mux.HandleFunc("/photos/delete", s.requireAuth(s.photoDelete))
	mux.HandleFunc("/api/mafia/auth", s.mafiaMiniAppAuth)
	mux.HandleFunc("/api/mafia/profile", s.mafiaMiniAppProfile)
	mux.HandleFunc("/api/mafia/profiles/", s.mafiaMiniAppProfileView)
	mux.HandleFunc("/api/mafia/rooms", s.mafiaMiniAppRooms)
	mux.HandleFunc("/api/mafia/rooms/join", s.mafiaMiniAppJoinRoom)
	mux.HandleFunc("/api/mafia/rooms/", s.mafiaMiniAppRoom)
	mux.HandleFunc("/api/mafia/ws", s.mafiaMiniAppWS)
	return mux
}

func (s *server) dashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if !s.authenticated(r) {
		s.render(w, "login.html", pageData{Title: "Laya Panel"})
		return
	}
	dashboard, err := s.cfg.Service.Dashboard(r.Context())
	if err != nil {
		s.renderError(w, r, "Панель", "dashboard", "dashboard.html", err)
		return
	}
	data := s.baseData(r, "Панель", "dashboard")
	data.Dashboard = dashboard
	data.PublicURL = dashboard.PublicURL
	s.render(w, "dashboard.html", data)
}

func (s *server) login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.render(w, "login.html", pageData{Title: "Laya Panel", Error: r.URL.Query().Get("err")})
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.render(w, "login.html", pageData{Title: "Laya Panel", Error: "Не удалось прочитать форму."})
		return
	}
	if !sameToken(r.FormValue("token"), s.cfg.Web.AdminToken) {
		s.render(w, "login.html", pageData{Title: "Laya Panel", Error: "Неверный токен."})
		return
	}
	http.SetCookie(w, s.sessionCookie())
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *server) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, expiredSessionCookie(s.cfg.Web))
	http.Redirect(w, r, "/login", http.StatusFound)
}

func (s *server) reload(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	if err := s.cfg.Service.Reload(r.Context()); err != nil {
		s.redirect(w, r, "/", "", err)
		return
	}
	s.redirect(w, r, "/", "Конфиг и базы перечитаны.", nil)
}

func (s *server) settings(w http.ResponseWriter, r *http.Request) {
	data := s.baseData(r, "Настройки панели", "settings")
	s.render(w, "settings.html", data)
}

func (s *server) discussion(w http.ResponseWriter, r *http.Request) {
	page, err := s.cfg.Service.Discussion(r.Context())
	if err != nil {
		s.renderError(w, r, "Комменты", "discussion", "discussion.html", err)
		return
	}
	data := s.baseData(r, "Комменты", "discussion")
	data.Discussion = page
	s.render(w, "discussion.html", data)
}

func (s *server) discussionUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	input := DiscussionInput{
		Enabled:            r.FormValue("enabled") == "on",
		ChatID:             formInt64Default(r, "chat_id"),
		SourceChatID:       formInt64Default(r, "source_chat_id"),
		StrictChecks:       r.FormValue("strict_checks") == "on",
		Text:               r.FormValue("text"),
		ChatButtonText:     r.FormValue("chat_button_text"),
		ChatButtonURL:      r.FormValue("chat_button_url"),
		FollowUpText:       r.FormValue("follow_up_text"),
		FollowUpButtonText: r.FormValue("follow_up_button_text"),
		FollowUpButtonURL:  r.FormValue("follow_up_button_url"),
		Pin:                r.FormValue("pin") == "on",
	}
	err := s.cfg.Service.UpdateDiscussion(r.Context(), input)
	s.redirect(w, r, "/discussion", "Настройки комментов сохранены.", err)
}
