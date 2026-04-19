package strava

import (
	"encoding/json"
	"fmt"
	"time"
)

// FetchActivitiesSince fetches all activities after the given time.
func (c *Client) FetchActivitiesSince(since time.Time) ([]Activity, error) {
	var all []Activity
	page := 1
	after := since.Unix()

	for {
		path := fmt.Sprintf("/athlete/activities?after=%d&page=%d&per_page=100", after, page)
		resp, err := c.doRequest(path)
		if err != nil {
			return nil, err
		}

		var activities []Activity
		if err := json.NewDecoder(resp.Body).Decode(&activities); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("parsing activities page %d: %w", page, err)
		}
		resp.Body.Close()

		if len(activities) == 0 {
			break
		}

		all = append(all, activities...)
		page++
	}

	return all, nil
}

// FetchActivity fetches full details for a single activity.
func (c *Client) FetchActivity(id int64) (*Activity, error) {
	path := fmt.Sprintf("/activities/%d", id)
	resp, err := c.doRequest(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("fetching activity %d: status %d", id, resp.StatusCode)
	}

	var activity Activity
	if err := json.NewDecoder(resp.Body).Decode(&activity); err != nil {
		return nil, fmt.Errorf("parsing activity %d: %w", id, err)
	}
	return &activity, nil
}

// FetchActivityZones fetches heart rate zone distribution for an activity.
// Returns the dominant HR zone (1-5), or 0 if no HR data.
func (c *Client) FetchActivityZones(id int64) (int, error) {
	path := fmt.Sprintf("/activities/%d/zones", id)
	resp, err := c.doRequest(path)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return 0, nil // Zones API not available (e.g., 402 for free accounts)
	}

	var zones []ZoneDistribution
	if err := json.NewDecoder(resp.Body).Decode(&zones); err != nil {
		return 0, nil
	}

	for _, z := range zones {
		if z.Type == "heartrate" && len(z.Buckets) > 0 {
			maxTime := 0
			maxZone := 0
			for i, bucket := range z.Buckets {
				if bucket.Time > maxTime {
					maxTime = bucket.Time
					maxZone = i + 1
				}
			}
			if maxZone > 5 {
				maxZone = 5
			}
			return maxZone, nil
		}
	}

	return 0, nil
}
