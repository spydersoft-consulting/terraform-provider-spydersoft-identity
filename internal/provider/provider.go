// Package provider implements the Spydersoft Identity Terraform provider.
package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/spyder007/terraform-provider-spydersoft-identity/internal/client"
)

var _ provider.Provider = &identityProvider{}

type identityProvider struct {
	version string
}

type identityProviderModel struct {
	URL          types.String `tfsdk:"url"`
	TokenURL     types.String `tfsdk:"token_url"`
	ClientID     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

// New returns a provider factory for the given build version.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &identityProvider{version: version}
	}
}

func (p *identityProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "spydersoftidentity"
	resp.Version = p.version
}

func (p *identityProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manage clients, API resources, identity resources and scopes on a Spydersoft Identity server through its admin API.",
		Attributes: map[string]schema.Attribute{
			"url": schema.StringAttribute{
				Optional:    true,
				Description: "Base URL of the admin API. May also be set with SPYDERSOFT_IDENTITY_URL.",
			},
			"token_url": schema.StringAttribute{
				Optional:    true,
				Description: "OAuth2 token endpoint of the identity server. May also be set with SPYDERSOFT_IDENTITY_TOKEN_URL.",
			},
			"client_id": schema.StringAttribute{
				Optional:    true,
				Description: "client_credentials client with the identity:admin:read/write scopes. May also be set with SPYDERSOFT_IDENTITY_CLIENT_ID.",
			},
			"client_secret": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Secret for client_id. May also be set with SPYDERSOFT_IDENTITY_CLIENT_SECRET.",
			},
		},
	}
}

func (p *identityProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg identityProviderModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	url := valueOrEnv(cfg.URL, "SPYDERSOFT_IDENTITY_URL")
	tokenURL := valueOrEnv(cfg.TokenURL, "SPYDERSOFT_IDENTITY_TOKEN_URL")
	clientID := valueOrEnv(cfg.ClientID, "SPYDERSOFT_IDENTITY_CLIENT_ID")
	clientSecret := valueOrEnv(cfg.ClientSecret, "SPYDERSOFT_IDENTITY_CLIENT_SECRET")

	for attr, v := range map[string]string{
		"url": url, "token_url": tokenURL, "client_id": clientID, "client_secret": clientSecret,
	} {
		if v == "" {
			resp.Diagnostics.AddAttributeError(path.Root(attr), "Missing provider configuration",
				"Set "+attr+" in the provider block or via its SPYDERSOFT_IDENTITY_* environment variable.")
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	c := client.New(ctx, client.Config{
		BaseURL:      url,
		TokenURL:     tokenURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"identity:admin:read", "identity:admin:write"},
	})
	resp.ResourceData = c
	resp.DataSourceData = c
}

func (p *identityProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewScopeResource,
	}
}

func (p *identityProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return nil
}

func valueOrEnv(v types.String, env string) string {
	if !v.IsNull() && !v.IsUnknown() && v.ValueString() != "" {
		return v.ValueString()
	}
	return os.Getenv(env)
}
