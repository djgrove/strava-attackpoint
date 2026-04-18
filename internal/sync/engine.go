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
	Status       string // "synced", "skipped", "failed"
	Error        error
	Warning      string
}

// Engine orchestrates the Strava-to-AP sync process.
type Engine struct {
	stravaClient *strava.Client
	apClient     *attackpoint.Client
	syncState    *config.SyncState
	formSchema   *attackpoint.FormSchema
}

// NewEngine creates a sync engine.
func NewEngine(sc *strava.Client, ap *attackpoint.Client, state *config.SyncState) *Engine {
	return &Engine{
		stravaClient: sc,
		apClient:     ap,
		syncState:    state,
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

	var results []Result
	for i, summary := range activities {
		idStr := strconv.FormatInt(summary.ID, 10)
		fmt.Printf("[%d/%d] Syncing \"%s\" (%s)... ", i+1, len(activities), summary.Name, summary.StartDateLocal[:10])

		// Check idempotency.
		if _, exists := e.syncState.Activities[idStr]; exists {
			fmt.Println("skipped (already synced)")
			results = append(results, Result{
				ActivityID:   summary.ID,
				ActivityName: summary.Name,
				Status:       "skipped",
			})
			continue
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
	// Discover AP form.
	schema, err := e.apClient.DiscoverForm()
	if err != nil {
		return nil, fmt.Errorf("discovering AP form: %w", err)
	}
	e.formSchema = schema

	// Fetch full activity details.
	activity, err := e.stravaClient.FetchActivity(activityID)
	if err != nil {
		return nil, fmt.Errorf("fetching activity: %w", err)
	}

	idStr := strconv.FormatInt(activityID, 10)

	// Check if previously synced — if so, update instead of create.
	if synced, exists := e.syncState.Activities[idStr]; exists {
		fmt.Printf("Re-syncing \"%s\" (updating existing AP entry)... ", activity.Name)
		result := e.updateActivity(activity, synced.APEntryURL)
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

	// Not previously synced — create new.
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

func (e *Engine) syncActivity(activity *strava.Activity) Result {
	// Fetch full details if we only have a summary (need description).
	if activity.Description == "" {
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
	}

	workout, warning := mapping.MapActivity(activity, e.formSchema.ActivityTypes)

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

func (e *Engine) updateActivity(activity *strava.Activity, editPath string) Result {
	workout, warning := mapping.MapActivity(activity, e.formSchema.ActivityTypes)

	if editPath == "" {
		// No edit path stored — fall back to creating new.
		return e.syncActivity(activity)
	}

	if err := e.apClient.UpdateWorkout(editPath, e.formSchema, workout); err != nil {
		return Result{
			ActivityID:   activity.ID,
			ActivityName: activity.Name,
			Status:       "failed",
			Error:        err,
			Warning:      warning,
		}
	}

	// Update sync state.
	idStr := strconv.FormatInt(activity.ID, 10)
	e.syncState.Activities[idStr] = config.SyncedActivity{
		APEntryURL:   editPath,
		LastSyncedAt: time.Now(),
	}

	return Result{
		ActivityID:   activity.ID,
		ActivityName: activity.Name,
		Status:       "synced",
		Warning:      warning,
	}
}
