//go:build !darwin

package quota

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// KeychainCredential holds a backup of a credential file for rollback.
type KeychainCredential struct {
	ServiceName string // config dir path (used as the "service" on Linux)
	Token       string // backed-up .credentials.json content
}

// KeychainServiceName returns the expanded config dir path.
// On Linux, the "service name" is the config dir path itself (no macOS keychain).
func KeychainServiceName(configDirPath string) string {
	return expandTilde(configDirPath)
}

// credentialsPath returns the path to .credentials.json inside the config dir.
func credentialsPath(configDir string) string {
	return filepath.Join(configDir, ".credentials.json")
}

// ReadKeychainToken reads the .credentials.json file from the given config dir path.
// The serviceName parameter is the expanded config dir path (from KeychainServiceName).
func ReadKeychainToken(serviceName string) (string, error) {
	path := credentialsPath(serviceName)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading credentials from %s: %w", path, err)
	}
	return string(data), nil
}

// WriteKeychainToken writes the token to .credentials.json in the config dir.
// The accountLabel parameter is ignored on Linux (kept for API compatibility).
// Writes atomically via temp file + rename, with 0600 permissions.
func WriteKeychainToken(serviceName, _ string, token string) error {
	path := credentialsPath(serviceName)

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	// Write atomically: temp file + rename
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(token), 0600); err != nil {
		return fmt.Errorf("writing temp credentials file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("renaming temp credentials file: %w", err)
	}
	return nil
}

// SwapKeychainCredential backs up the target's .credentials.json, then overwrites
// it with the source's credentials. Returns the backup for rollback.
func SwapKeychainCredential(targetConfigDir, sourceConfigDir string) (*KeychainCredential, error) {
	targetSvc := KeychainServiceName(targetConfigDir)
	sourceSvc := KeychainServiceName(sourceConfigDir)

	// Back up the target's current credentials
	backupToken, err := ReadKeychainToken(targetSvc)
	if err != nil {
		return nil, fmt.Errorf("backing up target credentials: %w", err)
	}

	// Read the source's credentials
	sourceToken, err := ReadKeychainToken(sourceSvc)
	if err != nil {
		return nil, fmt.Errorf("reading source credentials: %w", err)
	}

	// Write the source's credentials to the target
	if err := WriteKeychainToken(targetSvc, "", sourceToken); err != nil {
		return nil, fmt.Errorf("writing source credentials to target: %w", err)
	}

	return &KeychainCredential{
		ServiceName: targetSvc,
		Token:       backupToken,
	}, nil
}

// RestoreKeychainToken writes the backup credentials back to the config dir.
func RestoreKeychainToken(backup *KeychainCredential) error {
	if backup == nil {
		return nil
	}
	return WriteKeychainToken(backup.ServiceName, "", backup.Token)
}

// SwapOAuthAccount copies the oauthAccount field from the source config dir's
// .claude.json into the target's.
func SwapOAuthAccount(targetConfigDir, sourceConfigDir string) (json.RawMessage, error) {
	targetPath := filepath.Join(expandTilde(targetConfigDir), ".claude.json")
	sourcePath := filepath.Join(expandTilde(sourceConfigDir), ".claude.json")

	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return nil, nil
	}
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil, nil
	}

	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("reading source .claude.json: %w", err)
	}
	var sourceDoc map[string]json.RawMessage
	if err := json.Unmarshal(sourceData, &sourceDoc); err != nil {
		return nil, fmt.Errorf("parsing source .claude.json: %w", err)
	}
	sourceOAuth, ok := sourceDoc["oauthAccount"]
	if !ok {
		return nil, fmt.Errorf("source .claude.json has no oauthAccount")
	}

	targetData, err := os.ReadFile(targetPath)
	if err != nil {
		return nil, fmt.Errorf("reading target .claude.json: %w", err)
	}
	var targetDoc map[string]json.RawMessage
	if err := json.Unmarshal(targetData, &targetDoc); err != nil {
		return nil, fmt.Errorf("parsing target .claude.json: %w", err)
	}

	backup := targetDoc["oauthAccount"]
	targetDoc["oauthAccount"] = sourceOAuth

	out, err := json.MarshalIndent(targetDoc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling target .claude.json: %w", err)
	}
	if err := os.WriteFile(targetPath, out, 0600); err != nil {
		return nil, fmt.Errorf("writing target .claude.json: %w", err)
	}

	return backup, nil
}

// RestoreOAuthAccount writes the backup oauthAccount back to the target .claude.json.
func RestoreOAuthAccount(targetConfigDir string, backup json.RawMessage) error {
	if backup == nil {
		return nil
	}
	targetPath := filepath.Join(expandTilde(targetConfigDir), ".claude.json")

	data, err := os.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("reading target .claude.json: %w", err)
	}
	var doc map[string]json.RawMessage
	if err := json.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("parsing target .claude.json: %w", err)
	}
	doc["oauthAccount"] = backup
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling target .claude.json: %w", err)
	}
	return os.WriteFile(targetPath, out, 0600)
}

// ValidateKeychainToken checks if the OAuth token for a config dir is still usable.
// On Linux, reads .credentials.json and checks the claudeAiOauth.expiresAt field
// (milliseconds since epoch).
func ValidateKeychainToken(configDir string) error {
	svc := KeychainServiceName(configDir)
	raw, err := ReadKeychainToken(svc)
	if err != nil {
		// Can't read the token — don't block planning.
		return nil
	}
	if raw == "" {
		return nil
	}

	// Parse the credentials JSON. On Linux, Claude Code stores:
	// {"claudeAiOauth":{"accessToken":"...","refreshToken":"...","expiresAt":1772315120989,...}}
	// Note: expiresAt is in MILLISECONDS.
	var creds struct {
		ClaudeAiOauth struct {
			ExpiresAt int64 `json:"expiresAt"`
		} `json:"claudeAiOauth"`
	}
	if err := json.Unmarshal([]byte(raw), &creds); err == nil && creds.ClaudeAiOauth.ExpiresAt > 0 {
		expiryMs := creds.ClaudeAiOauth.ExpiresAt
		expiry := time.UnixMilli(expiryMs)
		if time.Now().After(expiry) {
			return fmt.Errorf("token expired at %s", expiry.Format(time.RFC3339))
		}
		return nil
	}

	// Fallback: try top-level expires_at (seconds)
	var cred struct {
		ExpiresAt int64 `json:"expires_at"`
	}
	if json.Unmarshal([]byte(raw), &cred) == nil && cred.ExpiresAt > 0 {
		if time.Now().Unix() >= cred.ExpiresAt {
			return fmt.Errorf("token expired at %s", time.Unix(cred.ExpiresAt, 0).Format(time.RFC3339))
		}
		return nil
	}

	// Token present but format unrecognized — assume valid.
	return nil
}
