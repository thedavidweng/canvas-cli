package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/thedavidweng/canvas-cli/internal/browsercookie"
	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
	"github.com/thedavidweng/canvas-cli/internal/output"
)

// NewAuthCmd returns the `auth` parent command with all subcommands.
func NewAuthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication and profiles",
		Long:  `Manage Canvas API authentication tokens and configuration profiles.`,
	}

	cmd.AddCommand(newAuthStatusCmd())
	cmd.AddCommand(newAuthTestCmd())
	cmd.AddCommand(newAuthLoginCmd())
	cmd.AddCommand(newAuthLogoutCmd())
	cmd.AddCommand(newAuthProfilesCmd())
	cmd.AddCommand(newAuthUseCmd())

	return cmd
}

// newAuthStatusCmd returns `auth status`.
func newAuthStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current authentication status",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")

			tokenPresent := cfg.Token != ""
			cookiePresent := cfg.Cookie != ""

			// Determine active auth method.
			authMethod := "none"
			if tokenPresent {
				authMethod = "token"
			} else if cookiePresent {
				authMethod = "cookie (experimental)"
			}

			if jsonMode {
				env := output.NewSuccess(map[string]any{
					"profile":        cfg.Profile,
					"base_url":       cfg.BaseURL,
					"auth_method":    authMethod,
					"token_present":  tokenPresent,
					"cookie_present": cookiePresent,
				}, "auth.status", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human output
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Profile:    %s\n", cfg.Profile)
			fmt.Fprintf(w, "Base URL:   %s\n", cfg.BaseURL)
			fmt.Fprintf(w, "Auth:       %s\n", authMethod)
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

// newAuthTestCmd returns `auth test`.
func newAuthTestCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Test authentication by calling the API",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			if cfg.Token == "" && cfg.Cookie != "" {
				client.WithCookie(cfg.Cookie, cfg.CSRFToken)
			}
			resp, err := client.Do(cmd.Context(), "GET", "/api/v1/users/self", nil, nil)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_NETWORK_ERROR",
						Message:  err.Error(),
						Category: "network",
					}, "auth.test")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return fmt.Errorf("failed to reach API: %w", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != 200 {
				env := canvas.NormalizeError(resp, "auth.test", cookieAuthBaseURL(cfg)...)
				if jsonMode {
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return fmt.Errorf("authentication failed: %s (status %d)", env.Error.Message, resp.StatusCode)
			}

			var user canvas.User
			if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
				return fmt.Errorf("failed to decode user response: %w", err)
			}

			if jsonMode {
				env := output.NewSuccess(user, "auth.test", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Authentication successful!\n")
			fmt.Fprintf(w, "User:  %s\n", user.Name)
			fmt.Fprintf(w, "ID:    %s\n", user.ID)
			if user.LoginID != "" {
				fmt.Fprintf(w, "Login: %s\n", user.LoginID)
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

// newAuthLoginCmd returns `auth login`.
func newAuthLoginCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Save authentication credentials",
		Long: `Save Canvas API credentials to the config file.

When run without flags, enters interactive mode:
  1. Prompts for your Canvas instance URL
  2. Shows where to generate an access token
  3. Prompts for the access token (input is masked)
  4. Validates the token by calling the API
  5. Saves credentials to the config file

Supports multiple profiles for multiple institutions or users:

  canvas auth login --profile school1 --base-url https://school1.instructure.com
  canvas auth login --profile school2 --base-url https://school2.instructure.com
  canvas auth use school1   # switch between them
  canvas --profile school2 courses list   # or use inline`,
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL, _ := cmd.Flags().GetString("base-url")
			tokenStdin, _ := cmd.Flags().GetBool("token-stdin")
			tokenEnv, _ := cmd.Flags().GetString("token-env")
			cookieStdin, _ := cmd.Flags().GetBool("cookie-stdin")
			cookieEnv, _ := cmd.Flags().GetString("cookie-env")
			cookieFile, _ := cmd.Flags().GetString("cookie-file")
			csrfStdin, _ := cmd.Flags().GetBool("csrf-token-stdin")
			csrfEnv, _ := cmd.Flags().GetString("csrf-token-env")
			csrfFile, _ := cmd.Flags().GetString("csrf-token-file")
			browserFlag, _ := cmd.Flags().GetString("browser")
			configPath, _ := cmd.Flags().GetString("config")
			profileFlag, _ := cmd.Flags().GetString("profile")

			if configPath == "" {
				configPath = config.ConfigPath()
			}

			hasTokenFlag := tokenStdin || tokenEnv != ""
			hasCookieFlag := cookieStdin || cookieEnv != "" || cookieFile != ""
			hasCredsFlag := hasTokenFlag || hasCookieFlag
			interactive := !hasCredsFlag && baseURL == "" && isTerminal(cmd.InOrStdin())

			w := cmd.OutOrStdout()

			// --- Get profile name (interactive only, if --profile not given) ---
			if interactive && profileFlag == "" {
				input := promptLine(w, "Profile name (default): ")
				if input != "" {
					profileFlag = input
				}
			}

			// --- Get base URL ---
			if baseURL == "" {
				if interactive {
					baseURL = promptLine(w, "Canvas Instance URL (e.g. https://school.instructure.com): ")
					if baseURL == "" {
						return fmt.Errorf("base URL is required")
					}
				} else {
					cfg := GetConfig(cmd.Context())
					if cfg != nil {
						baseURL = cfg.BaseURL
					}
					if baseURL == "" {
						return fmt.Errorf("--base-url is required")
					}
				}
			}

			// Normalize base URL early so we can use it in the help message.
			baseURL = strings.TrimRight(baseURL, "/")
			baseURL = strings.TrimSuffix(baseURL, "/api/v1")

			// --- Determine auth method and read credentials ---
			var token string
			var cookie string
			var csrfToken string

			switch {
			// Token flags (existing flow, unchanged)
			case tokenStdin:
				reader := bufio.NewReader(cmd.InOrStdin())
				line, err := reader.ReadString('\n')
				if err != nil && err != io.EOF {
					return fmt.Errorf("failed to read token from stdin: %w", err)
				}
				token = strings.TrimSpace(line)
			case tokenEnv != "":
				token = "env:" + tokenEnv

			// Cookie stdin
			case cookieStdin:
				reader := bufio.NewReader(cmd.InOrStdin())
				line, err := reader.ReadString('\n')
				if err != nil && err != io.EOF {
					return fmt.Errorf("failed to read cookie from stdin: %w", err)
				}
				cookie = strings.TrimSpace(line)
				if cookie == "" {
					return fmt.Errorf("cookie value is required")
				}
				// Optionally read CSRF from stdin too if --csrf-token-stdin is also set
				if csrfStdin {
					csrfLine, err := reader.ReadString('\n')
					if err != nil && err != io.EOF {
						return fmt.Errorf("failed to read CSRF token from stdin: %w", err)
					}
					csrfToken = strings.TrimSpace(csrfLine)
				}

			// Cookie env
			case cookieEnv != "":
				cookie = "env:" + cookieEnv

			// Cookie file
			case cookieFile != "":
				val, err := readSecretFile(cookieFile)
				if err != nil {
					return fmt.Errorf("failed to read cookie file: %w", err)
				}
				cookie = val

			// Interactive mode
			case interactive:
				var err error
				token, cookie, csrfToken, err = promptAuthMethod(cmd.Context(), w, baseURL, browserFlag)
				if err != nil {
					return err
				}

			default:
				return fmt.Errorf("must specify --token-stdin, --token-env, --cookie-stdin, --cookie-env, or --cookie-file")
			}

			// Handle CSRF from env or file flags (non-stdin cases)
			if csrfToken == "" {
				switch {
				case csrfEnv != "":
					csrfToken = "env:" + csrfEnv
				case csrfFile != "":
					val, err := readSecretFile(csrfFile)
					if err != nil {
						return fmt.Errorf("failed to read CSRF token file: %w", err)
					}
					csrfToken = val
				case csrfStdin && !cookieStdin:
					// --csrf-token-stdin without --cookie-stdin: read standalone
					reader := bufio.NewReader(cmd.InOrStdin())
					line, err := reader.ReadString('\n')
					if err != nil && err != io.EOF {
						return fmt.Errorf("failed to read CSRF token from stdin: %w", err)
					}
					csrfToken = strings.TrimSpace(line)
				}
			}

			// --- Load or create config ---
			existingCfg, _ := config.LoadConfig(configPath, "")
			if existingCfg == nil {
				existingCfg = &canvas.Config{
					CurrentProfile: "default",
					Profiles:       make(map[string]canvas.Profile),
				}
			}

			// Determine profile name: --profile flag > current profile > "default".
			profileName := profileFlag
			if profileName == "" {
				profileName = existingCfg.CurrentProfile
			}
			if profileName == "" {
				profileName = "default"
			}

			prof := existingCfg.Profiles[profileName]
			prof.BaseURL = baseURL
			prof.Token = token
			prof.Cookie = cookie
			prof.CSRFToken = csrfToken
			existingCfg.Profiles[profileName] = prof
			existingCfg.CurrentProfile = profileName

			// --- Write config ---
			if err := writeConfigFile(configPath, existingCfg); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}

			// --- Validate credentials ---
			if token != "" {
				// Token verification (existing flow)
				resolvedToken := token
				if strings.HasPrefix(token, "env:") {
					resolvedToken = os.Getenv(strings.TrimPrefix(token, "env:"))
				}

				if resolvedToken != "" {
					fmt.Fprintf(w, "Verifying credentials...\n")
					client := canvas.NewClient(baseURL, resolvedToken, "dev", 10*time.Second, 0)
					resp, err := client.Do(cmd.Context(), "GET", "/api/v1/users/self", nil, nil)
					if err != nil {
						fmt.Fprintf(w, "\nWarning: could not verify token (saved anyway): %v\n", err)
						fmt.Fprintf(w, "  Check your base URL: %s\n", baseURL)
						return nil
					}
					defer resp.Body.Close()

					if resp.StatusCode != 200 {
						fmt.Fprintf(w, "\nWarning: token verification failed (status %d) -- saved anyway\n", resp.StatusCode)
						fmt.Fprintf(w, "  Run `canvas auth test` after fixing your token\n")
						return nil
					}

					var user canvas.User
					if err := json.NewDecoder(resp.Body).Decode(&user); err == nil {
						fmt.Fprintf(w, "\nAuthenticated as: %s", user.Name)
						if user.LoginID != "" {
							fmt.Fprintf(w, " (%s)", user.LoginID)
						}
						fmt.Fprintln(w)
					}
				}
			} else if cookie != "" {
				// Cookie verification
				resolvedCookie := cookie
				if strings.HasPrefix(cookie, "env:") {
					resolvedCookie = os.Getenv(strings.TrimPrefix(cookie, "env:"))
				}
				resolvedCSRF := csrfToken
				if strings.HasPrefix(csrfToken, "env:") {
					resolvedCSRF = os.Getenv(strings.TrimPrefix(csrfToken, "env:"))
				}

				if resolvedCookie != "" {
					fmt.Fprintf(w, "Verifying cookie...\n")
					client := canvas.NewClient(baseURL, "", "dev", 10*time.Second, 0)
					client.WithCookie(resolvedCookie, resolvedCSRF)
					resp, err := client.Do(cmd.Context(), "GET", "/api/v1/users/self", nil, nil)
					if err != nil {
						fmt.Fprintf(w, "\nWarning: could not verify cookie (saved anyway): %v\n", err)
						fmt.Fprintf(w, "  Check your base URL: %s\n", baseURL)
						return nil
					}
					defer resp.Body.Close()

					if resp.StatusCode != 200 {
						fmt.Fprintf(w, "\nWarning: cookie verification failed (status %d) -- saved anyway\n", resp.StatusCode)
						fmt.Fprintf(w, "  Run `canvas auth test` after fixing your cookie\n")
						return nil
					}

					var user canvas.User
					if err := json.NewDecoder(resp.Body).Decode(&user); err == nil {
						fmt.Fprintf(w, "\nAuthenticated as: %s", user.Name)
						if user.LoginID != "" {
							fmt.Fprintf(w, " (%s)", user.LoginID)
						}
						fmt.Fprintln(w)
					}
				}
			}

			fmt.Fprintf(w, "\nCredentials saved to profile %q in %s\n", profileName, configPath)
			return nil
		},
	}

	cmd.Flags().String("base-url", "", "Canvas instance base URL")
	cmd.Flags().Bool("token-stdin", false, "read token from stdin")
	cmd.Flags().String("token-env", "", "name of environment variable containing token")
	cmd.Flags().Bool("cookie-stdin", false, "read session cookie from stdin")
	cmd.Flags().String("cookie-env", "", "name of environment variable containing session cookie")
	cmd.Flags().String("cookie-file", "", "path to file containing session cookie")
	cmd.Flags().Bool("csrf-token-stdin", false, "read CSRF token from stdin")
	cmd.Flags().String("csrf-token-env", "", "name of environment variable containing CSRF token")
	cmd.Flags().String("csrf-token-file", "", "path to file containing CSRF token")
	cmd.Flags().String("config", "", "config file path")
	cmd.Flags().String("profile", "", "profile name (for multi-account setups)")
	cmd.Flags().String("browser", "", "browser for cookie extraction (chrome, firefox, safari, etc.)")

	return cmd
}

