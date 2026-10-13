package client

import "context"

// SyncScopeClaims makes the scope's claims exactly equal to want. Claims are
// value-keyed on the server (duplicates are rejected with 409), so only the
// differences are created or deleted.
func (c *Client) SyncScopeClaims(ctx context.Context, scopeID int, want []string) error {
	have, err := c.ListScopeClaims(ctx, scopeID)
	if err != nil {
		return err
	}

	wanted := make(map[string]struct{}, len(want))
	for _, t := range want {
		wanted[t] = struct{}{}
	}
	existing := make(map[string]struct{}, len(have))
	for _, h := range have {
		existing[h.Type] = struct{}{}
		if _, keep := wanted[h.Type]; !keep {
			if err := c.DeleteScopeClaim(ctx, scopeID, h.ID); err != nil {
				return err
			}
		}
	}
	for t := range wanted {
		if _, ok := existing[t]; !ok {
			if err := c.CreateScopeClaim(ctx, scopeID, t); err != nil {
				return err
			}
		}
	}
	return nil
}

// SyncScopeProperties makes the scope's properties exactly equal to want. The
// API has no update call for properties, so a changed value is a delete + create.
func (c *Client) SyncScopeProperties(ctx context.Context, scopeID int, want map[string]string) error {
	have, err := c.ListScopeProperties(ctx, scopeID)
	if err != nil {
		return err
	}

	existing := make(map[string]string, len(have))
	for _, h := range have {
		existing[h.Key] = h.Value
		if v, keep := want[h.Key]; !keep || v != h.Value {
			if err := c.DeleteScopeProperty(ctx, scopeID, h.ID); err != nil {
				return err
			}
			delete(existing, h.Key)
		}
	}
	for k, v := range want {
		if _, ok := existing[k]; !ok {
			if err := c.CreateScopeProperty(ctx, scopeID, k, v); err != nil {
				return err
			}
		}
	}
	return nil
}
