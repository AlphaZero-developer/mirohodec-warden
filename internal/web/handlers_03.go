package web

import (
	"net/http"
	"strings"
)

func (s *server) users(w http.ResponseWriter, r *http.Request) {
	data, ok := s.chatPage(w, r, "Пользователи", "users")
	if !ok {
		return
	}
	rows, err := s.cfg.Service.Users(r.Context(), data.Chats.CurrentID)
	if err != nil {
		s.renderError(w, r, "Пользователи", "users", "users.html", err)
		return
	}
	data.Users = rows
	s.render(w, "users.html", data)
}

func (s *server) moderation(w http.ResponseWriter, r *http.Request) {
	data, ok := s.chatPage(w, r, "Модерация", "moderation")
	if !ok {
		return
	}
	page, err := s.cfg.Service.Moderation(r.Context(), data.Chats.CurrentID)
	if err != nil {
		s.renderError(w, r, "Модерация", "moderation", "moderation.html", err)
		return
	}
	data.Moderation = page
	s.render(w, "moderation.html", data)
}

func (s *server) foreverBan(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	input := ForeverBanInput{
		ChatID: formInt64Default(r, "chat_id"),
		UserID: formInt64Default(r, "user_id"),
		Reason: strings.TrimSpace(r.FormValue("reason")),
	}
	err := s.cfg.Service.AddForeverBan(r.Context(), input)
	redirect := chatURL("/users", input.ChatID)
	if strings.TrimSpace(r.FormValue("return_to")) == "moderation" {
		redirect = chatURL("/moderation", input.ChatID)
	}
	s.redirect(w, r, redirect, "Пользователь добавлен в вечный бан.", err)
}

func (s *server) warnings(w http.ResponseWriter, r *http.Request) {
	data, ok := s.chatPage(w, r, "Предупреждения", "warnings")
	if !ok {
		return
	}
	rows, err := s.cfg.Service.Warnings(r.Context(), data.Chats.CurrentID)
	if err != nil {
		s.renderError(w, r, "Предупреждения", "warnings", "warnings.html", err)
		return
	}
	data.Warnings = rows
	s.render(w, "warnings.html", data)
}

func (s *server) karma(w http.ResponseWriter, r *http.Request) {
	data, ok := s.chatPage(w, r, "Карма", "karma")
	if !ok {
		return
	}
	rows, err := s.cfg.Service.Karma(r.Context(), data.Chats.CurrentID)
	if err != nil {
		s.renderError(w, r, "Карма", "karma", "karma.html", err)
		return
	}
	data.Karma = rows
	s.render(w, "karma.html", data)
}

func (s *server) ownerSleep(w http.ResponseWriter, r *http.Request) {
	page, err := s.cfg.Service.OwnerSleep(r.Context())
	if err != nil {
		s.renderError(w, r, "Сон создателя", "sleep", "sleep.html", err)
		return
	}
	data := s.baseData(r, "Сон создателя", "sleep")
	data.OwnerSleep = page
	s.render(w, "sleep.html", data)
}

func (s *server) ownerSleepClear(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	userID, err := formInt64(r, "user_id")
	if err == nil {
		err = s.cfg.Service.ClearOwnerSleep(r.Context(), userID)
	}
	s.redirect(w, r, "/sleep", "Режим сна снят.", err)
}

func (s *server) laya(w http.ResponseWriter, r *http.Request) {
	page, err := s.cfg.Service.LayaControl(r.Context())
	if err != nil {
		s.renderError(w, r, "Лайя", "laya", "laya.html", err)
		return
	}
	data := s.baseData(r, "Лайя", "laya")
	data.Laya = page
	s.render(w, "laya.html", data)
}

func (s *server) layaUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	input := LayaControlInput{
		SystemPrompt:         r.FormValue("system_prompt"),
		ExtraPrompt:          r.FormValue("extra_prompt"),
		FallbackReply:        r.FormValue("fallback_reply"),
		BaseURL:              r.FormValue("base_url"),
		Model:                r.FormValue("model"),
		Timeout:              r.FormValue("timeout"),
		SoftTimeout:          r.FormValue("soft_timeout"),
		FasMute:              r.FormValue("fas_mute_duration"),
		MaxTokens:            formIntDefault(r, "max_tokens"),
		MaxReplyChars:        formIntDefault(r, "max_reply_chars"),
		LogConversations:     r.FormValue("log_conversations"),
		FatherUserID:         formInt64Default(r, "father_user_id"),
		MotherUserID:         formInt64Default(r, "mother_user_id"),
		BrotherUserID:        formInt64Default(r, "brother_user_id"),
		CreatorUserID:        formInt64Default(r, "creator_user_id"),
		FatherPrompt:         r.FormValue("father_prompt"),
		MotherPrompt:         r.FormValue("mother_prompt"),
		BrotherPrompt:        r.FormValue("brother_prompt"),
		CreatorPrompt:        r.FormValue("creator_prompt"),
		Relationships:        r.FormValue("relationships"),
		MoodMode:             r.FormValue("mood_mode"),
		MoodPrompt:           r.FormValue("mood_prompt"),
		DirectInsultMode:     r.FormValue("direct_insult_mode"),
		DirectInsultPrompt:   r.FormValue("direct_insult_prompt"),
		DirectInsultFallback: r.FormValue("direct_insult_fallback"),
		SafetyCooldownMode:   r.FormValue("safety_cooldown_mode"),
		SafetyPauseReply:     r.FormValue("safety_pause_reply"),
		TagReactionRules:     r.FormValue("tag_reaction_rules"),
		RetryAttempts:        formIntDefault(r, "retry_attempts"),
		RetryBackoff:         r.FormValue("retry_backoff"),
	}
	err := s.cfg.Service.UpdateLayaControl(r.Context(), input)
	s.redirect(w, r, "/laya", "Состояние Лайи сохранено.", err)
}

func (s *server) layaSay(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	input := LayaManualMessageInput{
		ChatID:           formInt64Default(r, "chat_id"),
		ThreadID:         formIntDefault(r, "thread_id"),
		ReplyToMessageID: formIntDefault(r, "reply_to_message_id"),
		Text:             r.FormValue("text"),
	}
	err := s.cfg.Service.SendLayaMessage(r.Context(), input)
	s.redirect(w, r, "/laya", "Сообщение от Лайи отправлено.", err)
}

func (s *server) layaUsers(w http.ResponseWriter, r *http.Request) {
	data, ok := s.chatPage(w, r, "Обращения Лайи", "laya-users")
	if !ok {
		return
	}
	page, err := s.cfg.Service.LayaUsers(r.Context(), data.Chats.CurrentID)
	if err != nil {
		s.renderError(w, r, "Обращения Лайи", "laya-users", "laya_users.html", err)
		return
	}
	data.LayaUsers = page
	s.render(w, "laya_users.html", data)
}
