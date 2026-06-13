package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/thedavidweng/canvas-cli/internal/canvas"
	"github.com/thedavidweng/canvas-cli/internal/config"
	"github.com/thedavidweng/canvas-cli/internal/output"
	"github.com/thedavidweng/canvas-cli/internal/safety"
)

// NewApiCmd returns the `api` parent command with all subcommands.
func NewApiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api",
		Short: "Raw Canvas API access",
		Long:  `Execute raw Canvas API requests. Useful for debugging or accessing endpoints not yet wrapped by named commands.`,
	}

	cmd.AddCommand(newApiGetCmd())
	cmd.AddCommand(newApiPostCmd())
	cmd.AddCommand(newApiPutCmd())
	cmd.AddCommand(newApiDeleteCmd())

	return cmd
}

// newApiGetCmd returns `api get PATH`.
func newApiGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get PATH",
		Short: "Execute a GET request to the Canvas API",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("path argument is required")
			}

			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			path := args[0]
			jsonMode, _ := cmd.Flags().GetBool("json")
			rawMode, _ := cmd.Flags().GetBool("raw")
			paginate, _ := cmd.Flags().GetBool("paginate")
			queryStr, _ := cmd.Flags().GetString("query")
			includeHeaders, _ := cmd.Flags().GetBool("include-headers")
			pageSize, _ := cmd.Flags().GetInt("page-size")
			limit, _ := cmd.Flags().GetInt("limit")

			// Parse query params
			query := url.Values{}
			if queryStr != "" {
				for _, pair := range strings.Split(queryStr, ",") {
					parts := strings.SplitN(pair, "=", 2)
					if len(parts) == 2 {
						query.Set(parts[0], parts[1])
					}
				}
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)

			if paginate {
				return handlePaginatedRequest(cmd, client, path, query, pageSize, limit, cfg, jsonMode, rawMode)
			}

			// Single request
			resp, err := client.Do(cmd.Context(), "GET", path, query, nil)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_NETWORK_ERROR",
						Message:  err.Error(),
						Category: "network",
					}, "api.get")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			if resp.StatusCode >= 400 {
				// Create error from the already-read body
				errInfo := createErrorFromResponse(resp, bodyBytes, "api.get")
				env := canvas.Envelope{
					OK:    false,
					Error: &errInfo,
					Meta: canvas.Meta{
						SchemaVersion: "2026-06-12",
						Command:       "api.get",
					},
				}
				if jsonMode {
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return fmt.Errorf("API error: %s (status %d)", errInfo.Message, resp.StatusCode)
			}

			// Raw mode: output body only
			if rawMode {
				w := cmd.OutOrStdout()
				_, err = w.Write(bodyBytes)
				return err
			}

			// Parse the response
			var data any
			if err := json.Unmarshal(bodyBytes, &data); err != nil {
				data = string(bodyBytes)
			}

			if jsonMode {
				meta := canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				}
				if includeHeaders {
					meta.Warnings = extractResponseHeaders(resp)
				}
				env := output.NewSuccess(data, "api.get", meta)
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			// Human mode: pretty print JSON
			w := cmd.OutOrStdout()
			pretty, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to format JSON: %w", err)
			}
			fmt.Fprintln(w, string(pretty))
			return nil
		},
	}

	cmd.Flags().Bool("json", false, "output JSON envelope")
	cmd.Flags().Bool("raw", false, "output raw response body only")
	cmd.Flags().Bool("paginate", false, "auto-paginate the response")
	cmd.Flags().String("query", "", "query parameters (key=value,key=value)")
	cmd.Flags().Bool("include-headers", false, "include response headers in meta")
	cmd.Flags().Int("page-size", 100, "items per page for paginated requests")
	cmd.Flags().Int("limit", 0, "max items to return (0 = no limit)")

	return cmd
}

// handlePaginatedRequest handles paginated API requests.
func handlePaginatedRequest(cmd *cobra.Command, client *canvas.Client, path string, query url.Values, pageSize, limit int, cfg *config.ResolvedConfig, jsonMode, rawMode bool) error {
	if pageSize == 0 {
		pageSize = 100
	}

	items, pagMeta, err := canvas.Paginate[any](cmd.Context(), client, path, query, limit, pageSize)
	if err != nil {
		if jsonMode {
			env := output.NewError(canvas.ErrorInfo{
				Code:     "CANVAS_API_ERROR",
				Message:  err.Error(),
				Category: "api",
			}, "api.get")
			return output.WriteJSON(cmd.OutOrStdout(), env, false)
		}
		return fmt.Errorf("pagination failed: %w", err)
	}

	if rawMode {
		w := cmd.OutOrStdout()
		pretty, err := json.MarshalIndent(items, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to format JSON: %w", err)
		}
		fmt.Fprintln(w, string(pretty))
		return nil
	}

	if jsonMode {
		env := output.NewSuccess(items, "api.get", canvas.Meta{
			Profile:      cfg.Profile,
			BaseURL:      cfg.BaseURL,
			Paginated:    true,
			PageSize:     pagMeta.PageSize,
			RequestCount: pagMeta.RequestCount,
		})
		return output.WriteJSON(cmd.OutOrStdout(), env, false)
	}

	// Human mode
	w := cmd.OutOrStdout()
	pretty, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to format JSON: %w", err)
	}
	fmt.Fprintln(w, string(pretty))
	return nil
}

