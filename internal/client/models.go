package client

// ApiResource mirrors the admin API's ApiResourceDto.
type ApiResource struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

// SaveApiResource mirrors SaveApiResourceDto.
type SaveApiResource struct {
	Name        string  `json:"name"`
	DisplayName *string `json:"displayName"`
	Description *string `json:"description"`
	Enabled     bool    `json:"enabled"`
}

// IdentityResource mirrors IdentityResourceDto.
type IdentityResource struct {
	ID                      int    `json:"id"`
	Name                    string `json:"name"`
	DisplayName             string `json:"displayName"`
	Description             string `json:"description"`
	Enabled                 bool   `json:"enabled"`
	Emphasize               bool   `json:"emphasize"`
	Required                bool   `json:"required"`
	ShowInDiscoveryDocument bool   `json:"showInDiscoveryDocument"`
}

// SaveIdentityResource mirrors SaveIdentityResourceDto.
type SaveIdentityResource struct {
	Name                    string  `json:"name"`
	DisplayName             *string `json:"displayName"`
	Description             *string `json:"description"`
	Enabled                 bool    `json:"enabled"`
	Emphasize               bool    `json:"emphasize"`
	Required                bool    `json:"required"`
	ShowInDiscoveryDocument bool    `json:"showInDiscoveryDocument"`
}

// OidcClient mirrors ClientDto (named to avoid clashing with the HTTP Client type).
type OidcClient struct {
	ID                                int    `json:"id"`
	ClientID                          string `json:"clientId"`
	ClientName                        string `json:"clientName"`
	Enabled                           bool   `json:"enabled"`
	ProtocolType                      string `json:"protocolType"`
	Description                       string `json:"description"`
	ClientURI                         string `json:"clientUri"`
	LogoURI                           string `json:"logoUri"`
	RequireClientSecret               bool   `json:"requireClientSecret"`
	RequireConsent                    bool   `json:"requireConsent"`
	RequirePkce                       bool   `json:"requirePkce"`
	AllowPlainTextPkce                bool   `json:"allowPlainTextPkce"`
	AllowOfflineAccess                bool   `json:"allowOfflineAccess"`
	AllowAccessTokensViaBrowser       bool   `json:"allowAccessTokensViaBrowser"`
	AllowRememberConsent              bool   `json:"allowRememberConsent"`
	AlwaysSendClientClaims            bool   `json:"alwaysSendClientClaims"`
	AlwaysIncludeUserClaimsInIDToken  bool   `json:"alwaysIncludeUserClaimsInIdToken"`
	EnableLocalLogin                  bool   `json:"enableLocalLogin"`
	IncludeJwtID                      bool   `json:"includeJwtId"`
	UpdateAccessTokenClaimsOnRefresh  bool   `json:"updateAccessTokenClaimsOnRefresh"`
	BackChannelLogoutSessionRequired  bool   `json:"backChannelLogoutSessionRequired"`
	FrontChannelLogoutSessionRequired bool   `json:"frontChannelLogoutSessionRequired"`
	AbsoluteRefreshTokenLifetime      int64  `json:"absoluteRefreshTokenLifetime"`
	AccessTokenLifetime               int64  `json:"accessTokenLifetime"`
	AccessTokenType                   int64  `json:"accessTokenType"`
	AuthorizationCodeLifetime         int64  `json:"authorizationCodeLifetime"`
	DeviceCodeLifetime                int64  `json:"deviceCodeLifetime"`
	IdentityTokenLifetime             int64  `json:"identityTokenLifetime"`
	RefreshTokenExpiration            int64  `json:"refreshTokenExpiration"`
	RefreshTokenUsage                 int64  `json:"refreshTokenUsage"`
	SlidingRefreshTokenLifetime       int64  `json:"slidingRefreshTokenLifetime"`
	ConsentLifetime                   *int64 `json:"consentLifetime"`
	UserSsoLifetime                   *int64 `json:"userSsoLifetime"`
	BackChannelLogoutURI              string `json:"backChannelLogoutUri"`
	FrontChannelLogoutURI             string `json:"frontChannelLogoutUri"`
	ClientClaimsPrefix                string `json:"clientClaimsPrefix"`
	PairWiseSubjectSalt               string `json:"pairWiseSubjectSalt"`
	UserCodeType                      string `json:"userCodeType"`
}

