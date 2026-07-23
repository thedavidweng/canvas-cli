package cli

import (
	"context"
	"fmt"
	"io"
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
			client, err := getClientFromContext(cmd.Context())
			if err != nil {
				return err
			}
			cfg := GetConfig(cmd.Context())

			jsonMode, _ := cmd.Flags().GetBool("json")
			courseID, _ := cmd.Flags().GetString("course")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}

			modules, _, err := canvas.ListModules(cmd.Context(), client, courseID, url.Values{})
			if err != nil {
				return writeError(cmd.OutOrStdout(), err, "modules.list", jsonMode)
			}

			return writeOutput(cmd.OutOrStdout(), cfg, modules, "modules.list", jsonMode, func(w io.Writer) error {
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
			})
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
			client, err := getClientFromContext(cmd.Context())
			if err != nil {
				return err
			}
			cfg := GetConfig(cmd.Context())

			jsonMode, _ := cmd.Flags().GetBool("json")
			courseID, _ := cmd.Flags().GetString("course")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			moduleID := args[0]

			mod, err := canvas.GetModule(cmd.Context(), client, courseID, moduleID)
			if err != nil {
				return writeError(cmd.OutOrStdout(), err, "modules.get", jsonMode)
			}

			return writeOutput(cmd.OutOrStdout(), cfg, mod, "modules.get", jsonMode, func(w io.Writer) error {
				fmt.Fprintf(w, "ID:        %s\n", mod.ID)
				fmt.Fprintf(w, "Name:      %s\n", mod.Name)
				fmt.Fprintf(w, "Position:  %d\n", mod.Position)
				fmt.Fprintf(w, "Published: %v\n", mod.Published)
				fmt.Fprintf(w, "Items:     %d\n", mod.ItemsCount)
				return nil
			})
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
			client, err := getClientFromContext(cmd.Context())
			if err != nil {
				return err
			}
			cfg := GetConfig(cmd.Context())

			jsonMode, _ := cmd.Flags().GetBool("json")
			courseID, _ := cmd.Flags().GetString("course")
			moduleID, _ := cmd.Flags().GetString("module")
			if courseID == "" {
				return fmt.Errorf("--course is required")
			}
			if moduleID == "" {
				return fmt.Errorf("--module is required")
			}

			items, _, err := canvas.ListModuleItems(cmd.Context(), client, courseID, moduleID, url.Values{})
			if err != nil {
				return writeError(cmd.OutOrStdout(), err, "modules.items", jsonMode)
			}

			return writeOutput(cmd.OutOrStdout(), cfg, items, "modules.items", jsonMode, func(w io.Writer) error {
				tbl := output.Table{
					Headers: []string{"ID", "Title", "Type", "Position"},
				}
				for _, item := range items {
					tbl.Rows = append(tbl.Rows, []string{item.ID, item.Title, item.Type, fmt.Sprintf("%d", item.Position)})
				}
				return tbl.Render(w, false)
			})
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
			client, err := getClientFromContext(cmd.Context())
			if err != nil {
				return err
			}
			cfg := GetConfig(cmd.Context())

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

			item, err := canvas.GetModuleItem(cmd.Context(), client, courseID, moduleID, itemID)
			if err != nil {
				return writeError(cmd.OutOrStdout(), err, "modules.item", jsonMode)
			}

			return writeOutput(cmd.OutOrStdout(), cfg, item, "modules.item", jsonMode, func(w io.Writer) error {
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
			})
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

	path := fmt.Sprintf("/api/v1/courses/%s/modules/%s", courseID, moduleID)
	action := "published"
	if !published {
		action = "unpublished"
	}
	payload := fmt.Sprintf(`{"module":{"published":%v}}`, published)

	spec := MutationSpec{
		Command:        fmt.Sprintf("modules.%s", action),
		Level:          safety.LowRiskWrite,
		Method:         "PUT",
		Path:           path,
		DryRun:         dryRun,
		Confirm:        confirm,
		ResourceIDs:    []string{courseID, moduleID},
		PayloadSummary: fmt.Sprintf("published=%v", published),
		AuditBody:      payload,
	}

	return Run(cmd.Context(), cfg, cmd.OutOrStdout(), false, spec,
		func(ctx context.Context, client *canvas.Client) (any, int, error) {
			_, err := canvas.PublishModule(ctx, client, courseID, moduleID, published)
			if err != nil {
				return nil, 0, err
			}
			return nil, 200, nil
		},
		func(w io.Writer, _ any) error {
			fmt.Fprintf(w, "Module %s %s\n", moduleID, action)
			return nil
		},
	)
}
