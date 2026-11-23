package main

import (
	"crypto/sha256"
	"fmt"
	"html"
	"net/http"
	"sort"
	"strings"
)

func (a *App) knownTopicButtons(chatID int64) string {
	seen := make(map[int]struct{})
	for _, member := range a.store.Members(chatID) {
		if member.LastThreadID != 0 {
			seen[member.LastThreadID] = struct{}{}
		}
	}
	if len(seen) == 0 {
		return `<p class="muted">Known topics: -</p>`
	}
	ids := make([]int, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	var b strings.Builder
	b.WriteString(`<div class="chips"><span class="muted">Known topics:</span>`)
	for _, id := range ids {
		b.WriteString(fmt.Sprintf(`<form method="post" action="/supergroups/rehab-topic"><input type="hidden" name="chat_id" value="%d"><input type="hidden" name="topic_id" value="%d"><button class="chip" title="Use topic %d as psych center">%d</button></form>`, chatID, id, id, id))
	}
	b.WriteString(`</div>`)
	return b.String()
}

func (a *App) webUserDatalist(chatID int64) string {
	var b strings.Builder
	b.WriteString(`<datalist id="known-users">`)
	for _, member := range a.store.TopActive(chatID, 200) {
		b.WriteString(fmt.Sprintf(`<option value="%d">%s</option>`, member.ID, esc(member.DisplayName)))
	}
	b.WriteString(`</datalist>`)
	return b.String()
}

func legacyRehabTopicFor(cfg Config, chatID int64) int {
	if cfg.Chat.ID != 0 && cfg.Chat.ID == chatID {
		return cfg.Chat.RehabTopicID
	}
	return 0
}

func chatSettingsTitle(settings ChatSettings) string {
	if strings.TrimSpace(settings.Title) != "" {
		return settings.Title
	}
	return fmt.Sprintf("chat %d", settings.ChatID)
}

func debugButton(enabled bool) string {
	if enabled {
		return "Switch to release"
	}
	return "Switch to debug"
}

func debugModeText(enabled bool) string {
	if enabled {
		return "debug"
	}
	return "release"
}

func selected(value bool) string {
	if value {
		return " selected"
	}
	return ""
}

func richEditorScript() string {
	return `<script>
function text(v){return (v||'').replace(/[&<>"]/g,function(c){return {'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'}[c]})}
function filterCards(){const q=(document.getElementById('filter')?.value||'').toLowerCase();document.querySelectorAll('.searchable').forEach(c=>{c.style.display=(c.dataset.search||'').includes(q)?'':'none'})}
function formID(f){return f.querySelector('[name=id]')?.value||''}
function dupe(f,value){const n=f.querySelector('.dupe');if(!n)return;value=(value||'').trim().toLowerCase();if(!value){n.textContent='';return}const id=formID(f);const hit=[...document.querySelectorAll('.searchable')].some(c=>(c.dataset.id||'')!==id&&(c.dataset.value||'')===value);n.textContent=hit?'Похоже, такая запись уже есть.':'';n.className=hit?'dupe warn':'dupe muted'}
function previewQuote(f){const t=f.querySelector('[name=text]')?.value||'';const a=f.querySelector('[name=author]')?.value||'';const p=f.querySelector('.preview');if(p)p.innerHTML=text(t)+(a?'<br><small>'+text(a)+'</small>':'');dupe(f,t)}
function previewFAQ(f){const p=f.querySelector('.preview');const a=f.querySelector('[name=answer]')?.value||'';const patterns=f.querySelector('[name=patterns]')?.value||'';if(p)p.innerHTML=text(a);dupe(f,patterns)}
function randomCard(){const cards=[...document.querySelectorAll('.searchable')].filter(c=>c.style.display!=='none');if(!cards.length)return;cards.forEach(c=>c.classList.remove('picked'));const c=cards[Math.floor(Math.random()*cards.length)];c.classList.add('picked');c.scrollIntoView({behavior:'smooth',block:'center'})}
document.querySelectorAll('form').forEach(f=>{if(f.querySelector('[name=text]'))previewQuote(f);if(f.querySelector('[name=answer]'))previewFAQ(f)})
</script>`
}

func (a *App) webLayout(w http.ResponseWriter, title, body string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprintf(w, `<!doctype html><html><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>%s</title><style>
body{margin:0;font:14px system-ui,Segoe UI,sans-serif;background:#f6f7f9;color:#111}nav{display:flex;gap:12px;align-items:center;padding:12px 18px;background:#fff;border-bottom:1px solid #ddd;position:sticky;top:0;z-index:2;overflow:auto}nav a{color:#135;text-decoration:none;white-space:nowrap}main{padding:18px;max-width:1240px;margin:auto}table{width:100%%;border-collapse:collapse;background:#fff;margin:12px 0}th,td{border-bottom:1px solid #e5e7eb;padding:8px;text-align:left;vertical-align:top}input,select,textarea,button{padding:8px;border:1px solid #cbd5e1;border-radius:6px;font:inherit}textarea{min-height:92px;resize:vertical}button{background:#123;color:#fff;cursor:pointer}.danger{background:#8b1a1a}.chip{background:#eef2f6;color:#123;border-color:#ccd5df}.grid,.cards{display:grid;grid-template-columns:repeat(auto-fit,minmax(260px,1fr));gap:12px}.card,.panel,.item-card{background:#fff;border:1px solid #e5e7eb;border-radius:8px;padding:16px}.card b{display:block;font-size:28px}.card span,.muted{color:#667}.error,.warn{color:#b00020}.inline{display:inline}.form-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(180px,1fr));gap:10px;align-items:end}.stack{display:grid;gap:10px}.toolbar{display:flex;gap:10px;margin-bottom:12px}.item-card header{display:flex;justify-content:space-between;gap:8px;margin-bottom:10px}.item-card form{display:grid;gap:8px}.actions,.chips{display:flex;gap:8px;flex-wrap:wrap;align-items:center}.chips{margin-top:10px}.preview{background:#f2f5f8;border:1px dashed #cbd5e1;border-radius:6px;padding:10px;min-height:38px}.picked{outline:3px solid #2f80ed}
</style></head><body><nav><b>Mirohodec Warden</b><a href="/">Dashboard</a><a href="/supergroups">Supergroups</a><a href="/rehab">Psych center</a><a href="/users">Users</a><a href="/moderation">Moderation</a><a href="/warnings">Warnings</a><a href="/karma">Karma</a><a href="/faq">FAQ</a><a href="/quotes">Quotes</a><a href="/logout">Logout</a></nav><main><h1>%s</h1>%s</main></body></html>`, esc(title), esc(title), body)
}

func webSessionHash(token string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(token)))
	return fmt.Sprintf("%x", sum[:])
}

func webCookieSecure(cfg WebConfig) bool {
	return cfg.CookieSecure || (cfg.BehindProxy && strings.HasPrefix(strings.ToLower(strings.TrimSpace(cfg.PublicURL)), "https://"))
}

func loginForm() string {
	return `<form method="post"><input type="password" name="token" placeholder="admin token" autofocus><button>Login</button></form>`
}

func esc(value string) string {
	return html.EscapeString(value)
}

func checked(value bool) string {
	if value {
		return "checked"
	}
	return ""
}