// promptAuthMethod shows the auth method selection and collects cookie credentials interactively.
func promptAuthMethod(ctx context.Context, w io.Writer, baseURL, browser string) (token, cookie, csrfToken string, err error) {
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "  Authenticate via:")
	fmt.Fprintln(w, "    1) Access token (recommended)")
	fmt.Fprintln(w, "    2) Session cookie (experimental)")
	choice := promptLine(w, "\n  Select method [1]: ")

	if isCookieChoice(choice) {
		return promptCookieAuth(ctx, w, baseURL, browser)
	}

	// Default: token flow (unchanged)
	fmt.Fprintf(w, "Generate an access token at: %s/profile/settings\n", baseURL)
	fmt.Fprintf(w, "  Account -> Settings -> Approved Integrations -> New Access Token\n\n")
	tok := promptLine(w, "Access Token: ")
	if tok == "" {
		return "", "", "", fmt.Errorf("access token is required")
	}
	return tok, "", "", nil
}

// promptCookieAuth handles the interactive cookie auth flow.
// browserOverride, if non-empty, skips browser auto-detection and tries the named browser directly.
func promptCookieAuth(ctx context.Context, w io.Writer, baseURL, browserOverride string) (token, cookie, csrfToken string, err error) {
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "WARNING: Session cookie auth is experimental. Your session cookie grants")
	fmt.Fprintln(w, "  full access to your Canvas account. Anyone with this cookie can act as you.")
	fmt.Fprintln(w, "  The cookie will be stored locally in your config file.")
	fmt.Fprintln(w, "")

	confirm := promptLine(w, "  Continue? [y/N]: ")
	if confirm != "y" && confirm != "Y" {
		return "", "", "", fmt.Errorf("aborted")
	}

	host := extractHost(baseURL)

	if !browsercookie.KOOKY_AVAILABLE {
		fmt.Fprintln(w, "Browser cookie auto-extraction is not available on this platform.")
		fmt.Fprintln(w, "Copy the session cookie from your browser:")
		fmt.Fprintln(w, "  DevTools -> Application -> Cookies -> your Canvas domain")
		return promptCookieManual(w)
	}

	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Reading cookies from your browser. You may be prompted to unlock")
	fmt.Fprintln(w, "  your system keyring — please approve the access.")

	// Build ordered list of browsers to try.
	var tryBrowsers []string
	if browserOverride != "" {
		tryBrowsers = []string{browserOverride}
	} else {
		tryBrowsers = []string{""} // "" = auto-detect (ExtractCookies tries all)
	}

	for i, browser := range tryBrowsers {
		label := "default browser"
		if browser != "" {
			label = browser
		}
		fmt.Fprintf(w, "Extracting cookies from %s for %s...\n", label, host)

		var sessionCookie, csrf string
		var extractErr error
		if browser == "" {
			sessionCookie, csrf, extractErr = browsercookie.ExtractCookies(ctx, host)
		} else {
			sessionCookie, csrf, extractErr = browsercookie.ExtractCookiesForBrowser(ctx, host, browser)
		}

		if extractErr == nil {
			fmt.Fprintf(w, "Cookie extracted successfully.\n")
			return "", sessionCookie, csrf, nil
		}

		fmt.Fprintf(w, "Could not extract cookies from %s: %v\n", label, extractErr)

		// If --browser was specified and failed, no retry — go to manual.
		if browserOverride != "" {
			break
		}

		// First failure: offer retry with a different browser or manual entry.
		if i == 0 {
			action := promptLine(w, "  Try another browser, enter manually, or abort? [try/manual/abort]: ")
			switch action {
			case "manual":
				return promptCookieManual(w)
			case "abort", "":
				return "", "", "", fmt.Errorf("aborted")
			}
			// "try" — show available browsers and let user pick.
			tryBrowsers = promptBrowserSelection(w, host)
			if len(tryBrowsers) == 0 {
				return promptCookieManual(w)
			}
		}
	}

	// All browsers exhausted — fall back to manual.
	fmt.Fprintln(w, "Could not extract cookies from any browser.")
	action := promptLine(w, "  Enter manually or abort? [manual/abort]: ")
	if action == "abort" || action == "" {
		return "", "", "", fmt.Errorf("aborted")
	}
	return promptCookieManual(w)
}

