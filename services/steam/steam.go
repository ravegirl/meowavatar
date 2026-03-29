// Package steam fetches a Steam user's avatar.
// No API key required - scrapes the og:image meta tag from the public profile page.
// Identifier: vanity name (e.g. "gaben") or SteamID64 (e.g. "76561197960287930")
// Numeric IDs are looked up via /profiles/, vanity names via /id/
package steam

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"meowavatar/httpclient"
)

var avatarRe = regexp.MustCompile(`(?:og:image"\s+content|image_src"\s+href)="(https://[^"]+steamstatic\.com/[^"]+)"`)

func Fetch(identifier string) ([]byte, string, error) {
	var profileURL string
	if isNumeric(identifier) {
		profileURL = "https://steamcommunity.com/profiles/" + identifier
	} else {
		profileURL = "https://steamcommunity.com/id/" + identifier
	}

	req, err := http.NewRequest(http.MethodGet, profileURL, nil)
	if err != nil {
		return nil, "", fmt.Errorf("steam: build request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")

	resp, err := httpclient.Proxied().Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("steam: fetch profile page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("steam: unexpected status %d for %q", resp.StatusCode, identifier)
	}

	// Read only the <head> - avatar meta tags appear well before </head>
	// Cap at 32 KB to avoid pulling the full page into memory
	limited := io.LimitReader(resp.Body, 32*1024)
	html, err := io.ReadAll(limited)
	if err != nil {
		return nil, "", fmt.Errorf("steam: read page: %w", err)
	}

	matches := avatarRe.FindSubmatch(html)
	if matches == nil {
		if strings.Contains(string(html), "The specified profile could not be found") {
			return nil, "", fmt.Errorf("steam: user %q not found", identifier)
		}
		return nil, "", fmt.Errorf("steam: could not find avatar for %q (profile may be private)", identifier)
	}

	imgURL := string(matches[1])
	return fetchImage(imgURL)
}

func isNumeric(s string) bool {
	return strings.Trim(s, "0123456789") == ""
}

func fetchImage(url string) ([]byte, string, error) {
	resp, err := httpclient.Proxied().Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("steam: fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("steam: image fetch status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("steam: read image: %w", err)
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/jpeg"
	}
	return data, ct, nil
}
