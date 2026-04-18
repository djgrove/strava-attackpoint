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
	Date           string // form's date field value
	Distance       string
	Duration       string // HH:MM:SS or similar
	AverageHR      string
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
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("workout submission failed with status %d", resp.StatusCode)
	}

	return nil
}

// UpdateWorkout edits an existing training entry.
func (c *Client) UpdateWorkout(editPath string, schema *FormSchema, workout *WorkoutData) error {
	// Fetch the edit form to get its specific action URL and any hidden fields.
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

	// Set known fields by searching for them in the discovered schema.
	for name := range schema.Fields {
		switch {
		case matchesField(name, "activitytype"):
			data.Set(name, workout.ActivityTypeID)
		case matchesField(name, "distance"):
			if workout.Distance != "" {
				data.Set(name, workout.Distance)
			}
		case matchesField(name, "sessionlength", "duration"):
			if workout.Duration != "" {
				data.Set(name, workout.Duration)
			}
		case matchesField(name, "ahr", "heartrate", "avghr"):
			if workout.AverageHR != "" {
				data.Set(name, workout.AverageHR)
			}
		case matchesField(name, "climb", "elevation", "gain"):
			if workout.ElevationGain != "" {
				data.Set(name, workout.ElevationGain)
			}
		case matchesField(name, "description", "notes", "logtextarea", "text"):
			data.Set(name, workout.Description)
		}
	}

	return data
}

// matchesField checks if a field name contains any of the given substrings (case-insensitive).
func matchesField(name string, substrings ...string) bool {
	lower := strings.ToLower(name)
	for _, s := range substrings {
		if strings.Contains(lower, strings.ToLower(s)) {
			return true
		}
	}
	return false
}
