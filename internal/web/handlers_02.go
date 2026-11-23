package web

import (
	"net/http"
	"strings"
)

func (s *server) content(w http.ResponseWriter, r *http.Request) {
	page, err := s.cfg.Service.ContentControl(r.Context())
	if err != nil {
		s.renderError(w, r, "Контент", "content", "content.html", err)
		return
	}
	data := s.baseData(r, "Контент", "content")
	data.Content = page
	s.render(w, "content.html", data)
}

func (s *server) contentUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	input := ContentControlInput{
		FAQEnabledMode:          strings.TrimSpace(r.FormValue("faq_enabled_mode")),
		FAQAutoReplyMode:        strings.TrimSpace(r.FormValue("faq_auto_reply_mode")),
		FAQRequireMentionMode:   strings.TrimSpace(r.FormValue("faq_require_mention_mode")),
		FAQEntryCooldown:        strings.TrimSpace(r.FormValue("faq_entry_cooldown")),
		FAQUserCooldown:         strings.TrimSpace(r.FormValue("faq_user_cooldown")),
		QuotesEnabledMode:       strings.TrimSpace(r.FormValue("quotes_enabled_mode")),
		QuotesAutoReplyMode:     strings.TrimSpace(r.FormValue("quotes_auto_reply_mode")),
		QuotesAutoReplyChance:   strings.TrimSpace(r.FormValue("quotes_auto_reply_chance")),
		QuotesAutoReplyCooldown: strings.TrimSpace(r.FormValue("quotes_auto_reply_cooldown")),
		QuotesActiveUserWindow:  strings.TrimSpace(r.FormValue("quotes_active_user_window")),
		QuotesPhotoChance:       strings.TrimSpace(r.FormValue("quotes_photo_chance")),
		PhotosEnabledMode:       strings.TrimSpace(r.FormValue("photos_enabled_mode")),
		PhotosAutoReplyMode:     strings.TrimSpace(r.FormValue("photos_auto_reply_mode")),
		PhotosAutoReplyChance:   strings.TrimSpace(r.FormValue("photos_auto_reply_chance")),
		PhotosCooldown:          strings.TrimSpace(r.FormValue("photos_cooldown")),
		PhotosTriggers:          strings.TrimSpace(r.FormValue("photos_triggers")),
		PhotosLocalDir:          strings.TrimSpace(r.FormValue("photos_local_dir")),
		PhotosLocalExtensions:   strings.TrimSpace(r.FormValue("photos_local_extensions")),
		IdeasEnabledMode:        strings.TrimSpace(r.FormValue("ideas_enabled_mode")),
		IdeasUserCooldown:       strings.TrimSpace(r.FormValue("ideas_user_cooldown")),
		IdeasChatCooldown:       strings.TrimSpace(r.FormValue("ideas_chat_cooldown")),
		IdeasMaxLength:          formIntDefault(r, "ideas_max_length"),
	}
	err := s.cfg.Service.UpdateContentControl(r.Context(), input)
	s.redirect(w, r, "/content", "Настройки контента сохранены.", err)
}

func (s *server) supergroups(w http.ResponseWriter, r *http.Request) {
	groups, err := s.cfg.Service.Groups(r.Context())
	if err != nil {
		s.renderError(w, r, "Супергруппы", "supergroups", "supergroups.html", err)
		return
	}
	data := s.baseData(r, "Супергруппы", "supergroups")
	data.Groups = groups
	s.render(w, "supergroups.html", data)
}

func (s *server) supergroupAdd(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	input, err := groupInputFromRequest(r)
	if err == nil {
		err = s.cfg.Service.AddGroup(r.Context(), input)
	}
	s.redirect(w, r, "/supergroups", "Группа добавлена.", err)
}

func (s *server) supergroupUpdate(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	input, err := groupInputFromRequest(r)
	if err == nil {
		err = s.cfg.Service.UpdateGroup(r.Context(), input)
	}
	s.redirect(w, r, "/supergroups", "Группа сохранена.", err)
}

