package web

import (
	"errors"
	"net/http"
	"strings"
)

func (s *server) layaUserUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	input := LayaUserInput{
		ChatID:      formInt64Default(r, "chat_id"),
		UserID:      formInt64Default(r, "user_id"),
		Username:    r.FormValue("username"),
		DisplayName: r.FormValue("display_name"),
		Gender:      r.FormValue("gender"),
		Address:     r.FormValue("address"),
		Tags:        r.FormValue("tags"),
		Note:        r.FormValue("note"),
	}
	err := s.cfg.Service.UpdateLayaUser(r.Context(), input)
	s.redirect(w, r, chatURL("/laya/users", input.ChatID), "Профиль обращения сохранён.", err)
}

func (s *server) layaCacheClear(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	err := s.cfg.Service.ClearAICache(r.Context())
	s.redirect(w, r, "/laya", "AI-кэш очищен.", err)
}

func (s *server) aiLogs(w http.ResponseWriter, r *http.Request) {
	page, err := s.cfg.Service.AILogs(r.Context())
	if err != nil {
		s.renderError(w, r, "AI-логи", "ai-logs", "ai_logs.html", err)
		return
	}
	data := s.baseData(r, "AI-логи", "ai-logs")
	data.AILogs = page
	s.render(w, "ai_logs.html", data)
}

func (s *server) aiLogsClear(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	err := s.cfg.Service.ClearAILogs(r.Context())
	s.redirect(w, r, "/ai-logs", "AI-логи очищены.", err)
}

func (s *server) faq(w http.ResponseWriter, r *http.Request) {
	rows, err := s.cfg.Service.FAQ(r.Context())
	if err != nil {
		s.renderError(w, r, "FAQ", "faq", "faq.html", err)
		return
	}
	data := s.baseData(r, "FAQ", "faq")
	data.FAQs = rows
	s.render(w, "faq.html", data)
}

func (s *server) faqAdd(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	input := FAQInput{Patterns: r.FormValue("patterns"), Answer: r.FormValue("answer"), Enabled: true}
	err := s.cfg.Service.AddFAQ(r.Context(), input)
	s.redirect(w, r, "/faq", "FAQ добавлен.", err)
}

func (s *server) faqUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	input := FAQInput{ID: r.FormValue("id"), Patterns: r.FormValue("patterns"), Answer: r.FormValue("answer"), Enabled: r.FormValue("enabled") == "on"}
	err := s.cfg.Service.UpdateFAQ(r.Context(), input)
	s.redirect(w, r, "/faq", "FAQ сохранен.", err)
}

func (s *server) faqDelete(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	err := s.cfg.Service.DeleteFAQ(r.Context(), r.FormValue("id"))
	s.redirect(w, r, "/faq", "FAQ удален.", err)
}

func (s *server) quotes(w http.ResponseWriter, r *http.Request) {
	rows, err := s.cfg.Service.Quotes(r.Context())
	if err != nil {
		s.renderError(w, r, "Цитаты", "quotes", "quotes.html", err)
		return
	}
	data := s.baseData(r, "Цитаты", "quotes")
	data.Quotes = rows
	s.render(w, "quotes.html", data)
}

func (s *server) quoteAdd(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	input := QuoteInput{Text: r.FormValue("text"), Author: r.FormValue("author"), Tags: r.FormValue("tags")}
	err := s.cfg.Service.AddQuote(r.Context(), input)
	s.redirect(w, r, "/quotes", "Цитата добавлена.", err)
}

func (s *server) quoteUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	input := QuoteInput{ID: r.FormValue("id"), Text: r.FormValue("text"), Author: r.FormValue("author"), Tags: r.FormValue("tags")}
	err := s.cfg.Service.UpdateQuote(r.Context(), input)
	s.redirect(w, r, "/quotes", "Цитата сохранена.", err)
}

func (s *server) quoteDelete(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	err := s.cfg.Service.DeleteQuote(r.Context(), r.FormValue("id"))
	s.redirect(w, r, "/quotes", "Цитата удалена.", err)
}

func (s *server) photos(w http.ResponseWriter, r *http.Request) {
	tag := strings.TrimSpace(r.URL.Query().Get("tag"))
	rows, err := s.cfg.Service.Photos(r.Context(), tag)
	if err != nil {
		s.renderError(w, r, "Фото", "photos", "photos.html", err)
		return
	}
	data := s.baseData(r, "Фото", "photos")
	data.Photos = rows
	data.PhotoTag = tag
	s.render(w, "photos.html", data)
}

func (s *server) photoAdd(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	input := PhotoInput{FileID: r.FormValue("file_id"), UniqueID: r.FormValue("unique_id"), Caption: r.FormValue("caption"), Tags: r.FormValue("tags")}
	err := s.cfg.Service.AddPhoto(r.Context(), input)
	s.redirect(w, r, "/photos", "Фото добавлено.", err)
}

func (s *server) photoDelete(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	err := s.cfg.Service.DeletePhoto(r.Context(), r.FormValue("id"))
	s.redirect(w, r, "/photos", "Фото удалено.", err)
}

func (s *server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.authenticated(r) {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next(w, r)
	}
}

func (s *server) chatPage(w http.ResponseWriter, r *http.Request, title, active string) (pageData, bool) {
	chats, err := s.cfg.Service.ChatSelection(r.Context(), queryInt64(r, "chat_id"))
	if err != nil {
		s.renderError(w, r, title, active, active+".html", err)
		return pageData{}, false
	}
	data := s.baseData(r, title, active)
	data.Chats = chats
	return data, true
}

func (s *server) baseData(r *http.Request, title, active string) pageData {
	return pageData{
		Title:   title,
		Active:  active,
		Message: r.URL.Query().Get("msg"),
		Error:   r.URL.Query().Get("err"),
	}
}

func (s *server) renderError(w http.ResponseWriter, r *http.Request, title, active, name string, err error) {
	data := s.baseData(r, title, active)
	if err == nil {
		err = errors.New("unknown error")
	}
	data.Error = err.Error()
	s.render(w, name, data)
}
