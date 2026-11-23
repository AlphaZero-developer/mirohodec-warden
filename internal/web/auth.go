package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const sessionCookieName = "warden_session"
const permanentSessionExpires = "forever"

func sameToken(got, expected string) bool {
	gotHash := sha256.Sum256([]byte(strings.TrimSpace(got)))
	expectedHash := sha256.Sum256([]byte(strings.TrimSpace(expected)))
	return subtle.ConstantTimeCompare(gotHash[:], expectedHash[:]) == 1
}

func (s *server) authenticated(r *http.Request) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return false
	}
	expires, sig, ok := strings.Cut(cookie.Value, ".")
	if !ok {
		return false
	}
	if expires == permanentSessionExpires {
		return subtle.ConstantTimeCompare([]byte(sig), []byte(sessionSignature(s.cfg.Web.AdminToken, expires))) == 1
	}
	expiresUnix, err := strconv.ParseInt(expires, 10, 64)
	if err != nil || time.Now().Unix() > expiresUnix {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(sig), []byte(sessionSignature(s.cfg.Web.AdminToken, expires))) == 1
}

func (s *server) sessionCookie() *http.Cookie {
	ttl := s.cfg.Web.SessionTTL
	if ttl == 0 {
		ttl = 24 * time.Hour
	}
	if ttl < 0 {
		expiresAt := time.Now().Add(permanentSessionTTL)
		return &http.Cookie{
			Name:     sessionCookieName,
			Value:    permanentSessionExpires + "." + sessionSignature(s.cfg.Web.AdminToken, permanentSessionExpires),
			Path:     "/",
			Expires:  expiresAt,
			MaxAge:   int(permanentSessionTTL.Seconds()),
			HttpOnly: true,
			Secure:   cookieSecure(s.cfg.Web),
			SameSite: http.SameSiteLaxMode,
		}
	}
	expiresAt := time.Now().Add(ttl)
	expires := strconv.FormatInt(expiresAt.Unix(), 10)
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    expires + "." + sessionSignature(s.cfg.Web.AdminToken, expires),
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(ttl.Seconds()),
		HttpOnly: true,
		Secure:   cookieSecure(s.cfg.Web),
		SameSite: http.SameSiteLaxMode,
	}
}

func expiredSessionCookie(cfg WebConfig) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cookieSecure(cfg),
		SameSite: http.SameSiteLaxMode,
	}
}

func sessionSignature(token, expires string) string {
	mac := hmac.New(sha256.New, []byte(strings.TrimSpace(token)))
	_, _ = mac.Write([]byte(expires))
	return hex.EncodeToString(mac.Sum(nil))
}

func cookieSecure(cfg WebConfig) bool {
	return cfg.CookieSecure || (cfg.BehindProxy && strings.HasPrefix(strings.ToLower(strings.TrimSpace(cfg.PublicURL)), "https://"))
}

func formatGroupTitle(group Group) string {
	title := strings.TrimSpace(group.Title)
	if title == "" {
		return fmt.Sprintf("Group %d", group.ID)
	}
	return title
}

func navClass(active, item string) string {
	if active == item {
		return "active"
	}
	return ""
}
