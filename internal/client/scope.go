package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

// Scope mirrors the admin API's ScopeDto.
type Scope struct {
	ID                      int    `json:"id"`
	Name                    string `json:"name"`
	DisplayName             string `json:"displayName,omitempty"`
	Description             string `json:"description,omitempty"`
	Enabled                 bool   `json:"enabled"`
	Emphasize               bool   `json:"emphasize"`
	Required                bool   `json:"required"`
	ShowInDiscoveryDocument bool   `json:"showInDiscoveryDocument"`
}

// SaveScope mirrors the admin API's SaveScopeDto.
type SaveScope struct {
	Name                    string  `json:"name"`
	DisplayName             *string `json:"displayName"`
	Description             *string `json:"description"`
	Enabled                 bool    `json:"enabled"`
	Emphasize               bool    `json:"emphasize"`
	Required                bool    `json:"required"`
	ShowInDiscoveryDocument bool    `json:"showInDiscoveryDocument"`
}

func (c *Client) CreateScope(ctx context.Context, in SaveScope) (*Scope, error) {
	var out Scope
	if err := c.do(ctx, http.MethodPost, "/api/v1/Scopes", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetScope(ctx context.Context, id int) (*Scope, error) {
	var out Scope
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/Scopes/%d", id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetScopeByName looks a scope up by its natural key.
func (c *Client) GetScopeByName(ctx context.Context, name string) (*Scope, error) {
	var out Scope
	if err := c.do(ctx, http.MethodGet, "/api/v1/Scopes/by-name/"+url.PathEscape(name), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateScope(ctx context.Context, id int, in SaveScope) error {
	return c.do(ctx, http.MethodPut, fmt.Sprintf("/api/v1/Scopes/%d", id), in, nil)
}

func (c *Client) DeleteScope(ctx context.Context, id int) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/Scopes/%d", id), nil, nil)
}
