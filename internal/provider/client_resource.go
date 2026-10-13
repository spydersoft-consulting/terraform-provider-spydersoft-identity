package provider

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/spyder007/terraform-provider-spydersoft-identity/internal/client"
)

var (
	_ resource.Resource                = &clientResource{}
	_ resource.ResourceWithConfigure   = &clientResource{}
	_ resource.ResourceWithImportState = &clientResource{}
)

// NewClientResource returns the spydersoftidentity_client resource.
func NewClientResource() resource.Resource { return &clientResource{} }

type clientResource struct {
	client *client.Client
}

var claimObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"type":  types.StringType,
	"value": types.StringType,
}}

type claimModel struct {
	Type  types.String `tfsdk:"type"`
	Value types.String `tfsdk:"value"`
}

type clientModel struct {
	ID                                types.String `tfsdk:"id"`
	ClientID                          types.String `tfsdk:"client_id"`
	ClientName                        types.String `tfsdk:"client_name"`
	Description                       types.String `tfsdk:"description"`
	ClientURI                         types.String `tfsdk:"client_uri"`
	LogoURI                           types.String `tfsdk:"logo_uri"`
	Enabled                           types.Bool   `tfsdk:"enabled"`
	ProtocolType                      types.String `tfsdk:"protocol_type"`
	RequireClientSecret               types.Bool   `tfsdk:"require_client_secret"`
	RequireConsent                    types.Bool   `tfsdk:"require_consent"`
	RequirePkce                       types.Bool   `tfsdk:"require_pkce"`
	AllowPlainTextPkce                types.Bool   `tfsdk:"allow_plain_text_pkce"`
	AllowOfflineAccess                types.Bool   `tfsdk:"allow_offline_access"`
	AllowAccessTokensViaBrowser       types.Bool   `tfsdk:"allow_access_tokens_via_browser"`
	AllowRememberConsent              types.Bool   `tfsdk:"allow_remember_consent"`
	AlwaysSendClientClaims            types.Bool   `tfsdk:"always_send_client_claims"`
	AlwaysIncludeUserClaimsInIDToken  types.Bool   `tfsdk:"always_include_user_claims_in_id_token"`
	EnableLocalLogin                  types.Bool   `tfsdk:"enable_local_login"`
	IncludeJwtID                      types.Bool   `tfsdk:"include_jwt_id"`
	UpdateAccessTokenClaimsOnRefresh  types.Bool   `tfsdk:"update_access_token_claims_on_refresh"`
	BackChannelLogoutURI              types.String `tfsdk:"back_channel_logout_uri"`
	BackChannelLogoutSessionRequired  types.Bool   `tfsdk:"back_channel_logout_session_required"`
	FrontChannelLogoutURI             types.String `tfsdk:"front_channel_logout_uri"`
	FrontChannelLogoutSessionRequired types.Bool   `tfsdk:"front_channel_logout_session_required"`
	AbsoluteRefreshTokenLifetime      types.Int64  `tfsdk:"absolute_refresh_token_lifetime"`
	AccessTokenLifetime               types.Int64  `tfsdk:"access_token_lifetime"`
	AccessTokenType                   types.Int64  `tfsdk:"access_token_type"`
	AuthorizationCodeLifetime         types.Int64  `tfsdk:"authorization_code_lifetime"`
	DeviceCodeLifetime                types.Int64  `tfsdk:"device_code_lifetime"`
	IdentityTokenLifetime             types.Int64  `tfsdk:"identity_token_lifetime"`
	RefreshTokenExpiration            types.Int64  `tfsdk:"refresh_token_expiration"`
	RefreshTokenUsage                 types.Int64  `tfsdk:"refresh_token_usage"`
	SlidingRefreshTokenLifetime       types.Int64  `tfsdk:"sliding_refresh_token_lifetime"`
	ConsentLifetime                   types.Int64  `tfsdk:"consent_lifetime"`
	UserSsoLifetime                   types.Int64  `tfsdk:"user_sso_lifetime"`
	ClientClaimsPrefix                types.String `tfsdk:"client_claims_prefix"`
	PairWiseSubjectSalt               types.String `tfsdk:"pair_wise_subject_salt"`
	UserCodeType                      types.String `tfsdk:"user_code_type"`

	AllowedGrantTypes            types.Set `tfsdk:"allowed_grant_types"`
	AllowedScopes                types.Set `tfsdk:"allowed_scopes"`
	RedirectURIs                 types.Set `tfsdk:"redirect_uris"`
	PostLogoutRedirectURIs       types.Set `tfsdk:"post_logout_redirect_uris"`
	AllowedCorsOrigins           types.Set `tfsdk:"allowed_cors_origins"`
	IdentityProviderRestrictions types.Set `tfsdk:"identity_provider_restrictions"`
	Claims                       types.Set `tfsdk:"claims"`
	Properties                   types.Map `tfsdk:"properties"`
}

