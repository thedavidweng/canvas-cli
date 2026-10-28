package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
	"github.com/thedavidweng/canvas-cli/internal/oauth"
)

type oauthURLWriter struct {
	bytes.Buffer
	visit func(string)
}

func (w *oauthURLWriter) Write(p []byte) (int, error) {
	n, e := w.Buffer.Write(p)
	for line := range strings.SplitSeq(string(p), "\n") {
		if strings.Contains(line, "/login/oauth2/auth?") {
			w.visit(strings.TrimSpace(line))
		}
	}
	return n, e
}

func oauthFixture(t *testing.T, base string) (string, *canvas.Config) {
	t.Helper()
	for _, k := range []string{"CANVAS_TOKEN", "CANVAS_COOKIE", "CANVAS_BASE_URL", "CANVAS_PROFILE"} {
		t.Setenv(k, "")
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	cfg := &canvas.Config{CurrentProfile: "school", Profiles: map[string]canvas.Profile{
		"school": {BaseURL: base, DefaultCourse: "42", OAuth: &canvas.OAuthConfig{ClientID: "test-client", RedirectURI: "http://127.0.0.1:39999/oauth/callback"}},
		"other":  {BaseURL: "https://other.example", Token: "unrelated-token"},
	}}
	if err := writeConfigFile(path, cfg); err != nil {
		t.Fatal(err)
	}
	return path, cfg
}

func oauthRun(path string, args ...string) (stdoutText, stderrText string, execErr error) {
	cmd := NewRootCmd("test")
	var out, stderr bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)
	cmd.SetArgs(append([]string{"--config", path, "--json"}, args...))
	err := cmd.Execute()
	return out.String(), stderr.String(), err
}

func TestOAuthLoginCommandAndRead(t *testing.T) {
	for _, confidential := range []bool{false, true} {
		t.Run(fmt.Sprint(confidential), func(t *testing.T) {
			var challenge string
			var tokenCalls int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/login/oauth2/token":
					tokenCalls++
					_ = r.ParseForm()
					if r.Form.Get("grant_type") != "authorization_code" || r.Form.Get("code") != "test-code" || oauth.Challenge(r.Form.Get("code_verifier")) != challenge {
						t.Error("invalid PKCE exchange")
					}
					wantSecret := ""
					if confidential {
						wantSecret = "private-client-secret"
					}
					if r.Form.Get("client_secret") != wantSecret || r.Form.Get("client_id") != "test-client" {
						t.Error("wrong client credentials")
					}
					fmt.Fprint(w, `{"access_token":"oauth-access-secret","refresh_token":"oauth-refresh-secret","expires_in":3600}`)
				case "/api/v1/users/self":
					if r.Header.Get("Authorization") != "Bearer oauth-access-secret" {
						t.Error("read did not use saved OAuth access token")
					}
					fmt.Fprint(w, `{"id":"42","name":"Test User"}`)
				default:
					t.Error(r.URL.Path)
					http.NotFound(w, r)
				}
			}))
			defer server.Close()
			path, cfg := oauthFixture(t, server.URL)
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			redirect := "http://" + listener.Addr().String() + "/oauth/callback"
			listener.Close()
			prof := cfg.Profiles["school"]
			prof.OAuth.RedirectURI = redirect
			if confidential {
				t.Setenv("TEST_OAUTH_SECRET", "private-client-secret")
				prof.OAuth.ClientSecret = "env:TEST_OAUTH_SECRET"
			}
			cfg.Profiles["school"] = prof
			if err := writeConfigFile(path, cfg); err != nil {
				t.Fatal(err)
			}
			callbackDone := make(chan error, 1)
			stderr := &oauthURLWriter{visit: func(raw string) {
				u, e := url.Parse(raw)
				if e != nil {
					callbackDone <- e
					return
				}
				if u.Query().Get("code_challenge_method") != "S256" || u.Query().Get("response_type") != "code" {
					t.Error("invalid authorization parameters")
				}
				challenge = u.Query().Get("code_challenge")
				callback, _ := url.Parse(u.Query().Get("redirect_uri"))
				q := callback.Query()
				q.Set("state", u.Query().Get("state"))
				q.Set("code", "test-code")
				callback.RawQuery = q.Encode()
				go func() {
					resp, e := http.Get(callback.String())
					if e == nil {
						resp.Body.Close()
					}
					callbackDone <- e
				}()
			}}
			cmd := NewRootCmd("test")
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(stderr)
			cmd.SetArgs([]string{"--config", path, "--json", "auth", "oauth", "login", "--login-timeout", "5s"})
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			if err := <-callbackDone; err != nil {
				t.Fatal(err)
			}
			if tokenCalls != 1 {
				t.Fatalf("token calls %d", tokenCalls)
			}
			saved, err := config.LoadConfig(path, "")
			if err != nil {
				t.Fatal(err)
			}
			o := saved.Profiles["school"].OAuth
			if o.AccessToken != "oauth-access-secret" || o.RefreshToken != "oauth-refresh-secret" || o.ExpiresAt == "" {
				t.Fatal("tokens or expiry not persisted")
			}
			if saved.Profiles["other"].Token != "unrelated-token" || saved.Profiles["school"].DefaultCourse != "42" {
				t.Fatal("config not preserved")
			}
			if confidential && o.ClientSecret != "env:TEST_OAUTH_SECRET" {
				t.Fatal("client env reference not preserved")
			}
			text, stderrText, err := oauthRun(path, "auth", "test")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(text, "Test User") {
				t.Fatal(text)
			}
			status, _, err := oauthRun(path, "auth", "status")
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(status, `"auth_method":"oauth"`) {
				t.Fatal(status)
			}
			for _, secret := range []string{"oauth-access-secret", "oauth-refresh-secret", "private-client-secret", "test-code"} {
				if strings.Contains(out.String()+stderr.String()+text+stderrText+status, secret) {
					t.Fatal("OAuth secret leaked")
				}
			}
		})
	}
}

