package provider

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/spydersoft-consulting/terraform-provider-spydersoft-identity/internal/client"
)

var (
	_ resource.Resource                = &identityResourceResource{}
	_ resource.ResourceWithConfigure   = &identityResourceResource{}
	_ resource.ResourceWithImportState = &identityResourceResource{}
)

// NewIdentityResourceResource returns the spydersoftidentity_identity_resource resource.
func NewIdentityResourceResource() resource.Resource { return &identityResourceResource{} }

type identityResourceResource struct {
	client *client.Client
}

type identityResourceModel struct {
	ID                      types.String `tfsdk:"id"`
	Name                    types.String `tfsdk:"name"`
	DisplayName             types.String `tfsdk:"display_name"`
	Description             types.String `tfsdk:"description"`
	Enabled                 types.Bool   `tfsdk:"enabled"`
	Emphasize               types.Bool   `tfsdk:"emphasize"`
	Required                types.Bool   `tfsdk:"required"`
	ShowInDiscoveryDocument types.Bool   `tfsdk:"show_in_discovery_document"`
	UserClaims              types.Set    `tfsdk:"user_claims"`
	Properties              types.Map    `tfsdk:"properties"`
}

func (r *identityResourceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_resource"
}

func (r *identityResourceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An identity resource (a group of user claims, such as profile or email). Import with the resource name.",
		Attributes: map[string]schema.Attribute{
			"id":                         idAttribute(),
			"name":                       schema.StringAttribute{Required: true, Description: "Unique identity resource name."},
			"display_name":               schema.StringAttribute{Optional: true, Description: "Name shown on the consent screen."},
			"description":                schema.StringAttribute{Optional: true, Description: "Description shown on the consent screen."},
			"enabled":                    boolWithDefault("Whether the identity resource is enabled.", true),
			"emphasize":                  boolWithDefault("Emphasize on the consent screen.", false),
			"required":                   boolWithDefault("Cannot be deselected on the consent screen.", false),
			"show_in_discovery_document": boolWithDefault("Whether it is listed in the discovery document.", true),
			"user_claims":                stringSetAttribute("User claim types in this identity resource."),
			"properties":                 stringMapAttribute("Custom key/value properties."),
		},
	}
}

func (r *identityResourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *identityResourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan identityResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var created client.IdentityResource
	if err := r.client.Post(ctx, "/api/v1/IdentityResources", plan.toAPI(), &created); err != nil {
		resp.Diagnostics.AddError("Creating identity resource", err.Error())
		return
	}
	if err := r.applyChildren(ctx, created.ID, plan); err != nil {
		resp.Diagnostics.AddError("Setting identity resource claims/properties", err.Error())
		return
	}
	got, err := r.load(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Reading created identity resource", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, got)...)
}

func (r *identityResourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state identityResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseID(state.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid identity resource id", err.Error())
		return
	}
	got, err := r.load(ctx, id)
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading identity resource", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, got)...)
}

func (r *identityResourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state identityResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseID(state.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid identity resource id", err.Error())
		return
	}
	if err := r.client.Put(ctx, fmt.Sprintf("/api/v1/IdentityResources/%d", id), plan.toAPI()); err != nil {
		resp.Diagnostics.AddError("Updating identity resource", err.Error())
		return
	}
	if err := r.applyChildren(ctx, id, plan); err != nil {
		resp.Diagnostics.AddError("Setting identity resource claims/properties", err.Error())
		return
	}
	got, err := r.load(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Reading updated identity resource", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, got)...)
}

func (r *identityResourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state identityResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseID(state.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid identity resource id", err.Error())
		return
	}
	if err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/IdentityResources/%d", id)); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Deleting identity resource", err.Error())
	}
}

// ImportState imports by identity resource name, resolved through the natural-key endpoint.
func (r *identityResourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var got client.IdentityResource
	if err := r.client.Get(ctx, "/api/v1/IdentityResources/by-name/"+url.PathEscape(req.ID), &got); err != nil {
		resp.Diagnostics.AddError("Importing identity resource", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.Itoa(got.ID))...)
}

func (r *identityResourceResource) applyChildren(ctx context.Context, id int, plan identityResourceModel) error {
	base := fmt.Sprintf("/api/v1/identityresources/%d", id)
	if err := syncStringSet(ctx, r.client, base+"/claims", "type", plan.UserClaims); err != nil {
		return err
	}
	return syncProperties(ctx, r.client, base+"/properties", plan.Properties)
}

func (r *identityResourceResource) load(ctx context.Context, id int) (identityResourceModel, error) {
	base := fmt.Sprintf("/api/v1/identityresources/%d", id)
	var i client.IdentityResource
	if err := r.client.Get(ctx, fmt.Sprintf("/api/v1/IdentityResources/%d", id), &i); err != nil {
		return identityResourceModel{}, err
	}
	claims, err := loadStringSet(ctx, r.client, base+"/claims", "type")
	if err != nil {
		return identityResourceModel{}, err
	}
	props, err := loadProperties(ctx, r.client, base+"/properties")
	if err != nil {
		return identityResourceModel{}, err
	}

	return identityResourceModel{
		ID:                      types.StringValue(strconv.Itoa(i.ID)),
		Name:                    types.StringValue(i.Name),
		DisplayName:             nullableString(i.DisplayName),
		Description:             nullableString(i.Description),
		Enabled:                 types.BoolValue(i.Enabled),
		Emphasize:               types.BoolValue(i.Emphasize),
		Required:                types.BoolValue(i.Required),
		ShowInDiscoveryDocument: types.BoolValue(i.ShowInDiscoveryDocument),
		UserClaims:              claims,
		Properties:              props,
	}, nil
}

func (m identityResourceModel) toAPI() client.SaveIdentityResource {
	return client.SaveIdentityResource{
		Name:                    m.Name.ValueString(),
		DisplayName:             optionalString(m.DisplayName),
		Description:             optionalString(m.Description),
		Enabled:                 m.Enabled.ValueBool(),
		Emphasize:               m.Emphasize.ValueBool(),
		Required:                m.Required.ValueBool(),
		ShowInDiscoveryDocument: m.ShowInDiscoveryDocument.ValueBool(),
	}
}