func (r *clientResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_client"
}

func (r *clientResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An OAuth2/OpenID Connect client. Secrets are managed with spydersoftidentity_client_secret. Import with the OAuth client_id.",
		Attributes: map[string]schema.Attribute{
			"id":          idAttribute(),
			"client_id":   schema.StringAttribute{Required: true, Description: "The OAuth client_id."},
			"client_name": schema.StringAttribute{Required: true, Description: "Display name."},

			"description":   serverString("Description."),
			"client_uri":    serverString("URI of a page with information about the client."),
			"logo_uri":      serverString("URI of the client logo."),
			"enabled":       boolWithDefault("Whether the client is enabled.", true),
			"protocol_type": stringWithDefault("Protocol type.", "oidc"),

			"require_client_secret":                  boolWithDefault("Whether the client must present a secret at the token endpoint.", true),
			"require_consent":                        boolWithDefault("Whether a consent screen is required.", true),
			"require_pkce":                           boolWithDefault("Whether authorization code flows require PKCE.", true),
			"allow_plain_text_pkce":                  serverBool("Allow the plain PKCE code challenge method."),
			"allow_offline_access":                   serverBool("Allow refresh tokens (offline_access)."),
			"allow_access_tokens_via_browser":        serverBool("Allow access tokens to be returned via the browser."),
			"allow_remember_consent":                 boolWithDefault("Allow users to remember consent decisions.", true),
			"always_send_client_claims":              serverBool("Always send client claims, not only for client_credentials."),
			"always_include_user_claims_in_id_token": serverBool("Include user claims in the ID token even when an access token is requested."),
			"enable_local_login":                     boolWithDefault("Allow local (username/password) login.", true),
			"include_jwt_id":                         serverBool("Include a jti claim in access tokens."),
			"update_access_token_claims_on_refresh":  serverBool("Refresh access token claims when a refresh token is used."),

			"back_channel_logout_uri":               serverString("Back-channel logout URI."),
			"back_channel_logout_session_required":  boolWithDefault("Whether the session id is sent on back-channel logout.", true),
			"front_channel_logout_uri":              serverString("Front-channel logout URI."),
			"front_channel_logout_session_required": boolWithDefault("Whether the session id is sent on front-channel logout.", true),

			"absolute_refresh_token_lifetime": int64WithDefault("Absolute refresh token lifetime in seconds.", 2592000),
			"access_token_lifetime":           int64WithDefault("Access token lifetime in seconds.", 3600),
			"access_token_type":               serverInt64("Access token type: 0 = JWT, 1 = reference."),
			"authorization_code_lifetime":     int64WithDefault("Authorization code lifetime in seconds.", 300),
			"device_code_lifetime":            int64WithDefault("Device code lifetime in seconds.", 300),
			"identity_token_lifetime":         int64WithDefault("Identity token lifetime in seconds.", 300),
			"refresh_token_expiration":        serverInt64("Refresh token expiration: 0 = sliding, 1 = absolute."),
			"refresh_token_usage":             serverInt64("Refresh token usage: 0 = re-use, 1 = one-time only."),
			"sliding_refresh_token_lifetime":  int64WithDefault("Sliding refresh token lifetime in seconds.", 1296000),
			"consent_lifetime":                schema.Int64Attribute{Optional: true, Description: "Consent lifetime in seconds; unset means no expiry."},
			"user_sso_lifetime":               schema.Int64Attribute{Optional: true, Description: "Maximum SSO lifetime in seconds; unset means no limit."},

			"client_claims_prefix":   serverString("Prefix for client claims."),
			"pair_wise_subject_salt": serverString("Salt for pair-wise subject identifiers."),
			"user_code_type":         serverString("User code type for the device flow."),

			"allowed_grant_types":            stringSetAttribute("Allowed grant types, e.g. authorization_code, client_credentials."),
			"allowed_scopes":                 stringSetAttribute("Scopes (API scopes, identity resources) the client may request."),
			"redirect_uris":                  stringSetAttribute("Allowed redirect URIs."),
			"post_logout_redirect_uris":      stringSetAttribute("Allowed post-logout redirect URIs."),
			"allowed_cors_origins":           stringSetAttribute("Allowed CORS origins."),
			"identity_provider_restrictions": stringSetAttribute("Identity providers the client may use; empty means all."),
			"claims": schema.SetNestedAttribute{
				Optional: true, Computed: true, Description: "Claims added to tokens issued to this client.",
				Default: setdefault.StaticValue(types.SetValueMust(claimObjectType, []attr.Value{})),
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"type":  schema.StringAttribute{Required: true},
					"value": schema.StringAttribute{Required: true},
				}},
			},
			"properties": stringMapAttribute("Custom key/value properties."),
		},
	}
}

