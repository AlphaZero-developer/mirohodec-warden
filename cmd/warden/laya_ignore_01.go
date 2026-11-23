package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	tele "gopkg.in/telebot.v3"
)

type layaIgnoreAction int

const (
	layaIgnoreNone layaIgnoreAction = iota
	layaIgnoreAdd
	layaIgnoreRemove
)

func normalizeLayaIgnoredUser(rec LayaIgnoredUser) LayaIgnoredUser {
	rec.Username = normalizeUsername(rec.Username)
	rec.DisplayName = strings.TrimSpace(rec.DisplayName)
	rec.AddedByName = strings.TrimSpace(rec.AddedByName)
	return rec
}

func layaIgnoredUserKey(chatID, userID int64) string {
	if chatID == 0 || userID == 0 {
		return ""
	}
	return fmt.Sprintf("%d:%d", chatID, userID)
}

func (a *App) maybeHandleLayaIgnoreCommand(c tele.Context) bool {
	msg := c.Message()
	chat := c.Chat()
	admin := c.Sender()
	if msg == nil || chat == nil || admin == nil || admin.IsBot || chat.Type == tele.ChatPrivate || a.store == nil || !a.isManagedChat(chat) {
		return false
	}

	action, args, ok := parseLayaIgnoreRequest(messageText(msg), a.botUsername())
	if !ok {
		return false
	}
	adminOK := a.isAdmin(c)
	if a.layaIsIgnoring(chat.ID, admin.ID) && !adminOK {
		return true
	}
	if !adminOK {
		if err := c.Send(a.t("no_admin")); err != nil && a.log != nil {
			a.log.Warn("send laya ignore no-admin notice failed", zap.Error(err))
		}
		return true
	}

	target, _, err := a.targetFromArgs(chat.ID, c, args)
	if err != nil || target.ID == 0 {
		if err := c.Send("Ответь на сообщение пользователя или укажи user_id/@username: Лайя, игнорируй этого человека."); err != nil && a.log != nil {
			a.log.Warn("send laya ignore target notice failed", zap.Error(err))
		}
		return true
	}

	switch action {
	case layaIgnoreAdd:
		if err := a.ensureModeratable(chat, target.ID); err != nil {
			if sendErr := c.Send(a.localizeErr(err)); sendErr != nil && a.log != nil {
				a.log.Warn("send laya ignore moderatable error failed", zap.Error(sendErr))
			}
			return true
		}
		rec := LayaIgnoredUser{
			ChatID:      chat.ID,
			UserID:      target.ID,
			Username:    target.Username,
			DisplayName: target.Name,
			AddedUnix:   time.Now().UTC().Unix(),
			AddedByID:   admin.ID,
			AddedByName: displayNameFromUser(admin),
		}
		if err := a.store.SetLayaIgnoredUser(rec); err != nil {
			if sendErr := c.Send("Не смогла включить игнор Лайи: " + err.Error()); sendErr != nil && a.log != nil {
				a.log.Warn("send laya ignore add error failed", zap.Error(sendErr))
			}
			return true
		}
		if a.metrics != nil {
			a.metrics.IncModeration("laya_ignore")
		}
		a.sendOwnerLog("Laya ignore",
			logField{"User", fmt.Sprintf("%s (%d)", target.Name, target.ID)},
			logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)},
			logField{"Action", "Laya stops replying, moderation stays active"},
			logField{"By", displayNameFromUser(admin)},
		)
		if err := c.Send(fmt.Sprintf("Запомнила. Для %s я молчу, но за правилами всё равно слежу.", target.Name)); err != nil && a.log != nil {
			a.log.Warn("send laya ignore add reply failed", zap.Error(err))
		}
		return true
	case layaIgnoreRemove:
		rec, existed, err := a.store.RemoveLayaIgnoredUser(chat.ID, target.ID)
		if err != nil {
			if sendErr := c.Send("Не смогла снять игнор Лайи: " + err.Error()); sendErr != nil && a.log != nil {
				a.log.Warn("send laya ignore remove error failed", zap.Error(sendErr))
			}
			return true
		}
		if a.metrics != nil {
			a.metrics.IncModeration("laya_unignore")
		}
		a.sendOwnerLog("Laya ignore removed",
			logField{"User", fmt.Sprintf("%s (%d)", layaIgnoreTargetName(target, rec), target.ID)},
			logField{"Chat", fmt.Sprintf("%s (%d)", textOrDash(chat.Title), chat.ID)},
			logField{"Action", "Laya can reply again"},
			logField{"By", displayNameFromUser(admin)},
		)
		name := layaIgnoreTargetName(target, rec)
		reply := fmt.Sprintf("Ок, снова отвечаю %s.", name)
		if !existed {
			reply = fmt.Sprintf("%s и так не был в моём игноре.", name)
		}
		if err := c.Send(reply); err != nil && a.log != nil {
			a.log.Warn("send laya ignore remove reply failed", zap.Error(err))
		}
		return true
	default:
		return false
	}
}

func parseLayaIgnoreRequest(text, botUsername string) (layaIgnoreAction, []string, bool) {
	text = strings.TrimSpace(text)
	if text == "" || !mentionsLaya(text, botUsername) {
		return layaIgnoreNone, nil, false
	}
	normalized := " " + normalizeFAQText(text) + " "
	args := layaIgnoreTargetArgsFromText(text, botUsername)

	if containsAnyNormalized(normalized,
		" не игнорируй", " перестань игнорировать", " хватит игнорировать", " прекрати игнорировать",
		" сними игнор", " убери игнор", " отмени игнор", " верни ответы", " снова отвечай",
		" отвечай", " unignore", " stop ignoring",
	) {
		return layaIgnoreRemove, args, true
	}
	if containsAnyNormalized(normalized,
		" игнорируй", " игнорь", " игнорировать", " добавь в игнор", " внеси в игнор",
		" не отвечай", " перестань отвечать", " молчи для", " замолчи для",
		" ignore", " stop replying",
	) {
		return layaIgnoreAdd, args, true
	}
	return layaIgnoreNone, nil, false
}

func layaIgnoreTargetArgsFromText(text, botUsername string) []string {
	botUsername = normalizeUsername(botUsername)
	for _, raw := range strings.Fields(text) {
		token := strings.TrimSpace(raw)
		token = strings.Trim(token, ".,:;!?()[]{}<>«»\"'`")
		if token == "" {
			continue
		}
		if username := usernameFromTarget(token); username != "" {
			if username == botUsername || username == "laya" {
				continue
			}
			return []string{token}
		}
		idToken := strings.TrimPrefix(token, "tg://user?id=")
		if id, err := strconv.ParseInt(idToken, 10, 64); err == nil && id != 0 {
			return []string{idToken}
		}
	}
	return nil
}

func (a *App) layaIsIgnoring(chatID, userID int64) bool {
	if a == nil || a.store == nil || chatID == 0 || userID == 0 {
		return false
	}
	_, ok := a.store.LayaIgnoredUser(chatID, userID)
	return ok
}
