package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Accessed under Server.mu. Sessions deliberately expire on server restart.
type adminAuth struct {
	username      string
	passwordHash  [32]byte
	sessions      map[string]time.Time
	attempts      int
	attemptWindow time.Time
}

func newAdminAuth(passwordPath string) (*adminAuth, error) {
	username := os.Getenv("ADMIN_USERNAME")
	if username == "" {
		username = "admin"
	}
	password := os.Getenv("ADMIN_PASSWORD")
	if password == "" {
		b, err := os.ReadFile(passwordPath)
		if os.IsNotExist(err) {
			raw := make([]byte, 18)
			if _, err = rand.Read(raw); err != nil {
				return nil, err
			}
			password = hex.EncodeToString(raw)
			if err = os.MkdirAll(filepath.Dir(passwordPath), 0700); err != nil {
				return nil, err
			}
			if err = os.WriteFile(passwordPath, []byte(password+"\n"), 0600); err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		} else {
			password = strings.TrimSpace(string(b))
		}
		fmt.Printf("Admin username: %s; password file: %s (not printed)\n", username, passwordPath)
	}
	if len(password) < 12 {
		return nil, fmt.Errorf("admin password must contain at least 12 characters")
	}
	return &adminAuth{username: username, passwordHash: sha256.Sum256([]byte(password)), sessions: map[string]time.Time{}}, nil
}
func allowedOrigin(r *http.Request) bool {
	o := r.Header.Get("Origin")
	return o == "" || o == "http://localhost:5173" || o == "http://127.0.0.1:5173"
}
func (a *adminAuth) loggedIn(r *http.Request) bool {
	cookie, err := r.Cookie("inspection_session")
	if err != nil {
		return false
	}
	expiry, ok := a.sessions[cookie.Value]
	if !ok {
		return false
	}
	if !time.Now().Before(expiry) {
		delete(a.sessions, cookie.Value)
		return false
	}
	return true
}
func (a *adminAuth) require(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Cache-Control", "no-store")
	if !allowedOrigin(r) {
		fail(w, 403, "请求来源不允许")
		return false
	}
	if !a.loggedIn(r) {
		fail(w, 401, "登录已失效，请重新登录")
		return false
	}
	return true
}
func (a *adminAuth) handle(w http.ResponseWriter, r *http.Request, s *Server) bool {
	switch r.URL.Path {
	case "/api/auth/login":
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != "POST" {
			fail(w, 405, "method not allowed")
			return true
		}
		if !allowedOrigin(r) {
			fail(w, 403, "请求来源不允许")
			return true
		}
		var in struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := decode(w, r, &in); err != nil {
			fail(w, 400, "请输入正确的账号和密码")
			return true
		}
		now := time.Now()
		if now.Sub(a.attemptWindow) > time.Minute {
			a.attemptWindow = now
			a.attempts = 0
		}
		if a.attempts >= 10 {
			w.Header().Set("Retry-After", "60")
			fail(w, 429, "尝试次数过多，请一分钟后重试")
			return true
		}
		hash := sha256.Sum256([]byte(in.Password))
		if in.Username != a.username || subtle.ConstantTimeCompare(hash[:], a.passwordHash[:]) != 1 {
			a.attempts++
			fail(w, 401, "账号或密码错误")
			return true
		}
		a.attempts = 0
		for token, expiry := range a.sessions {
			if !now.Before(expiry) {
				delete(a.sessions, token)
			}
		}
		if old, err := r.Cookie("inspection_session"); err == nil {
			delete(a.sessions, old.Value)
		}
		raw := make([]byte, 32)
		if _, err := rand.Read(raw); err != nil {
			fail(w, 500, "无法创建登录会话")
			return true
		}
		token := hex.EncodeToString(raw)
		expiry := now.Add(8 * time.Hour)
		a.sessions[token] = expiry
		http.SetCookie(w, &http.Cookie{Name: "inspection_session", Value: token, Path: "/api", HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode, MaxAge: 28800, Expires: expiry})
		reply(w, 200, map[string]string{"username": a.username})
		return true
	case "/api/auth/me":
		if r.Method != "GET" {
			fail(w, 405, "method not allowed")
			return true
		}
		if a.require(w, r) {
			reply(w, 200, map[string]string{"username": a.username})
		}
		return true
	case "/api/auth/logout":
		if r.Method != "POST" {
			fail(w, 405, "method not allowed")
			return true
		}
		if !allowedOrigin(r) {
			fail(w, 403, "请求来源不允许")
			return true
		}
		if cookie, err := r.Cookie("inspection_session"); err == nil {
			delete(a.sessions, cookie.Value)
		}
		http.SetCookie(w, &http.Cookie{Name: "inspection_session", Value: "", Path: "/api", HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode, MaxAge: -1})
		reply(w, 200, map[string]bool{"ok": true})
		return true
	case "/api/admin/issues":
		if r.Method != "GET" {
			fail(w, 405, "method not allowed")
			return true
		}
		if a.require(w, r) {
			reply(w, 200, s.issues)
		}
		return true
	}
	return false
}
