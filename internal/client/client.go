// Package client is a thin HTTP client for the Spydersoft Identity admin API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golang.org/x/oauth2/clientcredentials"
)

// ErrNotFound is returned (wrapped in an APIError) when the API answers 404.
var ErrNotFound = errors.New("not found")

// APIError describes a non-success response from the admin API.
type APIError struct {
	Method string
	Path   string
	Status int
	Body   string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s %s: HTTP %d: %s", e.Method, e.Path, e.Status, strings.TrimSpace(e.Body))
}

// Is lets errors.Is(err, ErrNotFound) match a 404 APIError.
func (e *APIError) Is(target error) bool {
	return target == ErrNotFound && e.Status == http.StatusNotFound
}

// Client calls the admin API using an OAuth2 client_credentials token.
type Client struct {
	baseURL string
	http    *http.Client
}

// Config holds the settings needed to reach and authenticate to the admin API.
type Config struct {
	BaseURL      string
	TokenURL     string
	ClientID     string
	ClientSecret string
	Scopes       []string
}

// New builds a Client. Tokens are fetched and refreshed automatically.
func New(ctx context.Context, cfg Config) *Client {
	cc := clientcredentials.Config{
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		TokenURL:     cfg.TokenURL,
		Scopes:       cfg.Scopes,
	}
	// The provider-configure context is cancelled once configuration completes,
	// but the token source must outlive it.
	httpClient := cc.Client(context.WithoutCancel(ctx))
	httpClient.Timeout = 30 * time.Second

	return &Client{baseURL: strings.TrimRight(cfg.BaseURL, "/"), http: httpClient}
}

// do sends a JSON request and decodes a JSON response into out (when non-nil).
func (c *Client) do(ctx context.Context, method, path string, in, out any) error {
	var body io.Reader
	if in != nil {
		buf, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("encoding request: %w", err)
		}
		body = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return &APIError{Method: method, Path: path, Status: resp.StatusCode, Body: string(raw)}
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			return fmt.Errorf("decoding response from %s %s: %w", method, path, err)
		}
	}
	return nil
}
