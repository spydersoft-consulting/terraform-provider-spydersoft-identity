package provider

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/spydersoft-consulting/terraform-provider-spydersoft-identity/internal/client"
)

// ---- Schema attribute builders ----

func idAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Computed:      true,
		Description:   "Numeric identifier assigned by the server.",
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
}

func boolWithDefault(desc string, def bool) schema.BoolAttribute {
	return schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(def), Description: desc}
}

// serverBool is an optional bool the server defaults when unset; the server's
// value is kept in state.
func serverBool(desc string) schema.BoolAttribute {
	return schema.BoolAttribute{
		Optional: true, Computed: true, Description: desc,
		PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
	}
}

func int64WithDefault(desc string, def int64) schema.Int64Attribute {
	return schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(def), Description: desc}
}

func serverInt64(desc string) schema.Int64Attribute {
	return schema.Int64Attribute{
		Optional: true, Computed: true, Description: desc,
		PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
	}
}

func stringWithDefault(desc, def string) schema.StringAttribute {
	return schema.StringAttribute{Optional: true, Computed: true, Default: stringdefault.StaticString(def), Description: desc}
}

// serverString is an optional string the server may default; the server's value
// is kept in state when the attribute is not configured.
func serverString(desc string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional: true, Computed: true, Description: desc,
		PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
}

func stringSetAttribute(desc string) schema.SetAttribute {
	return schema.SetAttribute{
		ElementType: types.StringType, Optional: true, Computed: true, Description: desc,
		Default: setdefault.StaticValue(types.SetValueMust(types.StringType, []attr.Value{})),
	}
}

func stringMapAttribute(desc string) schema.MapAttribute {
	return schema.MapAttribute{
		ElementType: types.StringType, Optional: true, Computed: true, Description: desc,
		Default: mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
	}
}

// ---- Value conversion ----

func parseID(v types.String) (int, error) {
	id, err := strconv.Atoi(v.ValueString())
	if err != nil {
		return 0, fmt.Errorf("invalid id %q: %w", v.ValueString(), err)
	}
	return id, nil
}

func optionalString(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

func optionalBool(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

func optionalInt64(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	i := v.ValueInt64()
	return &i
}

// nullableString maps the API's empty/absent string to a null Terraform value.
func nullableString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func nullableInt64(i *int64) types.Int64 {
	if i == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*i)
}

// ---- Value-keyed sub-resource sync/load ----

func setToSlice(ctx context.Context, s types.Set) ([]string, error) {
	var out []string
	if diags := s.ElementsAs(ctx, &out, false); diags.HasError() {
		return nil, fmt.Errorf("reading set: %v", diags)
	}
	return out, nil
}

func sliceToSet(vals []string) types.Set {
	elems := make([]attr.Value, 0, len(vals))
	for _, v := range vals {
		elems = append(elems, types.StringValue(v))
	}
	return types.SetValueMust(types.StringType, elems)
}

// syncStringSet reconciles a sub-resource collection whose items are keyed by a
// single string field (for example claims by "type").
func syncStringSet(ctx context.Context, c *client.Client, base, field string, s types.Set) error {
	vals, err := setToSlice(ctx, s)
	if err != nil {
		return err
	}
	desired := make([]map[string]string, 0, len(vals))
	for _, v := range vals {
		desired = append(desired, map[string]string{field: v})
	}
	return c.SyncChildren(ctx, base, desired)
}

func loadStringSet(ctx context.Context, c *client.Client, base, field string) (types.Set, error) {
	items, err := c.ListChildren(ctx, base)
	if err != nil {
		return types.Set{}, err
	}
	vals := make([]string, 0, len(items))
	for _, it := range items {
		if v, ok := it[field].(string); ok {
			vals = append(vals, v)
		}
	}
	sort.Strings(vals)
	return sliceToSet(vals), nil
}

func syncProperties(ctx context.Context, c *client.Client, base string, m types.Map) error {
	props := map[string]string{}
	if diags := m.ElementsAs(ctx, &props, false); diags.HasError() {
		return fmt.Errorf("reading properties: %v", diags)
	}
	desired := make([]map[string]string, 0, len(props))
	for k, v := range props {
		desired = append(desired, map[string]string{"key": k, "value": v})
	}
	return c.SyncChildren(ctx, base, desired)
}

func loadProperties(ctx context.Context, c *client.Client, base string) (types.Map, error) {
	items, err := c.ListChildren(ctx, base)
	if err != nil {
		return types.Map{}, err
	}
	vals := make(map[string]attr.Value, len(items))
	for _, it := range items {
		k, _ := it["key"].(string)
		v, _ := it["value"].(string)
		vals[k] = types.StringValue(v)
	}
	return types.MapValueMust(types.StringType, vals), nil
}
