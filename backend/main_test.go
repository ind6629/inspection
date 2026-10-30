package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func call(s *Server, method, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	return w
}
func TestIssueLifecycleAndRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issues.json")
	s, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	if w := call(s, "GET", "/health", ""); w.Code != 200 {
		t.Fatal(w.Body)
	}
	w := call(s, "POST", "/api/issues", `{"title":"  Cabinet hot  ","description":"test","priority":"high","position":{"x":1.25,"y":0.8,"z":2.4}}`)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body)
	}
	var issue Issue
	if err = json.Unmarshal(w.Body.Bytes(), &issue); err != nil {
		t.Fatal(err)
	}
	if issue.ID == "" || issue.Status != "open" || issue.Title != "Cabinet hot" || issue.CreatedAt == "" {
		t.Fatal(issue)
	}
	w = call(s, "PATCH", "/api/issues/"+issue.ID+"/status", `{"status":"resolved"}`)
	if w.Code != 200 {
		t.Fatal(w.Body)
	}
	restarted, err := load(path)
	if err != nil {
		t.Fatal(err)
	}
	w = call(restarted, "GET", "/api/issues", "")
	var list []Issue
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 || list[0].Status != "resolved" || list[0].Position.X != 1.25 || list[0].ID != issue.ID {
		t.Fatal(list)
	}
	second := call(s, "POST", "/api/issues", `{"title":"second","priority":"low","position":{"x":0,"y":0,"z":0}}`)
	var another Issue
	json.Unmarshal(second.Body.Bytes(), &another)
	if another.ID == issue.ID || second.Code != 201 {
		t.Fatal(second.Body)
	}
}
func TestInvalidRequests(t *testing.T) {
	s, _ := load(filepath.Join(t.TempDir(), "issues.json"))
	for _, body := range []string{
		`{"title":" ","priority":"low","position":{"x":0,"y":0,"z":0}}`,
		`{"title":"x","priority":"urgent","position":{"x":0,"y":0,"z":0}}`,
		`{"title":"x","priority":"low","position":{"x":0,"y":0}}`,
		`{"title":"x","priority":"low"}`, `{`,
		`{"title":"x","priority":"low","position":{"x":0,"y":0,"z":0}} {}`,
	} {
		if w := call(s, "POST", "/api/issues", body); w.Code != 400 {
			t.Fatalf("%s: %d", body, w.Code)
		}
	}
	if w := call(s, "PATCH", "/api/issues/missing/status", `{"status":"wrong"}`); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if w := call(s, "PATCH", "/api/issues/missing/status", `{"status":"open"}`); w.Code != 404 {
		t.Fatal(w.Code)
	}
	if len(s.issues) != 0 {
		t.Fatal("invalid request saved data")
	}
	r := httptest.NewRequest("OPTIONS", "/api/issues", nil)
	r.Header.Set("Origin", "http://localhost:5173")
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 204 || w.Header().Get("Access-Control-Allow-Origin") == "" {
		t.Fatal(w)
	}
}
func TestSaveFailureAndCorruptData(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "blocked")
	if err := os.WriteFile(path, []byte("file"), 0600); err != nil {
		t.Fatal(err)
	}
	s := &Server{path: filepath.Join(path, "issues.json"), issues: []Issue{}}
	w := call(s, "POST", "/api/issues", `{"title":"x","priority":"low","position":{"x":0,"y":0,"z":0}}`)
	if w.Code != 500 || len(s.issues) != 0 {
		t.Fatal("failed save reported success")
	}
	if _, err := load(path); err == nil {
		t.Fatal("corrupt data silently accepted")
	}
}
