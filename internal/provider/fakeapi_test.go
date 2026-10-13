package provider_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakeIdentity is an in-memory stand-in for the identity server's token
// endpoint and admin API. It mirrors the behaviours the provider relies on:
// numeric ids, lookup by natural key, 201/204/404/409 status codes, duplicate
// rejection on value-keyed sub-resources, and secrets whose value is never
// returned. It is deliberately dumb about everything else (no validation, no
// server-side defaults), so a passing run proves the provider's request/response
// handling and Terraform plumbing, not the server's business rules.
type fakeIdentity struct {
	mu     sync.Mutex
	nextID int
	// top-level objects: collection -> id -> object
	top map[string]map[int]map[string]any
	// sub-resources: collection/child -> parent id -> items
	children map[string]map[int][]map[string]any
	// roles have string ids and live outside the generic collections
	roles map[string]*fakeRole
}

const fakeToken = "fake-access-token"

// startFakeIdentity serves the fake and returns its base URL.
func startFakeIdentity(t *testing.T) string {
	t.Helper()
	f := &fakeIdentity{
		nextID:   1,
		top:      map[string]map[int]map[string]any{},
		children: map[string]map[int][]map[string]any{},
		roles:    map[string]*fakeRole{},
	}
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	return srv.URL
}

// topKey is the natural key field for each top-level collection.
var topKey = map[string]string{
	"scopes": "name", "apiresources": "name", "identityresources": "name", "clients": "clientId",
}

// lookupRoute maps the by-key route segment to its collections.
var lookupRoute = map[string]string{"by-name": "name", "by-client-id": "clientId"}

// allowedChildren lists the sub-resources each collection supports.
var allowedChildren = map[string][]string{
	"clients":           {"claims", "corsorigins", "granttypes", "idprestrictions", "postlogoutredirecturis", "properties", "redirecturis", "scopes", "secrets"},
	"apiresources":      {"claims", "properties", "scopes", "secrets"},
	"identityresources": {"claims", "properties"},
	"scopes":            {"claims", "properties"},
}

// childKeyFields are the fields whose combined value must be unique within a
// parent; the real API answers 409 for a duplicate.
func childKeyFields(collection, child string) []string {
	switch child {
	case "claims":
		if collection == "clients" {
			return []string{"type", "value"}
		}
		return []string{"type"}
	case "properties":
		return []string{"key"}
	case "scopes":
		return []string{"scope"}
	case "corsorigins":
		return []string{"origin"}
	case "granttypes":
		return []string{"grantType"}
	case "idprestrictions":
		return []string{"provider"}
	case "postlogoutredirecturis":
		return []string{"postLogoutRedirectUri"}
	case "redirecturis":
		return []string{"redirectUri"}
	}
	return nil // secrets are not unique
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

func (f *fakeIdentity) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if r.URL.Path == "/connect/token" {
		writeJSON(w, http.StatusOK, map[string]any{"access_token": fakeToken, "token_type": "Bearer", "expires_in": 3600})
		return
	}
	if r.Header.Get("Authorization") != "Bearer "+fakeToken {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	rest, ok := strings.CutPrefix(r.URL.Path, "/api/v1/")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if strings.EqualFold(strings.SplitN(rest, "/", 2)[0], "roles") {
		f.handleRoles(w, r)
		return
	}
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	collection := strings.ToLower(parts[0])
	if _, known := topKey[collection]; !known {
		http.NotFound(w, r)
		return
	}
	if f.top[collection] == nil {
		f.top[collection] = map[int]map[string]any{}
	}

	switch {
	case len(parts) == 1:
		f.handleCollection(w, r, collection)
	case len(parts) == 3 && lookupRoute[parts[1]] != "":
		f.handleLookup(w, collection, lookupRoute[parts[1]], parts[2])
	default:
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			http.NotFound(w, r)
			return
		}
		obj, exists := f.top[collection][id]
		if !exists {
			http.NotFound(w, r)
			return
		}
		switch len(parts) {
		case 2:
			f.handleItem(w, r, collection, id, obj)
		case 3, 4:
			f.handleChild(w, r, collection, id, strings.ToLower(parts[2]), parts[3:])
		default:
			http.NotFound(w, r)
		}
	}
}

func decodeBody(r *http.Request) map[string]any {
	body := map[string]any{}
	_ = json.NewDecoder(r.Body).Decode(&body)
	for k, v := range body {
		if v == nil { // null means "unset": the server applies its default
			delete(body, k)
		}
	}
	return body
}

func (f *fakeIdentity) handleCollection(w http.ResponseWriter, r *http.Request, collection string) {
	switch r.Method {
	case http.MethodGet:
		out := make([]map[string]any, 0, len(f.top[collection]))
		for _, o := range f.top[collection] {
			out = append(out, o)
		}
		writeJSON(w, http.StatusOK, out)
	case http.MethodPost:
		body := decodeBody(r)
		key, _ := body[topKey[collection]].(string)
		for _, o := range f.top[collection] {
			if o[topKey[collection]] == key {
				writeJSON(w, http.StatusConflict, map[string]any{"title": "duplicate"})
				return
			}
		}
		body["id"] = f.nextID
		f.top[collection][f.nextID] = body
		f.nextID++
		writeJSON(w, http.StatusCreated, body)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (f *fakeIdentity) handleLookup(w http.ResponseWriter, collection, field, value string) {
	for _, o := range f.top[collection] {
		if o[field] == value {
			writeJSON(w, http.StatusOK, o)
			return
		}
	}
	w.WriteHeader(http.StatusNotFound)
}

func (f *fakeIdentity) handleItem(w http.ResponseWriter, r *http.Request, collection string, id int, obj map[string]any) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, obj)
	case http.MethodPut:
		for k, v := range decodeBody(r) {
			obj[k] = v
		}
		w.WriteHeader(http.StatusNoContent)
	case http.MethodDelete:
		delete(f.top[collection], id)
		for k := range f.children {
			if strings.HasPrefix(k, collection+"/") {
				delete(f.children[k], id)
			}
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (f *fakeIdentity) handleChild(w http.ResponseWriter, r *http.Request, collection string, parentID int, child string, tail []string) {
	allowed := false
	for _, c := range allowedChildren[collection] {
		allowed = allowed || c == child
	}
	if !allowed {
		http.NotFound(w, r)
		return
	}
	key := collection + "/" + child
	if f.children[key] == nil {
		f.children[key] = map[int][]map[string]any{}
	}
	items := f.children[key][parentID]

	if len(tail) == 0 {
		switch r.Method {
		case http.MethodGet:
			if items == nil {
				items = []map[string]any{}
			}
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			body := decodeBody(r)
			for _, existing := range items {
				same := len(childKeyFields(collection, child)) > 0
				for _, k := range childKeyFields(collection, child) {
					same = same && existing[k] == body[k]
				}
				if same {
					writeJSON(w, http.StatusConflict, map[string]any{"title": "duplicate"})
					return
				}
			}
			if child == "secrets" {
				delete(body, "value") // secret values are write-only
			}
			body["id"] = f.nextID
			f.nextID++
			f.children[key][parentID] = append(items, body)
			writeJSON(w, http.StatusCreated, body)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}

	childID, err := strconv.Atoi(tail[0])
	if err != nil {
		http.NotFound(w, r)
		return
	}
	for i, it := range items {
		if it["id"] != childID {
			continue
		}
		switch r.Method {
		case http.MethodGet:
			writeJSON(w, http.StatusOK, it)
		case http.MethodDelete:
			f.children[key][parentID] = append(items[:i:i], items[i+1:]...)
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
		return
	}
	w.WriteHeader(http.StatusNotFound)
}
