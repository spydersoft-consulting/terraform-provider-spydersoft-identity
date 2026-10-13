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

	"github.com/spyder007/terraform-provider-spydersoft-identity/internal/client"
)

var (
	_ resource.Resource                = &apiResourceResource{}
	_ resource.ResourceWithConfigure   = &apiResourceResource{}
	_ resource.ResourceWithImportState = &apiResourceResource{}
)

// NewApiResourceResource returns the spydersoftidentity_api_resource resource.
func NewApiResourceResource() resource.Resource { return &apiResourceResource{} }

type apiResourceResource struct {
	client *client.Client
}

type apiResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DisplayName types.String `tfsdk:"display_name"`
	Description types.String `tfsdk:"description"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	UserClaims  types.Set    `tfsdk:"user_claims"`
	Scopes      types.Set    `tfsdk:"scopes"`
	Properties  types.Map    `tfsdk:"properties"`
}

func (r *apiResourceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_resource"
}

func (r *apiResourceResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An API resource. Import with the resource name.",
		Attributes: map[string]schema.Attribute{
			"id":           idAttribute(),
			"name":         schema.StringAttribute{Required: true, Description: "Unique API resource name."},
			"display_name": schema.StringAttribute{Optional: true, Description: "Display name."},
			"description":  schema.StringAttribute{Optional: true, Description: "Description."},
			"enabled":      boolWithDefault("Whether the API resource is enabled.", true),
			"user_claims":  stringSetAttribute("User claim types included in access tokens for this resource."),
			"scopes":       stringSetAttribute("Names of the scopes this API resource supports."),
			"properties":   stringMapAttribute("Custom key/value properties."),
		},
	}
}

func (r *apiResourceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *apiResourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var created client.ApiResource
	if err := r.client.Post(ctx, "/api/v1/ApiResources", plan.toAPI(), &created); err != nil {
		resp.Diagnostics.AddError("Creating API resource", err.Error())
		return
	}
	if err := r.applyChildren(ctx, created.ID, plan); err != nil {
		resp.Diagnostics.AddError("Setting API resource claims/scopes/properties", err.Error())
		return
	}
	got, err := r.load(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Reading created API resource", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, got)...)
}

func (r *apiResourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseID(state.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid API resource id", err.Error())
		return
	}
	got, err := r.load(ctx, id)
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading API resource", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, got)...)
}

func (r *apiResourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state apiResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseID(state.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid API resource id", err.Error())
		return
	}
	if err := r.client.Put(ctx, fmt.Sprintf("/api/v1/ApiResources/%d", id), plan.toAPI()); err != nil {
		resp.Diagnostics.AddError("Updating API resource", err.Error())
		return
	}
	if err := r.applyChildren(ctx, id, plan); err != nil {
		resp.Diagnostics.AddError("Setting API resource claims/scopes/properties", err.Error())
		return
	}
	got, err := r.load(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Reading updated API resource", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, got)...)
}

func (r *apiResourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseID(state.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid API resource id", err.Error())
		return
	}
	if err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/ApiResources/%d", id)); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Deleting API resource", err.Error())
	}
}

// ImportState imports by API resource name, resolved through the natural-key endpoint.
func (r *apiResourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var got client.ApiResource
	if err := r.client.Get(ctx, "/api/v1/ApiResources/by-name/"+url.PathEscape(req.ID), &got); err != nil {
		resp.Diagnostics.AddError("Importing API resource", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.Itoa(got.ID))...)
}

func (r *apiResourceResource) applyChildren(ctx context.Context, id int, plan apiResourceModel) error {
	base := fmt.Sprintf("/api/v1/apiresources/%d", id)
	if err := syncStringSet(ctx, r.client, base+"/claims", "type", plan.UserClaims); err != nil {
		return err
	}
	if err := syncStringSet(ctx, r.client, base+"/scopes", "scope", plan.Scopes); err != nil {
		return err
	}
	return syncProperties(ctx, r.client, base+"/properties", plan.Properties)
}

func (r *apiResourceResource) load(ctx context.Context, id int) (apiResourceModel, error) {
	base := fmt.Sprintf("/api/v1/apiresources/%d", id)
	var a client.ApiResource
	if err := r.client.Get(ctx, fmt.Sprintf("/api/v1/ApiResources/%d", id), &a); err != nil {
		return apiResourceModel{}, err
	}
	claims, err := loadStringSet(ctx, r.client, base+"/claims", "type")
	if err != nil {
		return apiResourceModel{}, err
	}
	scopes, err := loadStringSet(ctx, r.client, base+"/scopes", "scope")
	if err != nil {
		return apiResourceModel{}, err
	}
	props, err := loadProperties(ctx, r.client, base+"/properties")
	if err != nil {
		return apiResourceModel{}, err
	}

	return apiResourceModel{
		ID:          types.StringValue(strconv.Itoa(a.ID)),
		Name:        types.StringValue(a.Name),
		DisplayName: nullableString(a.DisplayName),
		Description: nullableString(a.Description),
		Enabled:     types.BoolValue(a.Enabled),
		UserClaims:  claims,
		Scopes:      scopes,
		Properties:  props,
	}, nil
}

func (m apiResourceModel) toAPI() client.SaveApiResource {
	return client.SaveApiResource{
		Name:        m.Name.ValueString(),
		DisplayName: optionalString(m.DisplayName),
		Description: optionalString(m.Description),
		Enabled:     m.Enabled.ValueBool(),
	}
}
