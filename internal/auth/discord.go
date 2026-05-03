// Package auth speaks Discord OAuth2 (authorize URL, code exchange,
// /users/@me, /users/@me/guilds) and produces opaque session tokens.
// It does not touch the database or HTTP cookies — those live in the
// server package.
package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

const (
	authorizeURL = "https://discord.com/oauth2/authorize"
	tokenURL     = "https://discord.com/api/oauth2/token"
	meURL        = "https://discord.com/api/users/@me"
	guildsURL    = "https://discord.com/api/users/@me/guilds"
)

type Client struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	GuildID      string
	HTTPClient   *http.Client // nil → http.DefaultClient
}

type DiscordUser struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	GlobalName string `json:"global_name"`
}

func (c *Client) AuthorizeURL(state string) string {
	v := url.Values{}
	v.Set("client_id", c.ClientID)
	v.Set("redirect_uri", c.RedirectURI)
	v.Set("response_type", "code")
	v.Set("scope", "identify guilds")
	v.Set("state", state)
	return authorizeURL + "?" + v.Encode()
}

func (c *Client) Exchange(ctx context.Context, code string) (string, error) {
	form := url.Values{}
	form.Set("client_id", c.ClientID)
	form.Set("client_secret", c.ClientSecret)
	form.Set("grant_type", "authorization_code")
	form.Set("code", code)
	form.Set("redirect_uri", c.RedirectURI)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("auth: build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", fmt.Errorf("auth: token exchange: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth: token exchange status %d", resp.StatusCode)
	}

	var body struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("auth: decode token response: %w", err)
	}
	if body.AccessToken == "" {
		return "", fmt.Errorf("auth: empty access_token in response")
	}
	return body.AccessToken, nil
}

func (c *Client) Me(ctx context.Context, accessToken string) (DiscordUser, error) {
	var u DiscordUser
	if err := c.getJSON(ctx, meURL, accessToken, &u); err != nil {
		return DiscordUser{}, err
	}
	return u, nil
}

func (c *Client) InGuild(ctx context.Context, accessToken string) (bool, error) {
	var guilds []struct {
		ID string `json:"id"`
	}
	if err := c.getJSON(ctx, guildsURL, accessToken, &guilds); err != nil {
		return false, err
	}
	for _, g := range guilds {
		if g.ID == c.GuildID {
			return true, nil
		}
	}
	return false, nil
}

func (c *Client) getJSON(ctx context.Context, url, accessToken string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("auth: build request %s: %w", url, err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return fmt.Errorf("auth: GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("auth: GET %s status %d", url, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("auth: decode %s: %w", url, err)
	}
	return nil
}

func (c *Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}
