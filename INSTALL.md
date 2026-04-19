# Installation

## Option 1: Download a Pre-built Binary

Download the latest binary for your platform from the [Releases](https://github.com/djgrove/strava-attackpoint/releases) page.

| Platform | File |
|---|---|
| macOS (Apple Silicon) | `strava-ap-darwin-arm64` |
| macOS (Intel) | `strava-ap-darwin-amd64` |
| Linux (x86_64) | `strava-ap-linux-amd64` |
| Windows (x86_64) | `strava-ap-windows-amd64.exe` |

After downloading, make it executable (macOS/Linux):

```bash
chmod +x strava-ap-darwin-arm64
mv strava-ap-darwin-arm64 /usr/local/bin/strava-ap
```

## Option 2: Build from Source

Requires [Go 1.25+](https://go.dev/dl/).

```bash
git clone https://github.com/djgrove/strava-attackpoint.git
cd strava-attackpoint
go build -o strava-ap .
```

Or install directly:

```bash
go install github.com/djgrove/strava-attackpoint@latest
```

## Setup

### 1. Connect to Strava

```bash
strava-ap setup
```

This opens your browser to authorize the app with Strava. No API keys or secrets needed — everything is handled automatically. Your OAuth tokens are stored securely in your OS keychain (macOS Keychain, Windows Credential Manager, or Linux secret-service).

### 2. Sync Activities

```bash
# Sync everything since a date
strava-ap sync --since 2025-01-01
```

You'll be prompted for your AttackPoint username and password. These are used to log in to AP for that session only and are never stored.

### 3. (Optional) Launch the TUI

```bash
strava-ap
```

Running with no arguments opens an interactive terminal UI with menus for setup and sync.

## Troubleshooting

### "Strava not configured"

Run `strava-ap setup` to authorize with Strava.

### "login failed — check your username and password"

Double-check your AttackPoint credentials. The username and password are the same ones you use to log in at attackpoint.org.

### Activity type warnings

If you see warnings like "no AP type matching Strava type 'X'", your AttackPoint account doesn't have a matching activity type configured. Add the type at https://attackpoint.org/settings.jsp.

### Token expired

Tokens refresh automatically. If you see persistent auth errors, run `strava-ap setup` again to re-authorize.
