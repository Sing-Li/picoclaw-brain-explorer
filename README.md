# PicoClaw Brain Explorer 
### (standalone native golang debugging/learning tool)

Stateful long running agents framework are still crafted art rather than coding science.  One often find the agents doing surprising things or locking up or aborting for apparently no reasons.   It really helps when you can take a look "inside" picoclaw's brain to see what exactly is going on at any time.  This tool aims to do exactly that.  First revision gives you real-time viewi and access into the short term memory within the sessions.

<img width="1328" height="815" alt="Screenshot 2026-05-09 at 8 53 19 PM" src="https://github.com/user-attachments/assets/6726c3da-5545-4250-b4c8-09ad42b3fb2c" />


This is a standalone Go web application to view and edit sessions created by PicoClaw. It uses `picoclaw` core packages for robust session management and native Go templates (`html/template`) for the UI, completely eliminating the need for Node.js or a separate frontend build process.

## Features
- List all PicoClaw sessions.
- View detailed message history for each session.
- Edit message content.
- Delete individual messages.
- Delete entire sessions.
- **Zero dependencies:** No Node.js, npm, or React required. Just standard Go.

## Prerequisites
- Go 1.20+

## Setup and Running

0. Start with your git clone of the picoclaw project, copy the `go-session.explorer` directory to the top level. 

1. **Start the Go Application:**

The application serves both the API and the HTML templates. Run it from the `go-session-explorer` directory.

```bash
cd go-session-explorer
go run main.go handlers.go
```

The application will start on http://localhost:3001. Open this URL in your browser.

## Configuration
By default, the application looks for sessions in `~/.picoclaw/workspace/sessions`. 
You can override this by setting the `PICOCLAW_HOME` environment variable before running the Go app.

```bash
export PICOCLAW_HOME=/path/to/your/picoclaw/data
go run main.go handlers.go
```

## Roadmap
- Add media explorer/manager
- Add cron explorer/editor
- Add memory.md explorer/editor
- Add heartbeat.md explorer/editor
- Add soul.md explorer/editor
- Add skills explorer/creator/editor
- Add mcp explorer/manager
- Add semantics layer that "knows" how "this brain works": a "live" bi-directional cross referenced overlay between logged events (including LLM queries) and all the above pieces that are observable 


