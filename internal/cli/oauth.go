package cli

import (
	"context"
	"crypto/subtle"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
	"github.com/thedavidweng/canvas-cli/internal/oauth"
	"github.com/thedavidweng/canvas-cli/internal/output"
)

func oauthProfile(cmd *cobra.Command) (configPath string, stored *canvas.Config, profileName string, loadErr error) {
	path, _ := cmd.Flags().GetString("config")
	if path == "" {
		path = config.ConfigPath()
	}
	name, _ := cmd.Flags().GetString("profile")
	if name == "" {
		name = os.Getenv("CANVAS_PROFILE")
	}
	cfg, err := config.LoadConfig(path, name)
	if err != nil {
		return "", nil, "", fmt.Errorf("OAuth requires a readable 0600 config file with Developer Key settings")
	}
	name = cfg.CurrentProfile
	prof, ok := cfg.Profiles[name]
	if !ok || prof.OAuth == nil {
		return "", nil, "", fmt.Errorf("configure profile oauth.client_id and oauth.redirect_uri before OAuth login")
	}
	return path, cfg, name, nil
}

func oauthClient(prof *canvas.Profile) (oauth.Client, error) {
	if _, err := oauth.ValidateBaseURL(prof.BaseURL); err != nil {
		return oauth.Client{}, err
	}
	id, err := oauthEnv(prof.OAuth.ClientID)
	if err != nil {
		return oauth.Client{}, err
	}
	secret, err := oauthEnv(prof.OAuth.ClientSecret)
	if err != nil {
		return oauth.Client{}, err
	}
	if id == "" {
		return oauth.Client{}, fmt.Errorf("OAuth client_id is required")
	}
	return oauth.Client{BaseURL: prof.BaseURL, ClientID: id, ClientSecret: secret, RedirectURI: prof.OAuth.RedirectURI}, nil
}

func oauthEnv(value string) (string, error) {
	if after, ok := strings.CutPrefix(value, "env:"); ok {
		value = os.Getenv(after)
		if value == "" {
			return "", fmt.Errorf("OAuth credential environment variable is not set")
		}
	}
	return value, nil
}

func saveOAuthTokens(path string, cfg *canvas.Config, name string, tokens oauth.Tokens) error {
	prof := cfg.Profiles[name]
	prof.OAuth.AccessToken = tokens.AccessToken
	if tokens.RefreshToken != "" {
		prof.OAuth.RefreshToken = tokens.RefreshToken
	}
	prof.OAuth.ExpiresAt = ""
	if tokens.ExpiresIn > 0 {
		prof.OAuth.ExpiresAt = time.Now().UTC().Add(time.Duration(tokens.ExpiresIn) * time.Second).Format(time.RFC3339)
	}
	prof.Token = ""
	prof.Cookie = ""
	prof.CSRFToken = ""
	cfg.Profiles[name] = prof
	if err := writeConfigFile(path, cfg); err != nil {
		return fmt.Errorf("could not persist OAuth credentials; reauthorize before retrying")
	}
	return nil
}

func refreshOAuth(ctx context.Context, path, name string) (string, error) {
	cfg, err := config.LoadConfig(path, name)
	if err != nil {
		return "", fmt.Errorf("could not load OAuth credentials")
	}
	prof, ok := cfg.Profiles[name]
	if !ok || prof.OAuth == nil || prof.OAuth.RefreshToken == "" {
		return "", fmt.Errorf("OAuth refresh token is missing; run canvas auth oauth login")
	}
	client, err := oauthClient(&prof)
	if err != nil {
		return "", err
	}
	refreshToken, err := oauthEnv(prof.OAuth.RefreshToken)
	if err != nil {
		return "", err
	}
	tokens, err := client.Refresh(ctx, refreshToken)
	if err != nil {
		return "", err
	}
	if err := saveOAuthTokens(path, cfg, name, tokens); err != nil {
		return "", err
	}
	return tokens.AccessToken, nil
}

