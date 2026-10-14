package client

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

// Role mirrors the admin API's RoleDto. Role ids are strings (GUIDs).
type Role struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// SaveRole mirrors SaveRoleDto.
type SaveRole struct {
	Name string `json:"name"`
}

// RoleClaim is a claim granted to every user in a role.
type RoleClaim struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// FindRoleByName looks a role up by name. The API has no by-name endpoint, so
// this lists the roles; an exact match wins over a case-insensitive one because
// role names are unique case-insensitively on the server.
func (c *Client) FindRoleByName(ctx context.Context, name string) (*Role, error) {
	var roles []Role
	if err := c.Get(ctx, "/api/v1/Roles", &roles); err != nil {
		return nil, err
	}
	var folded *Role
	for i := range roles {
		if roles[i].Name == name {
			return &roles[i], nil
		}
		if folded == nil && strings.EqualFold(roles[i].Name, name) {
			folded = &roles[i]
		}
	}
	if folded != nil {
		return folded, nil
	}
	return nil, &APIError{Method: "GET", Path: "/api/v1/Roles (name " + name + ")", Status: 404, Body: "role not found"}
}

func roleClaimsPath(roleID string) string {
	return fmt.Sprintf("/api/v1/Roles/%s/claims", url.PathEscape(roleID))
}

// ListRoleClaims returns the role's claims.
func (c *Client) ListRoleClaims(ctx context.Context, roleID string) ([]RoleClaim, error) {
	var out []RoleClaim
	if err := c.Get(ctx, roleClaimsPath(roleID), &out); err != nil {
		return nil, err
	}
	return out, nil
}

// SyncRoleClaims makes the role's claims exactly equal to desired.
//
// The API can add a claim with a value, but it can only remove a claim by its
// type, and it removes the first claim with that type. So a claim whose type is
// shared with another claim cannot be removed precisely. For every type that
// has a claim to remove, all of that type's claims are deleted and the wanted
// ones are added back; types with nothing to remove only get additions.
func (c *Client) SyncRoleClaims(ctx context.Context, roleID string, desired []RoleClaim) error {
	have, err := c.ListRoleClaims(ctx, roleID)
	if err != nil {
		return err
	}

	wanted := make(map[RoleClaim]bool, len(desired))
	for _, d := range desired {
		wanted[d] = true
	}
	existing := make(map[RoleClaim]bool, len(have))
	existingPerType := map[string]int{}
	rebuild := map[string]bool{}
	for _, h := range have {
		existing[h] = true
		existingPerType[h.Type]++
		if !wanted[h] {
			rebuild[h.Type] = true
		}
	}

	base := roleClaimsPath(roleID)
	for claimType := range rebuild {
		for i := 0; i < existingPerType[claimType]; i++ {
			if err := c.Delete(ctx, base+"/"+url.PathEscape(claimType)); err != nil {
				return fmt.Errorf("removing claim type %q: %w", claimType, err)
			}
		}
	}
	for d := range wanted {
		if existing[d] && !rebuild[d.Type] {
			continue // untouched
		}
		if err := c.Post(ctx, base, d, nil); err != nil {
			return fmt.Errorf("adding claim %q: %w", d.Type, err)
		}
	}
	return nil
}
