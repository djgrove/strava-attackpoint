package attackpoint

import (
	"fmt"
	"io"
	"net/url"
	"strings"
)

// WorkoutData holds the mapped values to submit to AP.
type WorkoutData struct {
	ActivityTypeID string
	Day            string // "01"-"31"
	Month          string // "01"-"12"
	Year           string // "2026"
	StartHour      string // "0"-"23" or "-1" for unset
	Distance       string
	DistanceUnits  string // "kilometers" or "miles"
	Duration       string // HH:MM:SS or similar
	AverageHR      string
	MaxHR          string
	ElevationGain  string
	Description    string
}

// SubmitWorkout creates a new training entry on AttackPoint.
func (c *Client) SubmitWorkout(schema *FormSchema, workout *WorkoutData) error {
	if schema.Action == "" {
		return fmt.Errorf("no form action URL discovered")
	}

	data := buildFormData(schema, workout)

	resp, err := c.PostForm(schema.Action, data)
	if err != nil {
		return fmt.Errorf("submitting workout: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		preview := string(body)
		if len(preview) > 500 {
			preview = preview[:500]
		}
		return fmt.Errorf("workout submission failed with status %d: %s", resp.StatusCode, preview)
	}

	return nil
}

// UpdateWorkout edits an existing training entry.
func (c *Client) UpdateWorkout(editPath string, schema *FormSchema, workout *WorkoutData) error {
	resp, err := c.Get(editPath)
	if err != nil {
		return fmt.Errorf("fetching edit form: %w", err)
	}
	defer resp.Body.Close()

	editSchema, err := ParseForm(resp.Body)
	if err != nil {
		return fmt.Errorf("parsing edit form: %w", err)
	}

	action := editSchema.Action
	if action == "" {
		action = schema.Action
	}

	data := buildFormData(editSchema, workout)

	postResp, err := c.PostForm(action, data)
	if err != nil {
		return fmt.Errorf("submitting workout update: %w", err)
	}
	defer postResp.Body.Close()
	io.Copy(io.Discard, postResp.Body)

	if postResp.StatusCode >= 400 {
		return fmt.Errorf("workout update failed with status %d", postResp.StatusCode)
	}

	return nil
}

func buildFormData(schema *FormSchema, workout *WorkoutData) url.Values {
	data := url.Values{}

	for name := range schema.Fields {
		lower := strings.ToLower(name)
		switch {
		case name == "activitytypeid":
			data.Set(name, workout.ActivityTypeID)
		case name == "session-day":
			data.Set(name, workout.Day)
		case name == "session-month":
			data.Set(name, workout.Month)
		case name == "session-year":
			data.Set(name, workout.Year)
		case name == "sessionstarthour":
			if workout.StartHour != "" {
				data.Set(name, workout.StartHour)
			}
		case name == "distance":
			if workout.Distance != "" {
				data.Set(name, workout.Distance)
			}
		case name == "distanceunits":
			data.Set(name, workout.DistanceUnits)
		case name == "sessionlength":
			if workout.Duration != "" {
				data.Set(name, workout.Duration)
			}
		case name == "ahr":
			if workout.AverageHR != "" {
				data.Set(name, workout.AverageHR)
			}
		case name == "mhr":
			if workout.MaxHR != "" {
				data.Set(name, workout.MaxHR)
			}
		case name == "climb":
			if workout.ElevationGain != "" {
				data.Set(name, workout.ElevationGain)
			}
		case name == "description":
			data.Set(name, workout.Description)
		case lower == "workouttypeid":
			data.Set(name, "1") // Default to "Training"
		case name == "isplan":
			data.Set(name, "0") // Not a planned workout
		case name == "intensity":
			data.Set(name, "0") // Default intensity
		case name == "map":
			data.Set(name, "0") // No map
		case name == "shoes":
			data.Set(name, "null") // Not specified
		case name == "restday", name == "sick", name == "injured",
			name == "spiked", name == "controls",
			name == "weight", name == "rhr", name == "sleep",
			name == "pace", name == "wunit",
			name == "climb_grade", name == "climb_angle",
			name == "newactivitytype", name == "activitymodifiers":
			data.Set(name, "") // Send empty for optional fields
		}
	}

	return data
}
