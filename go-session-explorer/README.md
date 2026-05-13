# Go Native PicoClaw Session Explorer

A standalone Go web application to view and edit sessions, logs, and cron jobs created by PicoClaw. It uses `picoclaw` core packages for robust data management and native Go templates (`html/template`) for the UI, completely eliminating the need for Node.js or a separate frontend build process.

## Features
- **Session Management:** List, search, view, edit, and delete PicoClaw sessions.
- **Log Viewer:** Browse all log files in the `logs` directory.
    - **Smart JSON Parsing:** Automatically formats JSON log lines for readability.
    - **Live View:** Auto-refresh toggle to watch logs in real-time.
- **Cron Job Manager:** View and manage scheduled tasks from `jobs.json`.
    - **Detailed View:** See schedule, payload, and execution status.
    - **Editing:** Modify job name, enabled status, schedule, and payload.
    - **Deletion:** Remove jobs from the scheduler.
- **Zero Dependencies:** No Node.js, npm, or React required. Just standard Go.

## Prerequisites
- Go 1.20+

## Setup and Running

1. **Start the Go Application:**

The application serves both the API and the HTML templates. Run it from the `go-session-explorer` directory.

```bash
cd go-session-explorer
go run main.go handlers.go
```

The application will start on http://localhost:3001. Open this URL in your browser.

## Configuration
By default, the application looks for data in `~/.picoclaw`. 
You can override this by setting the `PICOCLAW_HOME` environment variable before running the Go app.

```bash
export PICOCLAW_HOME=/path/to/your/picoclaw/data
go run main.go handlers.go
```

## Docker Support
You can also run the explorer as a Docker container:

```bash
docker build -t go-session-explorer -f go-session-explorer/Dockerfile .
docker run -p 3001:3001 -v ~/.picoclaw:/data go-session-explorer
```