func TestOAuthRefreshCommandAndAutomaticRotation(t *testing.T) {
	for _, manual := range []bool{false, true} {
		t.Run(fmt.Sprint(manual), func(t *testing.T) {
			var refreshCalls, readCalls int
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/login/oauth2/token" {
					refreshCalls++
					_ = r.ParseForm()
					if r.Form.Get("grant_type") != "refresh_token" || r.Form.Get("refresh_token") != "old-refresh" {
						t.Error("invalid refresh grant")
					}
					fmt.Fprint(w, `{"access_token":"new-access","refresh_token":"rotated-refresh","expires_in":7200}`)
					return
				}
				readCalls++
				if r.Header.Get("Authorization") == "Bearer old-access" {
					w.WriteHeader(401)
					return
				}
				if r.Header.Get("Authorization") != "Bearer new-access" {
					t.Error("new access token missing")
				}
				fmt.Fprint(w, `{"id":"42","name":"Refreshed User"}`)
			}))
			defer server.Close()
			path, cfg := oauthFixture(t, server.URL)
			cfg.Profiles["school"].OAuth.AccessToken = "old-access"
			cfg.Profiles["school"].OAuth.RefreshToken = "old-refresh"
			if err := writeConfigFile(path, cfg); err != nil {
				t.Fatal(err)
			}
			args := []string{"auth", "test"}
			if manual {
				args = []string{"auth", "refresh"}
			}
			out, stderr, err := oauthRun(path, args...)
			if err != nil {
				t.Fatal(err)
			}
			saved, err := config.LoadConfig(path, "")
			if err != nil {
				t.Fatal(err)
			}
			if refreshCalls != 1 || saved.Profiles["school"].OAuth.RefreshToken != "rotated-refresh" {
				t.Fatal("rotation was not persisted")
			}
			if !manual && readCalls != 2 {
				t.Fatalf("read calls %d", readCalls)
			}
			if strings.Contains(out+stderr, "new-access") || strings.Contains(out+stderr, "rotated-refresh") {
				t.Fatal("refresh leaked tokens")
			}
		})
	}
}

