package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	tele "gopkg.in/telebot.v3"
)

func (a *App) webSupergroupDebug(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		chatID, _ := strconv.ParseInt(strings.TrimSpace(r.FormValue("chat_id")), 10, 64)
		settings, ok := a.store.ChatSettings(chatID)
		if ok {
			settings.DebugCommands = r.FormValue("debug") == "true"
			_ = a.store.UpsertChatSettings(settings)
			go a.syncCommandMenus()
		}
	}
	http.Redirect(w, r, "/supergroups", http.StatusFound)
}

func (a *App) webSupergroupRehabTopic(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		chatID, _ := strconv.ParseInt(strings.TrimSpace(r.FormValue("chat_id")), 10, 64)
		topicID, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("topic_id")))
		if chatID != 0 && topicID != 0 {
			settings, ok := a.store.ChatSettings(chatID)
			if !ok {
				settings = ChatSettings{ChatID: chatID, Enabled: true, SpamNoRehabAction: spamNoRehabMuteOnDelete}
			}
			settings.RehabTopicID = topicID
			if settings.SpamNoRehabAction == "" {
				settings.SpamNoRehabAction = spamNoRehabMuteOnDelete
			}
			_ = a.store.UpsertChatSettings(settings)
		}
	}
	http.Redirect(w, r, "/supergroups", http.StatusFound)
}

func (a *App) webRehab(w http.ResponseWriter, r *http.Request) {
	chatID := a.webChatID(r)
	chat := &tele.Chat{ID: chatID, Type: tele.ChatSuperGroup}
	rehabTopicID := a.rehabTopicID(chat)
	var body strings.Builder
	body.WriteString(chatSelector(a, r))
	if rehabTopicID == 0 {
		body.WriteString(`<section class="panel"><h2>Send to psych center</h2><p class="error">Psych center topic is not configured for this chat. Open Supergroups and set rehab_topic_id first.</p></section>`)
	} else {
		body.WriteString(fmt.Sprintf(`<section class="panel"><h2>Send to psych center</h2><p class="muted">Psych center topic: %d</p><form class="form-grid" method="post" action="/rehab/add">
<input type="hidden" name="chat_id" value="%d"><input name="user_id" placeholder="user id" list="known-users" required><input name="duration" placeholder="3d / 2h / forever"><input name="reason" placeholder="reason"><button>Send</button></form>%s</section>`,
			rehabTopicID, chatID, a.webUserDatalist(chatID)))
	}
	body.WriteString(`<table><thead><tr><th>User</th><th>Reason</th><th>Release</th><th>By</th><th></th></tr></thead><tbody>`)
	for _, rec := range a.store.ModerationRecords(chatID, "jail", true, time.Now().UTC()) {
		body.WriteString(fmt.Sprintf(`<tr><td>%s (%d)</td><td>%s</td><td>%s</td><td>%s</td><td><form method="post" action="/rehab/release"><input type="hidden" name="chat_id" value="%d"><input type="hidden" name="user_id" value="%d"><button>Release</button></form></td></tr>`,
			esc(rec.DisplayName), rec.UserID, esc(rec.Reason), esc(releaseLine(rec.UntilUnix, time.Now().UTC())), esc(rec.ByAdminName), chatID, rec.UserID))
	}
	body.WriteString(`</tbody></table>`)
	a.webLayout(w, "Psych center", body.String())
}

