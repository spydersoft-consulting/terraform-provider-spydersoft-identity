package provider_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// fakeRole is a role with string id, as in the real API (ASP.NET Identity).
type fakeRole struct {
	ID     string
	Name   string
	Claims []map[string]string
}

// handleRoles mimics RolesController, including the quirks the provider has to
// work around: claim POST answers 204 and allows duplicates, and DELETE
// /claims/{type} removes only the first claim with that type. Called with
// f.mu held.
func (f *fakeIdentity) handleRoles(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.EscapedPath(), "/api/v1/")
	raw := strings.Split(strings.Trim(rest, "/"), "/")
	parts := make([]string, len(raw))
	for i, p := range raw {
		parts[i], _ = url.PathUnescape(p)
	}

	switch {
	case len(parts) == 1 && r.Method == http.MethodGet:
		out := []map[string]string{}
		for _, role := range f.roles {
			out = append(out, map[string]string{"id": role.ID, "name": role.Name})
		}
		writeJSON(w, http.StatusOK, out)
	case len(parts) == 1 && r.Method == http.MethodPost:
		body := decodeBody(r)
		name, _ := body["name"].(string)
		if f.roleNameTaken(name, "") {
			writeJSON(w, http.StatusBadRequest, map[string]any{"title": "DuplicateRoleName"})
			return
		}
		role := &fakeRole{ID: fmt.Sprintf("role-%d", f.nextID), Name: name}
		f.nextID++
		f.roles[role.ID] = role
		writeJSON(w, http.StatusCreated, map[string]string{"id": role.ID, "name": role.Name})
	default:
		role, ok := f.roles[parts[1]]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		f.handleRole(w, r, role, parts[2:])
	}
}

func (f *fakeIdentity) roleNameTaken(name, exceptID string) bool {
	for _, role := range f.roles {
		if role.ID != exceptID && strings.EqualFold(role.Name, name) {
			return true
		}
	}
	return false
}

func (f *fakeIdentity) handleRole(w http.ResponseWriter, r *http.Request, role *fakeRole, tail []string) {
	switch {
	case len(tail) == 0 && r.Method == http.MethodGet:
		writeJSON(w, http.StatusOK, map[string]string{"id": role.ID, "name": role.Name})
	case len(tail) == 0 && r.Method == http.MethodPut:
		name, _ := decodeBody(r)["name"].(string)
		if f.roleNameTaken(name, role.ID) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"title": "DuplicateRoleName"})
			return
		}
		role.Name = name
		w.WriteHeader(http.StatusNoContent)
	case len(tail) == 0 && r.Method == http.MethodDelete:
		delete(f.roles, role.ID)
		w.WriteHeader(http.StatusNoContent)
	case len(tail) == 1 && tail[0] == "claims" && r.Method == http.MethodGet:
		out := role.Claims
		if out == nil {
			out = []map[string]string{}
		}
		writeJSON(w, http.StatusOK, out)
	case len(tail) == 1 && tail[0] == "claims" && r.Method == http.MethodPost:
		var c map[string]string
		_ = json.NewDecoder(r.Body).Decode(&c)
		role.Claims = append(role.Claims, map[string]string{"type": c["type"], "value": c["value"]})
		w.WriteHeader(http.StatusNoContent)
	case len(tail) == 2 && tail[0] == "claims" && r.Method == http.MethodDelete:
		for i, c := range role.Claims {
			if c["type"] == tail[1] {
				role.Claims = append(role.Claims[:i:i], role.Claims[i+1:]...)
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}