func (r *clientResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected *client.Client, got %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *clientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var created client.OidcClient
	if err := r.client.Post(ctx, "/api/v1/Clients", plan.toAPI(), &created); err != nil {
		resp.Diagnostics.AddError("Creating client", err.Error())
		return
	}
	if err := r.applyChildren(ctx, created.ID, plan); err != nil {
		resp.Diagnostics.AddError("Setting client collections", err.Error())
		return
	}
	got, err := r.load(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Reading created client", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, got)...)
}

func (r *clientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseID(state.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid client id", err.Error())
		return
	}
	got, err := r.load(ctx, id)
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading client", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, got)...)
}

func (r *clientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state clientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseID(state.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid client id", err.Error())
		return
	}
	if err := r.client.Put(ctx, fmt.Sprintf("/api/v1/Clients/%d", id), plan.toAPI()); err != nil {
		resp.Diagnostics.AddError("Updating client", err.Error())
		return
	}
	if err := r.applyChildren(ctx, id, plan); err != nil {
		resp.Diagnostics.AddError("Setting client collections", err.Error())
		return
	}
	got, err := r.load(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Reading updated client", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, got)...)
}

func (r *clientResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseID(state.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid client id", err.Error())
		return
	}
	if err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/Clients/%d", id)); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Deleting client", err.Error())
	}
}

// ImportState imports by OAuth client_id, resolved through the natural-key endpoint.
func (r *clientResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var got client.OidcClient
	if err := r.client.Get(ctx, "/api/v1/Clients/by-client-id/"+url.PathEscape(req.ID), &got); err != nil {
		resp.Diagnostics.AddError("Importing client", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.Itoa(got.ID))...)
}

// setChild describes one value-keyed string collection on a client.
type setChild struct {
	route string
	field string
	get   func(clientModel) types.Set
	set   func(*clientModel, types.Set)
}

var clientSetChildren = []setChild{
	{"granttypes", "grantType", func(m clientModel) types.Set { return m.AllowedGrantTypes }, func(m *clientModel, s types.Set) { m.AllowedGrantTypes = s }},
	{"scopes", "scope", func(m clientModel) types.Set { return m.AllowedScopes }, func(m *clientModel, s types.Set) { m.AllowedScopes = s }},
	{"redirecturis", "redirectUri", func(m clientModel) types.Set { return m.RedirectURIs }, func(m *clientModel, s types.Set) { m.RedirectURIs = s }},
	{"postlogoutredirecturis", "postLogoutRedirectUri", func(m clientModel) types.Set { return m.PostLogoutRedirectURIs }, func(m *clientModel, s types.Set) { m.PostLogoutRedirectURIs = s }},
	{"corsorigins", "origin", func(m clientModel) types.Set { return m.AllowedCorsOrigins }, func(m *clientModel, s types.Set) { m.AllowedCorsOrigins = s }},
	{"idprestrictions", "provider", func(m clientModel) types.Set { return m.IdentityProviderRestrictions }, func(m *clientModel, s types.Set) { m.IdentityProviderRestrictions = s }},
}

func (r *clientResource) applyChildren(ctx context.Context, id int, plan clientModel) error {
	base := fmt.Sprintf("/api/v1/clients/%d", id)
	for _, ch := range clientSetChildren {
		if err := syncStringSet(ctx, r.client, base+"/"+ch.route, ch.field, ch.get(plan)); err != nil {
			return fmt.Errorf("%s: %w", ch.route, err)
		}
	}

	var claims []claimModel
	if diags := plan.Claims.ElementsAs(ctx, &claims, false); diags.HasError() {
		return fmt.Errorf("reading claims: %v", diags)
	}
	desired := make([]map[string]string, 0, len(claims))
	for _, c := range claims {
		desired = append(desired, map[string]string{"type": c.Type.ValueString(), "value": c.Value.ValueString()})
	}
	if err := r.client.SyncChildren(ctx, base+"/claims", desired); err != nil {
		return fmt.Errorf("claims: %w", err)
	}
	return syncProperties(ctx, r.client, base+"/properties", plan.Properties)
}

