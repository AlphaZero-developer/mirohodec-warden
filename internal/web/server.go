package web

import (
	"context"
	"embed"
	"errors"
	"html/template"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
)

//go:embed templates/*.html static/*
var assets embed.FS

type Logger interface {
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

type WebConfig struct {
	Enabled              bool
	Listen               string
	PublicURL            string
	MafiaMiniAppDir      string
	AdminToken           string
	SessionTTL           time.Duration
	BehindProxy          bool
	CookieSecure         bool
	SendPanelLinkToOwner bool
}

type Config struct {
	Web       WebConfig
	Context   context.Context
	Logger    Logger
	Service   Service
	OnStarted func()
}

const permanentSessionTTL = 10 * 365 * 24 * time.Hour

func Start(cfg Config) error {
	if !cfg.Web.Enabled {
		return nil
	}
	if strings.TrimSpace(cfg.Web.AdminToken) == "" {
		return errors.New("web.admin_token is empty")
	}
	if cfg.Service == nil {
		return errors.New("web service is nil")
	}
	if strings.TrimSpace(cfg.Web.Listen) == "" {
		cfg.Web.Listen = ":8090"
	}
	if cfg.Web.SessionTTL == 0 {
		cfg.Web.SessionTTL = 24 * time.Hour
	}

	handler, err := newServer(cfg)
	if err != nil {
		return err
	}

	httpServer := &http.Server{
		Addr:              cfg.Web.Listen,
		Handler:           handler.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	listener, err := net.Listen("tcp", cfg.Web.Listen)
	if err != nil {
		return err
	}

	if cfg.Context != nil {
		go func() {
			<-cfg.Context.Done()
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := httpServer.Shutdown(shutdownCtx); err != nil && cfg.Logger != nil {
				cfg.Logger.Warn("web server shutdown failed", zap.Error(err))
			}
		}()
	}

	if cfg.Logger != nil {
		cfg.Logger.Info("web server started",
			zap.String("listen", cfg.Web.Listen),
			zap.String("public_url", cfg.Web.PublicURL),
			zap.String("mafia_mini_app_dir", cfg.Web.MafiaMiniAppDir),
		)
	}
	if cfg.OnStarted != nil {
		go cfg.OnStarted()
	}

	if err := httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func newServer(cfg Config) (*server, error) {
	styleBytes, err := assets.ReadFile("static/style.css")
	if err != nil {
		return nil, err
	}
	styleCSS := string(styleBytes)
	tpl, err := template.New("").Funcs(template.FuncMap{
		"groupTitle":      formatGroupTitle,
		"inlineCSS":       func() template.CSS { return template.CSS(styleCSS) },
		"navClass":        navClass,
		"optionalInt":     optionalInt,
		"optionalInt64":   optionalInt64,
		"themeBootScript": func() template.JS { return template.JS(themeBootScript) },
	}).ParseFS(assets, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return &server{
		cfg:       cfg,
		templates: tpl,
		mafiaHub:  newMafiaWSHub(),
	}, nil
}

const themeBootScript = `(function(){try{var root=document.documentElement;var theme=localStorage.getItem("laya-panel-theme");if(theme!=="light"&&theme!=="dark"){theme=window.matchMedia&&window.matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light"}root.dataset.theme=theme;var raw=localStorage.getItem("laya-panel-settings");if(!raw){root.dataset.transition="fade";root.dataset.motionSpeed="normal";root.dataset.fontScale="normal";root.dataset.bgPreset="clean";root.dataset.rightPanel="hover";return}var settings=JSON.parse(raw)||{};var transitions=["fade","slide-left","slide-right","slide-up","slide-down","zoom","scale-soft","flip","blur","rotate","diagonal","none"];var bg=["clean","aurora","nebula","frost","image"];root.dataset.transition=transitions.indexOf(settings.transition)>=0?settings.transition:"fade";root.dataset.motionSpeed=settings.motionSpeed||"normal";root.dataset.fontScale=settings.fontScale||"normal";root.dataset.bgPreset=bg.indexOf(settings.backgroundPreset)>=0?settings.backgroundPreset:"clean";root.dataset.rightPanel=["hover","pinned","off"].indexOf(settings.rightPanel)>=0?settings.rightPanel:"hover";root.style.setProperty("--background-blur",settings.backgroundBlur||"0px");root.style.setProperty("--surface-alpha",settings.surfaceOpacity||"100%");root.style.setProperty("--panel-blur",settings.surfaceBlur||"0px");if(settings.backgroundImage&&/^https?:\/\//i.test(settings.backgroundImage)){root.style.setProperty("--panel-bg-image","url(\""+String(settings.backgroundImage).replace(/["\\\n\r]/g,"")+"\")")}if(settings.fontFamily){root.style.setProperty("--panel-font",settings.fontFamily)}if(settings.fontUrl&&/^https?:\/\//i.test(settings.fontUrl)){var link=document.createElement("link");link.id="panel-font-link";link.rel="stylesheet";link.href=settings.fontUrl;document.head.appendChild(link)}var colors=settings.colors||{};Object.keys(colors).forEach(function(key){if(/^--[a-z0-9-]+$/.test(key)&&/^#[0-9a-f]{6}$/i.test(colors[key]))root.style.setProperty(key,colors[key])})}catch(e){document.documentElement.dataset.theme="light";document.documentElement.dataset.transition="fade";document.documentElement.dataset.motionSpeed="normal";document.documentElement.dataset.fontScale="normal";document.documentElement.dataset.bgPreset="clean";document.documentElement.dataset.rightPanel="hover"}})();`

func optionalInt(value int) string {
	if value <= 0 {
		return ""
	}
	return strconv.Itoa(value)
}

func optionalInt64(value int64) string {
	if value <= 0 {
		return ""
	}
	return strconv.FormatInt(value, 10)
}