// createErrorFromResponse creates an ErrorInfo from an HTTP response and body bytes.
func createErrorFromResponse(resp *http.Response, bodyBytes []byte, command string) canvas.ErrorInfo {
	errInfo := canvas.ErrorInfo{
		Status: resp.StatusCode,
	}

	// Try to parse body as JSON
	var bodyMap map[string]any
	if err := json.Unmarshal(bodyBytes, &bodyMap); err == nil {
		errInfo.ResponseBody = bodyMap
		if msg, ok := bodyMap["message"].(string); ok {
			errInfo.Message = msg
		}
	}

	// Map status codes to error codes
	switch resp.StatusCode {
	case http.StatusUnauthorized:
		errInfo.Code = "CANVAS_AUTH_ERROR"
		errInfo.Category = "auth"
	case http.StatusForbidden:
		errInfo.Code = "CANVAS_PERMISSION_DENIED"
		errInfo.Category = "permission"
	case http.StatusNotFound:
		errInfo.Code = "CANVAS_NOT_FOUND"
		errInfo.Category = "not_found"
	case http.StatusUnprocessableEntity:
		errInfo.Code = "CANVAS_VALIDATION_ERROR"
		errInfo.Category = "validation"
	default:
		errInfo.Code = "CANVAS_API_ERROR"
		errInfo.Category = "api"
	}

	// Extract Canvas request ID
	if reqID := resp.Header.Get("X-Request-Id"); reqID != "" {
		errInfo.CanvasRequestID = reqID
	}

	return errInfo
}

// extractResponseHeaders extracts relevant response headers for metadata.
func extractResponseHeaders(resp *http.Response) []string {
	var headers []string
	for key, values := range resp.Header {
		for _, v := range values {
			headers = append(headers, fmt.Sprintf("%s: %s", key, v))
		}
	}
	return headers
}

// newApiPostCmd returns `api post PATH`.
func newApiPostCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "post PATH",
		Short: "Execute a POST request to the Canvas API",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			path := args[0]
			data, _ := cmd.Flags().GetString("data")
			jsonMode, _ := cmd.Flags().GetBool("json")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			confirm, _ := cmd.Flags().GetBool("confirm")

			if data == "" {
				return fmt.Errorf("--data is required")
			}

			if err := checkHighRiskSafety(cfg, dryRun, confirm); err != nil {
				return err
			}

			// Resolve data: @file reads from file, otherwise use as-is
			payload, err := resolveData(data)
			if err != nil {
				return fmt.Errorf("failed to read data: %w", err)
			}

			if dryRun {
				preview := safety.FormatPreview(safety.Preview{
					Method:         "POST",
					Path:           path,
					PayloadSummary: string(payload),
				})
				fmt.Fprintln(cmd.OutOrStdout(), preview)
				return nil
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			resp, err := client.Do(cmd.Context(), "POST", path, nil, bytes.NewReader(payload))
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_NETWORK_ERROR",
						Message:  err.Error(),
						Category: "network",
					}, "api.post")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			writeAudit(cfg, "api.post", "POST", path, string(payload), false)

			if resp.StatusCode >= 400 {
				errInfo := createErrorFromResponse(resp, bodyBytes, "api.post")
				env := canvas.Envelope{
					OK:    false,
					Error: &errInfo,
					Meta: canvas.Meta{
						SchemaVersion: "2026-06-12",
						Command:       "api.post",
					},
				}
				if jsonMode {
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return fmt.Errorf("API error: %s (status %d)", errInfo.Message, resp.StatusCode)
			}

			var dataOut any
			if err := json.Unmarshal(bodyBytes, &dataOut); err != nil {
				dataOut = string(bodyBytes)
			}

			if jsonMode {
				env := output.NewSuccess(dataOut, "api.post", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "POST %s succeeded (status %d)\n", path, resp.StatusCode)
			return nil
		},
	}

	cmd.Flags().String("data", "", "JSON body (use @filename to read from file)")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	cmd.Flags().Bool("dry-run", false, "show what would be sent without sending")
	cmd.Flags().Bool("confirm", false, "confirm the operation")

	return cmd
}

