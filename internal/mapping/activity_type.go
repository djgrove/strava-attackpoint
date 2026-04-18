package mapping

import (
	"strings"

	"github.com/djgrove/strava-attackpoint/internal/attackpoint"
)

// stravaToAPKeywords maps Strava sport_type to keywords that match AP activity type labels.
// We match if the AP label contains any of the keywords (case-insensitive).
var stravaToAPKeywords = map[string][]string{
	"Run":              {"run"},
	"TrailRun":         {"run"},
	"VirtualRun":       {"run"},
	"Ride":             {"bike", "cycl"},
	"MountainBikeRide": {"bike", "cycl", "mtb"},
	"GravelRide":       {"bike", "cycl"},
	"EBikeRide":        {"bike", "cycl"},
	"VirtualRide":      {"bike", "cycl"},
	"Swim":             {"swim"},
	"NordicSki":        {"ski"},
	"BackcountrySki":   {"ski"},
	"RollerSki":        {"ski"},
	"Rowing":           {"row"},
	"Canoeing":         {"paddl", "canoe", "kayak"},
	"Kayaking":         {"paddl", "kayak", "canoe"},
	"StandUpPaddling":  {"paddl"},
	"Hike":             {"hik"},
	"Walk":             {"walk", "hik"},
	"WeightTraining":   {"weight", "strength"},
	"Crossfit":         {"cross-training", "crossfit"},
	"Yoga":             {"stretch", "yoga"},
	"Workout":          {"cross-training", "core"},
}

// MapActivityType returns the AP activity type ID for a Strava activity.
// It checks for "orienteering" in the name/description first, then uses keyword matching.
func MapActivityType(sportType, name, description string, apTypes []attackpoint.SelectOption) (typeID string, typeName string, warning string) {
	// Filter out the "New Type" placeholder.
	var validTypes []attackpoint.SelectOption
	for _, opt := range apTypes {
		if opt.Value != "-1" {
			validTypes = append(validTypes, opt)
		}
	}

	// Check for orienteering override.
	nameLower := strings.ToLower(name)
	descLower := strings.ToLower(description)
	if strings.Contains(nameLower, "orienteering") || strings.Contains(descLower, "orienteering") {
		if id, label := findByKeyword(validTypes, "orient"); id != "" {
			return id, label, ""
		}
		warning = "activity looks like orienteering but no matching type in your AP account — add 'Orienteering' at https://attackpoint.org/settings.jsp"
	}

	// Try keyword matching from the Strava type.
	if keywords, ok := stravaToAPKeywords[sportType]; ok {
		for _, kw := range keywords {
			if id, label := findByKeyword(validTypes, kw); id != "" {
				return id, label, ""
			}
		}
		warning = "no AP type matching Strava type '" + sportType + "'"
	} else {
		warning = "no mapping for Strava type '" + sportType + "'"
	}

	// Fall back to first available type and warn.
	if len(validTypes) > 0 {
		return validTypes[0].Value, validTypes[0].Label, warning + " — using '" + validTypes[0].Label + "'"
	}
	return "", "", "no activity types found in AttackPoint form"
}

// findByKeyword returns the first AP type whose label contains the keyword.
func findByKeyword(types []attackpoint.SelectOption, keyword string) (string, string) {
	kw := strings.ToLower(keyword)
	for _, opt := range types {
		if strings.Contains(strings.ToLower(opt.Label), kw) {
			return opt.Value, opt.Label
		}
	}
	return "", ""
}
