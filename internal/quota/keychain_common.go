package quota

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// expandTilde expands a leading ~/ to the user's home directory.
func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return home + path[1:]
		}
	}
	return path
}

// TokenInfo holds parsed token metadata for display.
type TokenInfo struct {
	ExpiresAt time.Time // when the token expires
	Valid     bool      // whether the token is currently valid
	HasToken  bool      // whether a token file exists
}

// GetTokenInfo reads and parses the token expiry from a config dir.
// Works on both Linux (.credentials.json) and macOS (keychain via service name).
func GetTokenInfo(configDir string) TokenInfo {
	expanded := expandTilde(configDir)
	credPath := filepath.Join(expanded, ".credentials.json")

	data, err := os.ReadFile(credPath)
	if err != nil {
		return TokenInfo{}
	}

	// Try claudeAiOauth.expiresAt (milliseconds) — Linux format
	var creds struct {
		ClaudeAiOauth struct {
			ExpiresAt int64 `json:"expiresAt"`
		} `json:"claudeAiOauth"`
	}
	if json.Unmarshal(data, &creds) == nil && creds.ClaudeAiOauth.ExpiresAt > 0 {
		expiry := time.UnixMilli(creds.ClaudeAiOauth.ExpiresAt)
		return TokenInfo{
			ExpiresAt: expiry,
			Valid:     time.Now().Before(expiry),
			HasToken:  true,
		}
	}

	// Fallback: top-level expires_at (seconds)
	var cred struct {
		ExpiresAt int64 `json:"expires_at"`
	}
	if json.Unmarshal(data, &cred) == nil && cred.ExpiresAt > 0 {
		expiry := time.Unix(cred.ExpiresAt, 0)
		return TokenInfo{
			ExpiresAt: expiry,
			Valid:     time.Now().Before(expiry),
			HasToken:  true,
		}
	}

	// Token exists but format unrecognized
	return TokenInfo{HasToken: true, Valid: true}
}
