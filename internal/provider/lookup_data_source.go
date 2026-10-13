package provider

import (
	"context"
	"fmt"
	"net/url"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/spyder007/terraform-provider-spydersoft-identity/internal/client"
)

var (
	_ datasource.DataSource              = &lookupDataSource{}
	_ datasource.DataSourceWithConfigure = &lookupDataSource{}
)

// lookupDataSource reads one object by its natural key and exposes the same
// attributes as the matching resource, all computed. It reuses the resource's
// schema and load code so the two never drift apart.
type lookupDataSource struct {
	typeSuffix  string // e.g. "_scope"
	description string
	lookupAttr  string // natural-key attribute, e.g. "name" or "client_id"
	byKeyPath   string // lookup endpoint prefix, e.g. "/api/v1/Scopes/by-name/"
	newResource func() resource.Resource
	// load reads the full model for a server id using the resource's load logic.
	load   func(ctx context.Context, c *client.Client, id int) (any, error)
	client *client.Client
}

func NewScopeDataSource() datasource.DataSource {
	return &lookupDataSource{
		typeSuffix: "_scope", description: "Looks up an API scope by name.",
		lookupAttr: "name", byKeyPath: "/api/v1/Scopes/by-name/",
		newResource: NewScopeResource,
		load: func(ctx context.Context, c *client.Client, id int) (any, error) {
			return (&scopeResource{client: c}).load(ctx, id)
		},
	}
}

func NewApiResourceDataSource() datasource.DataSource {
	return &lookupDataSource{
		typeSuffix: "_api_resource", description: "Looks up an API resource by name.",
		lookupAttr: "name", byKeyPath: "/api/v1/ApiResources/by-name/",
		newResource: NewApiResourceResource,
		load: func(ctx context.Context, c *client.Client, id int) (any, error) {
			return (&apiResourceResource{client: c}).load(ctx, id)
		},
	}
}

func NewIdentityResourceDataSource() datasource.DataSource {
	return &lookupDataSource{
		typeSuffix: "_identity_resource", description: "Looks up an identity resource by name.",
		lookupAttr: "name", byKeyPath: "/api/v1/IdentityResources/by-name/",
		newResource: NewIdentityResourceResource,
		load: func(ctx context.Context, c *client.Client, id int) (any, error) {
			return (&identityResourceResource{client: c}).load(ctx, id)
		},
	}
}

func NewClientDataSource() datasource.DataSource {
	return &lookupDataSource{
		typeSuffix: "_client", description: "Looks up a client by its OAuth client_id.",
		lookupAttr: "client_id", byKeyPath: "/api/v1/Clients/by-client-id/",
		newResource: NewClientResource,
		load: func(ctx context.Context, c *client.Client, id int) (any, error) {
			return (&clientResource{client: c}).load(ctx, id)
		},
	}
}

func (d *lookupDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + d.typeSuffix
}

func (d *lookupDataSource) Schema(ctx context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	var rs resource.SchemaResponse
	d.newResource().Schema(ctx, resource.SchemaRequest{}, &rs)

	attrs := make(map[string]dschema.Attribute, len(rs.Schema.Attributes))
	for name, a := range rs.Schema.Attributes {
		attrs[name] = toDataSourceAttribute(a, name == d.lookupAttr)
	}
	resp.Schema = dschema.Schema{Description: d.description, Attributes: attrs}
}

func (d *lookupDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("expected *client.Client, got %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *lookupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var key types.String
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root(d.lookupAttr), &key)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var found struct {
		ID int `json:"id"`
	}
	if err := d.client.Get(ctx, d.byKeyPath+url.PathEscape(key.ValueString()), &found); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Looking up %s %q", d.lookupAttr, key.ValueString()), err.Error())
		return
	}
	model, err := d.load(ctx, d.client, found.ID)
	if err != nil {
		resp.Diagnostics.AddError("Reading "+d.typeSuffix[1:], err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// toDataSourceAttribute converts a resource attribute into a read-only data
// source attribute. The lookup attribute is the only required one.
func toDataSourceAttribute(a rschema.Attribute, lookup bool) dschema.Attribute {
	computed := !lookup
	switch v := a.(type) {
	case rschema.StringAttribute:
		return dschema.StringAttribute{Required: lookup, Computed: computed, Description: v.Description}
	case rschema.BoolAttribute:
		return dschema.BoolAttribute{Computed: true, Description: v.Description}
	case rschema.Int64Attribute:
		return dschema.Int64Attribute{Computed: true, Description: v.Description}
	case rschema.SetAttribute:
		return dschema.SetAttribute{ElementType: v.ElementType, Computed: true, Description: v.Description}
	case rschema.MapAttribute:
		return dschema.MapAttribute{ElementType: v.ElementType, Computed: true, Description: v.Description}
	case rschema.SetNestedAttribute:
		nested := make(map[string]dschema.Attribute, len(v.NestedObject.Attributes))
		for n, na := range v.NestedObject.Attributes {
			nested[n] = toDataSourceAttribute(na, false)
		}
		return dschema.SetNestedAttribute{
			Computed: true, Description: v.Description,
			NestedObject: dschema.NestedAttributeObject{Attributes: nested},
		}
	default:
		panic(fmt.Sprintf("unsupported attribute type %T in data source schema conversion", a))
	}
}