func (s *server) supergroupDelete(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	chatID, err := formInt64(r, "chat_id")
	if err == nil {
		err = s.cfg.Service.DeleteGroup(r.Context(), chatID)
	}
	s.redirect(w, r, "/supergroups", "Группа удалена.", err)
}

func (s *server) supergroupDebug(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	chatID, err := formInt64(r, "chat_id")
	debug := r.FormValue("debug") == "true"
	if err == nil {
		err = s.cfg.Service.SetGroupDebug(r.Context(), chatID, debug)
	}
	s.redirect(w, r, "/supergroups", "Режим меню изменен.", err)
}

func (s *server) supergroupRehabTopic(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	chatID, err := formInt64(r, "chat_id")
	topicID, topicErr := formInt(r, "topic_id")
	if err == nil {
		err = topicErr
	}
	if err == nil {
		err = s.cfg.Service.SetGroupRehabTopic(r.Context(), chatID, topicID)
	}
	s.redirect(w, r, "/supergroups", "Тема психиатрического центра сохранена.", err)
}

func (s *server) rehab(w http.ResponseWriter, r *http.Request) {
	chatID := queryInt64(r, "chat_id")
	chats, err := s.cfg.Service.ChatSelection(r.Context(), chatID)
	if err != nil {
		s.renderError(w, r, "Психиатрический центр", "rehab", "rehab.html", err)
		return
	}
	page, err := s.cfg.Service.Rehab(r.Context(), chats.CurrentID)
	if err != nil {
		s.renderError(w, r, "Психиатрический центр", "rehab", "rehab.html", err)
		return
	}
	data := s.baseData(r, "Психиатрический центр", "rehab")
	data.Chats = chats
	data.Rehab = page
	s.render(w, "rehab.html", data)
}

func (s *server) rehabAdd(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	input := RehabInput{
		ChatID:   formInt64Default(r, "chat_id"),
		UserID:   formInt64Default(r, "user_id"),
		Duration: strings.TrimSpace(r.FormValue("duration")),
		Reason:   strings.TrimSpace(r.FormValue("reason")),
	}
	err := s.cfg.Service.AddRehab(r.Context(), input)
	s.redirect(w, r, chatURL("/rehab", input.ChatID), "Пользователь отправлен в психиатрический центр.", err)
}

func (s *server) rehabRelease(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	chatID := formInt64Default(r, "chat_id")
	userID := formInt64Default(r, "user_id")
	err := s.cfg.Service.ReleaseRehab(r.Context(), chatID, userID)
	s.redirect(w, r, chatURL("/rehab", chatID), "Выход оформлен или назначен экзамен.", err)
}

func (s *server) rehabBrainwash(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	input := RehabBrainwashInput{
		ChatID:       formInt64Default(r, "chat_id"),
		Enabled:      r.FormValue("enabled") == "on",
		AIEnabled:    r.FormValue("ai_enabled") == "on",
		LeaderUserID: formInt64Default(r, "leader_user_id"),
		LeaderName:   r.FormValue("leader_name"),
		AIPrompt:     r.FormValue("ai_prompt"),
		Messages:     r.FormValue("messages"),
	}
	err := s.cfg.Service.UpdateRehabBrainwash(r.Context(), input)
	s.redirect(w, r, chatURL("/rehab", input.ChatID), "Режим промывки сохранен.", err)
}

func (s *server) rehabReleaseExam(w http.ResponseWriter, r *http.Request) {
	if !s.requirePost(w, r) {
		return
	}
	input := RehabReleaseExamInput{
		ChatID:       formInt64Default(r, "chat_id"),
		Enabled:      r.FormValue("enabled") == "on",
		Questions:    r.FormValue("questions"),
		AnswerWindow: strings.TrimSpace(r.FormValue("answer_window")),
		FailDuration: strings.TrimSpace(r.FormValue("fail_duration")),
		Scope:        strings.TrimSpace(r.FormValue("scope")),
	}
	err := s.cfg.Service.UpdateRehabReleaseExam(r.Context(), input)
	s.redirect(w, r, chatURL("/rehab", input.ChatID), "Экзамен перед выходом сохранен.", err)
}
