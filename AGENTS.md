# strava-ap

A Go CLI tool that syncs Strava activities to AttackPoint.org training log entries.

## Architecture

- **CLI + TUI frontends** over shared internal packages
- `cmd/` — Cobra commands (setup, sync) and TUI launcher
- `internal/strava/` — Strava OAuth2 and API client
- `internal/attackpoint/` — AP session login, HTML form parsing, form submission
- `internal/mapping/` — Strava-to-AP field and activity type mapping
- `internal/sync/` — Orchestration engine
- `internal/tui/` — Bubbletea interactive UI
- `internal/config/` — Config and sync state persistence

## Key Concepts

- **AttackPoint has no API.** We log in via HTTP POST, parse HTML forms at runtime to discover field names, and submit workouts via form POST. This "form discovery" approach makes us resilient to AP changing field names.
- **Idempotency** is tracked via a local `sync_state.json` file mapping Strava activity IDs to AP entries.
- **Strava OAuth tokens** are stored locally and auto-refreshed. AP credentials are prompted each run.

## Conventions

- Go module: `github.com/djgrove/strava-attackpoint`
- Use stdlib where possible; external deps: cobra, bubbletea/lipgloss/bubbles, x/net/html, pkg/browser
- Errors are wrapped with context: `fmt.Errorf("doing X: %w", err)`
- Activity type mapping: static table in `internal/mapping/activity_type.go`, validated against AP form options at runtime
- Orienteering override: if activity name/description contains "orienteering" (case-insensitive), type is set to Orienteering regardless of Strava sport type

## Building

```
go build -o strava-ap .
```

## Testing

```
go test ./...
```
