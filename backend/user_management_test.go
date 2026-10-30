package main

import (
	"crypto/sha256"
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAdminUserManagement(t *testing.T) {
	dir := t.TempDir()
	s, _ := load(filepath.Join(dir, "issues.json"))
	s.auth = &adminAuth{username: "admin", passwordHash: sha256.Sum256([]byte("test-admin-password")), sessions: map[string]time.Time{}}
	s.inspectors = &inspectorAuth{path: filepath.Join(dir, "inspectors.json"), sessions: map[string]inspectorSession{}}
	login := authCall(s, "POST", "/api/auth/login", `{"username":"admin","password":"test-admin-password"}`, nil, "")
	cookie := login.Result().Cookies()[0]
	create := `{"username":"managed_user","displayName":"原姓名","password":"test-user-password"}`
	if w := authCall(s, "POST", "/api/admin/users", create, nil, ""); w.Code != 401 {
		t.Fatal("anonymous admin operation allowed")
	}
	w := authCall(s, "POST", "/api/admin/users", create, cookie, "")
	if w.Code != 201 {
		t.Fatal(w.Body)
	}
	var user managedUser
	json.Unmarshal(w.Body.Bytes(), &user)
	id := user.ID
	if strings.Contains(w.Body.String(), "password") || id == "" || user.Disabled {
		t.Fatal("unsafe user response")
	}
	if w = authCall(s, "POST", "/api/admin/users", create, cookie, ""); w.Code != 409 {
		t.Fatal("duplicate accepted")
	}
	loginUser := func(password string) string {
		w := inspectorCall(s, "POST", "/api/inspector/login", `{"username":"managed_user","password":"`+password+`"}`, "")
		if w.Code != 200 {
			t.Fatal(w.Body)
		}
		var data struct {
			Token string `json:"token"`
		}
		json.Unmarshal(w.Body.Bytes(), &data)
		return data.Token
	}
	token := loginUser("test-user-password")
	if w = inspectorCall(s, "GET", "/api/admin/users", "", token); w.Code != 401 {
		t.Fatal("inspector accessed management")
	}
	inspectorCall(s, "POST", "/api/issues", `{"title":"history","priority":"low","position":{"x":0,"y":0,"z":0}}`, token)
	base := "/api/admin/users/" + id
	w = authCall(s, "PATCH", base, `{"displayName":"新姓名"}`, cookie, "")
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	if s.issues[0].Reporter.DisplayName != "原姓名" || s.inspectors.users[0].DisplayName != "新姓名" {
		t.Fatal("historical reporter changed")
	}
	if w = inspectorCall(s, "GET", "/api/issues", "", token); w.Code != 401 {
		t.Fatal("edit did not revoke login")
	}
	token = loginUser("test-user-password")
	w = authCall(s, "PATCH", base, `{"disabled":true}`, cookie, "")
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	if w = inspectorCall(s, "GET", "/api/issues", "", token); w.Code != 401 {
		t.Fatal("disabled session still active")
	}
	if w = inspectorCall(s, "POST", "/api/inspector/login", `{"username":"managed_user","password":"test-user-password"}`, ""); w.Code != 403 {
		t.Fatal("disabled login succeeded")
	}
	restored, err := newInspectorAuth(dir)
	if err != nil || !restored.users[0].Disabled {
		t.Fatal("disabled state not persistent", err)
	}
	authCall(s, "PATCH", base, `{"disabled":false}`, cookie, "")
	token = loginUser("test-user-password")
	w = authCall(s, "POST", base+"/password", `{"password":"replacement-test-password"}`, cookie, "")
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	if w = inspectorCall(s, "GET", "/api/issues", "", token); w.Code != 401 {
		t.Fatal("reset did not revoke login")
	}
	if w = inspectorCall(s, "POST", "/api/inspector/login", `{"username":"managed_user","password":"test-user-password"}`, ""); w.Code != 401 {
		t.Fatal("old password still valid")
	}
	loginUser("replacement-test-password")
	for _, body := range []string{`{}`, `{"displayName":" "}`, `{"username":"rename"}`, `{"role":"admin"}`} {
		if w = authCall(s, "PATCH", base, body, cookie, ""); w.Code != 400 {
			t.Fatal(w.Code, w.Body)
		}
	}
	if w = authCall(s, "POST", base+"/password", `{"password":"short"}`, cookie, ""); w.Code != 400 {
		t.Fatal("short password accepted")
	}
	if w = authCall(s, "PATCH", base, `{"disabled":true}`, cookie, "https://untrusted.invalid"); w.Code != 403 {
		t.Fatal("origin check missing")
	}
	if w = authCall(s, "PATCH", "/api/admin/users/missing", `{"disabled":true}`, cookie, ""); w.Code != 404 {
		t.Fatal(w.Code)
	}
	w = authCall(s, "GET", "/api/admin/users", "", cookie, "")
	var list []managedUser
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 || list[0].IssueCount != 1 || strings.Contains(w.Body.String(), "passwordHash") {
		t.Fatal("user list incorrect")
	}
	token = loginUser("replacement-test-password")
	s.inspectors.path = filepath.Join(dir, "missing", "users.json")
	w = authCall(s, "PATCH", base, `{"disabled":true}`, cookie, "")
	if w.Code != 500 || s.inspectors.users[0].Disabled {
		t.Fatal("failed save applied mutation")
	}
	if w = inspectorCall(s, "GET", "/api/issues", "", token); w.Code != http.StatusOK {
		t.Fatal("failed save revoked session")
	}
}
