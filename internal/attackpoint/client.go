package attackpoint

import (
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strings"
)

const baseURL = "https://www.attackpoint.org"

var userIDPattern = regexp.MustCompile(`/user[_/](\d+)`)

// Client is an authenticated AttackPoint HTTP client.
type Client struct {
	httpClient *http.Client
	loggedIn   bool
	UserID     string // discovered from log page link after login
}

// NewClient creates a new AP client with a cookie jar.
func NewClient() (*Client, error) {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return nil, fmt.Errorf("creating cookie jar: %w", err)
	}
	return &Client{
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 30_000_000_000, // 30s
		},
	}, nil
}

// Login authenticates with AttackPoint and discovers the user ID.
func (c *Client) Login(username, password string) error {
	data := url.Values{
		"username": {username},
		"password": {password},
		"returl":   {"/"},
	}

	resp, err := c.httpClient.PostForm(baseURL+"/dologin.jsp", data)
	if err != nil {
		return fmt.Errorf("posting login form: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// AP redirects on successful login.
	finalURL := resp.Request.URL.String()
	if strings.Contains(finalURL, "login.jsp") {
		return fmt.Errorf("login failed — check your username and password")
	}

	c.loggedIn = true

	// Try to extract user ID from the homepage body (look for log.jsp/user_XXXX link).
	bodyStr := string(body)
	matches := userIDPattern.FindStringSubmatch(bodyStr)
	if len(matches) >= 2 {
		c.UserID = matches[1]
	}

	return nil
}

// Get performs an authenticated GET request.
func (c *Client) Get(path string) (*http.Response, error) {
	if !c.loggedIn {
		return nil, fmt.Errorf("not logged in")
	}
	return c.httpClient.Get(baseURL + path)
}

// PostForm performs an authenticated POST with form data.
func (c *Client) PostForm(path string, data url.Values) (*http.Response, error) {
	if !c.loggedIn {
		return nil, fmt.Errorf("not logged in")
	}
	return c.httpClient.PostForm(baseURL+path, data)
}
