package client

import (
	"context"
	"fmt"
	"net/http"
)

// ScopeClaim mirrors the admin API's ScopeClaimDto.
type ScopeClaim struct {
	ID      int    `json:"id"`
	ScopeID int    `json:"scopeId"`
	Type    string `json:"type"`
}

// ScopeProperty mirrors the admin API's ScopePropertyDto.
type ScopeProperty struct {
	ID      int    `json:"id"`
	ScopeID int    `json:"scopeId"`
	Key     string `json:"key"`
	Value   string `json:"value"`
}

func (c *Client) ListScopeClaims(ctx context.Context, scopeID int) ([]ScopeClaim, error) {
	var out []ScopeClaim
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/scopes/%d/claims", scopeID), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) CreateScopeClaim(ctx context.Context, scopeID int, claimType string) error {
	body := map[string]string{"type": claimType}
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/scopes/%d/claims", scopeID), body, nil)
}

func (c *Client) DeleteScopeClaim(ctx context.Context, scopeID, claimID int) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/scopes/%d/claims/%d", scopeID, claimID), nil, nil)
}

func (c *Client) ListScopeProperties(ctx context.Context, scopeID int) ([]ScopeProperty, error) {
	var out []ScopeProperty
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/scopes/%d/properties", scopeID), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) CreateScopeProperty(ctx context.Context, scopeID int, key, value string) error {
	body := map[string]string{"key": key, "value": value}
	return c.do(ctx, http.MethodPost, fmt.Sprintf("/api/v1/scopes/%d/properties", scopeID), body, nil)
}

func (c *Client) DeleteScopeProperty(ctx context.Context, scopeID, propertyID int) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/scopes/%d/properties/%d", scopeID, propertyID), nil, nil)
}
