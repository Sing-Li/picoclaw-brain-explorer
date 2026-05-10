package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"html/template"

	"github.com/sipeed/picoclaw/pkg/memory"
	"github.com/sipeed/picoclaw/pkg/providers"
)

type SessionMeta struct {
	Key       string          `json:"key"`
	Summary   string          `json:"summary"`
	Skip      int             `json:"skip"`
	Count     int             `json:"count"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
	Scope     json.RawMessage `json:"scope,omitempty"`
	Aliases   []string        `json:"aliases,omitempty"`
}

type Session struct {
	Meta     SessionMeta         `json:"meta"`
	Messages []providers.Message `json:"messages"`
}

var (
	indexTmpl   *template.Template
	sessionTmpl *template.Template
)

func initTemplates() {
	indexTmpl = template.Must(template.ParseFiles("templates/index.html"))
	sessionTmpl = template.Must(template.ParseFiles("templates/session.html"))
}

func registerHandlers(mux *http.ServeMux, store *memory.JSONLStore, sessionsDir string) {
	initTemplates()

	// Serve static files
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Serve the main UI
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		
		sessions := getSessionsList(sessionsDir)
		data := struct {
			Sessions []SessionMeta
		}{
			Sessions: sessions,
		}
		indexTmpl.Execute(w, data)
	})

	// Serve the session HTML fragment
	mux.HandleFunc("/session", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		if key == "" {
			http.Error(w, "Missing key", http.StatusBadRequest)
			return
		}
		
		session, err := getSessionData(store, key, sessionsDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sessionTmpl.Execute(w, session)
	})

	mux.HandleFunc("/api/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		sessions := getSessionsList(sessionsDir)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sessions)
	})

	// Since net/http doesn't have path parameters easily until Go 1.22, we manually parse
	mux.HandleFunc("/api/sessions/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/sessions/")
		if path == "" {
			http.Error(w, "Missing session key", http.StatusBadRequest)
			return
		}
		
		key := path

		switch r.Method {
		case http.MethodGet:
			handleGetSession(w, r, store, key, sessionsDir)
		case http.MethodPut:
			handleUpdateSession(w, r, store, key, sessionsDir)
		case http.MethodDelete:
			handleDeleteSession(w, r, store, key, sessionsDir)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

// sanitizeKey converts a session key to a safe filename component.
func sanitizeKey(key string) string {
	s := strings.ReplaceAll(key, ":", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	return s
}

func getSessionsList(sessionsDir string) []SessionMeta {
	files, err := os.ReadDir(sessionsDir)
	if err != nil {
		return nil
	}

	var sessions []SessionMeta
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".meta.json") {
			metaPath := filepath.Join(sessionsDir, f.Name())
			data, err := os.ReadFile(metaPath)
			if err != nil {
				continue
			}
			var meta SessionMeta
			if err := json.Unmarshal(data, &meta); err == nil {
				sessions = append(sessions, meta)
			}
		}
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})

	return sessions
}

func getSessionData(store *memory.JSONLStore, key string, sessionsDir string) (*Session, error) {
	metaPath := filepath.Join(sessionsDir, sanitizeKey(key)+".meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("Session not found")
		}
		return nil, fmt.Errorf("Failed to read session meta")
	}

	var meta SessionMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, fmt.Errorf("Failed to parse session meta")
	}

	history, err := store.GetHistory(context.Background(), key)
	if err != nil {
		return nil, fmt.Errorf("Failed to get history")
	}

	return &Session{
		Meta:     meta,
		Messages: history,
	}, nil
}

func handleGetSession(w http.ResponseWriter, r *http.Request, store *memory.JSONLStore, key string, sessionsDir string) {
	session, err := getSessionData(store, key, sessionsDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func handleUpdateSession(w http.ResponseWriter, r *http.Request, store *memory.JSONLStore, key string, sessionsDir string) {
	var session Session
	if err := json.NewDecoder(r.Body).Decode(&session); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// In the Go backend, updating a session involves setting its history and summary using the store
	ctx := context.Background()
	
	// Set history overwrites the existing jsonl and handles the meta.skip logic internally if needed,
	// but the JSONLStore implementation of SetHistory in picoclaw:
	// Let's rely on store.SetHistory and store.SetSummary to manage files correctly.
	err := store.SetHistory(ctx, key, session.Messages)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update history: %v", err), http.StatusInternalServerError)
		return
	}

	err = store.SetSummary(ctx, key, session.Meta.Summary)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to update summary: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func handleDeleteSession(w http.ResponseWriter, r *http.Request, store *memory.JSONLStore, key string, sessionsDir string) {
	// The memory.Store interface doesn't have a direct Delete method, so we delete the files
	metaPath := filepath.Join(sessionsDir, sanitizeKey(key)+".meta.json")
	jsonlPath := filepath.Join(sessionsDir, sanitizeKey(key)+".jsonl")

	os.Remove(metaPath)
	os.Remove(jsonlPath)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