func (a *App) webRehabAdd(w http.ResponseWriter, r *http.Request) {
	chatID, _ := strconv.ParseInt(strings.TrimSpace(r.FormValue("chat_id")), 10, 64)
	redirect := "/rehab?chat_id=" + strconv.FormatInt(chatID, 10)
	if r.Method != http.MethodPost {
		http.Redirect(w, r, redirect, http.StatusFound)
		return
	}
	chat := &tele.Chat{ID: chatID, Type: tele.ChatSuperGroup}
	if a.rehabTopicID(chat) == 0 {
		http.Redirect(w, r, redirect, http.StatusFound)
		return
	}
	userID, _ := strconv.ParseInt(strings.TrimSpace(r.FormValue("user_id")), 10, 64)
	if userID == 0 {
		http.Redirect(w, r, redirect, http.StatusFound)
		return
	}
	member, _ := a.store.Member(chatID, userID)
	name := member.DisplayName
	if name == "" {
		name = fmt.Sprintf("user %d", userID)
	}
	untilUnix := int64(0)
	if token := strings.TrimSpace(r.FormValue("duration")); token != "" {
		if d, permanent, ok, err := parseDurationToken(token); err == nil && ok && !permanent && d > 0 {
			untilUnix = time.Now().UTC().Add(d).Unix()
		}
	}
	rec := ModerationRecord{
		UserID:      userID,
		Username:    member.Username,
		DisplayName: name,
		Reason:      strings.TrimSpace(r.FormValue("reason")),
		ByAdminName: "web panel",
		SinceUnix:   time.Now().UTC().Unix(),
		UntilUnix:   untilUnix,
	}
	rec.RehabWard = a.chooseRehabWard(chatID, targetUser{ID: userID, Username: member.Username, Name: name}, rec.Reason, strings.TrimSpace(r.FormValue("rehab_ward")), time.Now().UTC())
	a.store.SetJail(chatID, rec)
	a.clearRehabEscapeCooldown(chatID, rec.UserID)
	a.showRehabEscapeCommand(chatID, rec.UserID)
	a.sendRehabNotice(chat, rec)
	a.sendOwnerLog("Psych center", logField{"User", fmt.Sprintf("%s (%d)", name, userID)}, logField{"Chat", fmt.Sprintf("%d", chatID)}, logField{"Release", releaseLine(untilUnix, time.Now().UTC())}, logField{"Reason", textOrDash(rec.Reason)}, logField{"By", "web panel"})
	http.Redirect(w, r, redirect, http.StatusFound)
}

func (a *App) webRehabRelease(w http.ResponseWriter, r *http.Request) {
	chatID, _ := strconv.ParseInt(strings.TrimSpace(r.FormValue("chat_id")), 10, 64)
	userID, _ := strconv.ParseInt(strings.TrimSpace(r.FormValue("user_id")), 10, 64)
	if r.Method == http.MethodPost && chatID != 0 && userID != 0 {
		rec, released := a.store.RemoveJail(chatID, userID)
		if released {
			a.hideRehabEscapeCommand(chatID, userID)
			a.clearRehabEscapeCooldown(chatID, userID)
		}
		duration := a.rehabReleaseDurationText(rec, released, time.Now().UTC())
		if released {
			a.logRehabRelease(chatID, rec, duration, "web")
			a.sendRehabReleaseNotice(&tele.Chat{ID: chatID, Type: tele.ChatSuperGroup}, rec, duration, 0)
			a.schedulePostRehabCheck(chatID, rec, time.Now().UTC())
		}
		a.sendOwnerLog("Psych center released", logField{"User ID", fmt.Sprintf("%d", userID)}, logField{"Chat", fmt.Sprintf("%d", chatID)}, logField{"Duration", duration}, logField{"By", "web panel"})
	}
	http.Redirect(w, r, "/rehab?chat_id="+strconv.FormatInt(chatID, 10), http.StatusFound)
}

func (a *App) webModeration(w http.ResponseWriter, r *http.Request) {
	chatID := a.webChatID(r)
	now := time.Now().UTC()
	var body strings.Builder
	body.WriteString(chatSelector(a, r))
	for _, kind := range []string{"jail", "mute", "ban"} {
		body.WriteString("<h2>" + kind + "</h2><table><thead><tr><th>User</th><th>Reason</th><th>Until</th><th>By</th></tr></thead><tbody>")
		for _, rec := range a.store.ModerationRecords(chatID, kind, true, now) {
			body.WriteString(fmt.Sprintf("<tr><td>%s (%d)</td><td>%s</td><td>%s</td><td>%s</td></tr>",
				esc(rec.DisplayName), rec.UserID, esc(rec.Reason), esc(a.formatUntil(rec.UntilUnix)), esc(rec.ByAdminName)))
		}
		body.WriteString("</tbody></table>")
	}
	a.webLayout(w, "Moderation", body.String())
}

