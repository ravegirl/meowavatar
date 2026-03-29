// Package discord fetches a Discord user's avatar by user ID.
// Requires env: DISCORD_BOT_TOKEN
// Identifier: user snowflake ID  e.g. "1322412139966836821"
package discord

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	apiBase    = "https://discord.com/api/v10"
	cdnBase    = "https://cdn.discordapp.com"
	avatarSize = "?size=512"
)

type discordUser struct {
	ID     string `json:"id"`
	Avatar string `json:"avatar"`
}

func Fetch(userID string) ([]byte, string, error) {
	token := os.Getenv("DISCORD_BOT_TOKEN")
	if token == "" {
		return nil, "", fmt.Errorf("discord: DISCORD_BOT_TOKEN not set")
	}

	// Fetch user object
	req, err := http.NewRequest(http.MethodGet, apiBase+"/users/"+userID, nil)
	if err != nil {
		return nil, "", fmt.Errorf("discord: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bot "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("discord: api request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, "", fmt.Errorf("discord: user %q not found", userID)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("discord: unexpected status %d", resp.StatusCode)
	}

	var user discordUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, "", fmt.Errorf("discord: decode response: %w", err)
	}

	if user.Avatar == "" {
		// Default avatar: based on discriminator legacy or new system (index = (user_id >> 22) % 6)
		return fetchDefaultAvatar(userID)
	}

	// Animated avatars start with "a_"
	ext := "png"
	if len(user.Avatar) >= 2 && user.Avatar[:2] == "a_" {
		ext = "gif"
	}

	avatarURL := fmt.Sprintf("%s/avatars/%s/%s.%s%s", cdnBase, user.ID, user.Avatar, ext, avatarSize)
	return fetchImage(avatarURL)
}

func fetchDefaultAvatar(userID string) ([]byte, string, error) {
	// New Discord default avatar system: index = (snowflake_id >> 22) % 6
	var id uint64
	fmt.Sscanf(userID, "%d", &id)
	index := (id >> 22) % 6
	url := fmt.Sprintf("%s/embed/avatars/%d.png", cdnBase, index)
	return fetchImage(url)
}

func fetchImage(url string) ([]byte, string, error) {
	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return nil, "", fmt.Errorf("discord: fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("discord: image fetch status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", fmt.Errorf("discord: read image: %w", err)
	}

	ct := resp.Header.Get("Content-Type")
	if ct == "" {
		ct = "image/png"
	}
	return data, ct, nil
}
