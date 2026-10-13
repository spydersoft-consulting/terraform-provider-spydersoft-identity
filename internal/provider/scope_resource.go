package provider

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/spyder007/terraform-provider-spydersoft-identity/internal/client"
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
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Numeric identifier assigned by the server.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name":         schema.StringAttribute{Required: true, Description: "Unique scope name."},
			"display_name": schema.StringAttribute{Optional: true, Description: "Name shown on the consent screen."},
			"description":  schema.StringAttribute{Optional: true, Description: "Description shown on the consent screen."},
			"enabled": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(true),
				Description: "Whether the scope is enabled.",
			},
			"emphasize": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(false),
				Description: "Emphasize the scope on the consent screen.",
			},
			"required": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(false),
				Description: "The scope cannot be deselected on the consent screen.",
			},
			"show_in_discovery_document": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(true),
				Description: "Whether the scope is listed in the discovery document.",
			},
			"user_claims": schema.SetAttribute{
				ElementType: types.StringType, Optional: true, Computed: true,
				Default:     setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
				Description: "User claim types included in tokens issued for this scope.",
			},
			"properties": schema.MapAttribute{
				ElementType: types.StringType, Optional: true, Computed: true,
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
				Description: "Custom key/value properties on the scope.",
			},
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

	created, err := r.client.CreateScope(ctx, plan.toAPI())
	if err != nil {
		resp.Diagnostics.AddError("Creating scope", err.Error())
		return
	}
	if err := r.applyChildren(ctx, created.ID, plan); err != nil {
		resp.Diagnostics.AddError("Setting scope claims/properties", err.Error())
		return
	}
	// Re-read so the state reflects exactly what the server stored.
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

	id, err := strconv.Atoi(state.ID.ValueString())
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

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid scope id", err.Error())
		return
	}
	if err := r.client.UpdateScope(ctx, id, plan.toAPI()); err != nil {
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

	id, err := strconv.Atoi(state.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Invalid scope id", err.Error())
		return
	}
	if err := r.client.DeleteScope(ctx, id); err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Deleting scope", err.Error())
	}
}

// ImportState imports by scope name, resolved through the natural-key endpoint.
func (r *scopeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	got, err := r.client.GetScopeByName(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Importing scope", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), strconv.Itoa(got.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("name"), got.Name)...)
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

// applyChildren reconciles the scope's claims and properties with the plan.
func (r *scopeResource) applyChildren(ctx context.Context, id int, plan scopeModel) error {
	var claims []string
	if diags := plan.UserClaims.ElementsAs(ctx, &claims, false); diags.HasError() {
		return fmt.Errorf("reading user_claims: %v", diags)
	}
	props := map[string]string{}
	if diags := plan.Properties.ElementsAs(ctx, &props, false); diags.HasError() {
		return fmt.Errorf("reading properties: %v", diags)
	}

	if err := r.client.SyncScopeClaims(ctx, id, claims); err != nil {
		return err
	}
	return r.client.SyncScopeProperties(ctx, id, props)
}

// load reads the scope together with its claims and properties.
func (r *scopeResource) load(ctx context.Context, id int) (scopeModel, error) {
	s, err := r.client.GetScope(ctx, id)
	if err != nil {
		return scopeModel{}, err
	}
	claims, err := r.client.ListScopeClaims(ctx, id)
	if err != nil {
		return scopeModel{}, err
	}
	props, err := r.client.ListScopeProperties(ctx, id)
	if err != nil {
		return scopeModel{}, err
	}

	m := scopeFromAPI(s)
	claimVals := make([]attr.Value, 0, len(claims))
	for _, c := range claims {
		claimVals = append(claimVals, types.StringValue(c.Type))
	}
	m.UserClaims = types.SetValueMust(types.StringType, claimVals)
	propVals := make(map[string]attr.Value, len(props))
	for _, p := range props {
		propVals[p.Key] = types.StringValue(p.Value)
	}
	m.Properties = types.MapValueMust(types.StringType, propVals)
	return m, nil
}

func scopeFromAPI(s *client.Scope) scopeModel {
	return scopeModel{
		ID:                      types.StringValue(strconv.Itoa(s.ID)),
		Name:                    types.StringValue(s.Name),
		DisplayName:             nullableString(s.DisplayName),
		Description:             nullableString(s.Description),
		Enabled:                 types.BoolValue(s.Enabled),
		Emphasize:               types.BoolValue(s.Emphasize),
		Required:                types.BoolValue(s.Required),
		ShowInDiscoveryDocument: types.BoolValue(s.ShowInDiscoveryDocument),
	}
}

func optionalString(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

// nullableString maps the API's empty/absent string to a null Terraform value.
func nullableString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
