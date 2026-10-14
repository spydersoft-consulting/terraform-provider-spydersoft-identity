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

	"github.com/spydersoft-consulting/terraform-provider-spydersoft-identity/internal/client"
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
	newResource func() resource.Resource
	// read finds the object by its natural key and returns the resource's
	// full model for it.
	read   func(ctx context.Context, c *client.Client, key string) (any, error)
	client *client.Client
}

// byNaturalKey builds a read function for objects that have a by-key endpoint
// returning at least their numeric id, then loads them with the resource's
// load logic.
func byNaturalKey(byKeyPath string, load func(ctx context.Context, c *client.Client, id int) (any, error)) func(context.Context, *client.Client, string) (any, error) {
	return func(ctx context.Context, c *client.Client, key string) (any, error) {
		var found struct {
			ID int `json:"id"`
		}
		if err := c.Get(ctx, byKeyPath+url.PathEscape(key), &found); err != nil {
			return nil, err
		}
		return load(ctx, c, found.ID)
	}
}

func NewScopeDataSource() datasource.DataSource {
	return &lookupDataSource{
		typeSuffix: "_scope", description: "Looks up an API scope by name.",
		lookupAttr:  "name",
		newResource: NewScopeResource,
		read: byNaturalKey("/api/v1/Scopes/by-name/", func(ctx context.Context, c *client.Client, id int) (any, error) {
			return (&scopeResource{client: c}).load(ctx, id)
		}),
	}
}

func NewApiResourceDataSource() datasource.DataSource {
	return &lookupDataSource{
		typeSuffix: "_api_resource", description: "Looks up an API resource by name.",
		lookupAttr:  "name",
		newResource: NewApiResourceResource,
		read: byNaturalKey("/api/v1/ApiResources/by-name/", func(ctx context.Context, c *client.Client, id int) (any, error) {
			return (&apiResourceResource{client: c}).load(ctx, id)
		}),
	}
}

func NewIdentityResourceDataSource() datasource.DataSource {
	return &lookupDataSource{
		typeSuffix: "_identity_resource", description: "Looks up an identity resource by name.",
		lookupAttr:  "name",
		newResource: NewIdentityResourceResource,
		read: byNaturalKey("/api/v1/IdentityResources/by-name/", func(ctx context.Context, c *client.Client, id int) (any, error) {
			return (&identityResourceResource{client: c}).load(ctx, id)
		}),
	}
}

func NewClientDataSource() datasource.DataSource {
	return &lookupDataSource{
		typeSuffix: "_client", description: "Looks up a client by its OAuth client_id.",
		lookupAttr:  "client_id",
		newResource: NewClientResource,
		read: byNaturalKey("/api/v1/Clients/by-client-id/", func(ctx context.Context, c *client.Client, id int) (any, error) {
			return (&clientResource{client: c}).load(ctx, id)
		}),
	}
}

// NewRoleDataSource looks a role up by name. The API has no by-name endpoint,
// so the role list is searched.
func NewRoleDataSource() datasource.DataSource {
	return &lookupDataSource{
		typeSuffix: "_role", description: "Looks up a role by name.",
		lookupAttr:  "name",
		newResource: NewRoleResource,
		read: func(ctx context.Context, c *client.Client, key string) (any, error) {
			role, err := c.FindRoleByName(ctx, key)
			if err != nil {
				return nil, err
			}
			return (&roleResource{client: c}).load(ctx, role.ID)
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

	model, err := d.read(ctx, d.client, key.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Reading %s %q", d.typeSuffix[1:], key.ValueString()), err.Error())
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
