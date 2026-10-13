package client

// Scope mirrors the admin API's ScopeDto.
type Scope struct {
	ID                      int    `json:"id"`
	Name                    string `json:"name"`
	DisplayName             string `json:"displayName"`
	Description             string `json:"description"`
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
