package strava

import "time"

// Activity represents a Strava activity with the fields we need.
type Activity struct {
	ID                 int64   `json:"id"`
	Name               string  `json:"name"`
	Description        string  `json:"description"`
	Type               string  `json:"type"`
	SportType          string  `json:"sport_type"`
	StartDate          string  `json:"start_date"`
	StartDateLocal     string  `json:"start_date_local"`
	Timezone           string  `json:"timezone"`
	Distance           float64 `json:"distance"`            // meters
	MovingTime         int     `json:"moving_time"`          // seconds
	ElapsedTime        int     `json:"elapsed_time"`         // seconds
	TotalElevationGain float64 `json:"total_elevation_gain"` // meters
	AverageHeartrate   float64 `json:"average_heartrate"`
	MaxHeartrate       float64 `json:"max_heartrate"`
	HasHeartrate       bool    `json:"has_heartrate"`
	AverageSpeed       float64 `json:"average_speed"`
	MaxSpeed           float64 `json:"max_speed"`
}

// StartTime parses the local start date.
func (a *Activity) StartTime() (time.Time, error) {
	return time.Parse(time.RFC3339, a.StartDateLocal)
}

// TokenResponse is the Strava OAuth token exchange response.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	TokenType    string `json:"token_type"`
}
