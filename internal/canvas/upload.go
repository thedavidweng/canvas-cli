package canvas

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

// uploadInitResponse is the JSON returned by Canvas after step 1 of the upload flow.
type uploadInitResponse struct {
	UploadURL    string            `json:"upload_url"`
	UploadParams map[string]string `json:"upload_params"`
}

// UploadFile uploads a file to a Canvas course using the 3-step flow:
// 1. POST /api/v1/courses/{courseID}/files to notify Canvas (name, size, content_type).
// 2. POST to the returned upload_url with upload_params + file content (multipart).
// 3. Handle 201 (file JSON in body) or follow 3xx redirect for the final file info.
// Returns the file ID on success.
func UploadFile(ctx context.Context, client *Client, courseID, filePath string, content []byte) (string, error) {
	filename := filepath.Base(filePath)

	// Determine content type from extension, falling back to application/octet-stream.
	contentType := mime.TypeByExtension(filepath.Ext(filePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// --- Step 1: Notify Canvas ---
	initBody := url.Values{}
	initBody.Set("name", filename)
	initBody.Set("size", fmt.Sprintf("%d", len(content)))
	initBody.Set("content_type", contentType)

	initReq, err := http.NewRequestWithContext(ctx, "POST",
		client.baseURL+fmt.Sprintf("/api/v1/courses/%s/files", courseID),
		strings.NewReader(initBody.Encode()))
	if err != nil {
		return "", fmt.Errorf("initiate upload: %w", err)
	}
	if client.token != "" {
		initReq.Header.Set("Authorization", "Bearer "+client.token)
	} else if client.cookie != "" {
		initReq.Header.Set("Cookie", client.cookie)
	}
	initReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	initReq.Header.Set("User-Agent", client.userAgent)
	initReq.Header.Set("Accept", "application/json+canvas-string-ids")

	initResp, err := client.httpClient.Do(initReq)
	if err != nil {
		return "", fmt.Errorf("initiate upload: %w", err)
	}
	defer initResp.Body.Close()

	if initResp.StatusCode >= 400 {
		return "", fmt.Errorf("initiate upload: status %d", initResp.StatusCode)
	}

	var init uploadInitResponse
	if err := json.NewDecoder(initResp.Body).Decode(&init); err != nil {
		return "", fmt.Errorf("decode upload init response: %w", err)
	}

	if init.UploadURL == "" {
		return "", fmt.Errorf("upload init response missing upload_url")
	}

	// --- Step 2: Upload file content to upload_url ---
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add upload_params as form fields.
	for key, val := range init.UploadParams {
		if err := writer.WriteField(key, val); err != nil {
			return "", fmt.Errorf("write upload param %s: %w", key, err)
		}
	}

	// Add the file content.
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := part.Write(content); err != nil {
		return "", fmt.Errorf("write file content: %w", err)
	}
	writer.Close()

	// Use a client that does not follow POST redirects automatically,
	// so we can inspect 3xx responses and follow them ourselves.
	uploadClient := *client
	uploadClient.SetHTTPClient(&http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	})

	// Parse the upload_url to determine whether it is a full URL or a path.
	parsedURL, err := url.Parse(init.UploadURL)
	if err != nil {
		return "", fmt.Errorf("parse upload_url %s: %w", init.UploadURL, err)
	}

	var uploadPath string
	var uploadQuery url.Values
	if parsedURL.Scheme != "" && parsedURL.Host != "" {
		// Full URL: may point to an external host (e.g. S3). Do NOT send
		// Canvas auth token to external upload hosts.
		req, reqErr := http.NewRequestWithContext(ctx, "POST", init.UploadURL, &buf)
		if reqErr != nil {
			return "", fmt.Errorf("create upload request: %w", reqErr)
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())

		// Only send auth if the upload URL is on the same host as the Canvas API.
		canvasHost := ""
		if parsed, pErr := url.Parse(client.baseURL); pErr == nil {
			canvasHost = parsed.Host
		}
		if parsedURL.Host == canvasHost {
			if client.token != "" {
				req.Header.Set("Authorization", "Bearer "+client.token)
			} else if client.cookie != "" {
				req.Header.Set("Cookie", client.cookie)
			}
		}
		req.Header.Set("User-Agent", client.userAgent)

		uploadResp, doErr := uploadClient.httpClient.Do(req)
		if doErr != nil {
			return "", fmt.Errorf("upload file: %w", doErr)
		}
		return handleUploadResponse(ctx, uploadResp, client)
	}

	// Relative URL: use client.Do with the path.
	uploadPath = parsedURL.Path
	uploadQuery = parsedURL.Query()

	uploadResp, err := uploadClient.Do(ctx, "POST", uploadPath, uploadQuery, &buf)
	if err != nil {
		return "", fmt.Errorf("upload file: %w", err)
	}

	return handleUploadResponse(ctx, uploadResp, client)
}

// handleUploadResponse processes the response from step 2.
// It handles 201 (success with file JSON) and 3xx redirects.
// For 3xx redirects, it follows the Location URL to fetch the final file object.
func handleUploadResponse(ctx context.Context, resp *http.Response, client *Client) (string, error) {
	defer resp.Body.Close()

	// 201 Created: response body contains file JSON.
	if resp.StatusCode == http.StatusCreated {
		return extractFileID(resp)
	}

	// 3xx redirect: follow the Location header to get the final file JSON.
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		location := resp.Header.Get("Location")
		if location == "" {
			return "", fmt.Errorf("upload redirect missing Location header")
		}

		// Follow the redirect to get the final file object.
		followResp, err := client.DoURL(ctx, "GET", location, nil)
		if err != nil {
			// If we can't follow the redirect, try to extract ID from the URL.
			return extractFileIDFromLocation(location)
		}
		defer followResp.Body.Close()

		if followResp.StatusCode >= 400 {
			return "", fmt.Errorf("upload redirect follow failed: status %d", followResp.StatusCode)
		}

		return extractFileID(followResp)
	}

	return "", fmt.Errorf("upload returned unexpected status %d", resp.StatusCode)
}

// extractFileID decodes a file JSON response and returns the file ID.
func extractFileID(resp *http.Response) (string, error) {
	var file File
	if err := json.NewDecoder(resp.Body).Decode(&file); err != nil {
		return "", fmt.Errorf("decode upload response: %w", err)
	}
	if file.ID == "" {
		return "", fmt.Errorf("upload response missing file ID")
	}
	return file.ID, nil
}

// extractFileIDFromLocation extracts the file ID from a Canvas redirect Location
// header. The format is typically: /api/v1/files/{fileID} or a full URL ending
// with /api/v1/files/{fileID}.
func extractFileIDFromLocation(location string) (string, error) {
	// Strip query parameters.
	if idx := strings.Index(location, "?"); idx >= 0 {
		location = location[:idx]
	}

	// Handle full URLs by extracting the path.
	if strings.HasPrefix(location, "http://") || strings.HasPrefix(location, "https://") {
		if idx := strings.Index(location, "://"); idx >= 0 {
			rest := location[idx+3:]
			if slashIdx := strings.Index(rest, "/"); slashIdx >= 0 {
				location = rest[slashIdx:]
			}
		}
	}

	// Look for /files/{id} pattern.
	parts := strings.Split(location, "/")
	for i, part := range parts {
		if part == "files" && i+1 < len(parts) && parts[i+1] != "" {
			return parts[i+1], nil
		}
	}

	return "", fmt.Errorf("could not extract file ID from Location: %s", location)
}
