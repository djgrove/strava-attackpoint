package sync

import (
	"fmt"
	"strconv"
	"time"

	"github.com/djgrove/strava-attackpoint/internal/attackpoint"
	"github.com/djgrove/strava-attackpoint/internal/config"
	"github.com/djgrove/strava-attackpoint/internal/mapping"
	"github.com/djgrove/strava-attackpoint/internal/strava"
)

// Result tracks the outcome of syncing a single activity.
type Result struct {
	ActivityID   int64
	ActivityName string
	Status       string // "synced", "skipped", "updated", "failed"
	Error        error
	Warning      string
}

// Engine orchestrates the Strava-to-AP sync process.
type Engine struct {
	stravaClient    *strava.Client
	apClient        *attackpoint.Client
	syncState       *config.SyncState
	formSchema      *attackpoint.FormSchema
	existingEntries map[string]string // strava activity ID → AP session ID
}

// NewEngine creates a sync engine.
func NewEngine(sc *strava.Client, ap *attackpoint.Client, state *config.SyncState) *Engine {
	return &Engine{
		stravaClient:    sc,
		apClient:        ap,
		syncState:       state,
		existingEntries: make(map[string]string),
	}
}

// SyncSince syncs all activities after the given date.
func (e *Engine) SyncSince(since time.Time) ([]Result, error) {
	// Discover AP form.
	schema, err := e.apClient.DiscoverForm()
	if err != nil {
		return nil, fmt.Errorf("discovering AP form: %w", err)
	}
	e.formSchema = schema

	if len(schema.ActivityTypes) == 0 {
		return nil, fmt.Errorf("no activity types found in AP form — is the login session valid?")
	}

	// Fetch Strava activities.
	activities, err := e.stravaClient.FetchActivitiesSince(since)
	if err != nil {
		return nil, fmt.Errorf("fetching Strava activities: %w", err)
	}

	if len(activities) == 0 {
		return nil, nil
	}

	// Scan AP log for existing entries with Strava URLs.
	if e.apClient.UserID != "" {
		fmt.Print("Scanning AP log for existing entries... ")
		if err := e.scanExistingEntries(since); err != nil {
			fmt.Printf("warning: %v\n", err)
		} else {
			fmt.Printf("found %d synced entries\n", len(e.existingEntries))
		}
	}

	var results []Result
	for i, summary := range activities {
		idStr := strconv.FormatInt(summary.ID, 10)
		fmt.Printf("[%d/%d] Syncing \"%s\" (%s)... ", i+1, len(activities), summary.Name, summary.StartDateLocal[:10])

		// Check sync state first — skip if already synced successfully.
		if _, synced := e.syncState.Activities[idStr]; synced {
			fmt.Println("skipped (already synced)")
			results = append(results, Result{
				ActivityID:   summary.ID,
				ActivityName: summary.Name,
				Status:       "skipped",
			})
			continue
		}

		// Check if entry already exists on AP (has Strava URL in description) but
		// is not in our sync state (e.g., from a previous broken sync).
		if sessionID, exists := e.existingEntries[idStr]; exists {
			fmt.Print("replacing existing... ")
			if err := e.apClient.DeleteSession(sessionID); err != nil {
				fmt.Printf("FAILED to delete: %v\n", err)
				results = append(results, Result{
					ActivityID:   summary.ID,
					ActivityName: summary.Name,
					Status:       "failed",
					Error:        fmt.Errorf("deleting existing entry: %w", err),
				})
				continue
			}
		}

		result := e.syncActivity(&summary)
		results = append(results, result)

		if result.Error != nil {
			fmt.Printf("FAILED: %v\n", result.Error)
		} else {
			msg := "done"
			if result.Warning != "" {
				msg += " (warning: " + result.Warning + ")"
			}
			fmt.Println(msg)
		}
	}

	// Save updated sync state.
	if err := config.SaveSyncState(e.syncState); err != nil {
		return results, fmt.Errorf("saving sync state: %w", err)
	}

	return results, nil
}

