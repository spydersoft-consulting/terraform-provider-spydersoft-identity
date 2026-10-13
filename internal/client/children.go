package client

import (
	"context"
	"fmt"
	"net/http"
)

// Thin exported verbs so resource-specific code can stay in the provider package.

func (c *Client) Get(ctx context.Context, path string, out any) error {
	return c.do(ctx, http.MethodGet, path, nil, out)
}

func (c *Client) Post(ctx context.Context, path string, in, out any) error {
	return c.do(ctx, http.MethodPost, path, in, out)
}

func (c *Client) Put(ctx context.Context, path string, in any) error {
	return c.do(ctx, http.MethodPut, path, in, nil)
}

func (c *Client) Delete(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodDelete, path, nil, nil)
}

// ListChildren returns the raw items of a sub-resource collection.
func (c *Client) ListChildren(ctx context.Context, base string) ([]map[string]any, error) {
	var out []map[string]any
	if err := c.Get(ctx, base, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func childID(item map[string]any) int {
	f, _ := item["id"].(float64)
	return int(f)
}

func childMatches(item map[string]any, want map[string]string) bool {
	for k, v := range want {
		if s, _ := item[k].(string); s != v {
			return false
		}
	}
	return true
}

// SyncChildren makes a value-keyed sub-resource collection (claims, scopes,
// redirect URIs, properties, ...) exactly equal to desired. Each desired item
// is the POST body; an existing item matches when every field in the body is
// equal. The admin API has no update call for these, so a changed item is a
// delete + create, and deletes run first because duplicates are rejected (409).
func (c *Client) SyncChildren(ctx context.Context, base string, desired []map[string]string) error {
	have, err := c.ListChildren(ctx, base)
	if err != nil {
		return err
	}

	matched := make([]bool, len(desired))
	for _, h := range have {
		idx := -1
		for i, d := range desired {
			if !matched[i] && childMatches(h, d) {
				idx = i
				break
			}
		}
		if idx >= 0 {
			matched[idx] = true
			continue
		}
		if err := c.Delete(ctx, fmt.Sprintf("%s/%d", base, childID(h))); err != nil {
			return err
		}
	}
	for i, d := range desired {
		if !matched[i] {
			if err := c.Post(ctx, base, d, nil); err != nil {
				return err
			}
		}
	}
	return nil
}