// newApiPutCmd returns `api put PATH`.
func newApiPutCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "put PATH",
		Short: "Execute a PUT request to the Canvas API",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			path := args[0]
			data, _ := cmd.Flags().GetString("data")
			jsonMode, _ := cmd.Flags().GetBool("json")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			confirm, _ := cmd.Flags().GetBool("confirm")

			if data == "" {
				return fmt.Errorf("--data is required")
			}

			if err := checkHighRiskSafety(cfg, dryRun, confirm); err != nil {
				return err
			}

			// Resolve data: @file reads from file, otherwise use as-is
			payload, err := resolveData(data)
			if err != nil {
				return fmt.Errorf("failed to read data: %w", err)
			}

			if dryRun {
				preview := safety.FormatPreview(safety.Preview{
					Method:         "PUT",
					Path:           path,
					PayloadSummary: string(payload),
				})
				fmt.Fprintln(cmd.OutOrStdout(), preview)
				return nil
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			resp, err := client.Do(cmd.Context(), "PUT", path, nil, bytes.NewReader(payload))
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_NETWORK_ERROR",
						Message:  err.Error(),
						Category: "network",
					}, "api.put")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			writeAudit(cfg, "api.put", "PUT", path, string(payload), false)

			if resp.StatusCode >= 400 {
				errInfo := createErrorFromResponse(resp, bodyBytes, "api.put")
				env := canvas.Envelope{
					OK:    false,
					Error: &errInfo,
					Meta: canvas.Meta{
						SchemaVersion: "2026-06-12",
						Command:       "api.put",
					},
				}
				if jsonMode {
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return fmt.Errorf("API error: %s (status %d)", errInfo.Message, resp.StatusCode)
			}

			var dataOut any
			if err := json.Unmarshal(bodyBytes, &dataOut); err != nil {
				dataOut = string(bodyBytes)
			}

			if jsonMode {
				env := output.NewSuccess(dataOut, "api.put", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "PUT %s succeeded (status %d)\n", path, resp.StatusCode)
			return nil
		},
	}

	cmd.Flags().String("data", "", "JSON body (use @filename to read from file)")
	cmd.Flags().Bool("json", false, "output JSON envelope")
	cmd.Flags().Bool("dry-run", false, "show what would be sent without sending")
	cmd.Flags().Bool("confirm", false, "confirm the operation")

	return cmd
}

// newApiDeleteCmd returns `api delete PATH`.
func newApiDeleteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete PATH",
		Short: "Execute a DELETE request to the Canvas API",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := GetConfig(cmd.Context())
			if cfg == nil {
				return fmt.Errorf("no config loaded")
			}

			path := args[0]
			jsonMode, _ := cmd.Flags().GetBool("json")
			dryRun, _ := cmd.Flags().GetBool("dry-run")
			confirm, _ := cmd.Flags().GetBool("confirm")

			if err := checkHighRiskSafety(cfg, dryRun, confirm); err != nil {
				return err
			}

			if dryRun {
				preview := safety.FormatPreview(safety.Preview{
					Method: "DELETE",
					Path:   path,
				})
				fmt.Fprintln(cmd.OutOrStdout(), preview)
				return nil
			}

			client := canvas.NewClient(cfg.BaseURL, cfg.Token, "dev", cfg.TimeoutDuration, cfg.Retries)
			resp, err := client.Do(cmd.Context(), "DELETE", path, nil, nil)
			if err != nil {
				if jsonMode {
					env := output.NewError(canvas.ErrorInfo{
						Code:     "CANVAS_NETWORK_ERROR",
						Message:  err.Error(),
						Category: "network",
					}, "api.delete")
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			bodyBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				return fmt.Errorf("failed to read response: %w", err)
			}

			writeAudit(cfg, "api.delete", "DELETE", path, "", false)

			if resp.StatusCode >= 400 {
				errInfo := createErrorFromResponse(resp, bodyBytes, "api.delete")
				env := canvas.Envelope{
					OK:    false,
					Error: &errInfo,
					Meta: canvas.Meta{
						SchemaVersion: "2026-06-12",
						Command:       "api.delete",
					},
				}
				if jsonMode {
					return output.WriteJSON(cmd.OutOrStdout(), env, false)
				}
				return fmt.Errorf("API error: %s (status %d)", errInfo.Message, resp.StatusCode)
			}

			var dataOut any
			if err := json.Unmarshal(bodyBytes, &dataOut); err != nil {
				dataOut = string(bodyBytes)
			}

			if jsonMode {
				env := output.NewSuccess(dataOut, "api.delete", canvas.Meta{
					Profile: cfg.Profile,
					BaseURL: cfg.BaseURL,
				})
				return output.WriteJSON(cmd.OutOrStdout(), env, false)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "DELETE %s succeeded (status %d)\n", path, resp.StatusCode)
			return nil
		},
	}

	cmd.Flags().Bool("json", false, "output JSON envelope")
	cmd.Flags().Bool("dry-run", false, "show what would be sent without sending")
	cmd.Flags().Bool("confirm", false, "confirm the operation")

	return cmd
}

// resolveData handles --data values: if the value starts with "@", it reads the
// file path after "@"; otherwise it returns the value as-is.
func resolveData(data string) ([]byte, error) {
	if strings.HasPrefix(data, "@") {
		filePath := data[1:]
		return os.ReadFile(filePath)
	}
	return []byte(data), nil
}
