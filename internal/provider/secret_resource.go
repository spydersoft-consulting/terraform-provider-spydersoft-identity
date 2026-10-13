package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/spydersoft-consulting/terraform-provider-spydersoft-identity/internal/client"
)

var (
	_ resource.Resource              = &secretResource{}
	_ resource.ResourceWithConfigure = &secretResource{}
)

// secretKind describes which parent a secret resource attaches to.
type secretKind struct {
	typeSuffix  string // resource type suffix, e.g. "_client_secret"
	parentAttr  string // attribute naming the parent, e.g. "client"
	parentLabel string // human-readable parent, e.g. "client"
	route       string // fmt pattern for the secrets collection, taking the parent id
}

// NewClientSecretResource returns the spydersoftidentity_client_secret resource.
func NewClientSecretResource() resource.Resource {
	return &secretResource{kind: secretKind{"_client_secret", "client", "client", "/api/v1/clients/%d/secrets"}}
}

// NewApiResourceSecretResource returns the spydersoftidentity_api_resource_secret resource.
func NewApiResourceSecretResource() resource.Resource {
	return &secretResource{kind: secretKind{"_api_resource_secret", "api_resource", "API resource", "/api/v1/apiresources/%d/secrets"}}
}

// secretResource manages one secret on a client or API resource. The admin API
// has no update call and never returns the secret value, so every attribute
// forces replacement and the value is kept from configuration.
type secretResource struct {
	kind   secretKind
	client *client.Client
}

func (r *secretResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + r.kind.typeSuffix
}

func (r *secretResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "A secret on a " + r.kind.parentLabel + ". The value is hashed by the server and cannot be read back; changing any attribute replaces the secret.",
		Attributes: map[string]schema.Attribute{
			"id": idAttribute(),
			r.kind.parentAttr: schema.StringAttribute{
				Required: true, PlanModifiers: replace,
				Description: "The id of the " + r.kind.parentLabel + " that owns the secret.",
			},
			"type": schema.StringAttribute{
				Optional: true, Computed: true, PlanModifiers: replace,
				Default:     stringdefault.StaticString("SharedSecret"),
				Description: "Secret type: SharedSecret, X509Thumbprint, X509Name or X509CertificateBase64.",
			},
			"value": schema.StringAttribute{
				Required: true, Sensitive: true, PlanModifiers: replace,
				Description: "The secret value. Stored in state; protect the state file.",
			},
			"description": schema.StringAttribute{Optional: true, PlanModifiers: replace, Description: "Description."},
			"expiration":  schema.StringAttribute{Optional: true, PlanModifiers: replace, Description: "Expiration as an RFC 3339 timestamp."},
		},
	}
}

func (r *secretResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *secretResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var parent, typ, value types.String
	var description, expiration types.String
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root(r.kind.parentAttr), &parent)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("type"), &typ)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("value"), &value)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("description"), &description)...)
	resp.Diagnostics.Append(req.Plan.GetAttribute(ctx, path.Root("expiration"), &expiration)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parentID, err := parseID(parent)
	if err != nil {
		resp.Diagnostics.AddError("Invalid "+r.kind.parentLabel+" id", err.Error())
		return
	}
	var created client.SecretRef
	body := client.SaveSecret{
		Type:        typ.ValueString(),
		Value:       value.ValueString(),
		Description: optionalString(description),
		Expiration:  optionalString(expiration),
	}
	if err := r.client.Post(ctx, fmt.Sprintf(r.kind.route, parentID), body, &created); err != nil {
		resp.Diagnostics.AddError("Creating secret", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.Itoa(created.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root(r.kind.parentAttr), parent)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("type"), typ)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("value"), value)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("description"), description)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("expiration"), expiration)...)
}

// Read only checks that the secret still exists; its value and expiration
// formatting cannot be compared with the server, so they are left as configured.
func (r *secretResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var id, parent types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(r.kind.parentAttr), &parent)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parentID, err := parseID(parent)
	if err != nil {
		resp.Diagnostics.AddError("Invalid "+r.kind.parentLabel+" id", err.Error())
		return
	}
	var got client.SecretRef
	err = r.client.Get(ctx, fmt.Sprintf(r.kind.route+"/%s", parentID, id.ValueString()), &got)
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading secret", err.Error())
	}
}

func (r *secretResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError("Update not supported", "Secrets cannot be updated in place; this is a provider bug because all attributes force replacement.")
}

func (r *secretResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var id, parent types.String
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root("id"), &id)...)
	resp.Diagnostics.Append(req.State.GetAttribute(ctx, path.Root(r.kind.parentAttr), &parent)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parentID, err := parseID(parent)
	if err != nil {
		resp.Diagnostics.AddError("Invalid "+r.kind.parentLabel+" id", err.Error())
		return
	}
	err = r.client.Delete(ctx, fmt.Sprintf(r.kind.route+"/%s", parentID, id.ValueString()))
	if err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Deleting secret", err.Error())
	}
}
