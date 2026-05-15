package upstream

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HTTPDoer runs POST with LB + optional failover across instances.
type HTTPDoer struct {
	Balancer   Balancer
	Client     *http.Client
	MaxAttempts int // 0 = try all endpoints once
}

// NewHTTPDoer creates a shared-client doer. timeout applies per request.
func NewHTTPDoer(eps EndpointsConfig, timeout time.Duration) (*HTTPDoer, error) {
	urls := eps.Resolve()
	if len(urls) == 0 {
		return nil, fmt.Errorf("upstream: no endpoints")
	}
	if timeout <= 0 {
		timeout = 800 * time.Millisecond
	}
	return &HTTPDoer{
		Balancer: NewBalancer(urls, eps.LoadBalance),
		Client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        128,
				MaxIdleConnsPerHost: 32,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		MaxAttempts: len(urls),
	}, nil
}

// Post tries LB start host then failover on error or HTTP >= 500.
func (d *HTTPDoer) Post(ctx context.Context, path string, body io.Reader, contentType string) ([]byte, int, error) {
	if d == nil || d.Balancer == nil {
		return nil, 0, fmt.Errorf("upstream: nil doer")
	}
	attempts := d.MaxAttempts
	if attempts <= 0 {
		attempts = len(d.Balancer.All())
	}
	if attempts <= 0 {
		attempts = 1
	}
	var lastErr error
	var lastCode int
	tried := make(map[string]struct{}, attempts)
	for a := 0; a < attempts; a++ {
		base := d.Balancer.Next()
		if base == "" {
			break
		}
		if _, ok := tried[base]; ok && len(tried) >= len(d.Balancer.All()) {
			break
		}
		tried[base] = struct{}{}
		url := base + path
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
		if err != nil {
			lastErr = err
			continue
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		// body is one-shot; caller must rebuild reader per attempt — see PostBytes
		resp, err := d.Client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		b, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		lastCode = resp.StatusCode
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("upstream http %d: %s", resp.StatusCode, string(b))
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return b, resp.StatusCode, fmt.Errorf("upstream http %d: %s", resp.StatusCode, string(b))
		}
		return b, resp.StatusCode, nil
	}
	if lastErr != nil {
		return nil, lastCode, lastErr
	}
	return nil, lastCode, fmt.Errorf("upstream: all endpoints failed")
}

// PostBytes is Post with retry-safe body replay.
func (d *HTTPDoer) PostBytes(ctx context.Context, path string, raw []byte, contentType string) ([]byte, error) {
	attempts := d.MaxAttempts
	if attempts <= 0 {
		attempts = len(d.Balancer.All())
	}
	tried := make(map[string]struct{}, attempts)
	var lastErr error
	for a := 0; a < attempts; a++ {
		base := d.Balancer.Next()
		if base == "" {
			break
		}
		if _, ok := tried[base]; ok && len(tried) >= len(d.Balancer.All()) {
			break
		}
		tried[base] = struct{}{}
		url := base + path
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
		if err != nil {
			lastErr = err
			continue
		}
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}
		resp, err := d.Client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		b, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = readErr
			continue
		}
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("upstream http %d: %s", resp.StatusCode, string(b))
			continue
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("upstream http %d: %s", resp.StatusCode, string(b))
		}
		return b, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("upstream: all endpoints failed")
}
