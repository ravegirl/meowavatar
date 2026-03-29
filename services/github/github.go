// Package github fetches a GitHub user's avatar.
// No API key required - uses the public avatar URL directly, only here for consistency.
// Identifier: username  e.g. "torvalds"
package github

import (
	"fmt"
	"io"
	"net/http"
)

const avatarURL = "https://github.com/%s.png"

func Fetch(username string) ([]byte, string, error) {
	url := fmt.Sprintf(avatarURL, username)

	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return nil, "", fmt.Errorf("github: http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, "", fmt.Errorf("github: user %q not found", username)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("github: unexpected status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("github: read body: %w", err)
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/jpeg"
	}

	return data, ct, nil
}
