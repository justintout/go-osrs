// Package transport sends the HTTP requests shared by the API clients.
package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/justintout/go-osrs"
)

// Client sends GET requests with a required User-Agent.
type Client struct {
	HTTP      *http.Client
	UserAgent string
}

// GetJSON fetches url and decodes the JSON body into v.
func (c *Client) GetJSON(ctx context.Context, url string, v any) error {
	// The OSRS Wiki and Jagex block generic user agents.
	if c.UserAgent == "" {
		return errors.New("osrs: a descriptive user agent is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("osrs: build request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("osrs: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return &osrs.StatusError{StatusCode: res.StatusCode, URL: url}
	}
	if err := json.NewDecoder(res.Body).Decode(v); err != nil {
		return fmt.Errorf("osrs: decode %s: %w", url, err)
	}
	return nil
}
