// Package transport sends the HTTP requests shared by the API clients.
package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	res, err := c.get(ctx, url)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if err := json.NewDecoder(res.Body).Decode(v); err != nil {
		return fmt.Errorf("osrs: decode %s: %w", url, err)
	}
	return nil
}

// GetBytes fetches url and returns the body.
func (c *Client) GetBytes(ctx context.Context, url string) ([]byte, error) {
	res, err := c.get(ctx, url)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("osrs: read %s: %w", url, err)
	}
	return b, nil
}

// get returns the response for a 200 status. The caller closes the body.
func (c *Client) get(ctx context.Context, url string) (*http.Response, error) {
	// The OSRS Wiki and Jagex block generic user agents.
	if c.UserAgent == "" {
		return nil, errors.New("osrs: a descriptive user agent is required")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("osrs: build request: %w", err)
	}
	req.Header.Set("User-Agent", c.UserAgent)
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("osrs: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		res.Body.Close()
		return nil, &osrs.StatusError{StatusCode: res.StatusCode, URL: url}
	}
	return res, nil
}
