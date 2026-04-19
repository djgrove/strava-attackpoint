package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/zalando/go-keyring"
)

const (
	appName    = "strava-ap"
	keyService = "strava-ap"
)

// Config holds non-secret configuration.
type Config struct {
	TokenExpiry time.Time `json:"token_expiry"`
}

// SyncState tracks which Strava activities have been synced to AP.
type SyncState struct {
	Activities map[string]SyncedActivity `json:"activities"`
}

// SyncedActivity records a synced activity.
type SyncedActivity struct {
	APEntryURL   string    `json:"ap_entry_url"`
	LastSyncedAt time.Time `json:"last_synced_at"`
}

func configDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("finding config directory: %w", err)
	}
	return filepath.Join(base, appName), nil
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func syncStatePath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "sync_state.json"), nil
}

// LoadConfig reads the config file. Returns a zero Config if the file doesn't exist.
func LoadConfig() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}
	return &cfg, nil
}

// SaveConfig writes the config file.
func SaveConfig(cfg *Config) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("serializing config: %w", err)
	}

	path := filepath.Join(dir, "config.json")
	return os.WriteFile(path, data, 0600)
}

// Keychain helpers for secret storage.

func SetSecret(key, value string) error {
	return keyring.Set(keyService, key, value)
}

func GetSecret(key string) (string, error) {
	val, err := keyring.Get(keyService, key)
	if err == keyring.ErrNotFound {
		return "", nil
	}
	return val, err
}

func DeleteSecret(key string) error {
	err := keyring.Delete(keyService, key)
	if err == keyring.ErrNotFound {
		return nil
	}
	return err
}

// Convenience accessors for Strava secrets.

func GetAccessToken() (string, error)                { return GetSecret("strava-access-token") }
func SetAccessToken(v string) error                  { return SetSecret("strava-access-token", v) }
func GetRefreshToken() (string, error)               { return GetSecret("strava-refresh-token") }
func SetRefreshToken(v string) error                 { return SetSecret("strava-refresh-token", v) }

// LoadSyncState reads the sync state file.
func LoadSyncState() (*SyncState, error) {
	path, err := syncStatePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &SyncState{Activities: make(map[string]SyncedActivity)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading sync state: %w", err)
	}

	var state SyncState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing sync state: %w", err)
	}
	if state.Activities == nil {
		state.Activities = make(map[string]SyncedActivity)
	}
	return &state, nil
}

// SaveSyncState writes the sync state file.
func SaveSyncState(state *SyncState) error {
	dir, err := configDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("serializing sync state: %w", err)
	}

	path := filepath.Join(dir, "sync_state.json")
	return os.WriteFile(path, data, 0600)
}
