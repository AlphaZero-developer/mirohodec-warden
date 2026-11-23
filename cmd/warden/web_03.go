package main

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

func (a *App) webFAQUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		_, _, _ = a.faq.Update(r.FormValue("id"), splitPatternList(r.FormValue("patterns")), r.FormValue("answer"), r.FormValue("enabled") == "on")
	}
	http.Redirect(w, r, "/faq", http.StatusFound)
}

func (a *App) webFAQDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		_, _, _ = a.faq.Delete(r.FormValue("id"))
	}
	http.Redirect(w, r, "/faq", http.StatusFound)
}

func (a *App) webQuotes(w http.ResponseWriter, r *http.Request) {
	var body strings.Builder
	body.WriteString(`<section class="panel"><div class="toolbar"><input id="filter" placeholder="Search quotes..." oninput="filterCards()"><button type="button" onclick="randomCard()">Random preview</button></div>
<form class="stack" method="post" action="/quotes/add" oninput="previewQuote(this)"><textarea name="text" placeholder="quote"></textarea><input name="author" placeholder="author"><input name="tags" placeholder="tag1,tag2"><div class="preview"></div><p class="dupe muted"></p><button>Add quote</button></form></section><div class="cards" id="cards">`)
	for _, rec := range a.quotes.List("") {
		preview := esc(rec.Text)
		if strings.TrimSpace(rec.Author) != "" {
			preview += `<br><small>` + esc(rec.Author) + `</small>`
		}
		body.WriteString(fmt.Sprintf(`<article class="item-card searchable" data-id="%s" data-value="%s" data-search="%s"><form method="post" action="/quotes/update" oninput="previewQuote(this)">
<header><b>%s</b><span>uses %d</span></header>
<input name="id" value="%s" readonly><textarea name="text">%s</textarea><input name="author" value="%s"><input name="tags" value="%s"><div class="preview">%s</div><p class="dupe muted"></p><div class="actions"><button>Save</button></form><form method="post" action="/quotes/delete"><input type="hidden" name="id" value="%s"><button class="danger">Delete</button></form></div></article>`,
			esc(rec.ID), esc(strings.ToLower(rec.Text)),
			esc(strings.ToLower(rec.ID+" "+rec.Text+" "+rec.Author+" "+strings.Join(rec.Tags, " "))),
			esc(rec.ID), rec.Uses, esc(rec.ID), esc(rec.Text), esc(rec.Author), esc(strings.Join(rec.Tags, ",")), preview, esc(rec.ID)))
	}
	body.WriteString(`</div>` + richEditorScript())
	a.webLayout(w, "Quotes", body.String())
}

func (a *App) webQuoteAdd(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		_, _ = a.quotes.Add(r.FormValue("text"), r.FormValue("author"), splitPatternList(r.FormValue("tags")), nil)
	}
	http.Redirect(w, r, "/quotes", http.StatusFound)
}

func (a *App) webQuoteUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		_, _, _ = a.quotes.Update(r.FormValue("id"), r.FormValue("text"), r.FormValue("author"), splitPatternList(r.FormValue("tags")))
	}
	http.Redirect(w, r, "/quotes", http.StatusFound)
}

func (a *App) webQuoteDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		_, _, _ = a.quotes.Delete(r.FormValue("id"))
	}
	http.Redirect(w, r, "/quotes", http.StatusFound)
}

func (a *App) webChatID(r *http.Request) int64 {
	if raw := strings.TrimSpace(r.URL.Query().Get("chat_id")); raw != "" {
		if id, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return id
		}
	}
	if id := a.config().Chat.ID; id != 0 {
		return id
	}
	ids := a.store.ChatIDs()
	if len(ids) > 0 {
		return ids[0]
	}
	return 0
}

func chatSelector(a *App, r *http.Request) string {
	ids := a.store.ChatIDs()
	if len(ids) == 0 && a.config().Chat.ID != 0 {
		ids = []int64{a.config().Chat.ID}
	}
	if len(ids) <= 1 {
		return ""
	}
	current := a.webChatID(r)
	var b strings.Builder
	b.WriteString(`<form class="inline" method="get"><select name="chat_id" onchange="this.form.submit()">`)
	for _, id := range ids {
		selected := ""
		if id == current {
			selected = " selected"
		}
		b.WriteString(fmt.Sprintf(`<option value="%d"%s>%s</option>`, id, selected, esc(a.chatSelectLabel(id))))
	}
	b.WriteString(`</select></form>`)
	return b.String()
}

func (a *App) chatSelectLabel(chatID int64) string {
	if settings, ok := a.store.ChatSettings(chatID); ok && strings.TrimSpace(settings.Title) != "" {
		return fmt.Sprintf("%s (%d)", strings.TrimSpace(settings.Title), chatID)
	}
	if a.config().Chat.ID == chatID && chatID != 0 {
		return fmt.Sprintf("legacy config chat (%d)", chatID)
	}
	return fmt.Sprintf("chat %d", chatID)
}

func (a *App) webKnownChatSettings() []ChatSettings {
	byID := make(map[int64]ChatSettings)
	for _, settings := range a.store.ManagedChats(true) {
		byID[settings.ChatID] = settings
	}
	for _, id := range a.store.ChatIDs() {
		if _, ok := byID[id]; !ok {
			title := ""
			if id == a.config().Chat.ID {
				title = "legacy config chat"
			}
			byID[id] = ChatSettings{
				ChatID:            id,
				Title:             title,
				Enabled:           true,
				RehabTopicID:      legacyRehabTopicFor(a.config(), id),
				SpamNoRehabAction: spamNoRehabMuteOnDelete,
				DebugCommands:     a.config().Telegram.ShowDebugCommands,
			}
		}
	}
	if len(byID) == 0 && a.config().Chat.ID != 0 {
		byID[a.config().Chat.ID] = ChatSettings{
			ChatID:            a.config().Chat.ID,
			Title:             "legacy config chat",
			Enabled:           true,
			RehabTopicID:      a.config().Chat.RehabTopicID,
			SpamNoRehabAction: spamNoRehabMuteOnDelete,
			DebugCommands:     a.config().Telegram.ShowDebugCommands,
		}
	}
	out := make([]ChatSettings, 0, len(byID))
	for _, settings := range byID {
		if settings.SpamNoRehabAction == "" {
			settings.SpamNoRehabAction = spamNoRehabMuteOnDelete
		}
		out = append(out, settings)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Title == out[j].Title {
			return out[i].ChatID < out[j].ChatID
		}
		return out[i].Title < out[j].Title
	})
	return out
}

func chatSettingsFromRequest(r *http.Request, old ChatSettings) ChatSettings {
	chatID, _ := strconv.ParseInt(strings.TrimSpace(r.FormValue("chat_id")), 10, 64)
	rehabTopicID, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("rehab_topic_id")))
	settings := old
	settings.ChatID = chatID
	settings.Title = strings.TrimSpace(r.FormValue("title"))
	settings.Enabled = r.FormValue("enabled") == "on"
	settings.RehabTopicID = rehabTopicID
	settings.SpamNoRehabAction = strings.TrimSpace(r.FormValue("spam_no_rehab_action"))
	if settings.SpamNoRehabAction == "" {
		settings.SpamNoRehabAction = spamNoRehabMuteOnDelete
	}
	return settings
}