func (r *clientResource) load(ctx context.Context, id int) (clientModel, error) {
	var c client.OidcClient
	if err := r.client.Get(ctx, fmt.Sprintf("/api/v1/Clients/%d", id), &c); err != nil {
		return clientModel{}, err
	}
	m := clientFromAPI(&c)

	base := fmt.Sprintf("/api/v1/clients/%d", id)
	for _, ch := range clientSetChildren {
		s, err := loadStringSet(ctx, r.client, base+"/"+ch.route, ch.field)
		if err != nil {
			return clientModel{}, fmt.Errorf("%s: %w", ch.route, err)
		}
		ch.set(&m, s)
	}

	items, err := r.client.ListChildren(ctx, base+"/claims")
	if err != nil {
		return clientModel{}, fmt.Errorf("claims: %w", err)
	}
	claims := make([]claimModel, 0, len(items))
	for _, it := range items {
		t, _ := it["type"].(string)
		v, _ := it["value"].(string)
		claims = append(claims, claimModel{Type: types.StringValue(t), Value: types.StringValue(v)})
	}
	sort.Slice(claims, func(i, j int) bool {
		if claims[i].Type.ValueString() != claims[j].Type.ValueString() {
			return claims[i].Type.ValueString() < claims[j].Type.ValueString()
		}
		return claims[i].Value.ValueString() < claims[j].Value.ValueString()
	})
	claimSet, diags := types.SetValueFrom(ctx, claimObjectType, claims)
	if diags.HasError() {
		return clientModel{}, fmt.Errorf("building claims: %v", diags)
	}
	m.Claims = claimSet

	if m.Properties, err = loadProperties(ctx, r.client, base+"/properties"); err != nil {
		return clientModel{}, fmt.Errorf("properties: %w", err)
	}
	return m, nil
}

func clientFromAPI(c *client.OidcClient) clientModel {
	return clientModel{
		ID:                                types.StringValue(strconv.Itoa(c.ID)),
		ClientID:                          types.StringValue(c.ClientID),
		ClientName:                        types.StringValue(c.ClientName),
		Description:                       nullableString(c.Description),
		ClientURI:                         nullableString(c.ClientURI),
		LogoURI:                           nullableString(c.LogoURI),
		Enabled:                           types.BoolValue(c.Enabled),
		ProtocolType:                      types.StringValue(c.ProtocolType),
		RequireClientSecret:               types.BoolValue(c.RequireClientSecret),
		RequireConsent:                    types.BoolValue(c.RequireConsent),
		RequirePkce:                       types.BoolValue(c.RequirePkce),
		AllowPlainTextPkce:                types.BoolValue(c.AllowPlainTextPkce),
		AllowOfflineAccess:                types.BoolValue(c.AllowOfflineAccess),
		AllowAccessTokensViaBrowser:       types.BoolValue(c.AllowAccessTokensViaBrowser),
		AllowRememberConsent:              types.BoolValue(c.AllowRememberConsent),
		AlwaysSendClientClaims:            types.BoolValue(c.AlwaysSendClientClaims),
		AlwaysIncludeUserClaimsInIDToken:  types.BoolValue(c.AlwaysIncludeUserClaimsInIDToken),
		EnableLocalLogin:                  types.BoolValue(c.EnableLocalLogin),
		IncludeJwtID:                      types.BoolValue(c.IncludeJwtID),
		UpdateAccessTokenClaimsOnRefresh:  types.BoolValue(c.UpdateAccessTokenClaimsOnRefresh),
		BackChannelLogoutURI:              nullableString(c.BackChannelLogoutURI),
		BackChannelLogoutSessionRequired:  types.BoolValue(c.BackChannelLogoutSessionRequired),
		FrontChannelLogoutURI:             nullableString(c.FrontChannelLogoutURI),
		FrontChannelLogoutSessionRequired: types.BoolValue(c.FrontChannelLogoutSessionRequired),
		AbsoluteRefreshTokenLifetime:      types.Int64Value(c.AbsoluteRefreshTokenLifetime),
		AccessTokenLifetime:               types.Int64Value(c.AccessTokenLifetime),
		AccessTokenType:                   types.Int64Value(c.AccessTokenType),
		AuthorizationCodeLifetime:         types.Int64Value(c.AuthorizationCodeLifetime),
		DeviceCodeLifetime:                types.Int64Value(c.DeviceCodeLifetime),
		IdentityTokenLifetime:             types.Int64Value(c.IdentityTokenLifetime),
		RefreshTokenExpiration:            types.Int64Value(c.RefreshTokenExpiration),
		RefreshTokenUsage:                 types.Int64Value(c.RefreshTokenUsage),
		SlidingRefreshTokenLifetime:       types.Int64Value(c.SlidingRefreshTokenLifetime),
		ConsentLifetime:                   nullableInt64(c.ConsentLifetime),
		UserSsoLifetime:                   nullableInt64(c.UserSsoLifetime),
		ClientClaimsPrefix:                nullableString(c.ClientClaimsPrefix),
		PairWiseSubjectSalt:               nullableString(c.PairWiseSubjectSalt),
		UserCodeType:                      nullableString(c.UserCodeType),
	}
}

