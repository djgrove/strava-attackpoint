package attackpoint

import (
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// LogEntry represents a training entry found on the user's log page.
type LogEntry struct {
	SessionID   string
	Description string
	StravaID    string // extracted Strava activity ID from description, if present
}

var stravaURLPattern = regexp.MustCompile(`strava\.com/activities/(\d+)`)
var csrfPattern = regexp.MustCompile(`csrfToken=([^&'"]+)`)

// FetchLogEntries fetches the user's training log for a date range and returns entries
// that contain Strava activity URLs in their descriptions.
func (c *Client) FetchLogEntries(userID string, startDate, endDate time.Time) ([]LogEntry, error) {
	var allEntries []LogEntry

	// Iterate week by week to cover the date range.
	current := endDate
	for current.After(startDate) || current.Equal(startDate) {
		path := fmt.Sprintf("/viewlog.jsp/user_%s/period-7/enddate-%s", userID, current.Format("2006-01-02"))
		entries, err := c.fetchLogPage(path)
		if err != nil {
			return nil, fmt.Errorf("fetching log page for %s: %w", current.Format("2006-01-02"), err)
		}
		allEntries = append(allEntries, entries...)
		current = current.AddDate(0, 0, -7)
	}

	// Deduplicate by session ID.
	seen := make(map[string]bool)
	var unique []LogEntry
	for _, e := range allEntries {
		if !seen[e.SessionID] {
			seen[e.SessionID] = true
			unique = append(unique, e)
		}
	}

	return unique, nil
}

func (c *Client) fetchLogPage(path string) ([]LogEntry, error) {
	resp, err := c.Get(path)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("parsing log page: %w", err)
	}

	var entries []LogEntry
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "tlactivity") {
			entry := parseLogEntry(n)
			if entry.StravaID != "" {
				entries = append(entries, entry)
			}
			return
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)

	return entries, nil
}

func parseLogEntry(activityDiv *html.Node) LogEntry {
	entry := LogEntry{}

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "div" {
			if hasClass(n, "editutils") {
				entry.SessionID = getAttr(n, "data-sessionid")
			}
			if hasClass(n, "descrow") {
				entry.Description = textContent(n)
				matches := stravaURLPattern.FindStringSubmatch(entry.Description)
				if len(matches) >= 2 {
					entry.StravaID = matches[1]
				}
			}
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(activityDiv)

	return entry
}

func hasClass(n *html.Node, class string) bool {
	classes := getAttr(n, "class")
	for _, c := range strings.Fields(classes) {
		if c == class {
			return true
		}
	}
	return false
}

// DeleteSession deletes a training entry by session ID.
// It first fetches the edit page to get the CSRF token, then POSTs to /deltraining.jsp.
func (c *Client) DeleteSession(sessionID string) error {
	// Fetch the edit page to extract the CSRF token.
	editPath := fmt.Sprintf("/edittrainingsession.jsp?sessionid=%s", sessionID)
	resp, err := c.Get(editPath)
	if err != nil {
		return fmt.Errorf("fetching edit page for session %s: %w", sessionID, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading edit page: %w", err)
	}

	// Extract CSRF token from the page JS.
	matches := csrfPattern.FindSubmatch(body)
	if len(matches) < 2 {
		return fmt.Errorf("could not find CSRF token on edit page for session %s", sessionID)
	}

	csrfToken, err := url.QueryUnescape(string(matches[1]))
	if err != nil {
		csrfToken = string(matches[1])
	}

	// POST to /deltraining.jsp with the session ID and CSRF token.
	delURL := fmt.Sprintf("/deltraining.jsp?sessionid=%s", sessionID)
	delResp, err := c.PostForm(delURL, url.Values{
		"csrfToken": {csrfToken},
	})
	if err != nil {
		return fmt.Errorf("deleting session %s: %w", sessionID, err)
	}
	defer delResp.Body.Close()
	io.Copy(io.Discard, delResp.Body)

	if delResp.StatusCode >= 400 {
		return fmt.Errorf("delete failed for session %s (status %d)", sessionID, delResp.StatusCode)
	}

	return nil
}
