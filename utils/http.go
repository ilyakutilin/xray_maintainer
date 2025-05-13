package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"
)

// GetRequest makes a GET request to the specified URL with timeout support via context.
// Returns the response body as a string or an error if the request fails.
func GetRequest(ctx context.Context, urlStr string) (string, error) {
	// Validate URL first
	if urlStr == "" {
		return "", fmt.Errorf("empty URL provided")
	}

	// Parse the URL to validate its format
	parsedURL, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return "", fmt.Errorf("invalid URL format: %w", err)
	}

	// Validate the host part exists
	if parsedURL.Host == "" {
		return "", fmt.Errorf("missing host in URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create http request to %s: %w", urlStr, err)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		// Handle network errors
		var netErr net.Error
		if errors.As(err, &netErr) {
			return "", fmt.Errorf("network error: %w", netErr)
		}

		// Handle URL errors
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			return "", fmt.Errorf("request error: %w", urlErr)
		}

		return "", fmt.Errorf("http request to %s failed: %w", urlStr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("received bad status code when making http request to "+
			"%s: %d %s", urlStr, resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(bodyBytes), nil
}
