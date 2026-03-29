// Package reddit fetches a Reddit user's avatar/profile icon.
// No API key required - uses the public JSON API.
// Identifier: username  e.g. "spez"
package reddit

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"meowavatar/httpclient"
)

type redditAbout struct {
	Data struct {
		IconImg      string `json:"icon_img"`
		SnoovatarImg string `json:"snoovatar_img"`
	} `json:"data"`
}

func Fetch(username string) ([]byte, string, error) {
	url := fmt.Sprintf("https://www.reddit.com/user/%s/about.json", username)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, "", fmt.Errorf("reddit: build request: %w", err)
	}
	req.Header.Set("User-Agent", "meowavatar/1.0")

	resp, err := httpclient.Proxied().Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("reddit: api request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, "", fmt.Errorf("reddit: user %q not found", username)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("reddit: unexpected status %d", resp.StatusCode)
	}

	var about redditAbout
	if err := json.NewDecoder(resp.Body).Decode(&about); err != nil {
		return nil, "", fmt.Errorf("reddit: decode response: %w", err)
	}

	imgURL := about.Data.SnoovatarImg
	if imgURL == "" {
		imgURL = about.Data.IconImg
	}
	if imgURL == "" {
		return nil, "", fmt.Errorf("reddit: no avatar found for user %q", username)
	}

	imgURL = strings.ReplaceAll(imgURL, "&amp;", "&")

	return fetchImage(imgURL)
}

func fetchImage(url string) ([]byte, string, error) {
	resp, err := httpclient.Proxied().Get(url)
	if err != nil {
		return nil, "", fmt.Errorf("reddit: fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("reddit: image fetch status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("reddit: read image: %w", err)
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/png"
	}
	return data, ct, nil
}
