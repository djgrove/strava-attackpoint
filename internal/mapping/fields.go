package mapping

import (
	"fmt"

	"github.com/djgrove/strava-attackpoint/internal/attackpoint"
	"github.com/djgrove/strava-attackpoint/internal/strava"
)

// MapActivity converts a Strava activity into AP workout data.
func MapActivity(activity *strava.Activity, apTypes []attackpoint.SelectOption) (*attackpoint.WorkoutData, string) {
	typeID, _, warning := MapActivityType(activity.SportType, activity.Name, activity.Description, apTypes)

	description := buildDescription(activity)

	workout := &attackpoint.WorkoutData{
		ActivityTypeID: typeID,
		Distance:       formatDistance(activity.Distance),
		Duration:       formatDuration(activity.MovingTime),
		Description:    description,
	}

	if activity.HasHeartrate && activity.AverageHeartrate > 0 {
		workout.AverageHR = fmt.Sprintf("%.0f", activity.AverageHeartrate)
	}

	if activity.TotalElevationGain > 0 {
		workout.ElevationGain = fmt.Sprintf("%.0f", activity.TotalElevationGain)
	}

	return workout, warning
}

// formatDistance converts meters to kilometers with 2 decimal places.
func formatDistance(meters float64) string {
	if meters <= 0 {
		return ""
	}
	km := meters / 1000.0
	return fmt.Sprintf("%.2f", km)
}

// formatDuration converts seconds to HH:MM:SS.
func formatDuration(seconds int) string {
	if seconds <= 0 {
		return ""
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	s := seconds % 60
	if h > 0 {
		return fmt.Sprintf("%d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%d:%02d", m, s)
}

func buildDescription(activity *strava.Activity) string {
	var parts []string

	if activity.Name != "" {
		parts = append(parts, activity.Name)
	}
	if activity.Description != "" {
		parts = append(parts, activity.Description)
	}

	link := fmt.Sprintf("https://www.strava.com/activities/%d", activity.ID)
	parts = append(parts, link)

	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "\n\n"
		}
		result += p
	}
	return result
}
