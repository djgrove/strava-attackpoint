# strava-ap

Sync your [Strava](https://www.strava.com) activities to your [AttackPoint.org](https://attackpoint.org) training log.

## Features

- One-command sync of Strava activities to AttackPoint
- Automatic activity type mapping (Run, Bike, Swim, Orienteering, etc.)
- Detects orienteering events by name/description
- Heart rate, distance, elevation, and duration synced automatically
- Intensity estimated from heart rate data
- Idempotent — won't create duplicate entries
- Re-sync support for correcting activities after edits
- Interactive TUI or CLI flags
- No Strava API credentials needed — OAuth handled securely via proxy

## Quick Start

See [INSTALL.md](INSTALL.md) for detailed installation instructions.

```bash
# 1. Download the binary for your platform from Releases
# 2. Authorize with Strava
strava-ap setup

# 3. Sync your activities
strava-ap sync --since 2026-01-01
```

## Usage

### Interactive TUI

Run `strava-ap` with no arguments to launch the interactive terminal UI with menus for setup and sync.

### CLI

```bash
# Sync all activities since a date
strava-ap sync --since 2026-01-01

# Re-sync a specific activity (e.g., after editing title on Strava)
strava-ap sync --activity 12345678
```

You'll be prompted for your AttackPoint username and password each time (credentials are never stored).

## Activity Type Mapping

Activity types are matched by keyword against your AttackPoint account's configured types:

| Strava Type | Matches AP Types Containing |
|---|---|
| Run, TrailRun, VirtualRun | "run" |
| Ride, MountainBikeRide, GravelRide | "bike", "cycl" |
| NordicSki, BackcountrySki, RollerSki | "ski" |
| Swim | "swim" |
| Hike | "hik" |
| Rowing | "row" |
| Kayaking, Canoeing, StandUpPaddling | "paddl" |
| WeightTraining | "weight", "strength" |

If the activity name or description contains **"orienteering"** (case-insensitive), it maps to your Orienteering type regardless of Strava sport type.

Types not found in your AP account fall back to the first available type with a warning.

## How It Works

1. **Strava**: Fetches activities via the Strava API (OAuth tokens managed automatically)
2. **Mapping**: Converts activity type, distance (miles), duration, HR, elevation, and intensity
3. **AttackPoint**: Logs in via HTTP, discovers form fields at runtime, and submits workouts
4. **Idempotency**: Tracks synced activities locally; scans AP log for existing entries by Strava URL

AttackPoint has no API, so strava-ap parses HTML forms at runtime. This makes it resilient to AP changing field names.

## Building from Source

```bash
go build -o strava-ap .
make dist     # cross-compile for all platforms
```

## License

MIT