// SyncActivity re-syncs a single activity by Strava ID.
func (e *Engine) SyncActivity(activityID int64) (*Result, error) {
	schema, err := e.apClient.DiscoverForm()
	if err != nil {
		return nil, fmt.Errorf("discovering AP form: %w", err)
	}
	e.formSchema = schema

	activity, err := e.stravaClient.FetchActivity(activityID)
	if err != nil {
		return nil, fmt.Errorf("fetching activity: %w", err)
	}

	idStr := strconv.FormatInt(activityID, 10)

	// Scan AP log for existing entry with this Strava URL.
	if e.apClient.UserID != "" {
		startTime, err := activity.StartTime()
		if err == nil {
			_ = e.scanExistingEntries(startTime.AddDate(0, 0, -1))
		}
	}

	// If found on AP, delete it first.
	if sessionID, exists := e.existingEntries[idStr]; exists {
		fmt.Printf("Deleting existing AP entry for \"%s\"... ", activity.Name)
		if err := e.apClient.DeleteSession(sessionID); err != nil {
			return nil, fmt.Errorf("deleting existing entry: %w", err)
		}
		fmt.Println("done")
		delete(e.syncState.Activities, idStr)
	}

	fmt.Printf("Syncing \"%s\"... ", activity.Name)
	result := e.syncActivity(activity)
	if result.Error != nil {
		fmt.Printf("FAILED: %v\n", result.Error)
	} else {
		fmt.Println("done")
	}

	if err := config.SaveSyncState(e.syncState); err != nil {
		return &result, fmt.Errorf("saving sync state: %w", err)
	}
	return &result, nil
}

func (e *Engine) scanExistingEntries(since time.Time) error {
	entries, err := e.apClient.FetchLogEntries(e.apClient.UserID, since, time.Now())
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.StravaID != "" {
			e.existingEntries[entry.StravaID] = entry.SessionID
		}
	}
	return nil
}

// estimateIntensity estimates AP intensity (0-5) from avg/max HR ratio.
// Uses the ratio of average HR to max HR within the activity as a proxy for effort.
func estimateIntensity(avgHR, maxHR float64) int {
	if avgHR <= 0 || maxHR <= 0 {
		return 0
	}
	ratio := avgHR / maxHR
	switch {
	case ratio >= 0.95: // avg very close to max = all-out effort
		return 5
	case ratio >= 0.90:
		return 4
	case ratio >= 0.85:
		return 3
	case ratio >= 0.78:
		return 2
	default:
		return 1
	}
}

func (e *Engine) syncActivity(activity *strava.Activity) Result {
	// Fetch full details (need description field).
	detailed, err := e.stravaClient.FetchActivity(activity.ID)
	if err != nil {
		return Result{
			ActivityID:   activity.ID,
			ActivityName: activity.Name,
			Status:       "failed",
			Error:        fmt.Errorf("fetching activity details: %w", err),
		}
	}
	activity = detailed

	// Determine intensity from HR zones (premium) or avg/max HR ratio (fallback).
	dominantZone := 0
	if activity.HasHeartrate {
		zone, err := e.stravaClient.FetchActivityZones(activity.ID)
		if err == nil && zone > 0 {
			dominantZone = zone
		} else {
			dominantZone = estimateIntensity(activity.AverageHeartrate, activity.MaxHeartrate)
		}
	}

	workout, warning := mapping.MapActivity(activity, e.formSchema.ActivityTypes, dominantZone)

	if err := e.apClient.SubmitWorkout(e.formSchema, workout); err != nil {
		return Result{
			ActivityID:   activity.ID,
			ActivityName: activity.Name,
			Status:       "failed",
			Error:        err,
			Warning:      warning,
		}
	}

	// Record in sync state.
	idStr := strconv.FormatInt(activity.ID, 10)
	e.syncState.Activities[idStr] = config.SyncedActivity{
		LastSyncedAt: time.Now(),
	}

	return Result{
		ActivityID:   activity.ID,
		ActivityName: activity.Name,
		Status:       "synced",
		Warning:      warning,
	}
}
