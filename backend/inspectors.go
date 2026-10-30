package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Reporter struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}
type inspectorAccount struct {
	Reporter
	PasswordHash string `json:"passwordHash"`
}
type inspectorSession struct {
	User    Reporter
	Expires time.Time
}

// All access is serialized by Server.mu; tokens are memory-only and never persisted.
type inspectorAuth struct {
	users    []inspectorAccount
	sessions map[string]inspectorSession
	attempts int
	window   time.Time
}

func newInspectorAuth(dir string) (*inspectorAuth, error) {
	a := &inspectorAuth{sessions: map[string]inspectorSession{}}
	path := filepath.Join(dir, "inspectors.json")
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		credentials := "本机巡检演示账号（请勿上传或公开此文件）\n"
		for i := 1; i <= 2; i++ {
			raw := make([]byte, 12)
			if _, err = rand.Read(raw); err != nil {
				return nil, err
			}
			password := hex.EncodeToString(raw)
			hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return nil, err
			}
			user := Reporter{fmt.Sprintf("inspector-%02d", i), fmt.Sprintf("inspector%02d", i), fmt.Sprintf("巡检员%d", i)}
			a.users = append(a.users, inspectorAccount{user, string(hash)})
			credentials += fmt.Sprintf("%s（%s）\n密码：%s\n\n", user.Username, user.DisplayName, password)
		}
		if err = os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}
		// Write credentials first so a partially failed first setup cannot lose passwords.
		if err = os.WriteFile(filepath.Join(dir, "inspector-credentials.txt"), []byte(credentials), 0600); err != nil {
			return nil, err
		}
		b, err = json.MarshalIndent(a.users, "", "  ")
		if err != nil {
			return nil, err
		}
		if err = os.WriteFile(path, b, 0600); err != nil {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	} else if err = json.Unmarshal(b, &a.users); err != nil {
		return nil, fmt.Errorf("invalid inspectors file: %w", err)
	}
	seen := map[string]bool{}
	ids := map[string]bool{}
	for _, u := range a.users {
		if u.ID == "" || u.Username == "" || u.DisplayName == "" || seen[u.Username] || ids[u.ID] {
			return nil, fmt.Errorf("invalid or duplicate inspector identity")
		}
		if _, err = bcrypt.Cost([]byte(u.PasswordHash)); err != nil {
			return nil, fmt.Errorf("invalid inspector password hash")
		}
		seen[u.Username] = true
		ids[u.ID] = true
	}
	fmt.Printf("Inspector account file: %s; local credentials: %s (not printed)\n", path, filepath.Join(dir, "inspector-credentials.txt"))
	return a, nil
}
func (a *inspectorAuth) require(w http.ResponseWriter, r *http.Request) *Reporter {
	w.Header().Set("Cache-Control", "no-store")
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	session, ok := a.sessions[token]
	if !ok || !time.Now().Before(session.Expires) {
		delete(a.sessions, token)
		fail(w, 401, "巡检员登录已失效，请重新登录")
		return nil
	}
	user := session.User
	return &user
}
func (a *inspectorAuth) handle(w http.ResponseWriter, r *http.Request) bool {
	switch r.URL.Path {
	case "/api/inspector/login":
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != "POST" {
			fail(w, 405, "method not allowed")
			return true
		}
		var in struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := decode(w, r, &in); err != nil {
			fail(w, 400, "请输入账号和密码")
			return true
		}
		now := time.Now()
		if now.Sub(a.window) > time.Minute {
			a.window = now
			a.attempts = 0
		}
		if a.attempts >= 10 {
			fail(w, 429, "尝试次数过多，请一分钟后重试")
			return true
		}
		for _, u := range a.users {
			if u.Username == in.Username && bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in.Password)) == nil {
				raw := make([]byte, 32)
				if _, err := rand.Read(raw); err != nil {
					fail(w, 500, "登录失败，请重试")
					return true
				}
				token := hex.EncodeToString(raw)
				expiry := now.Add(8 * time.Hour)
				for key, value := range a.sessions {
					if !now.Before(value.Expires) {
						delete(a.sessions, key)
					}
				}
				a.sessions[token] = inspectorSession{u.Reporter, expiry}
				a.attempts = 0
				reply(w, 200, map[string]any{"token": token, "user": u.Reporter, "expiresAt": expiry.UTC().Format(time.RFC3339)})
				return true
			}
		}
		a.attempts++
		fail(w, 401, "账号或密码错误")
		return true
	case "/api/inspector/me":
		if r.Method != "GET" {
			fail(w, 405, "method not allowed")
			return true
		}
		if user := a.require(w, r); user != nil {
			reply(w, 200, user)
		}
		return true
	case "/api/inspector/logout":
		if r.Method != "POST" {
			fail(w, 405, "method not allowed")
			return true
		}
		delete(a.sessions, strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "))
		reply(w, 200, map[string]bool{"ok": true})
		return true
	}
	return false
}