func TestOAuthLoginFailureDoesNotSave(t *testing.T) {
	for _, scenario := range []string{"state", "denied", "token-error", "timeout"} {
		t.Run(scenario, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(400)
				fmt.Fprint(w, "authorization-code-secret verifier-secret access-secret refresh-secret")
			}))
			defer server.Close()
			path, cfg := oauthFixture(t, server.URL)
			l, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			cfg.Profiles["school"].OAuth.RedirectURI = "http://" + l.Addr().String() + "/oauth/callback"
			l.Close()
			if err := writeConfigFile(path, cfg); err != nil {
				t.Fatal(err)
			}
			stderr := &oauthURLWriter{visit: func(raw string) {
				if scenario == "timeout" {
					return
				}
				u, _ := url.Parse(raw)
				callback, _ := url.Parse(u.Query().Get("redirect_uri"))
				q := url.Values{"state": {u.Query().Get("state")}, "code": {"authorization-code-secret"}}
				if scenario == "state" {
					q.Set("state", "wrong")
				}
				if scenario == "denied" {
					q.Set("error", "access_denied-secret")
				}
				callback.RawQuery = q.Encode()
				go func() {
					resp, e := http.Get(callback.String())
					if e == nil {
						resp.Body.Close()
					}
				}()
			}}
			cmd := NewRootCmd("test")
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetErr(stderr)
			cmd.SetArgs([]string{"--config", path, "--json", "auth", "oauth", "login", "--login-timeout", oauthFailureTimeout(scenario)})
			err = cmd.Execute()
			if err == nil {
				t.Fatal("expected failure")
			}
			text := out.String() + stderr.String() + err.Error()
			for _, secret := range []string{"authorization-code-secret", "access_denied-secret", "verifier-secret", "access-secret", "refresh-secret"} {
				if strings.Contains(text, secret) {
					t.Fatal("OAuth failure leaked a secret")
				}
			}
			saved, e := config.LoadConfig(path, "")
			if e != nil {
				t.Fatal(e)
			}
			if saved.Profiles["school"].OAuth.AccessToken != "" {
				t.Fatal("failed login persisted token")
			}
		})
	}
}

func TestOAuthLogoutClearsTokens(t *testing.T) {
	path, cfg := oauthFixture(t, "https://canvas.example")
	cfg.Profiles["school"].OAuth.AccessToken = "access"
	cfg.Profiles["school"].OAuth.RefreshToken = "refresh"
	if err := writeConfigFile(path, cfg); err != nil {
		t.Fatal(err)
	}
	if _, _, err := oauthRun(path, "auth", "logout"); err != nil {
		t.Fatal(err)
	}
	saved, err := config.LoadConfig(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if saved.Profiles["school"].OAuth.AccessToken != "" || saved.Profiles["school"].OAuth.RefreshToken != "" {
		t.Fatal("logout retained OAuth tokens")
	}
}

func TestOAuthRefreshRetainsOmittedRefreshTokenAndEnvReferences(t *testing.T) {
	t.Setenv("TEST_REFRESH_TOKEN", "old-refresh-secret")
	t.Setenv("TEST_CLIENT_ID", "configured-client")
	t.Setenv("TEST_CLIENT_SECRET", "configured-secret")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		if r.Form.Get("refresh_token") != "old-refresh-secret" || r.Form.Get("client_id") != "configured-client" || r.Form.Get("client_secret") != "configured-secret" {
			t.Error("env credentials not resolved")
		}
		fmt.Fprint(w, `{"access_token":"new-access-secret","expires_in":3600}`)
	}))
	defer server.Close()
	path, cfg := oauthFixture(t, server.URL)
	o := cfg.Profiles["school"].OAuth
	o.ClientID = "env:TEST_CLIENT_ID"
	o.ClientSecret = "env:TEST_CLIENT_SECRET"
	o.RefreshToken = "env:TEST_REFRESH_TOKEN"
	if err := writeConfigFile(path, cfg); err != nil {
		t.Fatal(err)
	}
	out, stderr, err := oauthRun(path, "auth", "refresh")
	if err != nil {
		t.Fatal(err)
	}
	saved, err := config.LoadConfig(path, "")
	if err != nil {
		t.Fatal(err)
	}
	o = saved.Profiles["school"].OAuth
	if o.RefreshToken != "env:TEST_REFRESH_TOKEN" || o.ClientSecret != "env:TEST_CLIENT_SECRET" || o.ClientID != "env:TEST_CLIENT_ID" {
		t.Fatal("omitted refresh token or env references lost")
	}
	resolved, err := config.Resolve(config.Options{}, saved)
	if err != nil {
		t.Fatal(err)
	}
	text := out + stderr + o.String() + resolved.String()
	for _, secret := range []string{"old-refresh-secret", "configured-secret", "new-access-secret"} {
		if strings.Contains(text, secret) {
			t.Fatal("refresh/status/config leaked credential")
		}
	}
}

