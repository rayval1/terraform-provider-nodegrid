package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/rayval1/terraform-provider-nodegrid/internal/client"
)

// settingsResource manages an arbitrary set of Nodegrid settings-tree values
// on one device. It is deliberately generic: hostname, DNS, NTP, system
// preferences and TTY labels are all just paths in the same tree, so one
// resource type covers everything the old null_resource heredocs pushed.
type settingsResource struct {
	cfg ProviderConfig
}

func NewSettingsResource() resource.Resource {
	return &settingsResource{}
}

type settingsModel struct {
	Host     types.String `tfsdk:"host"`
	Settings types.Map    `tfsdk:"settings"`
}

func (r *settingsResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_settings"
}

func (r *settingsResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Declaratively manages Nodegrid CLI settings (full /settings/... paths) on a single device. " +
			"Reads real device state with export_settings, so drift shows up in terraform plan. " +
			"Removing a key from the map stops managing it but does not unset it on the device; " +
			"destroying the resource leaves device settings in place.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Required:    true,
				Description: "Device IP or hostname to SSH into.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"settings": schema.MapAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "Map of full setting path (e.g. /settings/network_settings/global_dns_servers) to desired value.",
			},
		},
	}
}

func (r *settingsResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cfg, ok := req.ProviderData.(ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	r.cfg = cfg
}

func (r *settingsResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan settingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	settings, diagErr := mapFromModel(ctx, plan)
	if diagErr != "" {
		resp.Diagnostics.AddError("Invalid settings map", diagErr)
		return
	}

	if err := r.cfg.ClientFor(plan.Host.ValueString()).ApplySettings(settings); err != nil {
		resp.Diagnostics.AddError("Failed to apply settings", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *settingsResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state settingsModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	managed, diagErr := mapFromModel(ctx, state)
	if diagErr != "" {
		resp.Diagnostics.AddError("Invalid settings map in state", diagErr)
		return
	}
	if len(managed) == 0 {
		return
	}

	device, err := r.cfg.ClientFor(state.Host.ValueString()).GetSettings(sectionsOf(managed))
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Failed to read settings from %s", state.Host.ValueString()),
			err.Error(),
		)
		return
	}

	refreshed := make(map[string]string, len(managed))
	for path, stateValue := range managed {
		if deviceValue, ok := device[path]; ok {
			refreshed[path] = deviceValue
		} else {
			// Some fields (secrets, defaults) never appear in export output;
			// keep the last known value rather than reporting false drift.
			refreshed[path] = stateValue
		}
	}

	state.Settings = mapToValue(ctx, refreshed, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, state)...)
}

func (r *settingsResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state settingsModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	planned, diagErr := mapFromModel(ctx, plan)
	if diagErr != "" {
		resp.Diagnostics.AddError("Invalid settings map", diagErr)
		return
	}
	current, _ := mapFromModel(ctx, state)

	changed := map[string]string{}
	for path, value := range planned {
		if old, ok := current[path]; !ok || old != value {
			changed[path] = value
		}
	}

	if len(changed) > 0 {
		if err := r.cfg.ClientFor(plan.Host.ValueString()).ApplySettings(changed); err != nil {
			resp.Diagnostics.AddError("Failed to apply settings", err.Error())
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *settingsResource) Delete(_ context.Context, _ resource.DeleteRequest, _ *resource.DeleteResponse) {
	// Intentionally a no-op on the device: there is no generic "unset" in the
	// Nodegrid settings tree, and reverting console-server config on destroy
	// would be more dangerous than leaving it in place.
}

func mapFromModel(ctx context.Context, m settingsModel) (map[string]string, string) {
	out := map[string]string{}
	if m.Settings.IsNull() || m.Settings.IsUnknown() {
		return out, ""
	}
	elements := map[string]types.String{}
	if diags := m.Settings.ElementsAs(ctx, &elements, false); diags.HasError() {
		return nil, fmt.Sprintf("%v", diags.Errors())
	}
	for path, v := range elements {
		if _, _, err := client.SplitPath(path); err != nil {
			return nil, err.Error()
		}
		out[path] = v.ValueString()
	}
	return out, ""
}

func sectionsOf(settings map[string]string) []string {
	seen := map[string]bool{}
	for path := range settings {
		section, _, err := client.SplitPath(path)
		if err == nil {
			seen[section] = true
		}
	}
	sections := make([]string, 0, len(seen))
	for s := range seen {
		sections = append(sections, s)
	}
	sort.Strings(sections)
	return sections
}