func configureOAuth(cfg *config.ResolvedConfig, path string) error {
	if cfg.OAuth == nil {
		return nil
	}
	origin, err := oauth.ValidateBaseURL(cfg.BaseURL)
	if err != nil {
		return err
	}
	if path == "" {
		path = config.ConfigPath()
	}
	transport := &oauth.Transport{Origin: origin, Token: cfg.Token, Base: http.DefaultTransport}
	if cfg.OAuth.RefreshToken != "" {
		transport.Refresh = func(ctx context.Context) (string, error) { return refreshOAuth(ctx, path, cfg.Profile) }
	}
	cfg.OAuthTransport = transport
	return nil
}

func newAuthOAuthCmd() *cobra.Command {
	parent := &cobra.Command{Use: "oauth", Short: "Authorize with a Canvas Developer Key and PKCE"}
	login := &cobra.Command{Use: "login", Short: "Authorize in your browser using a registered loopback callback", Long: `Authorize using OAuth2 authorization code and S256 PKCE.
Configure oauth.client_id and oauth.redirect_uri in the selected 0600 profile first.
Secretless login requires a public-client Developer Key. Confidential keys also
require oauth.client_secret (plain value or env:NAME). Register the exact HTTP
loopback URI, including its fixed port, with the institution. Open the printed
URL in your browser; this command waits for the callback and stores tokens.`, Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		path, cfg, name, err := oauthProfile(cmd)
		if err != nil {
			return oauthCommandError(cmd, err, "auth.oauth.login")
		}
		prof := cfg.Profiles[name]
		clearOAuthTokens(prof.OAuth)
		client, err := oauthClient(&prof)
		if err != nil {
			return oauthCommandError(cmd, err, "auth.oauth.login")
		}
		redirect, err := validateOAuthRedirect(client.RedirectURI)
		if err != nil {
			return oauthCommandError(cmd, err, "auth.oauth.login")
		}
		timeout, _ := cmd.Flags().GetDuration("login-timeout")
		if timeout <= 0 {
			return oauthCommandError(cmd, fmt.Errorf("login timeout must be positive"), "auth.oauth.login")
		}
		ctx, cancel := context.WithTimeout(cmd.Context(), timeout)
		defer cancel()
		pkce, err := oauth.NewPKCE()
		if err != nil {
			return oauthCommandError(cmd, fmt.Errorf("could not generate PKCE verifier"), "auth.oauth.login")
		}
		state, err := oauth.NewPKCE()
		if err != nil {
			return oauthCommandError(cmd, fmt.Errorf("could not generate OAuth state"), "auth.oauth.login")
		}
		code, err := receiveOAuthCode(ctx, cmd, client, redirect, pkce.Challenge, state.Verifier)
		if err != nil {
			return oauthCommandError(cmd, err, "auth.oauth.login")
		}
		tokens, err := client.Exchange(ctx, code, pkce.Verifier)
		if err != nil {
			return oauthCommandError(cmd, err, "auth.oauth.login")
		}
		cfg.CurrentProfile = name
		if err := saveOAuthTokens(path, cfg, name, tokens); err != nil {
			return oauthCommandError(cmd, err, "auth.oauth.login")
		}
		return oauthSuccess(cmd, "auth.oauth.login", name, prof.BaseURL, prof.OAuth)
	}}
	login.Flags().Duration("login-timeout", 5*time.Minute, "maximum time to wait for browser authorization and token exchange")
	parent.AddCommand(login)
	return parent
}

func validateOAuthRedirect(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "http" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Hostname() != "127.0.0.1" && u.Hostname() != "::1") || u.Path == "" || u.RawPath != "" {
		return nil, fmt.Errorf("OAuth redirect_uri must be an HTTP literal loopback URI with a fixed port and callback path")
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port < 1 || port > 65535 {
		return nil, fmt.Errorf("OAuth redirect_uri requires a registered fixed port (1-65535)")
	}
	return u, nil
}

type oauthCallback struct {
	code string
	err  error
}

