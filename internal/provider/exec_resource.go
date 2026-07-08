package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// execResource runs raw Nodegrid CLI command batches for configuration that
// is not expressible as key=value settings paths: firewall rules, DHCP
// scopes, NAT chains, bonded interfaces — anything built with add/delete.
// It complements nodegrid_settings, which should be preferred wherever the
// config is a plain settings path (settings get drift detection; exec does
// not).
type execResource struct {
	cfg ProviderConfig
}

func NewExecResource() resource.Resource {
	return &execResource{}
}

type execModel struct {
	Host            types.String `tfsdk:"host"`
	Commands        types.List   `tfsdk:"commands"`
	DestroyCommands types.List   `tfsdk:"destroy_commands"`
	Output          types.String `tfsdk:"output"`
}

func (r *execResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_exec"
}

func (r *execResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Runs a batch of Nodegrid CLI commands over SSH in one session. " +
			"Use for configuration the settings tree cannot express as key=value pairs " +
			"(firewall rules, DHCP scopes, NAT chains, bonding). Commands re-run whenever " +
			"they change (the resource is replaced). No drift detection — prefer " +
			"nodegrid_settings wherever possible. Include your own commit commands.",
		Attributes: map[string]schema.Attribute{
			"host": schema.StringAttribute{
				Required:    true,
				Description: "Device IP or hostname to SSH into.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"commands": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "CLI commands executed in order in a single session. Changing them re-runs the batch.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.RequiresReplace(),
				},
			},
			"destroy_commands": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Optional CLI commands run on destroy (e.g. delete the rules the create batch added). If omitted, destroy leaves the device untouched.",
			},
			"output": schema.StringAttribute{
				Computed:    true,
				Description: "Session transcript from the last run, for debugging.",
			},
		},
	}
}

func (r *execResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *execResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan execModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	out, err := r.runList(ctx, plan.Host.ValueString(), plan.Commands, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to run commands", err.Error())
		return
	}

	plan.Output = types.StringValue(out)
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *execResource) Read(_ context.Context, _ resource.ReadRequest, _ *resource.ReadResponse) {
	// Raw command batches have no readable remote representation.
}

func (r *execResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Only destroy_commands can change without replacement; nothing runs.
	var plan execModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state execModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if !resp.Diagnostics.HasError() {
		plan.Output = state.Output
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *execResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state execModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if state.DestroyCommands.IsNull() || len(state.DestroyCommands.Elements()) == 0 {
		return
	}

	_, err := r.runList(ctx, state.Host.ValueString(), state.DestroyCommands, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	if err != nil {
		resp.Diagnostics.AddError("Failed to run destroy_commands", err.Error())
	}
}

func (r *execResource) runList(ctx context.Context, host string, list types.List, diags *diag.Diagnostics) (string, error) {
	var cmds []string
	if d := list.ElementsAs(ctx, &cmds, false); d.HasError() {
		diags.Append(d...)
		return "", nil
	}
	return r.cfg.ClientFor(host).RunChecked(cmds)
}
