package main

import (
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDeviceAndAreaPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issues.json")
	s, _ := load(path)
	payload := `{"title":"设备异常","priority":"high","position":{"x":0,"y":1,"z":2},"deviceId":"cab-002","area":"机房A区"}`
	w := call(s, "POST", "/api/issues", payload)
	if w.Code != 201 {
		t.Fatal(w.Body)
	}
	restored, err := load(path)
	if err != nil || restored.issues[0].DeviceID != "CAB-002" || restored.issues[0].Area != "机房A区" {
		t.Fatal("metadata lost", err)
	}
	for _, body := range []string{strings.Replace(payload, "cab-002", "CAB-999", 1), strings.Replace(payload, "机房A区", "机房B区", 1), strings.Replace(payload, `"deviceId":"cab-002"`, `"deviceId":""`, 1)} {
		response := call(s, "POST", "/api/issues", body)
		if strings.Contains(body, `"deviceId":""`) {
			if response.Code != 201 {
				t.Fatal("surface location rejected")
			}
		} else if response.Code != 400 {
			t.Fatal("invalid equipment accepted")
		}
	}
	w = call(s, "POST", "/api/issues", strings.Replace(payload, `"area":"机房A区"`, `"area":""`, 1))
	var issue Issue
	json.Unmarshal(w.Body.Bytes(), &issue)
	if w.Code != 201 || issue.Area != "机房A区" {
		t.Fatal("catalog area not filled")
	}
}
func TestInspectorOwnPasswordChange(t *testing.T) {
	dir := t.TempDir()
	s, _ := load(filepath.Join(dir, "issues.json"))
	hash, _ := bcrypt.GenerateFromPassword([]byte("original-test-password"), bcrypt.MinCost)
	user := Reporter{"user-self", "self_user", "测试用户"}
	s.inspectors = &inspectorAuth{users: []inspectorAccount{{Reporter: user, PasswordHash: string(hash)}}, sessions: map[string]inspectorSession{"self-token": {user, time.Now().Add(time.Hour)}, "other-device": {user, time.Now().Add(time.Hour)}}, path: filepath.Join(dir, "inspectors.json")}
	path := "/api/inspector/password"
	valid := `{"currentPassword":"original-test-password","newPassword":"replacement-test-password"}`
	if w := inspectorCall(s, "POST", path, valid, ""); w.Code != 401 {
		t.Fatal("anonymous password change")
	}
	for _, body := range []string{strings.Replace(valid, "original-test-password", "wrong-password", 1), strings.Replace(valid, "replacement-test-password", "short", 1), strings.Replace(valid, "replacement-test-password", "original-test-password", 1), strings.TrimSuffix(valid, "}") + `,"userId":"other-user"}`} {
		if w := inspectorCall(s, "POST", path, body, "self-token"); w.Code != 400 {
			t.Fatal(w.Code, w.Body)
		}
	}
	if w := inspectorCall(s, "GET", "/api/inspector/me", "", "self-token"); w.Code != 200 {
		t.Fatal("wrong old password revoked session")
	}
	w := inspectorCall(s, "POST", path, valid, "self-token")
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	for _, token := range []string{"self-token", "other-device"} {
		if w = inspectorCall(s, "GET", "/api/inspector/me", "", token); w.Code != 401 {
			t.Fatal("password change did not revoke all sessions")
		}
	}
	a, err := newInspectorAuth(dir)
	if err != nil {
		t.Fatal(err)
	}
	s.inspectors = a
	if w = inspectorCall(s, "POST", "/api/inspector/login", `{"username":"self_user","password":"original-test-password"}`, ""); w.Code != 401 {
		t.Fatal("old password accepted")
	}
	w = inspectorCall(s, "POST", "/api/inspector/login", `{"username":"self_user","password":"replacement-test-password"}`, "")
	if w.Code != 200 {
		t.Fatal("new password failed", w.Body)
	}
	var result struct {
		Token string `json:"token"`
	}
	json.Unmarshal(w.Body.Bytes(), &result)
	a.path = filepath.Join(dir, "missing", "inspectors.json")
	w = inspectorCall(s, "POST", path, `{"currentPassword":"replacement-test-password","newPassword":"another-test-password"}`, result.Token)
	if w.Code != 500 {
		t.Fatal(w.Body)
	}
	if w = inspectorCall(s, "GET", "/api/inspector/me", "", result.Token); w.Code != 200 {
		t.Fatal("save failure revoked session")
	}
	if bcrypt.CompareHashAndPassword([]byte(a.users[0].PasswordHash), []byte("replacement-test-password")) != nil {
		t.Fatal("save failure mutated password")
	}
}