func receiveOAuthCode(ctx context.Context, cmd *cobra.Command, client oauth.Client, redirect *url.URL, challenge, state string) (string, error) {
	listener, err := net.Listen("tcp", redirect.Host)
	if err != nil {
		return "", fmt.Errorf("could not bind registered OAuth loopback port")
	}
	result := make(chan oauthCallback, 1)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Referrer-Policy", "no-referrer")
		if r.Method != http.MethodGet || r.URL.Path != redirect.Path {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		reply := oauthCallback{code: q.Get("code")}
		switch {
		case len(q["state"]) != 1 || subtle.ConstantTimeCompare([]byte(q.Get("state")), []byte(state)) != 1:
			reply.err = fmt.Errorf("OAuth callback state mismatch")
		case q.Get("error") != "":
			reply.err = fmt.Errorf("OAuth authorization was denied")
		case len(q["code"]) != 1 || reply.code == "":
			reply.err = fmt.Errorf("OAuth callback did not include one authorization code")
		}
		select {
		case result <- reply:
		default:
			http.Error(w, "Callback already received", http.StatusConflict)
			return
		}
		if reply.err != nil {
			http.Error(w, "Authorization failed. Return to the terminal.", http.StatusBadRequest)
			return
		}
		fmt.Fprint(w, "Authorization received. Return to the terminal.")
	})
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 5 * time.Second}
	defer func() { _ = server.Shutdown(context.Background()) }()
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()
	authURL, _ := url.Parse(strings.TrimRight(client.BaseURL, "/") + "/login/oauth2/auth")
	authURL.RawQuery = url.Values{"client_id": {client.ClientID}, "redirect_uri": {client.RedirectURI}, "response_type": {"code"}, "state": {state}, "code_challenge": {challenge}, "code_challenge_method": {"S256"}}.Encode()
	fmt.Fprintf(cmd.ErrOrStderr(), "Open this URL in your browser:\n%s\n", authURL.String())
	select {
	case reply := <-result:
		return reply.code, reply.err
	case <-ctx.Done():
		return "", fmt.Errorf("OAuth login canceled or timed out")
	case err := <-serveErr:
		return "", fmt.Errorf("OAuth callback listener stopped: %w", err)
	}
}

func newAuthRefreshCmd() *cobra.Command {
	return &cobra.Command{Use: "refresh", Short: "Refresh OAuth credentials and save any rotated refresh token", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		path, cfg, name, err := oauthProfile(cmd)
		if err != nil {
			return oauthCommandError(cmd, err, "auth.refresh")
		}
		if _, err := refreshOAuth(cmd.Context(), path, name); err != nil {
			return oauthCommandError(cmd, err, "auth.refresh")
		}
		saved, err := config.LoadConfig(path, name)
		if err != nil {
			return oauthCommandError(cmd, fmt.Errorf("could not read refreshed OAuth status"), "auth.refresh")
		}
		return oauthSuccess(cmd, "auth.refresh", name, cfg.Profiles[name].BaseURL, saved.Profiles[name].OAuth)
	}}
}

func oauthCommandError(cmd *cobra.Command, err error, command string) error {
	if emitErr := writeErrorWithCode(cmd.OutOrStdout(), GetConfig(cmd.Context()), err, command, "CANVAS_AUTH_ERROR", "auth", isJSONMode(cmd)); emitErr != nil {
		return emitErr
	}
	return &oauthOutputError{message: err.Error()}
}

type oauthOutputError struct{ message string }

func (e *oauthOutputError) Error() string { return e.message }
func (e *oauthOutputError) ExitCode() int { return output.ExitCodeForCategory("auth") }

func oauthSuccess(cmd *cobra.Command, command, name, base string, o *canvas.OAuthConfig) error {
	return writeOutput(cmd.OutOrStdout(), GetConfig(cmd.Context()), map[string]any{"profile": name, "base_url": base, "auth_method": "oauth", "token_present": o.AccessToken != "", "refresh_token_present": o.RefreshToken != "", "expires_at": o.ExpiresAt}, command, isJSONMode(cmd), func(w io.Writer) error {
		_, err := fmt.Fprintf(w, "OAuth credentials saved to profile %q (expires: %s)\n", name, o.ExpiresAt)
		return err
	})
}
