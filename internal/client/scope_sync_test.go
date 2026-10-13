package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
)

// fakeScopeAPI is a minimal in-memory stand-in for the scope claim/property endpoints.
type fakeScopeAPI struct {
	mu     sync.Mutex
	nextID int
	claims map[int]ScopeClaim
	props  map[int]ScopeProperty
}

func newFakeScopeAPI() *fakeScopeAPI {
	return &fakeScopeAPI{nextID: 1, claims: map[int]ScopeClaim{}, props: map[int]ScopeProperty{}}
}

func (f *fakeScopeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	isClaims := strings.Contains(r.URL.Path, "/claims")

	switch {
	case r.Method == http.MethodGet && isClaims:
		out := []ScopeClaim{}
		for _, c := range f.claims {
			out = append(out, c)
		}
		_ = json.NewEncoder(w).Encode(out)
	case r.Method == http.MethodGet:
		out := []ScopeProperty{}
		for _, p := range f.props {
			out = append(out, p)
		}
		_ = json.NewEncoder(w).Encode(out)
	case r.Method == http.MethodPost && isClaims:
		var body struct{ Type string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		for _, c := range f.claims {
			if c.Type == body.Type {
				http.Error(w, "duplicate", http.StatusConflict)
				return
			}
		}
		f.claims[f.nextID] = ScopeClaim{ID: f.nextID, Type: body.Type}
		f.nextID++
		w.WriteHeader(http.StatusCreated)
	case r.Method == http.MethodPost:
		var body struct{ Key, Value string }
		_ = json.NewDecoder(r.Body).Decode(&body)
		for _, p := range f.props {
			if p.Key == body.Key {
				http.Error(w, "duplicate", http.StatusConflict)
				return
			}
		}
		f.props[f.nextID] = ScopeProperty{ID: f.nextID, Key: body.Key, Value: body.Value}
		f.nextID++
		w.WriteHeader(http.StatusCreated)
	case r.Method == http.MethodDelete:
		parts := strings.Split(r.URL.Path, "/")
		var id int
		for _, ch := range parts[len(parts)-1] {
			id = id*10 + int(ch-'0')
		}
		if isClaims {
			delete(f.claims, id)
		} else {
			delete(f.props, id)
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func newTestClient(t *testing.T, h http.Handler) *Client {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &Client{baseURL: srv.URL, http: srv.Client()}
}

func TestSyncScopeClaims(t *testing.T) {
	api := newFakeScopeAPI()
	c := newTestClient(t, api)
	ctx := context.Background()

	if err := c.SyncScopeClaims(ctx, 1, []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	// Drop "a", keep "b" (would 409 if re-created), add "c".
	if err := c.SyncScopeClaims(ctx, 1, []string{"b", "c"}); err != nil {
		t.Fatal(err)
	}

	got := []string{}
	for _, cl := range api.claims {
		got = append(got, cl.Type)
	}
	sort.Strings(got)
	if strings.Join(got, ",") != "b,c" {
		t.Fatalf("claims = %v, want [b c]", got)
	}
}

func TestSyncScopePropertiesReplacesChangedValue(t *testing.T) {
	api := newFakeScopeAPI()
	c := newTestClient(t, api)
	ctx := context.Background()

	if err := c.SyncScopeProperties(ctx, 1, map[string]string{"k": "v1", "gone": "x"}); err != nil {
		t.Fatal(err)
	}
	if err := c.SyncScopeProperties(ctx, 1, map[string]string{"k": "v2"}); err != nil {
		t.Fatal(err)
	}

	if len(api.props) != 1 {
		t.Fatalf("got %d properties, want 1", len(api.props))
	}
	for _, p := range api.props {
		if p.Key != "k" || p.Value != "v2" {
			t.Fatalf("property = %+v, want k=v2", p)
		}
	}
}

func TestNotFoundMatchesErrNotFound(t *testing.T) {
	c := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	}))

	_, err := c.GetScopeByName(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
