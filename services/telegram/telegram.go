// Package telegram fetches a Telegram user/channel's profile photo.
// No API key required - scrapes the og:image meta tag from t.me.
// Identifier: username (without @)  e.g. "paul"
package telegram

import (
	"fmt"
	"io"
	"net/http"
	"regexp"

	"meowavatar/httpclient"
)

var metaImageRe = regexp.MustCompile(`<meta\s+property="(?:og|twitter):image"\s+content="([^"]+)"`)

func Fetch(username string) ([]byte, string, error) {
	pageURL := "https://t.me/" + username

	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("telegram: build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; meowavatar/1.0)")

	resp, err := httpclient.Proxied().Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("telegram: page request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, "", fmt.Errorf("telegram: user %q not found", username)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("telegram: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("telegram: read page: %w", err)
	}

	m := metaImageRe.FindSubmatch(body)
	if m == nil {
		return nil, "", fmt.Errorf("telegram: no profile image found for user %q", username)
	}
	imgURL := string(m[1])

	return fetchImage(imgURL)
}

func fetchImage(imgURL string) ([]byte, string, error) {
	resp, err := httpclient.Proxied().Get(imgURL)
	if err != nil {
		return nil, "", fmt.Errorf("telegram: fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("telegram: image fetch status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("telegram: read image: %w", err)
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/jpeg"
	}
	return data, ct, nil
}
