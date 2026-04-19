package strava

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/djgrove/strava-attackpoint/internal/config"
	"github.com/pkg/browser"
)

const (
	authorizeURL = "https://www.strava.com/oauth/authorize"
	callbackPort = "8089"

	// ClientID is public and safe to embed in the binary.
	ClientID = "200536"

	// ProxyURL is the API Gateway endpoint for the OAuth proxy Lambda.
	ProxyURL = "https://m28wz39mbc.execute-api.us-west-2.amazonaws.com"
)

// RunOAuthFlow performs the Strava OAuth2 authorization flow.
// The client secret is held by the proxy — the CLI never sees it.
func RunOAuthFlow() error {
	state, err := randomState()
	if err != nil {
		return fmt.Errorf("generating state: %w", err)
	}

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			errCh <- fmt.Errorf("OAuth state mismatch")
			return
		}
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			http.Error(w, "Authorization denied", http.StatusForbidden)
			errCh <- fmt.Errorf("authorization denied: %s", errMsg)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "No code received", http.StatusBadRequest)
			errCh <- fmt.Errorf("no authorization code received")
			return
		}

		fmt.Fprint(w, "<html><body><h2>Authorization successful!</h2><p>You can close this tab and return to the terminal.</p></body></html>")
		codeCh <- code
	})

	listener, err := net.Listen("tcp", ":"+callbackPort)
	if err != nil {
		return fmt.Errorf("starting callback server: %w", err)
	}

	server := &http.Server{Handler: mux}
	go func() { _ = server.Serve(listener) }()
	defer server.Shutdown(context.Background())

	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&scope=read,activity:read_all&state=%s",
		authorizeURL,
		url.QueryEscape(ClientID),
		url.QueryEscape("http://localhost:"+callbackPort+"/callback"),
		url.QueryEscape(state),
	)

	fmt.Println("Opening browser for Strava authorization...")
	fmt.Printf("If the browser doesn't open, visit this URL:\n%s\n\n", authURL)
	_ = browser.OpenURL(authURL)

	fmt.Println("Waiting for authorization...")

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return err
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("authorization timed out after 5 minutes")
	}

	// Exchange code for tokens via the proxy.
	token, err := exchangeToken(code)
	if err != nil {
		return err
	}

	// Save tokens.
	cfg := &config.Config{
		TokenExpiry: time.Unix(token.ExpiresAt, 0),
	}
	if err := config.SaveConfig(cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	if err := config.SetAccessToken(token.AccessToken); err != nil {
		return fmt.Errorf("saving access token to keychain: %w", err)
	}
	if err := config.SetRefreshToken(token.RefreshToken); err != nil {
		return fmt.Errorf("saving refresh token to keychain: %w", err)
	}

	fmt.Println("Authorization successful! Tokens saved securely.")
	return nil
}

// exchangeToken sends the auth code to the proxy, which injects the secret.
func exchangeToken(code string) (*TokenResponse, error) {
	payload, _ := json.Marshal(map[string]string{
		"code":       code,
		"grant_type": "authorization_code",
	})

	resp, err := http.Post(ProxyURL+"/token", "application/json", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("exchanging token via proxy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed with status %d", resp.StatusCode)
	}

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("parsing token response: %w", err)
	}
	return &token, nil
}

// RefreshAccessToken refreshes the access token if expired, via the proxy.
func RefreshAccessToken(cfg *config.Config) (string, error) {
	accessToken, err := config.GetAccessToken()
	if err != nil {
		return "", fmt.Errorf("reading access token: %w", err)
	}

	// If token is still valid (more than 1 minute remaining), return it.
	if time.Until(cfg.TokenExpiry) > time.Minute {
		return accessToken, nil
	}

	refreshToken, err := config.GetRefreshToken()
	if err != nil {
		return "", fmt.Errorf("reading refresh token: %w", err)
	}

	payload, _ := json.Marshal(map[string]string{
		"refresh_token": refreshToken,
	})

	resp, err := http.Post(ProxyURL+"/refresh", "application/json", bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("refreshing token via proxy: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token refresh failed with status %d — try running 'strava-ap setup' again", resp.StatusCode)
	}

	var token TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return "", fmt.Errorf("parsing refresh response: %w", err)
	}

	// Save updated tokens.
	cfg.TokenExpiry = time.Unix(token.ExpiresAt, 0)
	if err := config.SaveConfig(cfg); err != nil {
		return "", fmt.Errorf("saving config after refresh: %w", err)
	}
	if err := config.SetAccessToken(token.AccessToken); err != nil {
		return "", fmt.Errorf("saving refreshed access token: %w", err)
	}
	if err := config.SetRefreshToken(token.RefreshToken); err != nil {
		return "", fmt.Errorf("saving refreshed refresh token: %w", err)
	}

	return token.AccessToken, nil
}

func randomState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
