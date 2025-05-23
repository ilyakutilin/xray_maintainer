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

type HTTPProxy struct {
	IP   string
	Port int
}

// GetRequestWithProxy makes a GET request to the specified URL with timeout support via context.
// Returns the response body as bytes slice or an error if the request fails.
func GetRequestWithProxy(ctx context.Context, urlStr string, proxy *HTTPProxy) ([]byte, error) {
	// Validate URL first
	if urlStr == "" {
		return nil, fmt.Errorf("empty URL provided")
	}

	// Parse the URL to validate its format
	parsedURL, err := url.ParseRequestURI(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid URL format: %w", err)
	}

	// Validate the host part exists
	if parsedURL.Host == "" {
		return nil, fmt.Errorf("missing host in URL")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, urlStr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create http request to %s: %w", urlStr, err)
	}

	var client *http.Client

	if proxy != nil {
		proxyURL, err := url.Parse(fmt.Sprintf("http://%s:%d", proxy.IP, proxy.Port))
		if err != nil {
			return nil, fmt.Errorf("could not parse the http proxy credentials: "+
				"provided IP is %s, provided port is %d, the resulting proxy URL is "+
				"%s, and apparently it is not a valid proxy URL",
				proxy.IP, proxy.Port, proxyURL)
		}

		client = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}
	} else {
		client = &http.Client{}
	}

	client.Timeout = 10 * time.Second

	resp, err := client.Do(req)
	if err != nil {
		// Handle network errors
		var netErr net.Error
		if errors.As(err, &netErr) {
			return nil, fmt.Errorf("network error: %w", netErr)
		}

		// Handle URL errors
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			return nil, fmt.Errorf("request error: %w", urlErr)
		}

		return nil, fmt.Errorf("http request to %s failed: %w", urlStr, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("received bad status code when making http request to "+
			"%s: %d %s", urlStr, resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return bodyBytes, nil
}
