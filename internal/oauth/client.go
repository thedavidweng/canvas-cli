package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

type Tokens struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

func (c Client) String() string { return "OAuth client (credentials redacted)" }
func (t Tokens) String() string { return "OAuth tokens (redacted)" }

func ValidateBaseURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return nil, fmt.Errorf("OAuth base URL must be an origin without credentials, query or fragment")
	}
	loopback := u.Hostname() == "127.0.0.1" || u.Hostname() == "::1"
	if u.Scheme != "https" && (u.Scheme != "http" || !loopback) {
		return nil, fmt.Errorf("OAuth requires HTTPS (HTTP is allowed only on a literal loopback address)")
	}
	return u, nil
}

func (c Client) Exchange(ctx context.Context, code, verifier string) (Tokens, error) {
	return c.request(ctx, url.Values{
		"grant_type": {"authorization_code"}, "code": {code},
		"code_verifier": {verifier}, "redirect_uri": {c.RedirectURI},
	})
}

func (c Client) Refresh(ctx context.Context, refreshToken string) (Tokens, error) {
	return c.request(ctx, url.Values{"grant_type": {"refresh_token"}, "refresh_token": {refreshToken}})
}

func (c Client) request(ctx context.Context, form url.Values) (Tokens, error) {
	if _, err := ValidateBaseURL(c.BaseURL); err != nil {
		return Tokens{}, err
	}
	if c.ClientID == "" {
		return Tokens{}, fmt.Errorf("OAuth client_id is required")
	}
	form.Set("client_id", c.ClientID)
	if c.ClientSecret != "" {
		form.Set("client_secret", c.ClientSecret)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(c.BaseURL, "/")+"/login/oauth2/token", strings.NewReader(form.Encode()))
	if err != nil {
		return Tokens{}, fmt.Errorf("could not construct OAuth token request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	resp, err := client.Do(req)
	if err != nil {
		return Tokens{}, fmt.Errorf("OAuth token request failed (network, cancellation or timeout)")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Tokens{}, fmt.Errorf("OAuth token request rejected (HTTP %d); check Developer Key, grant and redirect configuration", resp.StatusCode)
	}
	var tokens Tokens
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&tokens); err != nil || tokens.AccessToken == "" || tokens.ExpiresIn < 0 || tokens.ExpiresIn > int64((1<<63-1)/time.Second) {
		return Tokens{}, fmt.Errorf("invalid OAuth token response")
	}
	return tokens, nil
}
