package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sipeed/picoclaw/pkg/memory"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	picoHome := os.Getenv("PICOCLAW_HOME")
	if picoHome == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			log.Fatalf("Failed to get user home dir: %v", err)
		}
		picoHome = filepath.Join(homeDir, ".picoclaw")
	}

	sessionsDir := filepath.Join(picoHome, "workspace", "sessions")
	logsDir := filepath.Join(picoHome, "logs")
	cronDir := filepath.Join(picoHome, "workspace", "cron")
	log.Printf("Using sessions directory: %s", sessionsDir)
	log.Printf("Using logs directory: %s", logsDir)
	log.Printf("Using cron directory: %s", cronDir)

	// Ensure directories exist
	os.MkdirAll(logsDir, 0o755)
	os.MkdirAll(cronDir, 0o755)

	store, err := memory.NewJSONLStore(sessionsDir)
	if err != nil {
		log.Fatalf("Failed to initialize session store: %v", err)
	}

	mux := http.NewServeMux()
	registerHandlers(mux, store, sessionsDir, logsDir, cronDir)

	// Wrap mux with CORS middleware
	handler := corsMiddleware(mux)

	log.Printf("Go Session Explorer listening at http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// Simple CORS middleware for development
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
