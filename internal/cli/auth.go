package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

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

			if jsonMode {
				env := output.NewSuccess(map[string]any{
					"profile":       cfg.Profile,
					"base_url":      cfg.BaseURL,
					"token_present": tokenPresent,
				}, "auth.status", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human output
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Profile:  %s\n", cfg.Profile)
			fmt.Fprintf(w, "Base URL: %s\n", cfg.BaseURL)
			tokenStr := "no"
			if tokenPresent {
				tokenStr = "yes"
			}
			fmt.Fprintf(w, "Token:    %s\n", tokenStr)
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
				env := canvas.NormalizeError(resp, "auth.test")
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

Use --token-stdin to read the token from stdin (safe for scripting).
Use --token-env to reference an environment variable name.
Do NOT use --token flag to avoid tokens in shell history.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL, _ := cmd.Flags().GetString("base-url")
			tokenStdin, _ := cmd.Flags().GetBool("token-stdin")
			tokenEnv, _ := cmd.Flags().GetString("token-env")
			configPath, _ := cmd.Flags().GetString("config")

			if configPath == "" {
				configPath = config.XDGConfigPath()
			}

			// Read token
			var token string
			switch {
			case tokenStdin:
				reader := bufio.NewReader(cmd.InOrStdin())
				line, err := reader.ReadString('\n')
				if err != nil && err != io.EOF {
					return fmt.Errorf("failed to read token from stdin: %w", err)
				}
				token = strings.TrimSpace(line)
			case tokenEnv != "":
				token = "env:" + tokenEnv
			default:
				return fmt.Errorf("must specify --token-stdin or --token-env")
			}

			if baseURL == "" {
				cfg := GetConfig(cmd.Context())
				if cfg != nil {
					baseURL = cfg.BaseURL
				}
				if baseURL == "" {
					return fmt.Errorf("--base-url is required")
				}
			}

			// Normalize base URL
			baseURL = strings.TrimRight(baseURL, "/")
			baseURL = strings.TrimSuffix(baseURL, "/api/v1")

			// Load or create config
			existingCfg, _ := config.LoadConfig(configPath, "")
			if existingCfg == nil {
				existingCfg = &canvas.Config{
					CurrentProfile: "default",
					Profiles:       make(map[string]canvas.Profile),
				}
			}

			profileName := existingCfg.CurrentProfile
			if profileName == "" {
				profileName = "default"
				existingCfg.CurrentProfile = profileName
			}

			prof := existingCfg.Profiles[profileName]
			prof.BaseURL = baseURL
			prof.Token = token
			existingCfg.Profiles[profileName] = prof

			// Write config
			if err := writeConfigFile(configPath, existingCfg); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Credentials saved to profile %q in %s\n", profileName, configPath)
			return nil
		},
	}

	cmd.Flags().String("base-url", "", "Canvas instance base URL")
	cmd.Flags().Bool("token-stdin", false, "read token from stdin")
	cmd.Flags().String("token-env", "", "name of environment variable containing token")
	cmd.Flags().String("config", "", "config file path")

	return cmd
}

// newAuthLogoutCmd returns `auth logout`.
func newAuthLogoutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logout",
		Short: "Remove token from current profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			configPath, _ := cmd.Flags().GetString("config")
			if configPath == "" {
				configPath = config.XDGConfigPath()
			}

			existingCfg, err := config.LoadConfig(configPath, "")
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			prof := existingCfg.Profiles[cfg.Profile]
			prof.Token = ""
			existingCfg.Profiles[cfg.Profile] = prof

			if err := writeConfigFile(configPath, existingCfg); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}

			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Token removed from profile %q\n", cfg.Profile)
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
				configPath = config.XDGConfigPath()
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
				configPath = config.XDGConfigPath()
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

// Ensure context is used (imported but may not be directly referenced in all paths).
var _ = context.Background
