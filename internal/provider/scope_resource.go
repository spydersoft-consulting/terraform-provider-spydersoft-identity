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
	_ resource.Resource                = &scopeResource{}
	_ resource.ResourceWithConfigure   = &scopeResource{}
	_ resource.ResourceWithImportState = &scopeResource{}
)

// NewScopeResource returns the spydersoftidentity_scope resource.
func NewScopeResource() resource.Resource { return &scopeResource{} }

type scopeResource struct {
	client *client.Client
}

type scopeModel struct {
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

func (r *scopeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scope"
}

func (r *scopeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "An API scope. Import with the scope name.",
		Attributes: map[string]schema.Attribute{
			"id":                         idAttribute(),
			"name":                       schema.StringAttribute{Required: true, Description: "Unique scope name."},
			"display_name":               schema.StringAttribute{Optional: true, Description: "Name shown on the consent screen."},
			"description":                schema.StringAttribute{Optional: true, Description: "Description shown on the consent screen."},
			"enabled":                    boolWithDefault("Whether the scope is enabled.", true),
			"emphasize":                  boolWithDefault("Emphasize the scope on the consent screen.", false),
			"required":                   boolWithDefault("The scope cannot be deselected on the consent screen.", false),
			"show_in_discovery_document": boolWithDefault("Whether the scope is listed in the discovery document.", true),
			"user_claims":                stringSetAttribute("User claim types included in tokens issued for this scope."),
			"properties":                 stringMapAttribute("Custom key/value properties on the scope."),
		},
	}
}

func (r *scopeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *scopeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan scopeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var created client.Scope
	if err := r.client.Post(ctx, "/api/v1/Scopes", plan.toAPI(), &created); err != nil {
		resp.Diagnostics.AddError("Creating scope", err.Error())
		return
	}
	if err := r.applyChildren(ctx, created.ID, plan); err != nil {
		resp.Diagnostics.AddError("Setting scope claims/properties", err.Error())
		return
	}
	got, err := r.load(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Reading created scope", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, got)...)
}

func (r *scopeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state scopeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseID(state.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid scope id", err.Error())
		return
	}
	got, err := r.load(ctx, id)
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading scope", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, got)...)
}

func (r *scopeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state scopeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseID(state.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid scope id", err.Error())
		return
	}
	if err := r.client.Put(ctx, fmt.Sprintf("/api/v1/Scopes/%d", id), plan.toAPI()); err != nil {
		resp.Diagnostics.AddError("Updating scope", err.Error())
		return
	}
	if err := r.applyChildren(ctx, id, plan); err != nil {
		resp.Diagnostics.AddError("Setting scope claims/properties", err.Error())
		return
	}
	got, err := r.load(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Reading updated scope", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, got)...)
}

func (r *scopeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state scopeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := parseID(state.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid scope id", err.Error())
		return
	}
	if err := r.client.Delete(ctx, fmt.Sprintf("/api/v1/Scopes/%d", id)); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Deleting scope", err.Error())
	}
}

// ImportState imports by scope name, resolved through the natural-key endpoint.
func (r *scopeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var got client.Scope
	if err := r.client.Get(ctx, "/api/v1/Scopes/by-name/"+url.PathEscape(req.ID), &got); err != nil {
		resp.Diagnostics.AddError("Importing scope", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.Itoa(got.ID))...)
}

// applyChildren reconciles the scope's claims and properties with the plan.
func (r *scopeResource) applyChildren(ctx context.Context, id int, plan scopeModel) error {
	if err := syncStringSet(ctx, r.client, fmt.Sprintf("/api/v1/scopes/%d/claims", id), "type", plan.UserClaims); err != nil {
		return err
	}
	return syncProperties(ctx, r.client, fmt.Sprintf("/api/v1/scopes/%d/properties", id), plan.Properties)
}

// load reads the scope together with its claims and properties.
func (r *scopeResource) load(ctx context.Context, id int) (scopeModel, error) {
	var s client.Scope
	if err := r.client.Get(ctx, fmt.Sprintf("/api/v1/Scopes/%d", id), &s); err != nil {
		return scopeModel{}, err
	}
	claims, err := loadStringSet(ctx, r.client, fmt.Sprintf("/api/v1/scopes/%d/claims", id), "type")
	if err != nil {
		return scopeModel{}, err
	}
	props, err := loadProperties(ctx, r.client, fmt.Sprintf("/api/v1/scopes/%d/properties", id))
	if err != nil {
		return scopeModel{}, err
	}

	return scopeModel{
		ID:                      types.StringValue(strconv.Itoa(s.ID)),
		Name:                    types.StringValue(s.Name),
		DisplayName:             nullableString(s.DisplayName),
		Description:             nullableString(s.Description),
		Enabled:                 types.BoolValue(s.Enabled),
		Emphasize:               types.BoolValue(s.Emphasize),
		Required:                types.BoolValue(s.Required),
		ShowInDiscoveryDocument: types.BoolValue(s.ShowInDiscoveryDocument),
		UserClaims:              claims,
		Properties:              props,
	}, nil
}

func (m scopeModel) toAPI() client.SaveScope {
	return client.SaveScope{
		Name:                    m.Name.ValueString(),
		DisplayName:             optionalString(m.DisplayName),
		Description:             optionalString(m.Description),
		Enabled:                 m.Enabled.ValueBool(),
		Emphasize:               m.Emphasize.ValueBool(),
		Required:                m.Required.ValueBool(),
		ShowInDiscoveryDocument: m.ShowInDiscoveryDocument.ValueBool(),
	}
}
