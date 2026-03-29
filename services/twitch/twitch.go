// Package twitch fetches a Twitch user's profile image.
// Requires env: TWITCH_CLIENT_ID, TWITCH_CLIENT_SECRET
// Identifier: channel login name  e.g. "xqc"
package twitch

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

var tokenCache struct {
	sync.Mutex
	token   string
	expires time.Time
}

type tokenResp struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

type usersResp struct {
	Data []struct {
		ProfileImageURL string `json:"profile_image_url"`
	} `json:"data"`
}

func getAppToken() (string, error) {
	tokenCache.Lock()
	defer tokenCache.Unlock()

	if tokenCache.token != "" && time.Now().Before(tokenCache.expires) {
		return tokenCache.token, nil
	}

	clientID := os.Getenv("TWITCH_CLIENT_ID")
	clientSecret := os.Getenv("TWITCH_CLIENT_SECRET")
	if clientID == "" || clientSecret == "" {
		return "", fmt.Errorf("twitch: TWITCH_CLIENT_ID and TWITCH_CLIENT_SECRET must be set")
	}

	form := url.Values{
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"grant_type":    {"client_credentials"},
	}

	resp, err := http.PostForm("https://id.twitch.tv/oauth2/token", form)
	if err != nil {
		return "", fmt.Errorf("twitch: token request: %w", err)
	}
	defer resp.Body.Close()

	var tr tokenResp
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("twitch: decode token: %w", err)
	}

	tokenCache.token = tr.AccessToken
	tokenCache.expires = time.Now().Add(time.Duration(tr.ExpiresIn-60) * time.Second)
	return tr.AccessToken, nil
}

func Fetch(username string) ([]byte, string, error) {
	clientID := os.Getenv("TWITCH_CLIENT_ID")

	token, err := getAppToken()
	if err != nil {
		return nil, "", err
	}

	req, err := http.NewRequest(http.MethodGet,
		"https://api.twitch.tv/helix/users?login="+url.QueryEscape(username), nil)
	if err != nil {
		return nil, "", fmt.Errorf("twitch: build request: %w", err)
	}
	req.Header.Set("Client-Id", clientID)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("twitch: api request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("twitch: unexpected status %d", resp.StatusCode)
	}

	var ur usersResp
	if err := json.NewDecoder(resp.Body).Decode(&ur); err != nil {
		return nil, "", fmt.Errorf("twitch: decode response: %w", err)
	}

	if len(ur.Data) == 0 {
		return nil, "", fmt.Errorf("twitch: user %q not found", username)
	}

	imgURL := ur.Data[0].ProfileImageURL
	if imgURL == "" {
		return nil, "", fmt.Errorf("twitch: no profile image for %q", username)
	}

	imgURL = strings.ReplaceAll(imgURL, "{width}", "300")
	imgURL = strings.ReplaceAll(imgURL, "{height}", "300")

	return fetchImage(imgURL)
}

func fetchImage(url string) ([]byte, string, error) {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return nil, "", fmt.Errorf("twitch: fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("twitch: image fetch status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("twitch: read image: %w", err)
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/jpeg"
	}
	return data, ct, nil
}
