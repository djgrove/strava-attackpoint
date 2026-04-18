# strava-ap

Sync your [Strava](https://www.strava.com) activities to your [AttackPoint.org](https://attackpoint.org) training log.

## Features

- Syncs activities from Strava to AttackPoint automatically
- Maps activity types (Running, Cycling, XC Skiing, etc.)
- Detects orienteering events by name/description
- Idempotent — won't create duplicate entries
- Re-sync support for correcting activities
- Interactive TUI or CLI flags

## Setup

### 1. Create a Strava API Application

1. Go to https://www.strava.com/settings/api
2. Fill in the form (set **Authorization Callback Domain** to `localhost`)
3. Note your **Client ID** and **Client Secret**

### 2. Install

Download the latest binary for your platform from [Releases](https://github.com/djgrove/strava-attackpoint/releases), or build from source:

```bash
go install github.com/djgrove/strava-attackpoint@latest
```

### 3. Configure

Run the setup wizard:

```bash
strava-ap setup
```

This will prompt for your Strava Client ID and Client Secret, then open your browser to authorize the app. Credentials are stored securely in your OS keychain.

## Usage

### Interactive TUI

Just run `strava-ap` with no arguments to launch the interactive terminal UI.

### CLI

```bash
# Sync all activities since a date
strava-ap sync --since 2026-01-01

# Re-sync a specific activity (e.g., after editing on Strava)
strava-ap sync --activity 12345678
```

You'll be prompted for your AttackPoint username and password each time (not stored).

## Activity Type Mapping

| Strava Type | AttackPoint Type |
|---|---|
| Run, TrailRun | Running |
| Ride, MountainBikeRide, GravelRide | Cycling |
| NordicSki, BackcountrySki, RollerSki | XC Skiing |
| Swim | Swimming |
| Rowing | Rowing |
| Kayaking, Canoeing, StandUpPaddling | Paddling |
| Everything else | Other |

If the activity name or description contains "orienteering" (case-insensitive), it will be mapped to **Orienteering** regardless of Strava type.

## Building

```bash
make build    # build for current platform
make test     # run tests
make dist     # cross-compile for all platforms
```
