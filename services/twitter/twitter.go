// Package twitter fetches a Twitter/X user's profile image.
// No API key required - seeds a guest session from x.com then calls the GQL endpoint.
// Identifier: username (without @)  e.g. "jack"
package twitter

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"meowavatar/httpclient"
)

// Public bearer token used by the Twitter web app - not a secret.
const bearerToken = "AAAAAAAAAAAAAAAAAAAAANRILgAAAAAAnNwIzUejRCOuH5E6I8xnZz4puTs%3D1Zv7ttfk8LF81IUq16cHjhLTvJu4FA33AGWWjCpTnA"

const (
	gqlURL       = "https://api.x.com/graphql/IGgvgiOx4QZndDHuD3x9TQ/UserByScreenName"
	variables    = `{"screen_name":"%s","withGrokTranslatedBio":false}`
	features     = `{"hidden_profile_subscriptions_enabled":true,"profile_label_improvements_pcf_label_in_post_enabled":true,"responsive_web_profile_redirect_enabled":false,"rweb_tipjar_consumption_enabled":false,"verified_phone_label_enabled":false,"subscriptions_verification_info_is_identity_verified_enabled":true,"subscriptions_verification_info_verified_since_enabled":true,"highlights_tweets_tab_ui_enabled":true,"responsive_web_twitter_article_notes_tab_enabled":true,"subscriptions_feature_can_gift_premium":true,"creator_subscriptions_tweet_preview_api_enabled":true,"responsive_web_graphql_skip_user_profile_image_extensions_enabled":false,"responsive_web_graphql_timeline_navigation_enabled":true}`
	fieldToggles = `{"withPayments":false,"withAuxiliaryUserLabels":true}`
)

var session struct {
	sync.Mutex
	client  *http.Client
	ct0     string
	expires time.Time
}

// seedSession visits x.com to collect guest cookies, then extracts ct0 for CSRF.
func seedSession() error {
	jar, err := cookiejar.New(nil)
	if err != nil {
		return fmt.Errorf("twitter: create cookie jar: %w", err)
	}

	base := httpclient.Proxied().Transport
	client := &http.Client{
		Transport: base,
		Jar:       jar,
	}

	req, err := http.NewRequest(http.MethodGet, "https://x.com/", nil)
	if err != nil {
		return fmt.Errorf("twitter: build seed request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("twitter: seed request: %w", err)
	}
	resp.Body.Close()

	// Extract ct0 from cookies
	xURL, _ := url.Parse("https://x.com")
	var ct0 string
	for _, c := range jar.Cookies(xURL) {
		if c.Name == "ct0" && c.Value != "" {
			ct0 = c.Value
			break
		}
	}

	// ct0 may be empty on first visit; that's fine - x.com sets it after JS runs.
	// We still have guest_id which is the important one.
	session.client = client
	session.ct0 = ct0
	session.expires = time.Now().Add(90 * time.Minute)
	return nil
}

func getSession() (*http.Client, string, error) {
	session.Lock()
	defer session.Unlock()

	if session.client != nil && time.Now().Before(session.expires) {
		return session.client, session.ct0, nil
	}

	if err := seedSession(); err != nil {
		return nil, "", err
	}
	return session.client, session.ct0, nil
}

type gqlResp struct {
	Data struct {
		User struct {
			Result struct {
				Avatar struct {
					ImageURL string `json:"image_url"`
				} `json:"avatar"`
			} `json:"result"`
		} `json:"user"`
	} `json:"data"`
}

func Fetch(username string) ([]byte, string, error) {
	client, ct0, err := getSession()
	if err != nil {
		return nil, "", err
	}

	params := url.Values{
		"variables":    {fmt.Sprintf(variables, username)},
		"features":     {features},
		"fieldToggles": {fieldToggles},
	}

	req, err := http.NewRequest(http.MethodGet, gqlURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, "", fmt.Errorf("twitter: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+bearerToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36")
	if ct0 != "" {
		req.Header.Set("X-Csrf-Token", ct0)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("twitter: api request: %w", err)
	}
	defer resp.Body.Close()

	// If we get 403, the session expired - clear it and return an error so the
	// next request re-seeds.
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnauthorized {
		session.Lock()
		session.client = nil
		session.Unlock()
		return nil, "", fmt.Errorf("twitter: session expired, will retry on next request")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("twitter: unexpected status %d", resp.StatusCode)
	}

	var gr gqlResp
	if err := json.NewDecoder(resp.Body).Decode(&gr); err != nil {
		return nil, "", fmt.Errorf("twitter: decode response: %w", err)
	}

	imgURL := gr.Data.User.Result.Avatar.ImageURL
	if imgURL == "" {
		return nil, "", fmt.Errorf("twitter: no avatar found for user %q", username)
	}

	imgURL = strings.Replace(imgURL, "_normal", "_400x400", 1)
	return fetchImage(imgURL)
}

func fetchImage(imgURL string) ([]byte, string, error) {
	resp, err := httpclient.Proxied().Get(imgURL)
	if err != nil {
		return nil, "", fmt.Errorf("twitter: fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("twitter: image fetch status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("twitter: read image: %w", err)
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/jpeg"
	}
	return data, ct, nil
}
