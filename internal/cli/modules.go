package cli

import (
	"fmt"
	"net/url"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/output"
	"github.com/thedavidweng/canvas-cli/internal/safety"
)

// NewModulesCmd returns the `modules` parent command with all subcommands.
func NewModulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "modules",
		Short: "Manage course modules",
		Long:  `List, get, and manage Canvas course modules and module items.`,
	}

	cmd.AddCommand(newModulesListCmd())
	cmd.AddCommand(newModulesGetCmd())
	cmd.AddCommand(newModulesItemsCmd())
	cmd.AddCommand(newModulesItemCmd())
	cmd.AddCommand(newModulesPublishCmd())
	cmd.AddCommand(newModulesUnpublishCmd())

	return cmd
}

func newModulesListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List modules for a course",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")
			courseID, _ := cmd.Flags().GetString("course")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			modules, _, err := canvas.ListModules(cmd.Context(), client, courseID, url.Values{})
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "modules.list")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(modules, "modules.list", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human mode
			w := cmd.OutOrStdout()
			tbl := output.Table{
				Headers: []string{"ID", "Name", "Position", "Published", "Items"},
			}
			for _, m := range modules {
				published := "no"
				if m.Published {
					published = "yes"
				}
				tbl.Rows = append(tbl.Rows, []string{
					m.ID, m.Name, fmt.Sprintf("%d", m.Position), published, fmt.Sprintf("%d", m.ItemsCount),
				})
			}
			return tbl.Render(w, false)
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	cmd.Flags().String("course", "", "course ID (required)")
	return cmd
}

func newModulesGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get MODULE_ID",
		Short: "Get a module by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")
			courseID, _ := cmd.Flags().GetString("course")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			moduleID := args[0]

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			mod, err := canvas.GetModule(cmd.Context(), client, courseID, moduleID)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "modules.get")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(mod, "modules.get", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human mode
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "ID:        %s\n", mod.ID)
			fmt.Fprintf(w, "Name:      %s\n", mod.Name)
			fmt.Fprintf(w, "Position:  %d\n", mod.Position)
			fmt.Fprintf(w, "Published: %v\n", mod.Published)
			fmt.Fprintf(w, "Items:     %d\n", mod.ItemsCount)
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	cmd.Flags().String("course", "", "course ID (required)")
	return cmd
}

func newModulesItemsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "items",
		Short: "List items in a module",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")
			courseID, _ := cmd.Flags().GetString("course")
			moduleID, _ := cmd.Flags().GetString("module")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			if moduleID == "" {
				return fmt.Errorf("--module is required")
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			items, _, err := canvas.ListModuleItems(cmd.Context(), client, courseID, moduleID, url.Values{})
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "modules.items")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(items, "modules.items", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human mode
			w := cmd.OutOrStdout()
			tbl := output.Table{
				Headers: []string{"ID", "Title", "Type", "Position"},
			}
			for _, item := range items {
				tbl.Rows = append(tbl.Rows, []string{item.ID, item.Title, item.Type, fmt.Sprintf("%d", item.Position)})
			}
			return tbl.Render(w, false)
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("module", "", "module ID (required)")
	return cmd
}

func newModulesItemCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "item ITEM_ID",
		Short: "Get a module item by ID",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			jsonMode, _ := cmd.Flags().GetBool("json")
			courseID, _ := cmd.Flags().GetString("course")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			moduleID, _ := cmd.Flags().GetString("module")
			if moduleID == "" {
				return fmt.Errorf("--module is required")
			}
			itemID := args[0]

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			item, err := canvas.GetModuleItem(cmd.Context(), client, courseID, moduleID, itemID)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_API_ERROR",
						Message:  err.Error(),
						Category: "api",
					}, "modules.item")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return err
			}

			if jsonMode {
				env := output.NewSuccess(item, "modules.item", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human mode
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "ID:         %s\n", item.ID)
			fmt.Fprintf(w, "Title:      %s\n", item.Title)
			fmt.Fprintf(w, "Type:       %s\n", item.Type)
			fmt.Fprintf(w, "Position:   %d\n", item.Position)
			fmt.Fprintf(w, "ContentID:  %s\n", item.ContentID)
			fmt.Fprintf(w, "HTMLURL:    %s\n", item.HTMLURL)
			published := "n/a"
			if item.Published != nil {
				if *item.Published {
					published = "yes"
				} else {
					published = "no"
				}
			}
			fmt.Fprintf(w, "Published:  %s\n", published)
			return nil
		},
	}
	cmd.Flags().Bool("json", false, "output JSON envelope")
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().String("module", "", "module ID (required)")
	return cmd
}

func newModulesPublishCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "publish MODULE_ID",
		Short: "Publish a module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModulePublish(cmd, args, true)
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().Bool("dry-run", false, "preview mutation without sending")
	cmd.Flags().Bool("confirm", false, "confirm write operation")
	return cmd
}

func newModulesUnpublishCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unpublish MODULE_ID",
		Short: "Unpublish a module",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runModulePublish(cmd, args, false)
		},
	}
	cmd.Flags().String("course", "", "course ID (required)")
	cmd.Flags().Bool("dry-run", false, "preview mutation without sending")
	cmd.Flags().Bool("confirm", false, "confirm write operation")
	return cmd
}

func runModulePublish(cmd *cobra.Command, args []string, published bool) error {
	cfg := GetConfig(cmd.Context())
	if cfg == nil {
		return fmt.Errorf("no config loaded")
	}

	courseID, _ := cmd.Flags().GetString("course")
	if courseID == "" {
		return fmt.Errorf("--course is required")
	}
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	confirm, _ := cmd.Flags().GetBool("confirm")

	moduleID := args[0]

	if err := checkSafety(cfg, dryRun, confirm); err != nil {
		return err
	}

	path := fmt.Sprintf("/api/v1/courses/%s/modules/%s", courseID, moduleID)

	if dryRun {
		preview := safety.FormatPreview(safety.Preview{
			Method:         "PUT",
			Path:           path,
			ResourceIDs:    []string{courseID, moduleID},
			PayloadSummary: fmt.Sprintf("published=%v", published),
		})
		fmt.Fprintln(cmd.OutOrStdout(), preview)
		return nil
	}

	client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
	_, err := canvas.PublishModule(cmd.Context(), client, courseID, moduleID, published)
	if err != nil {
		return err
	}

	action := "published"
	if !published {
		action = "unpublished"
	}
	writeAudit(cfg, fmt.Sprintf("modules.%s", action), "PUT", path, fmt.Sprintf(`{"module":{"published":%v}}`, published), false)

	fmt.Fprintf(cmd.OutOrStdout(), "Module %s %s\n", moduleID, action)
	return nil
}
