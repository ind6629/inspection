package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Opt-in: exercise the real Windows Unity player against an isolated HTTP server.
// Uses disposable test credentials, never reads local operator passwords.
func TestUnityPlayerIntegration(t *testing.T) {
	runUnityIntegration(t, false, false)
}
func TestUnityPlayerRegistration(t *testing.T)   { runUnityIntegration(t, true, false) }
func TestUnityPlayerPasswordChange(t *testing.T) { runUnityIntegration(t, false, true) }
func runUnityIntegration(t *testing.T, registration bool, passwordChange bool) {
	executable := os.Getenv("INSPECTION_UNITY_EXE")
	if executable == "" {
		t.Skip("set INSPECTION_UNITY_EXE to enable player integration")
	}
	path := filepath.Join(t.TempDir(), "issues.json")
	s, _ := load(path)
	hash, _ := bcrypt.GenerateFromPassword([]byte("unity-test-password"), bcrypt.MinCost)
	s.inspectors = &inspectorAuth{users: []inspectorAccount{{Reporter{"unity-test-user", "unity-test", "测试巡检员"}, string(hash), false}}, sessions: map[string]inspectorSession{}}
	s.inspectors.path = filepath.Join(filepath.Dir(path), "inspectors.json")
	username := "unity-test"
	registrationFlag := "0"
	if registration {
		username = "unity_test"
		registrationFlag = "1"
		s.inspectors.users = nil
		s.inspectors.path = filepath.Join(filepath.Dir(path), "inspectors.json")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("Unity HTTP: %s %s", r.Method, r.URL.Path)
		s.ServeHTTP(w, r)
	}))
	defer server.Close()
	logPath := filepath.Join(t.TempDir(), "unity.log")
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, executable, "-batchmode", "-nographics", "-inspection-smoke", "-logFile", logPath)
	previewPrefix := os.Getenv("INSPECTION_UNITY_PREVIEW_PREFIX")
	if passwordChange && previewPrefix != "" {
		command = exec.CommandContext(ctx, executable, "-screen-fullscreen", "0", "-screen-width", "1280", "-screen-height", "800", "-inspection-smoke", "-logFile", logPath)
	}
	localURL := strings.Replace(server.URL, "127.0.0.1", "localhost", 1)
	command.Env = append(os.Environ(), "INSPECTION_SMOKE_USER="+username, "INSPECTION_SMOKE_REGISTER="+registrationFlag, "INSPECTION_SMOKE_PASSWORD=unity-test-password", "INSPECTION_SMOKE_BACKEND="+localURL, "NO_PROXY=localhost,127.0.0.1", "no_proxy=localhost,127.0.0.1", "HTTP_PROXY=", "HTTPS_PROXY=", "ALL_PROXY=")
	if passwordChange {
		command.Env = append(command.Env, "INSPECTION_SMOKE_CHANGE_PASSWORD=1")
		if previewPrefix != "" {
			command.Env = append(command.Env, "INSPECTION_CAPTURE_PREFIX="+previewPrefix)
		}
	} else {
		command.Env = append(command.Env, "INSPECTION_SMOKE_CHANGE_PASSWORD=0")
	}
	err := command.Run()
	logBytes, _ := os.ReadFile(logPath)
	if err != nil || !strings.Contains(string(logBytes), "INSPECTION_SMOKE_PASS") {
		t.Fatalf("Unity integration failed: %v\n%s", err, logBytes)
	}
	restored, err := load(path)
	if err != nil || len(restored.issues) != 1 || restored.issues[0].Reporter == nil || restored.issues[0].Reporter.Username != username {
		t.Fatal("Unity reporter not persisted", err)
	}
	if restored.issues[0].DeviceID != "CAB-002" || restored.issues[0].Area != "机房A区" {
		t.Fatal("Unity hit metadata not persisted", restored.issues[0])
	}
	if passwordChange && !strings.Contains(string(logBytes), "INSPECTION_PASSWORD_PASS") {
		t.Fatal("Unity password flow not verified")
	}
}

func TestInspectorRegistration(t *testing.T) {
	dir := t.TempDir()
	s, _ := load(filepath.Join(dir, "issues.json"))
	s.inspectors = &inspectorAuth{path: filepath.Join(dir, "inspectors.json"), sessions: map[string]inspectorSession{}}
	request := `{"username":" New_User ","displayName":" 新巡检员 ","password":"test-only-password"}`
	w := inspectorCall(s, "POST", "/api/inspector/register", request, "")
	if w.Code != 201 || strings.Contains(w.Body.String(), "password") {
		t.Fatal(w.Code, w.Body)
	}
	var user Reporter
	json.Unmarshal(w.Body.Bytes(), &user)
	if user.ID == "" || user.Username != "new_user" || user.DisplayName != "新巡检员" {
		t.Fatal(user)
	}
	if w = inspectorCall(s, "POST", "/api/inspector/register", request, ""); w.Code != 409 {
		t.Fatal("duplicate account accepted")
	}
	for _, body := range []string{`{"username":"ab","displayName":"名字","password":"long-password"}`, `{"username":"admin","displayName":"名字","password":"long-password"}`, `{"username":"valid","displayName":" ","password":"long-password"}`, `{"username":"valid","displayName":"名字","password":"short"}`, `{"username":"valid","displayName":"名字","password":"long-password","role":"admin"}`} {
		if w = inspectorCall(s, "POST", "/api/inspector/register", body, ""); w.Code != 400 {
			t.Fatal(w.Code, w.Body)
		}
	}
	persisted, err := newInspectorAuth(dir)
	if err != nil || len(persisted.users) != 1 || persisted.users[0].ID != user.ID {
		t.Fatal("account not durable", err)
	}
	if bcrypt.CompareHashAndPassword([]byte(persisted.users[0].PasswordHash), []byte("test-only-password")) != nil {
		t.Fatal("password not hashed correctly")
	}
	s.inspectors = persisted
	w = inspectorCall(s, "POST", "/api/inspector/login", `{"username":"NEW_USER","password":"test-only-password"}`, "")
	if w.Code != 200 {
		t.Fatal("registered account cannot login after restart", w.Body)
	}
	s.inspectors.path = filepath.Join(dir, "missing", "inspectors.json")
	before := len(s.inspectors.users)
	w = inspectorCall(s, "POST", "/api/inspector/register", `{"username":"save_failure","displayName":"测试用户","password":"test-only-password"}`, "")
	if w.Code != 500 || len(s.inspectors.users) != before {
		t.Fatal("save failure changed account list")
	}
}