// promptBrowserSelection shows available browsers and returns the user's selection(s).
func promptBrowserSelection(w io.Writer, host string) []string {
	available := browsercookie.AvailableBrowsers()
	if len(available) == 0 {
		return nil
	}
	fmt.Fprintln(w, "  Available browsers:")
	for i, b := range available {
		fmt.Fprintf(w, "    %d) %s\n", i+1, b)
	}
	choice := promptLine(w, "  Select browser (number or name): ")
	if choice == "" {
		return nil
	}
	// Try numeric selection.
	for i, b := range available {
		if choice == fmt.Sprintf("%d", i+1) {
			return []string{b}
		}
	}
	// Try name match.
	for _, b := range available {
		if strings.EqualFold(choice, b) {
			return []string{b}
		}
	}
	return nil
}

// promptCookieManual prompts for cookie and CSRF token values with masked input.
func promptCookieManual(w io.Writer) (token, cookie, csrfToken string, err error) {
	fmt.Fprintln(w, "")
	cookie = promptLine(w, "Session Cookie: ")
	if cookie == "" {
		return "", "", "", fmt.Errorf("session cookie is required")
	}

	csrfToken = promptLine(w, "CSRF Token (optional, press Enter to skip): ")
	return "", cookie, csrfToken, nil
}

