package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHomeReturnsLoginHTML(t *testing.T) {
	srv, err := newServer(Config{Web: WebConfig{AdminToken: "secret", SessionTTL: time.Hour}, Service: fakeService{}})
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}

	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("unexpected status: %d", rec.Code)
	}
	if body := rec.Body.String(); !strings.Contains(body, "Laya Panel") || !strings.Contains(body, `name="token"`) {
		t.Fatalf("expected login html, body=%s", body)
	}
}

func TestLoginSetsSessionAndShowsPanel(t *testing.T) {
	srv, err := newServer(Config{
		Web:     WebConfig{AdminToken: "secret", SessionTTL: time.Hour},
		Service: fakeService{},
	})
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}

	login := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader("token=secret"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	srv.routes().ServeHTTP(login, req)
	if login.Code != http.StatusFound {
		t.Fatalf("unexpected login status: %d", login.Code)
	}
	cookies := login.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie")
	}

	panel := httptest.NewRecorder()
	panelReq := httptest.NewRequest(http.MethodGet, "/", nil)
	panelReq.AddCookie(cookies[0])
	srv.routes().ServeHTTP(panel, panelReq)
	if body := panel.Body.String(); !strings.Contains(body, "Known users") || !strings.Contains(body, "@laya") {
		t.Fatalf("expected panel html, body=%s", body)
	}
}

func TestPermanentSessionCookie(t *testing.T) {
	srv, err := newServer(Config{
		Web:     WebConfig{AdminToken: "secret", SessionTTL: -1},
		Service: fakeService{},
	})
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}

	cookie := srv.sessionCookie()
	if !strings.HasPrefix(cookie.Value, permanentSessionExpires+".") {
		t.Fatalf("cookie value = %q, want permanent marker", cookie.Value)
	}
	if cookie.MaxAge < int((9 * 365 * 24 * time.Hour).Seconds()) {
		t.Fatalf("permanent cookie MaxAge = %d, want about 10 years", cookie.MaxAge)
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(cookie)
	if !srv.authenticated(req) {
		t.Fatal("permanent session cookie should authenticate")
	}
}

func TestAuthenticatedPagesRender(t *testing.T) {
	srv, err := newServer(Config{
		Web:     WebConfig{AdminToken: "secret", SessionTTL: time.Hour},
		Service: fakeService{},
	})
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	cookie := srv.sessionCookie()
	for _, path := range []string{"/supergroups", "/rehab", "/users", "/moderation", "/warnings", "/karma", "/sleep", "/laya", "/laya/users", "/ai-logs", "/content", "/faq", "/quotes", "/photos", "/settings"} {
		t.Run(path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.AddCookie(cookie)
			srv.routes().ServeHTTP(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("unexpected status for %s: %d body=%s", path, rec.Code, rec.Body.String())
			}
			if !strings.Contains(rec.Body.String(), "Laya Panel") {
				t.Fatalf("expected shell html for %s", path)
			}
		})
	}
}

func TestProtectedPageRedirectsToLogin(t *testing.T) {
	srv, err := newServer(Config{Web: WebConfig{AdminToken: "secret", SessionTTL: time.Hour}, Service: fakeService{}})
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}

	rec := httptest.NewRecorder()
	srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/users", nil))
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/login" {
		t.Fatalf("expected login redirect, status=%d location=%q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestMafiaMiniAppServesBuiltFrontend(t *testing.T) {
	dist := t.TempDir()
	if err := os.WriteFile(filepath.Join(dist, "index.html"), []byte(`<div id="root">mafia</div>`), 0o600); err != nil {
		t.Fatalf("write index: %v", err)
	}
	assetsDir := filepath.Join(dist, "assets")
	if err := os.Mkdir(assetsDir, 0o700); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "app.js"), []byte(`console.log("mafia")`), 0o600); err != nil {
		t.Fatalf("write asset: %v", err)
	}

	srv, err := newServer(Config{Web: WebConfig{AdminToken: "secret", SessionTTL: time.Hour, MafiaMiniAppDir: dist}, Service: fakeService{}})
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}
	for _, tc := range []struct {
		path string
		want string
	}{
		{path: "/mafia-mini-app/", want: `<div id="root">mafia</div>`},
		{path: "/mafia-mini-app/assets/app.js", want: `console.log("mafia")`},
		{path: "/mafia-mini-app/room/MF-1", want: `<div id="root">mafia</div>`},
	} {
		t.Run(tc.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			srv.routes().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tc.path, nil))
			if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), tc.want) {
				t.Fatalf("unexpected response for %s: status=%d body=%s", tc.path, rec.Code, rec.Body.String())
			}
		})
	}
}

func TestPostInvalidInputRedirectsWithError(t *testing.T) {
	srv, err := newServer(Config{Web: WebConfig{AdminToken: "secret", SessionTTL: time.Hour}, Service: fakeService{}})
	if err != nil {
		t.Fatalf("newServer: %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/rehab/add", strings.NewReader("chat_id=-1001"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(srv.sessionCookie())
	srv.routes().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound || !strings.Contains(rec.Header().Get("Location"), "err=") {
		t.Fatalf("expected redirect with error, status=%d location=%q", rec.Code, rec.Header().Get("Location"))
	}
}