func inspectorCall(s *Server, method, path, body, token string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func TestInspectorAttributionIsolationAndPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issues.json")
	s, _ := load(path)
	hash, _ := bcrypt.GenerateFromPassword([]byte("test-user-password"), bcrypt.MinCost)
	a := Reporter{"u-a", "inspector-a", "巡检员甲"}
	b := Reporter{"u-b", "inspector-b", "巡检员乙"}
	s.inspectors = &inspectorAuth{users: []inspectorAccount{{a, string(hash), false}, {b, string(hash), false}}, sessions: map[string]inspectorSession{}}
	s.auth = &adminAuth{username: "admin", passwordHash: sha256.Sum256([]byte("test-admin-password")), sessions: map[string]time.Time{}}
	login := func(username string) string {
		w := inspectorCall(s, "POST", "/api/inspector/login", `{"username":"`+username+`","password":"test-user-password"}`, "")
		if w.Code != 200 {
			t.Fatal(w.Body)
		}
		var result struct {
			Token string   `json:"token"`
			User  Reporter `json:"user"`
		}
		json.Unmarshal(w.Body.Bytes(), &result)
		if result.Token == "" || result.User.Username != username {
			t.Fatal("bad login")
		}
		return result.Token
	}
	tokenA := login(a.Username)
	tokenB := login(b.Username)
	payload := `{"title":"inspection","priority":"high","position":{"x":1,"y":2,"z":3}}`
	if w := inspectorCall(s, "POST", "/api/issues", payload, ""); w.Code != 401 {
		t.Fatal("anonymous create allowed")
	}
	if w := inspectorCall(s, "GET", "/api/issues", "", ""); w.Code != 401 {
		t.Fatal("anonymous list allowed")
	}
	if w := inspectorCall(s, "POST", "/api/issues", strings.TrimSuffix(payload, "}")+`,"reporter":{"id":"u-b"}}`, tokenA); w.Code != 400 {
		t.Fatal("forged reporter accepted")
	}
	for _, token := range []string{tokenA, tokenB} {
		if w := inspectorCall(s, "POST", "/api/issues", payload, token); w.Code != 201 {
			t.Fatal(w.Body)
		}
	}
	if s.issues[0].Reporter.ID != a.ID || s.issues[1].Reporter.ID != b.ID {
		t.Fatal("wrong reporter")
	}
	for token, id := range map[string]string{tokenA: a.ID, tokenB: b.ID} {
		w := inspectorCall(s, "GET", "/api/issues", "", token)
		var list []Issue
		json.Unmarshal(w.Body.Bytes(), &list)
		if w.Code != 200 || len(list) != 1 || list[0].Reporter.ID != id {
			t.Fatal("user data isolation failed")
		}
	}
	if w := inspectorCall(s, "PATCH", "/api/issues/"+s.issues[0].ID+"/status", `{"status":"resolved"}`, tokenA); w.Code != 401 {
		t.Fatal("inspector can change status")
	}
	restored, err := load(path)
	if err != nil || restored.issues[0].Reporter.ID != a.ID || restored.issues[1].Reporter.DisplayName != b.DisplayName {
		t.Fatal("reporter not persistent", err)
	}
	w := authCall(s, "POST", "/api/auth/login", `{"username":"admin","password":"test-admin-password"}`, nil, "")
	cookie := w.Result().Cookies()[0]
	for _, route := range []string{"/api/admin/issues", "/api/admin/inspectors"} {
		w = authCall(s, "GET", route, "", cookie, "")
		if w.Code != 200 || strings.Contains(w.Body.String(), "passwordHash") {
			t.Fatal("admin view incorrect")
		}
	}
	w = authCall(s, "GET", "/api/admin/issues", "", &http.Cookie{Name: "inspection_session", Value: "invalid"}, "")
	if w.Code != 401 {
		t.Fatal("invalid admin cookie accepted")
	}
	inspectorCall(s, "POST", "/api/inspector/logout", "", tokenA)
	if w := inspectorCall(s, "GET", "/api/issues", "", tokenA); w.Code != 401 {
		t.Fatal("logout ineffective")
	}
	session := s.inspectors.sessions[tokenB]
	session.Expires = time.Now().Add(-time.Second)
	s.inspectors.sessions[tokenB] = session
	if w := inspectorCall(s, "GET", "/api/issues", "", tokenB); w.Code != 401 {
		t.Fatal("expired inspector accepted")
	}
}
func TestInspectorAccountsSurviveRestart(t *testing.T) {
	dir := t.TempDir()
	first, err := newInspectorAuth(dir)
	if err != nil {
		t.Fatal(err)
	}
	second, err := newInspectorAuth(dir)
	if err != nil || len(second.users) != 2 || first.users[0].ID != second.users[0].ID || first.users[0].PasswordHash != second.users[0].PasswordHash {
		t.Fatal("accounts not stable", err)
	}
}
