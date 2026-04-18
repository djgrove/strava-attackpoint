package strava

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/djgrove/strava-attackpoint/internal/config"
)

const baseURL = "https://www.strava.com/api/v3"

// Client is an authenticated Strava API client.
type Client struct {
	httpClient  *http.Client
	cfg         *config.Config
	accessToken string
}

// NewClient creates a Strava client, refreshing the token if needed.
func NewClient(cfg *config.Config) (*Client, error) {
	token, err := RefreshAccessToken(cfg)
	if err != nil {
		return nil, fmt.Errorf("authenticating with Strava: %w", err)
	}

	return &Client{
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		cfg:         cfg,
		accessToken: token,
	}, nil
}

// doRequest executes an authenticated GET request with rate limit handling.
func (c *Client) doRequest(path string) (*http.Response, error) {
	url := baseURL + path

	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.accessToken)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("requesting %s: %w", path, err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			wait := rateLimitWait(resp)
			fmt.Printf("Rate limited, waiting %s...\n", wait)
			time.Sleep(wait)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("request to %s failed after 3 rate limit retries", path)
}

func rateLimitWait(resp *http.Response) time.Duration {
	// Check for Retry-After header first.
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if secs, err := strconv.Atoi(ra); err == nil {
			return time.Duration(secs) * time.Second
		}
	}

	// Fall back to calculating from rate limit headers.
	// X-RateLimit-Usage: "short,daily" e.g. "98,450"
	if usage := resp.Header.Get("X-RateLimit-Usage"); usage != "" {
		parts := strings.SplitN(usage, ",", 2)
		if len(parts) >= 1 {
			if shortUsage, err := strconv.Atoi(parts[0]); err == nil && shortUsage >= 95 {
				// Wait until next 15-minute boundary.
				now := time.Now()
				mins := now.Minute() % 15
				wait := time.Duration(15-mins)*time.Minute - time.Duration(now.Second())*time.Second
				if wait < time.Minute {
					wait = time.Minute
				}
				return wait
			}
		}
	}

	// Default wait.
	return 60 * time.Second
}
