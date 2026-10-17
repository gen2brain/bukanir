package bukanir

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strconv"
	"time"
)

// Retry policy for transient responses (429 and 5xx).
var (
	retryMax     = 3
	retryBackoff = 500 * time.Millisecond
)

// doBytes performs an HTTP request and returns the body, retrying transient failures.
func (c *Client) doBytes(ctx context.Context, method, url string, headers map[string]string, body []byte) ([]byte, error) {
	hc, err := c.tr.clientFor(url)
	if err != nil {
		return nil, err
	}
	logf("%s: %s %s", sourceTag(url), method, url)
	for attempt := 0; ; attempt++ {
		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, url, reader)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", c.cfg.UserAgent)
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := hc.Do(req)
		if err != nil {
			return nil, err
		}
		respBody, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		switch {
		case resp.StatusCode >= 200 && resp.StatusCode < 300:
			if readErr != nil {
				return nil, readErr
			}
			return respBody, nil
		case (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500) && attempt < retryMax:
			delay, ok := parseRetryAfter(resp.Header.Get("Retry-After"), time.Now())
			if !ok {
				delay = retryBackoff * time.Duration(1<<attempt)
			}
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		default:
			logf("ERROR: %s %s: %s", method, url, resp.Status)
			return nil, fmt.Errorf("bukanir: %s %s: %s", method, url, resp.Status)
		}
	}
}

func (c *Client) getBytes(ctx context.Context, url string) ([]byte, error) {
	return c.doBytes(ctx, http.MethodGet, url, nil, nil)
}

func (c *Client) getJSON(ctx context.Context, url string, out any) error {
	return c.doJSON(ctx, http.MethodGet, url, nil, nil, out)
}

// doJSON performs a request with optional JSON request and response bodies.
func (c *Client) doJSON(ctx context.Context, method, url string, headers map[string]string, reqBody, out any) error {
	var body []byte
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return err
		}
		body = b
		merged := map[string]string{"Content-Type": "application/json"}
		maps.Copy(merged, headers)
		headers = merged
	}
	respBody, err := c.doBytes(ctx, method, url, headers, body)
	if err != nil {
		return err
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return fmt.Errorf("bukanir: decode %s: %w", url, err)
	}
	return nil
}

// parseRetryAfter interprets a Retry-After header (delay seconds or HTTP date).
func parseRetryAfter(v string, now time.Time) (time.Duration, bool) {
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(v); err == nil {
		if secs < 0 {
			return 0, false
		}
		return time.Duration(secs) * time.Second, true
	}
	if t, err := http.ParseTime(v); err == nil {
		if d := t.Sub(now); d > 0 {
			return d, true
		}
		return 0, true
	}
	return 0, false
}