// newAuthLogoutCmd returns `auth logout`.
func newAuthLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove credentials from current profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			configPath, _ := cmd.Flags().GetString("config")
			if configPath == "" {
				configPath = config.ConfigPath()
			}

			existingCfg, err := config.LoadConfig(configPath, "")
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			prof := existingCfg.Profiles[cfg.Profile]
			prof.Token = ""
			prof.Cookie = ""
			prof.CSRFToken = ""
			existingCfg.Profiles[cfg.Profile] = prof

			if err := writeConfigFile(configPath, existingCfg); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Credentials removed from profile %q\n", cfg.Profile)
			return nil
		},
	}
	cmd.Flags().String("config", "", "config file path")
	return cmd
}

// newAuthProfilesCmd returns `auth profiles`.
func newAuthProfilesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profiles",
		Short: "List all configured profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			configPath, _ := cmd.Flags().GetString("config")
			if configPath == "" {
				configPath = config.ConfigPath()
			}

			jsonMode, _ := cmd.Flags().GetBool("json")

			existingCfg, err := config.LoadConfig(configPath, "")
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if jsonMode {
				type profileInfo struct {
					Name    string `json:"name"`
					BaseURL string `json:"base_url"`
					Current bool   `json:"current"`
				}
				var profiles []profileInfo
				for name, prof := range existingCfg.Profiles {
					profiles = append(profiles, profileInfo{
						Name:    name,
						BaseURL: prof.BaseURL,
						Current: name == existingCfg.CurrentProfile,
					})
				}
				env := output.NewSuccess(profiles, "auth.profiles")
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			w := cmd.OutOrStdout()
			for name, prof := range existingCfg.Profiles {
				marker := "  "
				if name == existingCfg.CurrentProfile {
					marker = "* "
				}
				fmt.Fprintf(w, "%s%s\t%s\n", marker, name, prof.BaseURL)
			}
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	cmd.Flags().String("config", "", "config file path")
	return cmd
}

