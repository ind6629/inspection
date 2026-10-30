package main

import (
	"crypto/sha256"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func authCall(s *Server, method, path, body string, cookie *http.Cookie, origin string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if cookie != nil {
		r.AddCookie(cookie)
	}
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func TestAdminSessionLifecycle(t *testing.T) {
	s, _ := load(filepath.Join(t.TempDir(), "issues.json"))
	s.auth = &adminAuth{username: "admin", passwordHash: sha256.Sum256([]byte("test-only-password")), sessions: map[string]time.Time{}}
	if w := authCall(s, "GET", "/api/admin/issues", "", nil, ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := authCall(s, "PATCH", "/api/issues/missing/status", `{"status":"open"}`, nil, ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
	if w := authCall(s, "GET", "/api/issues", "", nil, ""); w.Code != 200 {
		t.Fatal("Unity read broken")
	}
	if w := authCall(s, "POST", "/api/issues", `{"title":"Unity","priority":"low","position":{"x":0,"y":0,"z":0}}`, nil, ""); w.Code != 201 {
		t.Fatal("Unity upload broken", w.Body)
	}
	if w := authCall(s, "POST", "/api/auth/login", `{"username":"admin","password":"incorrect"}`, nil, ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
	w := authCall(s, "POST", "/api/auth/login", `{"username":"admin","password":"test-only-password"}`, nil, "http://localhost:5173")
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	cookies := w.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatal("no session cookie")
	}
	cookie := cookies[0]
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteStrictMode || cookie.MaxAge != 28800 {
		t.Fatal("cookie options incorrect")
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("credential CORS missing")
	}
	for _, path := range []string{"/api/auth/me", "/api/admin/issues"} {
		if w = authCall(s, "GET", path, "", cookie, "http://localhost:5173"); w.Code != 200 {
			t.Fatal(w.Body)
		}
	}
	if w = authCall(s, "PATCH", "/api/issues/"+s.issues[0].ID+"/status", `{"status":"resolved"}`, cookie, "http://localhost:5173"); w.Code != 200 || s.issues[0].Status != "resolved" {
		t.Fatal(w.Body)
	}
	if w = authCall(s, "PATCH", "/api/issues/"+s.issues[0].ID+"/status", `{"status":"open"}`, cookie, "https://untrusted.invalid"); w.Code != 403 {
		t.Fatal("cross-origin mutation allowed")
	}
	if w = authCall(s, "POST", "/api/auth/logout", "", cookie, ""); w.Code != 200 {
		t.Fatal(w.Body)
	}
	if w = authCall(s, "GET", "/api/auth/me", "", cookie, ""); w.Code != 401 {
		t.Fatal("logout did not revoke session")
	}
	s.auth.sessions[cookie.Value] = time.Now().Add(-time.Second)
	if w = authCall(s, "GET", "/api/auth/me", "", cookie, ""); w.Code != 401 {
		t.Fatal("expired session accepted")
	}
}
func TestPasswordConfigurationAndLoginLimit(t *testing.T) {
	t.Setenv("ADMIN_USERNAME", "")
	t.Setenv("ADMIN_PASSWORD", "")
	path := filepath.Join(t.TempDir(), "admin-password.txt")
	first, err := newAdminAuth(path)
	if err != nil {
		t.Fatal(err)
	}
	second, err := newAdminAuth(path)
	if err != nil || first.passwordHash != second.passwordHash {
		t.Fatal("generated password not reusable")
	}
	s := &Server{auth: first}
	for i := 0; i < 10; i++ {
		if w := authCall(s, "POST", "/api/auth/login", `{"username":"admin","password":"wrong"}`, nil, ""); w.Code != 401 {
			t.Fatal(w.Code)
		}
	}
	if w := authCall(s, "POST", "/api/auth/login", `{"username":"admin","password":"wrong"}`, nil, ""); w.Code != 429 {
		t.Fatal("missing login limit")
	}
	t.Setenv("ADMIN_PASSWORD", "short")
	if _, err = newAdminAuth(path); err == nil {
		t.Fatal("weak password accepted")
	}
}
