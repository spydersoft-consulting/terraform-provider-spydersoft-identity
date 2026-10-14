package provider

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/spydersoft-consulting/terraform-provider-spydersoft-identity/internal/client"
)

var (
	_ resource.Resource                = &roleResource{}
	_ resource.ResourceWithConfigure   = &roleResource{}
	_ resource.ResourceWithImportState = &roleResource{}
)

// NewRoleResource returns the spydersoftidentity_role resource.
func NewRoleResource() resource.Resource { return &roleResource{} }

type roleResource struct {
	client *client.Client
}

type roleModel struct {
	ID     types.String `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Claims types.Set    `tfsdk:"claims"`
}

func (r *roleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (r *roleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "A role that users can be assigned to, with the claims every member receives. Import with the role name. Assigning users to roles is not managed by this resource.",
		Attributes: map[string]schema.Attribute{
			"id":   idAttribute(),
			"name": schema.StringAttribute{Required: true, Description: "Role name. Unique, ignoring case. Renaming keeps the role id and its members."},
			"claims": schema.SetNestedAttribute{
				Optional: true, Computed: true,
				Description: "Claims granted to every member of the role.",
				Default:     setdefault.StaticValue(types.SetValueMust(claimObjectType, []attr.Value{})),
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{
						Required: true, Description: "Claim type. Must not contain \"/\".",
						Validators: []validator.String{claimTypeValidator{}},
					},
					"value": schema.StringAttribute{
						Optional: true, Computed: true, Default: stringdefault.StaticString(""),
						Description: "Claim value. Defaults to an empty string.",
					},
				}},
			},
		},
	}
}

func (r *roleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *roleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var created client.Role
	if err := r.client.Post(ctx, "/api/v1/Roles", client.SaveRole{Name: plan.Name.ValueString()}, &created); err != nil {
		resp.Diagnostics.AddError("Creating role", err.Error())
		return
	}
	if err := r.applyClaims(ctx, created.ID, plan); err != nil {
		resp.Diagnostics.AddError("Setting role claims", err.Error())
		return
	}
	got, err := r.load(ctx, created.ID)
	if err != nil {
		resp.Diagnostics.AddError("Reading created role", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, got)...)
}

func (r *roleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state roleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	got, err := r.load(ctx, state.ID.ValueString())
	if errors.Is(err, client.ErrNotFound) {
		resp.State.RemoveResource(ctx)
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Reading role", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, got)...)
}

func (r *roleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state roleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id := state.ID.ValueString()
	if err := r.client.Put(ctx, "/api/v1/Roles/"+url.PathEscape(id), client.SaveRole{Name: plan.Name.ValueString()}); err != nil {
		resp.Diagnostics.AddError("Updating role", err.Error())
		return
	}
	if err := r.applyClaims(ctx, id, plan); err != nil {
		resp.Diagnostics.AddError("Setting role claims", err.Error())
		return
	}
	got, err := r.load(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError("Reading updated role", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, got)...)
}

func (r *roleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := r.client.Delete(ctx, "/api/v1/Roles/"+url.PathEscape(state.ID.ValueString()))
	if err != nil && !errors.Is(err, client.ErrNotFound) {
		resp.Diagnostics.AddError("Deleting role", err.Error())
	}
}

// ImportState imports by role name; the API has no by-name endpoint, so the
// role list is searched.
func (r *roleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	role, err := r.client.FindRoleByName(ctx, req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Importing role", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), role.ID)...)
}

func (r *roleResource) applyClaims(ctx context.Context, id string, plan roleModel) error {
	var claims []claimModel
	if diags := plan.Claims.ElementsAs(ctx, &claims, false); diags.HasError() {
		return fmt.Errorf("reading claims: %v", diags)
	}
	desired := make([]client.RoleClaim, 0, len(claims))
	for _, c := range claims {
		desired = append(desired, client.RoleClaim{Type: c.Type.ValueString(), Value: c.Value.ValueString()})
	}
	return r.client.SyncRoleClaims(ctx, id, desired)
}

func (r *roleResource) load(ctx context.Context, id string) (roleModel, error) {
	var role client.Role
	if err := r.client.Get(ctx, "/api/v1/Roles/"+url.PathEscape(id), &role); err != nil {
		return roleModel{}, err
	}
	have, err := r.client.ListRoleClaims(ctx, id)
	if err != nil {
		return roleModel{}, err
	}

	claims := make([]claimModel, 0, len(have))
	for _, c := range have {
		claims = append(claims, claimModel{Type: types.StringValue(c.Type), Value: types.StringValue(c.Value)})
	}
	sort.Slice(claims, func(i, j int) bool {
		if claims[i].Type.ValueString() != claims[j].Type.ValueString() {
			return claims[i].Type.ValueString() < claims[j].Type.ValueString()
		}
		return claims[i].Value.ValueString() < claims[j].Value.ValueString()
	})
	claimSet, diags := types.SetValueFrom(ctx, claimObjectType, claims)
	if diags.HasError() {
		return roleModel{}, fmt.Errorf("building claims: %v", diags)
	}

	return roleModel{ID: types.StringValue(role.ID), Name: types.StringValue(role.Name), Claims: claimSet}, nil
}

// claimTypeValidator rejects role claim types containing "/". The admin API
// removes a role claim through a URL path segment ("DELETE .../claims/{type}"),
// and an encoded slash in a segment makes the server answer 404, so such a
// claim could be created but never removed again.
type claimTypeValidator struct{}

var _ validator.String = claimTypeValidator{}

func (claimTypeValidator) Description(context.Context) string {
	return "must not contain \"/\" (the admin API cannot delete such a claim)"
}

func (v claimTypeValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (claimTypeValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if strings.Contains(req.ConfigValue.ValueString(), "/") {
		resp.Diagnostics.AddAttributeError(req.Path, "Unsupported claim type",
			fmt.Sprintf("Role claim type %q contains \"/\". The identity server's admin API cannot remove such a claim, so it could never be updated or deleted. Use a type without slashes (for example a urn: style name).", req.ConfigValue.ValueString()))
	}
}
