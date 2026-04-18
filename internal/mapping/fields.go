package mapping

import (
	"fmt"
	"strconv"
	"time"

	"github.com/djgrove/strava-attackpoint/internal/attackpoint"
	"github.com/djgrove/strava-attackpoint/internal/strava"
)

// MapActivity converts a Strava activity into AP workout data.
func MapActivity(activity *strava.Activity, apTypes []attackpoint.SelectOption) (*attackpoint.WorkoutData, string) {
	typeID, _, warning := MapActivityType(activity.SportType, activity.Name, activity.Description, apTypes)

	description := buildDescription(activity)

	// Parse activity date.
	startTime, err := activity.StartTime()
	if err != nil {
		startTime = time.Now()
	}

	workout := &attackpoint.WorkoutData{
		ActivityTypeID: typeID,
		Day:            fmt.Sprintf("%02d", startTime.Day()),
		Month:          fmt.Sprintf("%02d", int(startTime.Month())),
		Year:           strconv.Itoa(startTime.Year()),
		StartHour:      strconv.Itoa(startTime.Hour()),
		Distance:       formatDistanceMiles(activity.Distance),
		DistanceUnits:  "miles",
		Duration:       formatDuration(activity.MovingTime),
		Description:    description,
	}

	if activity.HasHeartrate && activity.AverageHeartrate > 0 {
		workout.AverageHR = fmt.Sprintf("%.0f", activity.AverageHeartrate)
	}

	if activity.HasHeartrate && activity.MaxHeartrate > 0 {
		workout.MaxHR = fmt.Sprintf("%.0f", activity.MaxHeartrate)
	}

	if activity.TotalElevationGain > 0 {
		workout.ElevationGain = fmt.Sprintf("%.0f", activity.TotalElevationGain)
	}

	return workout, warning
}

// formatDistanceMiles converts meters to miles with 2 decimal places.
func formatDistanceMiles(meters float64) string {
	if meters <= 0 {
		return ""
	}
	miles := meters / 1609.344
	return fmt.Sprintf("%.2f", miles)
}

// formatDuration converts seconds to H:MM format (what AP expects).
func formatDuration(seconds int) string {
	if seconds <= 0 {
		return ""
	}
	h := seconds / 3600
	m := (seconds % 3600) / 60
	return fmt.Sprintf("%d:%02d", h, m)
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