func (a *App) webWarnings(w http.ResponseWriter, r *http.Request) {
	chatID := a.webChatID(r)
	var rows strings.Builder
	for _, rec := range a.store.Warnings(chatID, 0, false, 100, time.Now().UTC()) {
		rows.WriteString(fmt.Sprintf("<tr><td>%s</td><td>%s (%d)</td><td>%s</td><td>%s</td><td>%s</td></tr>",
			esc(rec.ID), esc(rec.DisplayName), rec.UserID, esc(rec.Reason), esc(a.formatUntil(rec.ExpiresUnix)), esc(rec.ByAdminName)))
	}
	a.webLayout(w, "Warnings", chatSelector(a, r)+`<table><thead><tr><th>ID</th><th>User</th><th>Reason</th><th>Expires</th><th>By</th></tr></thead><tbody>`+rows.String()+`</tbody></table>`)
}

func (a *App) webKarma(w http.ResponseWriter, r *http.Request) {
	chatID := a.webChatID(r)
	var rows strings.Builder
	for i, rec := range a.store.TopKarma(chatID, 100) {
		rows.WriteString(fmt.Sprintf("<tr><td>%d</td><td>%s (%d)</td><td>%d</td></tr>", i+1, esc(rec.DisplayName), rec.UserID, rec.Score))
	}
	a.webLayout(w, "Karma", chatSelector(a, r)+`<table><thead><tr><th>#</th><th>User</th><th>Karma</th></tr></thead><tbody>`+rows.String()+`</tbody></table>`)
}

func (a *App) webFAQ(w http.ResponseWriter, r *http.Request) {
	var body strings.Builder
	body.WriteString(`<section class="panel"><div class="toolbar"><input id="filter" placeholder="Search FAQ..." oninput="filterCards()"><button type="button" onclick="randomCard()">Random preview</button></div>
<form class="stack" method="post" action="/faq/add" oninput="previewFAQ(this)"><input name="patterns" placeholder="pattern1; pattern2"><textarea name="answer" placeholder="answer"></textarea><div class="preview"></div><p class="dupe muted"></p><button>Add FAQ</button></form></section><div class="cards" id="cards">`)
	for _, rec := range a.faq.List() {
		body.WriteString(fmt.Sprintf(`<article class="item-card searchable" data-id="%s" data-value="%s" data-search="%s"><form method="post" action="/faq/update" oninput="previewFAQ(this)">
<header><b>%s</b><span>uses %d</span></header>
<input name="id" value="%s" readonly><input name="patterns" value="%s"><textarea name="answer">%s</textarea><label><input type="checkbox" name="enabled" %s> enabled</label><div class="preview">%s</div><p class="dupe muted"></p><div class="actions"><button>Save</button></form><form method="post" action="/faq/delete"><input type="hidden" name="id" value="%s"><button class="danger">Delete</button></form></div></article>`,
			esc(rec.ID), esc(strings.ToLower(strings.Join(rec.Patterns, "; "))),
			esc(strings.ToLower(rec.ID+" "+strings.Join(rec.Patterns, " ")+" "+rec.Answer)),
			esc(rec.ID), rec.Uses,
			esc(rec.ID), esc(strings.Join(rec.Patterns, "; ")), esc(rec.Answer), checked(rec.Enabled), esc(rec.Answer), esc(rec.ID)))
	}
	body.WriteString(`</div>` + richEditorScript())
	a.webLayout(w, "FAQ", body.String())
}

func (a *App) webFAQAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		_, _ = a.faq.Add(splitPatternList(r.FormValue("patterns")), r.FormValue("answer"))
	}
	http.Redirect(w, r, "/faq", http.StatusFound)
}
