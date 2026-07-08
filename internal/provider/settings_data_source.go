package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// settingsDataSource exports a subtree of a device's settings — handy for
// inspecting live state (`terraform console`) or feeding values elsewhere.
type settingsDataSource struct {
	cfg ProviderConfig
}

func NewSettingsDataSource() datasource.DataSource {
	return &settingsDataSource{}
}

type settingsDataModel struct {
	Host     types.String `tfsdk:"host"`
	Path     types.String `tfsdk:"path"`
	Settings types.Map    `tfsdk:"settings"`
}

func (d *settingsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_settings"
}

func (d *settingsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads a Nodegrid settings subtree from a device via export_settings.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Required:    true,
				Description: "Device IP or hostname to SSH into.",
			},
			"path": schema.StringAttribute{
				Required:    true,
				Description: "Settings tree prefix to export, e.g. /settings/network_settings.",
			},
			"settings": schema.MapAttribute{
				Computed:    true,
				ElementType: types.StringType,
				Description: "Full setting path to value, as currently configured on the device.",
			},
		},
	}
}

func (d *settingsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	cfg, ok := req.ProviderData.(ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	d.cfg = cfg
}

func (d *settingsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var model settingsDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	device, err := d.cfg.ClientFor(model.Host.ValueString()).GetSettings([]string{model.Path.ValueString()})
	if err != nil {
		resp.Diagnostics.AddError(
			fmt.Sprintf("Failed to read settings from %s", model.Host.ValueString()),
			err.Error(),
		)
		return
	}

	model.Settings = mapToValue(ctx, device, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

// mapToValue converts a plain string map into a framework Map value.
func mapToValue(ctx context.Context, m map[string]string, diags *diag.Diagnostics) types.Map {
	value, d := types.MapValueFrom(ctx, types.StringType, m)
	diags.Append(d...)
	return value
}
