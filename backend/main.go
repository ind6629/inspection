package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}
type Issue struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Priority    string   `json:"priority"`
	Status      string   `json:"status"`
	Position    Position `json:"position"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}
type Server struct {
	mu     sync.Mutex
	path   string
	issues []Issue
}

func load(path string) (*Server, error) {
	s := &Server{path: path, issues: []Issue{}}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err = json.Unmarshal(b, &s.issues); err != nil {
		return nil, fmt.Errorf("invalid data file: %w", err)
	}
	if s.issues == nil {
		s.issues = []Issue{}
	}
	return s, nil
}
func (s *Server) save(next []Issue) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.path), "issues-*.tmp")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(name, s.path); err != nil {
		return err
	}
	s.issues = next
	return nil
}
func reply(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}
func fail(w http.ResponseWriter, code int, msg string) {
	reply(w, code, map[string]any{"error": map[string]string{"message": msg}})
}
func decode(w http.ResponseWriter, r *http.Request, v any) error {
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	d.DisallowUnknownFields()
	if err := d.Decode(v); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return errors.New("expected one JSON object")
	}
	return nil
}
func valid(s string, values ...string) bool {
	for _, v := range values {
		if s == v {
			return true
		}
	}
	return false
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin == "http://localhost:5173" || origin == "http://127.0.0.1:5173" {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
	}
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	if r.Method == "OPTIONS" {
		w.WriteHeader(204)
		return
	}
	if r.URL.Path == "/health" && r.Method == "GET" {
		reply(w, 200, map[string]string{"status": "ok"})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if r.URL.Path == "/api/issues" {
		switch r.Method {
		case "GET":
			reply(w, 200, s.issues)
		case "POST":
			var in struct {
				Title       string `json:"title"`
				Description string `json:"description"`
				Priority    string `json:"priority"`
				Position    *struct {
					X *float64 `json:"x"`
					Y *float64 `json:"y"`
					Z *float64 `json:"z"`
				} `json:"position"`
			}
			if err := decode(w, r, &in); err != nil {
				fail(w, 400, "invalid JSON: "+err.Error())
				return
			}
			in.Title = strings.TrimSpace(in.Title)
			if in.Title == "" {
				fail(w, 400, "title is required")
				return
			}
			if !valid(in.Priority, "low", "medium", "high") {
				fail(w, 400, "priority must be low, medium or high")
				return
			}
			if in.Position == nil || in.Position.X == nil || in.Position.Y == nil || in.Position.Z == nil {
				fail(w, 400, "position.x/y/z are required")
				return
			}
			p := Position{*in.Position.X, *in.Position.Y, *in.Position.Z}
			for _, v := range []float64{p.X, p.Y, p.Z} {
				if math.IsNaN(v) || math.IsInf(v, 0) {
					fail(w, 400, "position must be finite")
					return
				}
			}
			id := make([]byte, 16)
			if _, err := rand.Read(id); err != nil {
				fail(w, 500, "ID generation failed")
				return
			}
			now := time.Now().UTC().Format(time.RFC3339Nano)
			issue := Issue{hex.EncodeToString(id), in.Title, in.Description, in.Priority, "open", p, now, now}
			next := append(append([]Issue{}, s.issues...), issue)
			if err := s.save(next); err != nil {
				log.Print(err)
				fail(w, 500, "could not persist issue")
				return
			}
			reply(w, 201, issue)
		default:
			fail(w, 405, "method not allowed")
		}
		return
	}
	if strings.HasPrefix(r.URL.Path, "/api/issues/") && strings.HasSuffix(r.URL.Path, "/status") {
		if r.Method != "PATCH" {
			fail(w, 405, "method not allowed")
			return
		}
		id := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/api/issues/"), "/status")
		var in struct {
			Status string `json:"status"`
		}
		if err := decode(w, r, &in); err != nil {
			fail(w, 400, "invalid JSON")
			return
		}
		if !valid(in.Status, "open", "in_progress", "resolved") {
			fail(w, 400, "status must be open, in_progress or resolved")
			return
		}
		for i, v := range s.issues {
			if v.ID == id {
				next := append([]Issue{}, s.issues...)
				next[i].Status = in.Status
				next[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339Nano)
				if err := s.save(next); err != nil {
					fail(w, 500, "could not persist status")
					return
				}
				reply(w, 200, next[i])
				return
			}
		}
		fail(w, 404, "issue not found")
		return
	}
	fail(w, 404, "route not found")
}
func main() {
	path := os.Getenv("DATA_FILE")
	if path == "" {
		path = "data/issues.json"
	}
	s, err := load(path)
	if err != nil {
		log.Fatal(err)
	}
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = "127.0.0.1:8080"
	}
	log.Printf("Listening on http://%s; data: %s", addr, path)
	log.Fatal((&http.Server{Addr: addr, Handler: s, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 15 * time.Second}).ListenAndServe())
}
