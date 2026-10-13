package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
)

// fakeRoleClaims mimics the real role claim API: POST appends (204, duplicates
// allowed) and DELETE removes only the FIRST claim with the given type.
type fakeRoleClaims struct {
	mu     sync.Mutex
	claims []RoleClaim
	calls  []string
}

func (f *fakeRoleClaims) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	switch r.Method {
	case http.MethodGet:
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(f.claims)
	case http.MethodPost:
		var c RoleClaim
		_ = json.NewDecoder(r.Body).Decode(&c)
		f.claims = append(f.claims, c)
		f.calls = append(f.calls, "POST "+c.Type+"="+c.Value)
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		claimType := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		f.calls = append(f.calls, "DELETE "+claimType)
		for i, c := range f.claims {
			if c.Type == claimType {
				f.claims = append(f.claims[:i:i], f.claims[i+1:]...)
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fakeRoleClaims) sorted() []string {
	out := []string{}
	for _, c := range f.claims {
		out = append(out, c.Type+"="+c.Value)
	}
	sort.Strings(out)
	return out
}

func TestSyncRoleClaimsRemovesOneOfSeveralWithSameType(t *testing.T) {
	api := &fakeRoleClaims{claims: []RoleClaim{{"perm", "read"}, {"perm", "write"}, {"dept", "x"}}}
	c := newTestClient(t, api)

	// Keep perm=write and dept=x; drop perm=read. The API can only delete by
	// type, so perm=write must be deleted and re-added.
	err := c.SyncRoleClaims(context.Background(), "r1", []RoleClaim{{"perm", "write"}, {"dept", "x"}})
	if err != nil {
		t.Fatal(err)
	}

	got := strings.Join(api.sorted(), ",")
	if got != "dept=x,perm=write" {
		t.Fatalf("claims = %s, want dept=x,perm=write (calls: %v)", got, api.calls)
	}
	for _, call := range api.calls {
		if call == "DELETE dept" {
			t.Fatalf("the untouched type dept must not be deleted: %v", api.calls)
		}
	}
}

func TestSyncRoleClaimsAddsWithoutDeleting(t *testing.T) {
	api := &fakeRoleClaims{claims: []RoleClaim{{"perm", "read"}}}
	c := newTestClient(t, api)

	err := c.SyncRoleClaims(context.Background(), "r1", []RoleClaim{{"perm", "read"}, {"perm", "write"}})
	if err != nil {
		t.Fatal(err)
	}

	if strings.Join(api.sorted(), ",") != "perm=read,perm=write" {
		t.Fatalf("claims = %v", api.sorted())
	}
	for _, call := range api.calls {
		if strings.HasPrefix(call, "DELETE") {
			t.Fatalf("pure additions must not delete anything: %v", api.calls)
		}
	}
}

func TestSyncRoleClaimsNoChangeMakesNoWrites(t *testing.T) {
	api := &fakeRoleClaims{claims: []RoleClaim{{"perm", "read"}}}
	c := newTestClient(t, api)

	if err := c.SyncRoleClaims(context.Background(), "r1", []RoleClaim{{"perm", "read"}}); err != nil {
		t.Fatal(err)
	}
	if len(api.calls) != 0 {
		t.Fatalf("expected no writes, got %v", api.calls)
	}
}

func TestSyncRoleClaimsRemovesAllWhenEmpty(t *testing.T) {
	api := &fakeRoleClaims{claims: []RoleClaim{{"perm", "read"}, {"perm", "write"}}}
	c := newTestClient(t, api)

	if err := c.SyncRoleClaims(context.Background(), "r1", nil); err != nil {
		t.Fatal(err)
	}
	if len(api.claims) != 0 {
		t.Fatalf("claims = %v, want none", api.claims)
	}
}

func TestFindRoleByName(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode([]Role{{ID: "1", Name: "Admins"}, {ID: "2", Name: "readers"}})
	}))
	t.Cleanup(srv.Close)
	c := &Client{baseURL: srv.URL, http: srv.Client()}

	if r, err := c.FindRoleByName(context.Background(), "readers"); err != nil || r.ID != "2" {
		t.Fatalf("exact: %+v %v", r, err)
	}
	if r, err := c.FindRoleByName(context.Background(), "admins"); err != nil || r.ID != "1" {
		t.Fatalf("case-insensitive: %+v %v", r, err)
	}
	if _, err := c.FindRoleByName(context.Background(), "missing"); err == nil {
		t.Fatal("expected not found")
	}
}
