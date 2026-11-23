package web

import (
	"errors"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

func (s *server) render(w http.ResponseWriter, name string, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		if s.cfg.Logger != nil {
			s.cfg.Logger.Warn("web template render failed", zap.String("template", name), zap.Error(err))
		}
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func (s *server) requirePost(w http.ResponseWriter, r *http.Request) bool {
	if r.Method == http.MethodPost {
		_ = r.ParseForm()
		return true
	}
	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	return false
}

func (s *server) redirect(w http.ResponseWriter, r *http.Request, target, message string, err error) {
	u, parseErr := url.Parse(target)
	if parseErr != nil {
		u = &url.URL{Path: target}
	}
	q := u.Query()
	if err != nil {
		q.Set("err", err.Error())
	} else if strings.TrimSpace(message) != "" {
		q.Set("msg", message)
	}
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func groupInputFromRequest(r *http.Request) (GroupInput, error) {
	chatID, err := formInt64(r, "chat_id")
	rehabTopicID, topicErr := formInt(r, "rehab_topic_id")
	if topicErr != nil && strings.TrimSpace(r.FormValue("rehab_topic_id")) == "" {
		topicErr = nil
	}
	if err == nil {
		err = topicErr
	}
	return GroupInput{
		ChatID:                chatID,
		Title:                 strings.TrimSpace(r.FormValue("title")),
		Enabled:               r.FormValue("enabled") == "on",
		RehabTopicID:          rehabTopicID,
		SpamNoRehabAction:     strings.TrimSpace(r.FormValue("spam_no_rehab_action")),
		AIEnabledMode:         strings.TrimSpace(r.FormValue("ai_enabled_mode")),
		SpamEnabledMode:       strings.TrimSpace(r.FormValue("spam_enabled_mode")),
		WarnEnabledMode:       strings.TrimSpace(r.FormValue("warn_enabled_mode")),
		KarmaEnabledMode:      strings.TrimSpace(r.FormValue("karma_enabled_mode")),
		FAQEnabledMode:        strings.TrimSpace(r.FormValue("faq_enabled_mode")),
		QuotesEnabledMode:     strings.TrimSpace(r.FormValue("quotes_enabled_mode")),
		PhotosEnabledMode:     strings.TrimSpace(r.FormValue("photos_enabled_mode")),
		WarnExpireAfter:       strings.TrimSpace(r.FormValue("warn_expire_after")),
		WarnMuteThreshold:     formIntDefault(r, "warn_mute_threshold"),
		WarnMuteDuration:      strings.TrimSpace(r.FormValue("warn_mute_duration")),
		WarnRehabThreshold:    formIntDefault(r, "warn_rehab_threshold"),
		WarnRehabDuration:     strings.TrimSpace(r.FormValue("warn_rehab_duration")),
		KarmaMessagesPerPoint: formIntDefault(r, "karma_messages_per_point"),
		KarmaDailyLimit:       formIntDefault(r, "karma_daily_limit"),
		KarmaManualCooldown:   strings.TrimSpace(r.FormValue("karma_manual_cooldown")),
	}, err
}

func formInt64(r *http.Request, name string) (int64, error) {
	value := strings.TrimSpace(r.FormValue(name))
	if value == "" {
		return 0, errors.New(name + " is required")
	}
	return strconv.ParseInt(value, 10, 64)
}

func formInt64Default(r *http.Request, name string) int64 {
	n, _ := formInt64(r, name)
	return n
}

func formInt(r *http.Request, name string) (int, error) {
	value := strings.TrimSpace(r.FormValue(name))
	if value == "" {
		return 0, errors.New(name + " is required")
	}
	return strconv.Atoi(value)
}

func formIntDefault(r *http.Request, name string) int {
	n, _ := formInt(r, name)
	return n
}

func queryInt64(r *http.Request, name string) int64 {
	value := strings.TrimSpace(r.URL.Query().Get(name))
	if value == "" {
		return 0
	}
	n, _ := strconv.ParseInt(value, 10, 64)
	return n
}

func chatURL(path string, chatID int64) string {
	if chatID == 0 {
		return path
	}
	return path + "?chat_id=" + strconv.FormatInt(chatID, 10)
}

func staticSubtree() fs.FS {
	sub, err := fs.Sub(assets, "static")
	if err != nil {
		return assets
	}
	return sub
}
