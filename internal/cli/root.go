// Package cli implements all canvas-cli commands using cobra.
package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
	"github.com/thedavidweng/canvas-cli/internal/output"
)

// contextKey is an unexported type for context keys in this package,
// avoiding collisions with keys from other packages.
type contextKey struct{}

// configKey is the context key for the resolved configuration.
var configKey = contextKey{}

// WithConfig returns a new context with the given ResolvedConfig attached.
func WithConfig(ctx context.Context, cfg *config.ResolvedConfig) context.Context {
	return context.WithValue(ctx, configKey, cfg)
}

// GetConfig retrieves the ResolvedConfig from the context, or nil if absent.
func GetConfig(ctx context.Context) *config.ResolvedConfig {
	cfg, _ := ctx.Value(configKey).(*config.ResolvedConfig)
	return cfg
}

// NewRootCmd constructs the root cobra.Command with all global persistent flags,
// config-loading PersistentPreRunE, and built-in subcommands (version, completion).
func NewRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "canvas",
		Short:         "Canvas LMS Command Line Interface",
		Long:          `A fast and agent-friendly CLI for Canvas LMS. Automate course management, assignments, submissions, and more from your terminal or scripts.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	// Register all global persistent flags.
	flags := cmd.PersistentFlags()
	flags.Bool("json", false, "output JSON envelope")
	flags.Bool("pretty", false, "pretty-print JSON output")
	flags.Bool("compact", false, "compact JSON output")
	flags.Bool("ndjson", false, "output newline-delimited JSON")
	flags.Bool("full", false, "include full Canvas response fields")
	flags.Int("limit", 0, "max items to return (0 = no limit)")
	flags.Int("page-size", 100, "items per page for paginated requests")
	flags.Bool("no-paginate", false, "disable auto-pagination")
	flags.String("timeout", "", "request timeout duration (e.g. 30s)")
	flags.Int("retries", 3, "max retries for transient failures")
	flags.Bool("dry-run", false, "preview mutations without sending")
	flags.Bool("confirm", false, "confirm write operations")
	flags.Bool("read-only", false, "block all write operations")
	flags.Bool("events", false, "emit NDJSON progress events on stderr")
	flags.Bool("verbose", false, "enable verbose output")
	flags.Bool("debug", false, "enable debug output")
	flags.Bool("quiet", false, "suppress non-essential output")
	flags.Bool("no-color", false, "disable color output")
	flags.Bool("confirm-delete", false, "confirm destructive operations")
	flags.String("config", "", "config file path (default: OS config dir/canvas-cli/config.yaml)")
	flags.String("profile", "", "config profile name")
	flags.String("base-url", "", "Canvas instance base URL")

	// PersistentPreRunE: resolve config from flags + env + file.
	cmd.PersistentPreRunE = func(c *cobra.Command, args []string) error {
		configPath, _ := c.Flags().GetString("config")
		profileName, _ := c.Flags().GetString("profile")
		baseURLFlag, _ := c.Flags().GetString("base-url")

		cfg, err := config.LoadConfig(configPath, profileName)
		if err != nil {
			cfg = &canvas.Config{
				Profiles: make(map[string]canvas.Profile),
			}
			if profileName != "" {
				cfg.CurrentProfile = profileName
			}
		}

		opts := config.Options{
			BaseURL: baseURLFlag,
			Profile: profileName,
		}

		resolved, err := config.Resolve(opts, cfg)
		if err != nil {
			// Commands that tolerate missing config get a best-effort resolution.
			if commandSkipsFullConfig(c) {
				resolved = &config.ResolvedConfig{
					Profile: cfg.CurrentProfile,
				}
				if baseURLFlag != "" {
					resolved.BaseURL = baseURLFlag
				}
			} else {
				return fmt.Errorf("config error: %w (run `canvas auth login` to set up credentials)", err)
			}
		}

		// Merge global flags into the resolved config.
		if timeoutStr, _ := c.Flags().GetString("timeout"); timeoutStr != "" {
			if d, parseErr := time.ParseDuration(timeoutStr); parseErr == nil {
				resolved.TimeoutDuration = d
			}
		}
		if v, _ := c.Flags().GetInt("retries"); v != 0 {
			resolved.Retries = v
		}
		if v, _ := c.Flags().GetInt("page-size"); v != 0 {
			resolved.PageSize = v
		}
		if v, _ := c.Flags().GetInt("limit"); v != 0 {
			resolved.Limit = v
		}
		if v, _ := c.Flags().GetBool("no-paginate"); v {
			resolved.NoPaginate = v
		}
		if v, _ := c.Flags().GetBool("read-only"); v {
			resolved.ReadOnly = v
		}
		if v, _ := c.Flags().GetBool("pretty"); v {
			resolved.OutputJSONPretty = v
		}
		if v, _ := c.Flags().GetBool("no-color"); v {
			resolved.OutputNoColor = v
		}

		c.SetContext(WithConfig(c.Context(), resolved))
		return nil
	}

	// Add built-in subcommands.
	cmd.AddCommand(newVersionCmd(version))
	cmd.AddCommand(newCompletionCmd(cmd))

	// Add Phase 1 commands.
	cmd.AddCommand(NewAuthCmd())
	cmd.AddCommand(NewDoctorCmd())
	cmd.AddCommand(NewMeCmd())
	cmd.AddCommand(NewApiCmd())

	// Resource commands.
	cmd.AddCommand(NewCoursesCmd())
	cmd.AddCommand(NewModulesCmd())
	cmd.AddCommand(NewAssignmentsCmd())
	cmd.AddCommand(NewAnnouncementsCmd())
	cmd.AddCommand(NewDiscussionsCmd())
	cmd.AddCommand(NewFilesCmd())
	cmd.AddCommand(NewPagesCmd())
	cmd.AddCommand(NewInboxCmd())
	cmd.AddCommand(NewEnrollmentsCmd())
	cmd.AddCommand(NewSectionsCmd())
	cmd.AddCommand(NewUsersCmd())
	cmd.AddCommand(NewRubricsCmd())
	cmd.AddCommand(NewSubmissionsCmd())
	cmd.AddCommand(NewGradingCmd())

	return cmd
}

// newVersionCmd returns the `version` subcommand.
func newVersionCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Print the CLI version",
		Run: func(cmd *cobra.Command, args []string) {
			jsonMode, _ := cmd.Flags().GetBool("json")
			if jsonMode {
				env := canvas.Envelope{
					OK: true,
					Data: map[string]string{
						"version": version,
						"commit":  Commit,
						"date":    Date,
					},
					Meta: canvas.Meta{SchemaVersion: output.SchemaVersion, Command: "version"},
				}
				_ = output.WriteJSON(cmd.OutOrStdout(), env, false)
				return
			}
			fmt.Fprintf(cmd.OutOrStdout(), "canvas %s (commit: %s, built: %s)\n", version, Commit, Date)
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	return cmd
}

// newCompletionCmd returns the `completion` subcommand with bash/zsh/fish/powershell support.
func newCompletionCmd(rootCmd *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for canvas-cli.

To load completions:

Bash:
  $ source <(canvas completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ canvas completion bash > /etc/bash_completion.d/canvas
  # macOS:
  $ canvas completion bash > $(brew --prefix)/etc/bash_completion.d/canvas

Zsh:
  # If shell completion is not already enabled, enable it:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc
  # To load completions for each session, execute once:
  $ canvas completion zsh > "${fpath[1]}/_canvas"

Fish:
  $ canvas completion fish | source
  # To load completions for each session, execute once:
  $ canvas completion fish > ~/.config/fish/completions/canvas.fish

PowerShell:
  PS> canvas completion powershell | Out-String | Invoke-Expression
  # To load completions for every new session, run:
  PS> canvas completion powershell > canvas.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			w := cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletionV2(w, true)
			case "zsh":
				return rootCmd.GenZshCompletion(w)
			case "fish":
				return rootCmd.GenFishCompletion(w, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(w)
			default:
				return fmt.Errorf("unsupported shell: %s", args[0])
			}
		},
	}
	return cmd
}

// commandSkipsFullConfig returns true for commands that can run without a
// fully resolved config (base URL + token). These commands either manage
// credentials, diagnose configuration, or are built-in shell helpers.
func commandSkipsFullConfig(cmd *cobra.Command) bool {
	path := commandPath(cmd)
	switch path {
	case "canvas version",
		"canvas completion",
		"canvas auth login",
		"canvas auth logout",
		"canvas auth status",
		"canvas auth profiles",
		"canvas auth use",
		"canvas doctor":
		return true
	}
	return false
}

// commandPath returns the full command path (e.g. "canvas auth login").
func commandPath(cmd *cobra.Command) string {
	parts := []string{cmd.Name()}
	for p := cmd.Parent(); p != nil; p = p.Parent() {
		parts = append([]string{p.Name()}, parts...)
	}
	return strings.Join(parts, " ")
}

// Execute is the top-level entry point used by main.go.
// It constructs the root command and runs it, returning an exit code.
func Execute(version string) int {
	cmd := NewRootCmd(version)

	if err := cmd.Execute(); err != nil {
		if exitErr, ok := err.(interface{ ExitCode() int }); ok {
			return exitErr.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}