// SaveOidcClient mirrors SaveClientDto. Nullable server fields are pointers so
// an unset value is sent as null and the server default applies.
type SaveOidcClient struct {
	ClientID                          string  `json:"clientId"`
	ClientName                        string  `json:"clientName"`
	Description                       *string `json:"description"`
	ClientURI                         *string `json:"clientUri"`
	LogoURI                           *string `json:"logoUri"`
	RequireClientSecret               bool    `json:"requireClientSecret"`
	RequireConsent                    bool    `json:"requireConsent"`
	RequirePkce                       bool    `json:"requirePkce"`
	AllowPlainTextPkce                *bool   `json:"allowPlainTextPkce"`
	AllowOfflineAccess                *bool   `json:"allowOfflineAccess"`
	AllowAccessTokensViaBrowser       *bool   `json:"allowAccessTokensViaBrowser"`
	AllowRememberConsent              bool    `json:"allowRememberConsent"`
	AlwaysSendClientClaims            *bool   `json:"alwaysSendClientClaims"`
	AlwaysIncludeUserClaimsInIDToken  *bool   `json:"alwaysIncludeUserClaimsInIdToken"`
	EnableLocalLogin                  bool    `json:"enableLocalLogin"`
	IncludeJwtID                      *bool   `json:"includeJwtId"`
	UpdateAccessTokenClaimsOnRefresh  *bool   `json:"updateAccessTokenClaimsOnRefresh"`
	BackChannelLogoutSessionRequired  bool    `json:"backChannelLogoutSessionRequired"`
	FrontChannelLogoutSessionRequired bool    `json:"frontChannelLogoutSessionRequired"`
	Enabled                           bool    `json:"enabled"`
	AbsoluteRefreshTokenLifetime      int64   `json:"absoluteRefreshTokenLifetime"`
	AccessTokenLifetime               int64   `json:"accessTokenLifetime"`
	AccessTokenType                   *int64  `json:"accessTokenType"`
	AuthorizationCodeLifetime         int64   `json:"authorizationCodeLifetime"`
	DeviceCodeLifetime                int64   `json:"deviceCodeLifetime"`
	IdentityTokenLifetime             int64   `json:"identityTokenLifetime"`
	RefreshTokenExpiration            *int64  `json:"refreshTokenExpiration"`
	RefreshTokenUsage                 *int64  `json:"refreshTokenUsage"`
	SlidingRefreshTokenLifetime       int64   `json:"slidingRefreshTokenLifetime"`
	ConsentLifetime                   *int64  `json:"consentLifetime"`
	UserSsoLifetime                   *int64  `json:"userSsoLifetime"`
	BackChannelLogoutURI              *string `json:"backChannelLogoutUri"`
	ClientClaimsPrefix                *string `json:"clientClaimsPrefix"`
	FrontChannelLogoutURI             *string `json:"frontChannelLogoutUri"`
	PairWiseSubjectSalt               *string `json:"pairWiseSubjectSalt"`
	ProtocolType                      string  `json:"protocolType"`
	UserCodeType                      *string `json:"userCodeType"`
}

// SecretRef is the non-sensitive part of a client/API resource secret.
type SecretRef struct {
	ID          int    `json:"id"`
	Type        string `json:"type"`
	Description string `json:"description"`
	Expiration  string `json:"expiration"`
}

// SaveSecret mirrors SaveClientSecretDto / SaveApiResourceSecretDto.
type SaveSecret struct {
	Type        string  `json:"type"`
	Value       string  `json:"value"`
	Description *string `json:"description"`
	Expiration  *string `json:"expiration"`
}
