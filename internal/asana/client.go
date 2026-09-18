package asana

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/time/rate"
)

const defaultBaseURL = "https://app.asana.com/api/1.0"

type Client struct {
	httpClient      *http.Client
	token           string
	baseURL         string
	workspaceGID    string
	limiter         *rate.Limiter
	maxAttempts     int
	paginationLimit int
}

// NewClient builds a Client authenticated with token, using httpClient for
// requests and limiter to throttle outgoing requests. An empty baseURL
// defaults to Asana's production API. workspaceGID is embedded as the
// "workspace" query param on every request. maxAttempts is the number of
// times a request is attempted before giving up (must be >= 1). paginationLimit
// is the page size used when listing paginated resources.
func NewClient(httpClient *http.Client, token, baseURL, workspaceGID string, limiter *rate.Limiter, maxAttempts, paginationLimit int) *Client {
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		httpClient:      httpClient,
		token:           token,
		baseURL:         baseURL,
		workspaceGID:    workspaceGID,
		limiter:         limiter,
		maxAttempts:     maxAttempts,
		paginationLimit: paginationLimit,
	}
}

// getAllPages calls getWithRetries for path starting at pagination, follows
// next_page.offset, and accumulates every page's data until next_page is
// empty.
func GetAllPages[T any](ctx context.Context, a *Client, path string, pagination Pagination) ([]T, error) {
	var all []T

	for {
		body, err := a.getWithRetries(ctx, path, pagination.QueryParams())
		if err != nil {
			return nil, err
		}

		var parsed paginatedResponse[T]
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
		all = append(all, parsed.Data...)

		if parsed.NextPage == nil || parsed.NextPage.Offset == "" {
			return all, nil
		}
		pagination.Offset = parsed.NextPage.Offset
	}
}

// Dummy backoff. Overridable in tests to avoid real sleeps.
//
//	TODO: We want an exponential backoff on 5xx;
var backoff = func() time.Duration {
	return time.Duration(rand.Intn(5)+1) * time.Second
}

// sleep pauses for d, returning early with ctx.Err() if ctx is done first.
func sleep(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// getWithRetries calls get, retrying up to maxAttempts times.
//
// A 429 response is retried after the delay from the Retry-After header (if
// present); a 5xx response is retried after a backoff delay. Any other
// non-200 status returns immediately.
func (a *Client) getWithRetries(ctx context.Context, path string, query url.Values) ([]byte, error) {
	var lastErr error

	for attempt := 1; attempt <= a.maxAttempts; attempt++ {
		status, header, body, err := a.get(ctx, path, query)
		if err != nil {
			return nil, err
		}

		// TODO: Refactor; Move into a switch case on more HTTP Statuses in another method

		if status == http.StatusOK {
			return body, nil
		}

		// 429
		if status == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("GET %s: rate limited (429): %s", path, string(body))
			delay := backoff()
			if secs, err := strconv.Atoi(header.Get("Retry-After")); err == nil {
				delay = time.Duration(secs) * time.Second
			}
			if err := sleep(ctx, delay); err != nil {
				return nil, err
			}
			continue
		}

		// >= 400 < 500 (429 handled above)
		if status >= http.StatusBadRequest && status < http.StatusInternalServerError {
			return nil, fmt.Errorf("GET %s: client error %d: %s", path, status, string(body))
		}

		// >= 500
		if status >= http.StatusInternalServerError {
			lastErr = fmt.Errorf("GET %s: server error %d: %s", path, status, string(body))
			if err := sleep(ctx, backoff()); err != nil {
				return nil, err
			}
			continue
		}

		return nil, fmt.Errorf("GET %s: unexpected status %d: %s", path, status, string(body))
	}

	return nil, fmt.Errorf("GET %s: giving up after %d attempts: %w", path, a.maxAttempts, lastErr)
}

// get performs a single authenticated GET against baseURL+path, respecting
// the rate limiter, and returns the status code, response headers, and body.
func (a *Client) get(ctx context.Context, path string, query url.Values) (int, http.Header, []byte, error) {
	if err := a.limiter.Wait(ctx); err != nil {
		return 0, nil, nil, fmt.Errorf("rate limiter: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, a.baseURL+path, nil)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("build request: %w", err)
	}
	query.Set("workspace", a.workspaceGID)
	req.URL.RawQuery = query.Encode()
	req.Header.Set("Authorization", "Bearer "+a.token)
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("GET %s: %w", path, err)
	}

	// From docs: The default HTTP client's Transport may not reuse HTTP/1.x "keep-alive" TCP connections if the Body is not read to completion and closed.
	// That is why we store in memory the body bytes below and close it;
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return 0, nil, nil, fmt.Errorf("read response body: %w", err)
	}

	return resp.StatusCode, resp.Header, body, nil
}
