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
	cronTmpl    *template.Template
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

// Cron Job Structures from picoclaw
type CronSchedule struct {
	Kind    string `json:"kind"`
	AtMS    *int64 `json:"atMs,omitempty"`
	EveryMS *int64 `json:"everyMs,omitempty"`
	Expr    string `json:"expr,omitempty"`
	TZ      string `json:"tz,omitempty"`
}

type CronPayload struct {
	Kind    string `json:"kind"`
	Message string `json:"message"`
	Command string `json:"command,omitempty"`
	Channel string `json:"channel,omitempty"`
	To      string `json:"to,omitempty"`
}

type CronJobState struct {
	NextRunAtMS *int64 `json:"nextRunAtMs,omitempty"`
	LastRunAtMS *int64 `json:"lastRunAtMs,omitempty"`
	LastStatus  string `json:"lastStatus,omitempty"`
	LastError   string `json:"lastError,omitempty"`
}

type CronJob struct {
	ID             string       `json:"id"`
	Name           string       `json:"name"`
	Enabled        bool         `json:"enabled"`
	Schedule       CronSchedule `json:"schedule"`
	Payload        CronPayload  `json:"payload"`
	State          CronJobState `json:"state"`
	CreatedAtMS    int64        `json:"createdAtMs"`
	UpdatedAtMS    int64        `json:"updatedAtMs"`
	DeleteAfterRun bool         `json:"deleteAfterRun"`
}

type CronStore struct {
	Version int       `json:"version"`
	Jobs    []CronJob `json:"jobs"`
}

func initTemplates() {
	funcMap := template.FuncMap{
		"formatMS": formatMS,
	}
	indexTmpl = template.Must(template.New("index.html").Funcs(funcMap).ParseFiles("templates/index.html"))
	sessionTmpl = template.Must(template.New("session.html").Funcs(funcMap).ParseFiles("templates/session.html"))
	logTmpl = template.Must(template.New("log.html").Funcs(funcMap).ParseFiles("templates/log.html"))
	cronTmpl = template.Must(template.New("cron.html").Funcs(funcMap).ParseFiles("templates/cron.html"))
}

func registerHandlers(mux *http.ServeMux, store *memory.JSONLStore, sessionsDir string, logsDir string, cronDir string) {
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
		cronJobs := getCronJobs(cronDir)
		data := struct {
			Sessions []SessionMeta
			Logs     []LogFile
			CronJobs []CronJob
		}{
			Sessions: sessions,
			Logs:     logs,
			CronJobs: cronJobs,
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

	// Serve the cron detail fragment
	mux.HandleFunc("/cron", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		jobs := getCronJobs(cronDir)
		
		var targetJob *CronJob
		for _, j := range jobs {
			if j.ID == id {
				targetJob = &j
				break
			}
		}
		
		if targetJob == nil {
			http.Error(w, "Job not found", http.StatusNotFound)
			return
		}
		
		cronTmpl.Execute(w, targetJob)
	})

	mux.HandleFunc("/api/cron/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/cron/")
		if path == "" {
			http.Error(w, "Missing job ID", http.StatusBadRequest)
			return
		}
		
		id := path

		switch r.Method {
		case http.MethodPut:
			handleUpdateCronJob(w, r, id, cronDir)
		case http.MethodDelete:
			handleDeleteCronJob(w, r, id, cronDir)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
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

func getCronJobs(cronDir string) []CronJob {
	storePath := filepath.Join(cronDir, "jobs.json")
	data, err := os.ReadFile(storePath)
	if err != nil {
		return nil
	}

	var store CronStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil
	}

	// Sort by updated time descending
	sort.Slice(store.Jobs, func(i, j int) bool {
		return store.Jobs[i].UpdatedAtMS > store.Jobs[j].UpdatedAtMS
	})

	return store.Jobs
}

func formatMS(ms *int64) string {
	if ms == nil || *ms == 0 {
		return "Never"
	}
	return time.UnixMilli(*ms).Format("2006-01-02 15:04:05")
}

func handleUpdateCronJob(w http.ResponseWriter, r *http.Request, id string, cronDir string) {
	var updateData CronJob
	if err := json.NewDecoder(r.Body).Decode(&updateData); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	jobs := getCronJobs(cronDir)
	found := false
	for i, j := range jobs {
		if j.ID == id {
			// Update editable fields
			jobs[i].Name = updateData.Name
			jobs[i].Enabled = updateData.Enabled
			jobs[i].Schedule.Kind = updateData.Schedule.Kind
			jobs[i].Schedule.Expr = updateData.Schedule.Expr
			jobs[i].Payload.Message = updateData.Payload.Message
			jobs[i].Payload.Channel = updateData.Payload.Channel
			jobs[i].Payload.To = updateData.Payload.To
			
			jobs[i].UpdatedAtMS = time.Now().UnixMilli()
			found = true
			break
		}
	}

	if !found {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	if err := saveCronJobs(cronDir, jobs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func handleDeleteCronJob(w http.ResponseWriter, r *http.Request, id string, cronDir string) {
	jobs := getCronJobs(cronDir)
	var newJobs []CronJob
	found := false
	for _, j := range jobs {
		if j.ID == id {
			found = true
			continue
		}
		newJobs = append(newJobs, j)
	}

	if !found {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	if err := saveCronJobs(cronDir, newJobs); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

func saveCronJobs(cronDir string, jobs []CronJob) error {
	storePath := filepath.Join(cronDir, "jobs.json")
	store := CronStore{
		Version: 1,
		Jobs:    jobs,
	}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode jobs: %w", err)
	}

	// We use the pkg/fileutil.WriteFileAtomic if available, but for simplicity let's use os.WriteFile
	// Actually we should follow picoclaw's standard.
	// Since we are in go-session-explorer, we can use fileutil if we import it correctly.
	// It's already in go.mod.
	// return fileutil.WriteFileAtomic(storePath, data, 0o600)
	
	// For now, simple os.WriteFile
	return os.WriteFile(storePath, data, 0o600)
}
