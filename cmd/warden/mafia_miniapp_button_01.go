package main

import (
	"net/url"
	"os"
	"strings"

	tele "gopkg.in/telebot.v3"
)

const mafiaMiniAppEnvURL = "MAFIA_MINI_APP_URL"

const mafiaMiniAppPublicPath = "/mafia-mini-app/"

func (a *App) mafiaMiniAppEnabled() bool {
	return a.mafiaMiniAppURL("") != ""
}

func (a *App) mafiaMiniAppURL(roomID string) string {
	base := a.mafiaMiniAppBaseURL()
	if base == "" {
		return ""
	}
	u, err := url.Parse(base)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return ""
	}
	roomID = normalizeMafiaRoomID(roomID)
	if roomID != "" {
		q := u.Query()
		q.Set("room_id", roomID)
		u.RawQuery = q.Encode()
	}
	return u.String()
}

func (a *App) mafiaMiniAppTelegramURL(roomID string) string {
	miniURL := a.mafiaMiniAppURL(roomID)
	if !mafiaMiniAppURLIsTelegramReady(miniURL) {
		return ""
	}
	return miniURL
}

func mafiaMiniAppURLIsTelegramReady(rawURL string) bool {
	u, err := url.Parse(strings.TrimSpace(rawURL))
	return err == nil && u.Scheme == "https" && u.Host != ""
}

func (a *App) mafiaMiniAppBaseURL() string {
	if envURL := strings.TrimSpace(os.Getenv(mafiaMiniAppEnvURL)); envURL != "" {
		return envURL
	}
	if a == nil {
		return ""
	}
	loaded := a.cfg.Load()
	cfg, ok := loaded.(Config)
	if !ok {
		return ""
	}
	if explicitURL := strings.TrimSpace(cfg.Web.MafiaMiniAppURL); explicitURL != "" {
		return explicitURL
	}
	return mafiaMiniAppURLFromPublicURL(cfg.Web.PublicURL)
}

func mafiaMiniAppURLFromPublicURL(publicURL string) string {
	publicURL = strings.TrimSpace(publicURL)
	if publicURL == "" {
		return ""
	}
	u, err := url.Parse(publicURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return ""
	}
	u.RawQuery = ""
	u.Fragment = ""
	u.Path = strings.TrimRight(u.Path, "/") + mafiaMiniAppPublicPath
	return u.String()
}

func (a *App) mafiaMiniAppEntryMarkup(roomID string) *tele.ReplyMarkup {
	miniURL := a.mafiaMiniAppTelegramURL(roomID)
	if miniURL == "" {
		return nil
	}
	markup := &tele.ReplyMarkup{}
	markup.Inline(markup.Row(markup.WebApp("Open Mini App", &tele.WebApp{URL: miniURL})))
	return markup
}

func (a *App) maybeSendMafiaMiniAppEntry(c tele.Context, roomID string) error {
	if c == nil {
		return nil
	}
	chat := c.Chat()
	if chat == nil || chat.Type != tele.ChatPrivate {
		return nil
	}
	markup := a.mafiaMiniAppEntryMarkup(roomID)
	if markup == nil {
		if warning := a.mafiaMiniAppHTTPSWarning(roomID); warning != "" {
			return c.Send(warning)
		}
		return nil
	}
	a.maybeSetMafiaMiniAppMenuButton(c, roomID)
	return c.Send("Mini App: единый экран партии с чатом, ролью, игроками, логом и действиями Лайи.", markup)
}

func (a *App) handleMafiaMiniAppOpen(c tele.Context, roomID string) error {
	if c == nil {
		return nil
	}
	chat := c.Chat()
	if chat == nil || chat.Type != tele.ChatPrivate {
		return c.Send("Mini App открывается из лички с Лайей. Напиши боту /mafia, и там будет кнопка входа.")
	}
	markup := a.mafiaMiniAppEntryMarkup(roomID)
	if markup == nil {
		if warning := a.mafiaMiniAppHTTPSWarning(roomID); warning != "" {
			return c.Send(warning)
		}
		return c.Send("Mini App ещё не настроен: добавь web.mafia_mini_app_url в config.yaml или переменную MAFIA_MINI_APP_URL.")
	}
	a.maybeSetMafiaMiniAppMenuButton(c, roomID)
	return c.Send("Открывай Mini App. Лайя уже держит весь экран партии в одном месте.", markup)
}

func (a *App) maybeSetMafiaMiniAppMenuButton(c tele.Context, roomID string) {
	if a == nil || a.bot == nil || c == nil {
		return
	}
	chat := c.Chat()
	sender := c.Sender()
	if chat == nil || chat.Type != tele.ChatPrivate || sender == nil {
		return
	}
	miniURL := a.mafiaMiniAppTelegramURL(roomID)
	if miniURL == "" {
		return
	}
	_ = a.bot.SetMenuButton(sender, &tele.MenuButton{
		Type:   tele.MenuButtonWebApp,
		Text:   "Mafia",
		WebApp: &tele.WebApp{URL: miniURL},
	})
}

func (a *App) sendMafiaMiniAppLinkToCreator(reason string) {
	_ = reason
	if a == nil || a.bot == nil {
		return
	}
	cfg := a.config()
	if !cfg.Web.Enabled || !cfg.Web.SendPanelLinkToOwner {
		return
	}
	miniURL := a.mafiaMiniAppURL("")
	if miniURL == "" {
		return
	}
	text := "Mafia Mini App запущен:\n" + miniURL
	if !mafiaMiniAppURLIsTelegramReady(miniURL) {
		text = a.mafiaMiniAppHTTPSWarning("")
	}
	sent := map[int64]bool{}
	if creatorID := a.activeLayaConfig().CreatorUserID; creatorID != 0 {
		a.sendPlainToUser(creatorID, text)
		sent[creatorID] = true
	}
	if a.store != nil {
		logChatID := a.store.LogChatID()
		if logChatID != 0 && !sent[logChatID] {
			a.sendPlainToUser(logChatID, text)
		}
	}
}
