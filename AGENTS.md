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
- **Strava OAuth** uses a Lambda proxy (`infra/`) that holds the client secret. The CLI never sees the secret — it sends auth codes and refresh tokens to the proxy, which injects the secret and forwards to Strava. Tokens are stored in the OS keychain.
- AP credentials are prompted each run (not stored).

## Conventions

- Go module: `github.com/djgrove/strava-attackpoint`
- Use stdlib where possible; external deps: cobra, bubbletea/lipgloss/bubbles, x/net/html, pkg/browser
- Errors are wrapped with context: `fmt.Errorf("doing X: %w", err)`
- Activity type mapping: static table in `internal/mapping/activity_type.go`, validated against AP form options at runtime
- Orienteering override: if activity name/description contains "orienteering" (case-insensitive), type is set to Orienteering regardless of Strava sport type

## Infrastructure

The OAuth proxy lives in `infra/`, managed with Pulumi:
- `infra/index.ts` — Pulumi stack (Lambda + Function URL + IAM)
- `infra/lambda/index.mjs` — Lambda handler (token exchange + refresh)
- Deploy: `cd infra && pulumi up`
- The Lambda Function URL is compiled into the Go binary as `strava.ProxyURL`

## Building

```
go build -o strava-ap .
```

## Testing

```
go test ./...
```
