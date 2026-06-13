package cli

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
	"github.com/thedavidweng/canvas-cli/internal/output"
)

// DoctorCheck represents a single diagnostic check result.
type DoctorCheck struct {
	Check   string `json:"check"`
	Status  string `json:"status"` // "pass", "fail", "warn"
	Message string `json:"message"`
}

// NewDoctorCmd returns the `doctor` command for the root command.
func NewDoctorCmd() *cobra.Command {
	return newDoctorCmd()
}

// newDoctorCmd returns the `doctor` command.
func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Validate CLI environment and configuration",
		Long:  `Check config, auth, base URL, token presence, API reachability, and write-safety settings.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			jsonMode, _ := cmd.Flags().GetBool("json")
			timeoutStr, _ := cmd.Flags().GetString("timeout")

			timeout := 10 * time.Second
			if timeoutStr != "" {
				if d, err := time.ParseDuration(timeoutStr); err == nil {
					timeout = d
				}
			}

			checks := make([]DoctorCheck, 0, 7)

			// 1. Config file check
			checks = append(checks, checkConfigFile())

			// 2. Config permissions check
			checks = append(checks, checkConfigPermissions())

			// 3. Token present check
			checks = append(checks, checkTokenPresent(cfg))

			// 4. Base URL check
			checks = append(checks, checkBaseURL(cfg))

			// 5. API reachable check
			checks = append(checks, checkAPIReachable(cfg, timeout))

			// 6. Token valid check (same as API reachable for token auth)
			checks = append(checks, checkTokenValid(cfg, timeout))

			// 7. Write safety check
			checks = append(checks, checkWriteSafety(cfg))

			if jsonMode {
				ok := true
				for _, c := range checks {
					if c.Status == "fail" {
						ok = false
						break
					}
				}
				env := output.NewSuccess(checks, "doctor")
				env.OK = ok
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human output
			w := cmd.OutOrStdout()
			for _, c := range checks {
				var icon string
				switch c.Status {
				case "pass":
					icon = "ok"
				case "fail":
					icon = "X"
				case "warn":
					icon = "!"
				}
				msg := c.Message
				if msg == "" {
					msg = c.Status
				}
				fmt.Fprintf(w, "  [%s] %-20s %s\n", icon, c.Check, msg)
			}
			return nil
		},
	}

	cmd.Flags().Bool("json", false, "output JSON envelope")
	cmd.Flags().String("timeout", "", "timeout for API checks (e.g. 10s)")

	return cmd
}

func checkConfigFile() DoctorCheck {
	configPath := config.XDGConfigPath()
	_, err := os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return DoctorCheck{
				Check:   "config_file",
				Status:  "warn",
				Message: "config file not found (env vars may suffice)",
			}
		}
		return DoctorCheck{
			Check:   "config_file",
			Status:  "fail",
			Message: fmt.Sprintf("cannot read config: %v", err),
		}
	}
	return DoctorCheck{
		Check:   "config_file",
		Status:  "pass",
		Message: "config file exists",
	}
}

func checkConfigPermissions() DoctorCheck {
	configPath := config.XDGConfigPath()
	info, err := os.Stat(configPath)
	if err != nil {
		return DoctorCheck{
			Check:   "config_permissions",
			Status:  "warn",
			Message: "config file not found, skipping permission check",
		}
	}
	perm := info.Mode().Perm()
	if perm != 0o600 {
		return DoctorCheck{
			Check:   "config_permissions",
			Status:  "warn",
			Message: fmt.Sprintf("config file has permissions %o (expected 0600)", perm),
		}
	}
	return DoctorCheck{
		Check:   "config_permissions",
		Status:  "pass",
		Message: "config file has correct permissions (0600)",
	}
}

func checkTokenPresent(cfg *config.ResolvedConfig) DoctorCheck {
	if cfg == nil || cfg.Token == "" {
		return DoctorCheck{
			Check:   "token_present",
			Status:  "fail",
			Message: "no token configured",
		}
	}
	return DoctorCheck{
		Check:   "token_present",
		Status:  "pass",
		Message: "token is present",
	}
}

func checkBaseURL(cfg *config.ResolvedConfig) DoctorCheck {
	if cfg == nil || cfg.BaseURL == "" {
		return DoctorCheck{
			Check:   "base_url",
			Status:  "fail",
			Message: "no base URL configured",
		}
	}
	u, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return DoctorCheck{
			Check:   "base_url",
			Status:  "fail",
			Message: fmt.Sprintf("invalid base URL: %v", err),
		}
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return DoctorCheck{
			Check:   "base_url",
			Status:  "warn",
			Message: fmt.Sprintf("base URL scheme is %q (expected https)", u.Scheme),
		}
	}
	if u.Scheme == "http" {
		return DoctorCheck{
			Check:   "base_url",
			Status:  "warn",
			Message: "base URL uses HTTP (consider HTTPS)",
		}
	}
	return DoctorCheck{
		Check:   "base_url",
		Status:  "pass",
		Message: fmt.Sprintf("base URL: %s", cfg.BaseURL),
	}
}

func checkAPIReachable(cfg *config.ResolvedConfig, timeout time.Duration) DoctorCheck {
	if cfg == nil || cfg.BaseURL == "" || cfg.Token == "" {
		return DoctorCheck{
			Check:   "api_reachable",
			Status:  "fail",
			Message: "skipped (missing base URL or token)",
		}
	}

	client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", timeout, 0)
	ctx := context.Background()
	resp, err := client.Do(ctx, "GET", "/api/v1/users/self", nil, nil)
	if err != nil {
		return DoctorCheck{
			Check:   "api_reachable",
			Status:  "fail",
			Message: fmt.Sprintf("API unreachable: %v", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return DoctorCheck{
			Check:   "api_reachable",
			Status:  "pass",
			Message: "API is reachable",
		}
	}
	return DoctorCheck{
		Check:   "api_reachable",
		Status:  "fail",
		Message: fmt.Sprintf("API returned status %d", resp.StatusCode),
	}
}

func checkTokenValid(cfg *config.ResolvedConfig, timeout time.Duration) DoctorCheck {
	if cfg == nil || cfg.BaseURL == "" || cfg.Token == "" {
		return DoctorCheck{
			Check:   "token_valid",
			Status:  "fail",
			Message: "skipped (missing base URL or token)",
		}
	}

	client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", timeout, 0)
	ctx := context.Background()
	resp, err := client.Do(ctx, "GET", "/api/v1/users/self", nil, nil)
	if err != nil {
		return DoctorCheck{
			Check:   "token_valid",
			Status:  "fail",
			Message: fmt.Sprintf("API call failed: %v", err),
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return DoctorCheck{
			Check:   "token_valid",
			Status:  "pass",
			Message: "token is valid",
		}
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return DoctorCheck{
			Check:   "token_valid",
			Status:  "fail",
			Message: "token is invalid or expired",
		}
	}
	return DoctorCheck{
		Check:   "token_valid",
		Status:  "fail",
		Message: fmt.Sprintf("API returned status %d", resp.StatusCode),
	}
}

func checkWriteSafety(cfg *config.ResolvedConfig) DoctorCheck {
	readOnly := false
	if cfg != nil {
		readOnly = cfg.ReadOnly
	}
	envReadOnly := os.Getenv("CANVAS_READ_ONLY") == "1"

	if readOnly || envReadOnly {
		return DoctorCheck{
			Check:   "write_safety",
			Status:  "pass",
			Message: "read-only mode enabled (--read-only or CANVAS_READ_ONLY)",
		}
	}
	return DoctorCheck{
		Check:   "write_safety",
		Status:  "pass",
		Message: "write operations allowed (use --read-only to restrict)",
	}
}
