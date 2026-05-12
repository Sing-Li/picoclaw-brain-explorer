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

	"bufio"
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
	logTmpl     *template.Template
)

type LogFile struct {
	Name      string
	UpdatedAt time.Time
}

type LogEntry struct {
	Raw    string
	Data   map[string]interface{}
	IsJSON bool
}

func initTemplates() {
	indexTmpl = template.Must(template.ParseFiles("templates/index.html"))
	sessionTmpl = template.Must(template.ParseFiles("templates/session.html"))
	logTmpl = template.Must(template.ParseFiles("templates/log.html"))
}

func registerHandlers(mux *http.ServeMux, store *memory.JSONLStore, sessionsDir string, logsDir string) {
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
		logs := getLogsList(logsDir)
		data := struct {
			Sessions []SessionMeta
			Logs     []LogFile
		}{
			Sessions: sessions,
			Logs:     logs,
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

	// Serve the log HTML fragment
	mux.HandleFunc("/log", func(w http.ResponseWriter, r *http.Request) {
		name := r.URL.Query().Get("name")
		if name == "" {
			http.Error(w, "Missing name", http.StatusBadRequest)
			return
		}
		
		logData, err := getLogData(name, logsDir)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		data := struct {
			Name    string
			Entries []LogEntry
		}{
			Name:    name,
			Entries: logData,
		}
		logTmpl.Execute(w, data)
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

func getLogsList(logsDir string) []LogFile {
	files, err := os.ReadDir(logsDir)
	if err != nil {
		return nil
	}

	var logs []LogFile
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".log") {
			info, err := f.Info()
			if err != nil {
				continue
			}
			logs = append(logs, LogFile{
				Name:      f.Name(),
				UpdatedAt: info.ModTime(),
			})
		}
	}

	sort.Slice(logs, func(i, j int) bool {
		return logs[i].UpdatedAt.After(logs[j].UpdatedAt)
	})

	return logs
}

func getLogData(name string, logsDir string) ([]LogEntry, error) {
	logPath := filepath.Join(logsDir, name)
	file, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var entries []LogEntry
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		entry := LogEntry{Raw: line}
		
		// Attempt to parse as JSON
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(line), &data); err == nil {
			entry.Data = data
			entry.IsJSON = true
		}
		
		entries = append(entries, entry)
	}

	return entries, scanner.Err()
}