func (m clientModel) toAPI() client.SaveOidcClient {
	return client.SaveOidcClient{
		ClientID:                          m.ClientID.ValueString(),
		ClientName:                        m.ClientName.ValueString(),
		Description:                       optionalString(m.Description),
		ClientURI:                         optionalString(m.ClientURI),
		LogoURI:                           optionalString(m.LogoURI),
		RequireClientSecret:               m.RequireClientSecret.ValueBool(),
		RequireConsent:                    m.RequireConsent.ValueBool(),
		RequirePkce:                       m.RequirePkce.ValueBool(),
		AllowPlainTextPkce:                optionalBool(m.AllowPlainTextPkce),
		AllowOfflineAccess:                optionalBool(m.AllowOfflineAccess),
		AllowAccessTokensViaBrowser:       optionalBool(m.AllowAccessTokensViaBrowser),
		AllowRememberConsent:              m.AllowRememberConsent.ValueBool(),
		AlwaysSendClientClaims:            optionalBool(m.AlwaysSendClientClaims),
		AlwaysIncludeUserClaimsInIDToken:  optionalBool(m.AlwaysIncludeUserClaimsInIDToken),
		EnableLocalLogin:                  m.EnableLocalLogin.ValueBool(),
		IncludeJwtID:                      optionalBool(m.IncludeJwtID),
		UpdateAccessTokenClaimsOnRefresh:  optionalBool(m.UpdateAccessTokenClaimsOnRefresh),
		BackChannelLogoutSessionRequired:  m.BackChannelLogoutSessionRequired.ValueBool(),
		FrontChannelLogoutSessionRequired: m.FrontChannelLogoutSessionRequired.ValueBool(),
		Enabled:                           m.Enabled.ValueBool(),
		AbsoluteRefreshTokenLifetime:      m.AbsoluteRefreshTokenLifetime.ValueInt64(),
		AccessTokenLifetime:               m.AccessTokenLifetime.ValueInt64(),
		AccessTokenType:                   optionalInt64(m.AccessTokenType),
		AuthorizationCodeLifetime:         m.AuthorizationCodeLifetime.ValueInt64(),
		DeviceCodeLifetime:                m.DeviceCodeLifetime.ValueInt64(),
		IdentityTokenLifetime:             m.IdentityTokenLifetime.ValueInt64(),
		RefreshTokenExpiration:            optionalInt64(m.RefreshTokenExpiration),
		RefreshTokenUsage:                 optionalInt64(m.RefreshTokenUsage),
		SlidingRefreshTokenLifetime:       m.SlidingRefreshTokenLifetime.ValueInt64(),
		ConsentLifetime:                   optionalInt64(m.ConsentLifetime),
		UserSsoLifetime:                   optionalInt64(m.UserSsoLifetime),
		BackChannelLogoutURI:              optionalString(m.BackChannelLogoutURI),
		ClientClaimsPrefix:                optionalString(m.ClientClaimsPrefix),
		FrontChannelLogoutURI:             optionalString(m.FrontChannelLogoutURI),
		PairWiseSubjectSalt:               optionalString(m.PairWiseSubjectSalt),
		ProtocolType:                      m.ProtocolType.ValueString(),
		UserCodeType:                      optionalString(m.UserCodeType),
	}
}
