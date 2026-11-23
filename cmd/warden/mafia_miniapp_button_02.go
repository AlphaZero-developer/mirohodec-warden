package main

import (
	"net"
	"net/url"
	"strings"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

func (a *App) mafiaMiniAppHTTPSWarning(roomID string) string {
	miniURL := a.mafiaMiniAppURL(roomID)
	if miniURL == "" || mafiaMiniAppURLIsTelegramReady(miniURL) {
		return ""
	}
	suggestedDomain, suggestedURL := mafiaMiniAppFreeHTTPSSuggestion(miniURL)
	command := "powershell -ExecutionPolicy Bypass -File .\\scripts\\deploy.ps1 -Server 5.34.208.171 -User root -SetupHttps"
	if suggestedDomain != "" && !strings.HasSuffix(suggestedDomain, ".sslip.io") {
		command += " -HttpsDomain " + suggestedDomain
	}
	if suggestedURL == "" {
		suggestedURL = "https://5-34-208-171.sslip.io/mafia-mini-app/"
	}
	return strings.Join([]string{
		"Mafia Mini App собран, но Telegram не откроет его по HTTP.",
		"Сейчас URL такой:",
		miniURL,
		"",
		"Нужен HTTPS. Если денег на домен нет, используем бесплатный sslip.io:",
		command,
		"",
		"В BotFather потом укажи:",
		suggestedURL,
	}, "\n")
}

func mafiaMiniAppFreeHTTPSSuggestion(rawURL string) (string, string) {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || u.Host == "" {
		return "", ""
	}
	host := u.Hostname()
	if ip := net.ParseIP(host); ip != nil && ip.To4() != nil {
		domain := strings.ReplaceAll(host, ".", "-") + ".sslip.io"
		u.Scheme = "https"
		u.Host = domain
		u.Path = strings.TrimRight(u.Path, "/") + "/"
		u.RawQuery = ""
		u.Fragment = ""
		return domain, u.String()
	}
	u.Scheme = "https"
	u.RawQuery = ""
	u.Fragment = ""
	return host, u.String()
}

func (a *App) sendPlainToUser(userID int64, text string) {
	if a == nil || a.bot == nil || userID == 0 || strings.TrimSpace(text) == "" {
		return
	}
	go func() {
		if _, err := a.bot.Send(&tele.Chat{ID: userID, Type: tele.ChatPrivate}, text); err != nil && a.log != nil {
			a.log.Warn("send direct message failed", zap.Int64("user", userID), zap.Error(err))
		}
	}()
}
