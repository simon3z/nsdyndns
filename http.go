package nsdyndns

// cspell:ignore nsdyndns

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// httpClient is the shared HTTP client used by all external API calls.
var httpClient = &http.Client{Timeout: 30 * time.Second}

const (
	maxRequestAttempts = 3
	retryBaseDelay     = time.Second
)

// requestBuilder builds a fresh request, since request bodies are
// single-use and must be re-created on retry.
type requestBuilder func(ctx context.Context) (*http.Request, error)

// doRequest performs an HTTP request, retrying with exponential backoff on
// 429 and 5xx responses as well as transient transport errors. It returns
// the response body for 2xx responses and an error otherwise.
func doRequest(ctx context.Context, build requestBuilder) ([]byte, error) {
	var lastErr error

	for attempt := 1; attempt <= maxRequestAttempts; attempt++ {
		if attempt > 1 {
			delay := time.Duration(1<<uint(attempt-2)) * retryBaseDelay

			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		req, err := build(ctx)

		if err != nil {
			return nil, err
		}

		res, err := httpClient.Do(req)

		if err != nil {
			lastErr = err
			continue
		}

		body, readErr := io.ReadAll(res.Body)
		res.Body.Close()

		if readErr != nil {
			lastErr = readErr
			continue
		}

		switch {
		case res.StatusCode >= 200 && res.StatusCode < 300:
			return body, nil
		case res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= 500:
			lastErr = fmt.Errorf("request failed with status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
		default:
			return nil, fmt.Errorf("request failed with status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
		}
	}

	return nil, lastErr
}