func TestOAuthFailureExitAndConfigPreservation(t *testing.T) {
	for _, scenario := range []string{"refresh-rejected", "missing-env", "invalid-redirect", "permissions", "malformed-config"} {
		t.Run(scenario, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(400)
				fmt.Fprint(w, "old-access-secret old-refresh-secret client-secret")
			}))
			defer server.Close()
			path, cfg := oauthFixture(t, server.URL)
			o := cfg.Profiles["school"].OAuth
			o.AccessToken = "old-access-secret"
			o.RefreshToken = "old-refresh-secret"
			args := []string{"auth", "refresh"}
			switch scenario {
			case "missing-env":
				t.Setenv("TEST_MISSING_OAUTH", "")
				o.ClientSecret = "env:TEST_MISSING_OAUTH"
			case "invalid-redirect":
				o.RedirectURI = "http://0.0.0.0:8765/oauth/callback"
				args = []string{"auth", "oauth", "login"}
			}
			if err := writeConfigFile(path, cfg); err != nil {
				t.Fatal(err)
			}
			if scenario == "permissions" {
				if runtime.GOOS == "windows" {
					t.Skip("Unix permissions")
				}
				if err := os.Chmod(path, 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "malformed-config" {
				if err := os.WriteFile(path, []byte("oauth: [client-secret"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			before, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			out, stderr, err := oauthRun(path, args...)
			if err == nil {
				t.Fatal("expected failure")
			}
			exit, ok := err.(interface{ ExitCode() int })
			if !ok || exit.ExitCode() != 3 {
				t.Fatalf("expected auth exit 3, got %v", err)
			}
			var env canvas.Envelope
			if e := json.Unmarshal([]byte(out), &env); e != nil || env.OK {
				t.Fatal("expected error envelope")
			}
			if !strings.Contains(out, "CANVAS_AUTH_ERROR") || stderr != "" {
				t.Fatal("wrong JSON failure output")
			}
			for _, secret := range []string{"old-access-secret", "old-refresh-secret", "client-secret"} {
				if strings.Contains(out+stderr+err.Error(), secret) {
					t.Fatal("secret leaked")
				}
			}
			after, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("failed auth modified config")
			}
		})
	}
}

func TestOAuthAPIRefreshFailureRedaction(t *testing.T) {
	var refreshes int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login/oauth2/token" {
			refreshes++
			w.WriteHeader(400)
			fmt.Fprint(w, "api-access-secret api-refresh-secret client-secret")
			return
		}
		w.WriteHeader(401)
	}))
	defer server.Close()
	path, cfg := oauthFixture(t, server.URL)
	o := cfg.Profiles["school"].OAuth
	o.AccessToken = "api-access-secret"
	o.RefreshToken = "api-refresh-secret"
	if err := writeConfigFile(path, cfg); err != nil {
		t.Fatal(err)
	}
	out, stderr, err := oauthRun(path, "auth", "test")
	text := out + stderr
	if err != nil {
		text += err.Error()
	}
	if refreshes != 1 || !strings.Contains(text, "OAuth token request rejected") {
		t.Fatal("refresh failure missing")
	}
	for _, secret := range []string{"api-access-secret", "api-refresh-secret", "client-secret"} {
		if strings.Contains(text, secret) {
			t.Fatal("refresh failure leaked")
		}
	}
}

func oauthFailureTimeout(scenario string) string {
	if scenario == "timeout" {
		return "30ms"
	}
	return "2s"
}
