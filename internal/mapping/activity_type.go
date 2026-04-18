package mapping

import (
	"strings"

	"github.com/djgrove/strava-attackpoint/internal/attackpoint"
)

// stravaToAP maps Strava sport_type values to AttackPoint activity type names.
var stravaToAP = map[string]string{
	"Run":               "Running",
	"TrailRun":          "Running",
	"VirtualRun":        "Running",
	"Ride":              "Cycling",
	"MountainBikeRide":  "Cycling",
	"GravelRide":        "Cycling",
	"EBikeRide":         "Cycling",
	"VirtualRide":       "Cycling",
	"Swim":              "Swimming",
	"NordicSki":         "XC Skiing",
	"BackcountrySki":    "XC Skiing",
	"RollerSki":         "XC Skiing",
	"Rowing":            "Rowing",
	"Canoeing":          "Paddling",
	"Kayaking":          "Paddling",
	"StandUpPaddling":   "Paddling",
	"Hike":              "Running",
	"Walk":              "Running",
	"AlpineSki":         "Other",
	"Snowshoe":          "Other",
	"RockClimbing":      "Other",
	"Crossfit":          "Other",
	"WeightTraining":    "Other",
	"Yoga":              "Other",
	"Workout":           "Other",
	"Elliptical":        "Other",
	"StairStepper":      "Other",
	"Handcycle":         "Other",
	"IceSkate":          "Other",
	"InlineSkate":       "Other",
	"Skateboard":        "Other",
	"Snowboard":         "Other",
	"Soccer":            "Other",
	"Surfing":           "Other",
	"Velomobile":        "Other",
	"Wheelchair":        "Other",
	"Windsurf":          "Other",
	"Golf":              "Other",
	"Tennis":            "Other",
	"Badminton":         "Other",
	"Pickleball":        "Other",
	"Squash":            "Other",
	"TableTennis":       "Other",
}

// MapActivityType returns the AP activity type ID for a Strava activity.
// It checks for "orienteering" in the name/description first, then uses the static map.
// Returns the matched AP type name and its form value, plus any warnings.
func MapActivityType(sportType, name, description string, apTypes []attackpoint.SelectOption) (typeID string, typeName string, warning string) {
	// Check for orienteering override.
	nameLower := strings.ToLower(name)
	descLower := strings.ToLower(description)
	targetName := ""
	if strings.Contains(nameLower, "orienteering") || strings.Contains(descLower, "orienteering") {
		targetName = "Orienteering"
	} else if mapped, ok := stravaToAP[sportType]; ok {
		targetName = mapped
	} else {
		targetName = "Other"
		warning = "no mapping for Strava type '" + sportType + "', using 'Other'"
	}

	// Look up the target name in the AP form options.
	for _, opt := range apTypes {
		if strings.EqualFold(opt.Label, targetName) {
			return opt.Value, targetName, warning
		}
	}

	// Target type not found in user's AP account — fall back to "Other".
	if targetName != "Other" {
		warning = "activity type '" + targetName + "' not found in your AttackPoint account — using 'Other'. Add it at https://attackpoint.org/settings.jsp"
	}
	for _, opt := range apTypes {
		if strings.EqualFold(opt.Label, "Other") {
			return opt.Value, "Other", warning
		}
	}

	// Last resort: use first available type.
	if len(apTypes) > 0 {
		return apTypes[0].Value, apTypes[0].Label, "could not find 'Other' type, using '" + apTypes[0].Label + "'"
	}
	return "", "", "no activity types found in AttackPoint form"
}