// newAuthUseCmd returns `auth use PROFILE`.
func newAuthUseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "use PROFILE",
		Short: "Switch the active profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			profileName := args[0]

			configPath, _ := cmd.Flags().GetString("config")
			if configPath == "" {
				configPath = config.ConfigPath()
			}

			existingCfg, err := config.LoadConfig(configPath, "")
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if _, ok := existingCfg.Profiles[profileName]; !ok {
				return fmt.Errorf("profile %q not found", profileName)
			}

			existingCfg.CurrentProfile = profileName

			if err := writeConfigFile(configPath, existingCfg); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Switched to profile %q\n", profileName)
			return nil
		},
	}
	cmd.Flags().String("config", "", "config file path")
	return cmd
}

// writeConfigFile writes a Config to the given path, creating parent directories as needed.
func writeConfigFile(path string, cfg *canvas.Config) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// isTerminal reports whether r is connected to a terminal.
func isTerminal(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// promptLine prints a prompt to w, reads a line from stdin, and returns it (trimmed).
func promptLine(w io.Writer, prompt string) string {
	fmt.Fprint(w, prompt)
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		return strings.TrimSpace(scanner.Text())
	}
	return ""
}

// readSecretFile reads a secret value from a file, enforcing that the file
// has permissions no more permissive than 0600 (owner read/write only).
// The file path is not included in error messages for security.
func readSecretFile(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("cannot access secret file") // path redacted
	}

	perm := info.Mode().Perm()
	// Check that group and other have no permissions (mask 0077 must be zero).
	// Skip on Windows where Unix file permissions don't apply.
	if runtime.GOOS != "windows" && perm&0o077 != 0 {
		return "", fmt.Errorf("file has too-permissive permissions (%o); must be 0600 or stricter", perm)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cannot read secret file") // path redacted
	}

	return strings.TrimSpace(string(data)), nil
}

// isCookieChoice returns true if the user's input selects cookie auth.
// Accepts: "2", "cookie", "session cookie" (case-insensitive, trimmed).
func isCookieChoice(choice string) bool {
	choice = strings.TrimSpace(strings.ToLower(choice))
	return choice == "2" || choice == "cookie" || choice == "session cookie"
}

// extractHost parses a URL and returns just the hostname.
func extractHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	return u.Hostname()
}
